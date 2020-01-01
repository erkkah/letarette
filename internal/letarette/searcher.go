// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package letarette

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/nats-io/nats.go"

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
	db     *database
	cache  *Cache
}

func (s *searcher) Close() {
	assert(s.closer != nil, "Searcher close channel is not nil")
	s.closer <- true
	<-s.closer
}

const minPagesize = 1
const maxPagesize = 500

func (s *searcher) spellSearch(
	ctx context.Context, phrases []Phrase, query protocol.SearchRequest,
) (protocol.SearchResult, error) {
	result, err := s.db.search(ctx, phrases, query.Spaces, query.PageLimit, query.PageOffset)
	if err != nil || result.TotalHits != 0 {
		return result, err
	}
	phrases, distance, changed, err := s.db.fixPhraseSpelling(ctx, phrases)
	if err != nil || !changed {
		return result, err
	}
	terms := []string{}
	for _, phrase := range phrases {
		terms = append(terms, phrase.String())
	}
	result.Respelt = strings.Join(terms, " ")
	result.RespeltDistance = distance
	if !query.Autocorrect {
		return result, nil
	}
	result, err = s.db.search(ctx, phrases, query.Spaces, query.PageLimit, query.PageOffset)
	return result, err
}

func (s *searcher) parseAndExecute(ctx context.Context, query protocol.SearchRequest) (protocol.SearchResponse, error) {
	var err error
	var status protocol.SearchStatusCode

	start := time.Now()
	query.PageLimit = uint16(max(minPagesize, int(query.PageLimit)))
	query.PageLimit = uint16(min(maxPagesize, int(query.PageLimit)))
	phrases := ParseQuery(query.Query)
	phrases = ReducePhraseList(phrases)

	var result protocol.SearchResult

	if len(query.Spaces) > 0 && len(phrases) > 0 {
		cacheKey := fmt.Sprintf("%s", CanonicalizePhraseList(phrases))
		var cached bool
		result, cached = s.cache.Get(cacheKey, query.Spaces, query.PageLimit, query.PageOffset)

		if cached {
			status = protocol.SearchStatusCacheHit
		} else {
			result, err = s.spellSearch(ctx, phrases, query)
			if err == nil {
				status = protocol.SearchStatusIndexHit
				s.cache.Put(cacheKey, query.Spaces, query.PageLimit, query.PageOffset, result)
			}
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
	} else if len(result.Hits) == 0 {
		status = protocol.SearchStatusNoHit
	}

	response := protocol.SearchResponse{
		Result:   result,
		Status:   status,
		Duration: duration,
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

	maxSize := cfg.Search.CacheMaxsizeMB * 1000 * 1000
	cache := NewCache(cfg.Search.CacheTimeout, maxSize)
	self := &searcher{
		closer,
		cfg,
		ec,
		db.(*database),
		cache,
	}

	type searchWork struct {
		req   protocol.SearchRequest
		reply string
	}

	// ??? Worker pool = 4 * GOMAXPROCS
	// I/O vs CPU
	numWorkers := 4 * runtime.GOMAXPROCS(-1)
	workChannel := make(chan searchWork, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for work := range workChannel {
				// Handle query
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Search.Timeout)
				response, err := self.parseAndExecute(ctx, work.req)
				cancel()
				if err != nil {
					logger.Error.Printf("Failed to execute query: %v", err)
				}
				// Reply
				ec.Publish(work.reply, response)
			}
		}()
	}

	subscription, err := ec.QueueSubscribe(
		cfg.Nats.Topic+".q", cfg.Nats.SearchGroup,
		func(sub, reply string, query *protocol.SearchRequest) {
			workChannel <- searchWork{
				req:   *query,
				reply: reply,
			}
		})

	if err != nil {
		return nil, err
	}

	go func() {
		logger.Info.Printf("Searcher starting")
		<-closer
		close(workChannel)
		subscription.Unsubscribe()
		logger.Info.Printf("Searcher exiting")
		closer <- true
	}()

	return self, nil
}
