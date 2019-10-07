package letarette

import (
	"context"
	"strings"

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
	q := escapeQuotes(query.Query)
	result, err := s.db.search(ctx, q, query.Spaces, query.Limit)
	return protocol.SearchResponse{Documents: result}, err
}

// StartSearcher creates and starts a searcher instance.
func StartSearcher(nc *nats.Conn, db Database, cfg Config) (Searcher, error) {
	closer := make(chan bool, 0)

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return &searcher{}, err
	}

	self := &searcher{
		closer,
		cfg,
		ec,
		db,
	}

	ec.Subscribe(cfg.Nats.Topic+".q", func(sub, reply string, query *protocol.SearchRequest) {
		// Handle query
		response, err := self.parseAndExecute(context.Background(), *query)
		if err != nil {
			logger.Error.Printf("Failed to execute query: %v", err)
		}
		// Reply
		ec.Publish(reply, response)
	})

	go func() {
		// for ever:
		logger.Info.Printf("Searcher starting")
		<-closer
		logger.Info.Printf("Searcher exiting")
		closer <- true
	}()

	return self, nil
}
