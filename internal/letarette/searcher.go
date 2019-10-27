package letarette

import (
	"context"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/nats-io/go-nats"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

// Searcher continuously runs the search process, until Close is called.
type Searcher interface {
	Close()
}

type searcher struct {
	closer chan bool
	cfg    Config
	conn   *nats.EncodedConn
	db     Database
	cache  *Cache
}

func (s *searcher) Close() {
	assert(s.closer != nil, "Searcher close channel is not nil")
	s.closer <- true
	<-s.closer
}

func escapeQuotes(q string) string {
	return strings.ReplaceAll(q, `"`, `""`)
}

func (s *searcher) parseAndExecute(ctx context.Context, query protocol.SearchRequest) (protocol.SearchResponse, error) {
	var err error
	var status protocol.SearchStatusCode

	q := escapeQuotes(query.Query)
	start := time.Now()
	result, cached := s.cache.Get(q, query.Spaces, query.Limit, query.Offset)

	if cached {
		status = protocol.SearchStatusCacheHit
	} else {
		result, err = s.db.search(ctx, q, query.Spaces, query.Limit, query.Offset)
		if err == nil {
			status = protocol.SearchStatusIndexHit
			s.cache.Put(q, query.Spaces, query.Limit, query.Offset, result)
		}
	}
	duration := float32(time.Since(start)) / float32(time.Second)

	if err != nil {
		if sqliteError, ok := err.(sqlite3.Error); ok && sqliteError.Code == sqlite3.ErrInterrupt {
			status = protocol.SearchStatusTimeout
		} else if err == context.DeadlineExceeded {
			status = protocol.SearchStatusTimeout
		} else {
			status = protocol.SearchStatusServerError
		}
	} else if len(result) == 0 {
		status = protocol.SearchStatusNoHit
	}

	response := protocol.SearchResponse{
		Documents: result,
		Status:    status,
		Duration:  duration,
	}
	return response, err
}

// StartSearcher creates and starts a searcher instance.
func StartSearcher(nc *nats.Conn, db Database, cfg Config) (Searcher, error) {
	closer := make(chan bool, 0)

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return &searcher{}, err
	}

	cache := NewCache(cfg.Search.CacheTimeout)
	self := &searcher{
		closer,
		cfg,
		ec,
		db,
		cache,
	}

	subscription, err := ec.QueueSubscribe(
		cfg.Nats.Topic+".q", cfg.Nats.SearchGroup,
		func(sub, reply string, query *protocol.SearchRequest) {
			// Handle query
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Search.Timeout)
			response, err := self.parseAndExecute(ctx, *query)
			cancel()
			if err != nil {
				logger.Error.Printf("Failed to execute query: %v", err)
			}
			// Reply
			ec.Publish(reply, response)
		})

	if err != nil {
		return nil, err
	}

	go func() {
		// for ever:
		logger.Info.Printf("Searcher starting")
		<-closer
		subscription.Unsubscribe()
		logger.Info.Printf("Searcher exiting")
		closer <- true
	}()

	return self, nil
}
