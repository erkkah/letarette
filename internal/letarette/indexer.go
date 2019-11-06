package letarette

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

// Indexer continuously runs the indexing process, until Close is called.
type Indexer interface {
	Close()
}

// StartIndexer creates and starts an indexer instance. This is really a singleton
// in that only one instance with the same database or config can be run at the
// same time.
func StartIndexer(nc *nats.Conn, db Database, cfg Config) (Indexer, error) {

	for _, space := range cfg.Index.Spaces {
		err := db.clearInterestList(context.Background(), space)
		if err != nil {
			return nil, fmt.Errorf("Failed to clear interest list: %w", err)
		}
	}

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	mainContext, cancel := context.WithCancel(context.Background())

	self := &indexer{
		context: mainContext,
		close:   cancel,
		cfg:     cfg,
		conn:    ec,
		db:      db,
	}

	updates := make(chan protocol.DocumentUpdate, 10)

	go func() {
		self.waiter.Add(1)
		for update := range updates {
			err := db.addDocumentUpdates(mainContext, update.Space, update.Documents)
			if err != nil {
				logger.Error.Printf("Failed to add document update: %v", err)
			}
		}
		self.waiter.Done()
	}()

	subscription, err := ec.Subscribe(cfg.Nats.Topic+".document.update", func(update *protocol.DocumentUpdate) {
		updates <- *update
	})
	if err != nil {
		return nil, err
	}

	go func() {
		logger.Info.Printf("Indexer starting")
		self.waiter.Add(1)

		var lastDocumentRequest time.Time
		for {
			cycleThrottle := time.After(cfg.Index.CycleWait)
			totalInterests := 0

			for _, space := range cfg.Index.Spaces {
				interests, err := db.getInterestList(mainContext, space)
				if err != nil {
					logger.Error.Printf("Failed to fetch current interest list: %v", err)
				} else {
					totalInterests += len(interests)

					numPending := 0
					numRequested := 0
					numServed := 0
					pendingIDs := []protocol.DocumentID{}
					maxOutstanding := int(cfg.Index.MaxOutstanding)

					for _, interest := range interests {
						switch interest.State {
						case served:
							numServed++
						case pending:
							numPending++
							pendingIDs = append(pendingIDs, interest.DocID)
						case requested:
							numRequested++
						}
					}

					docsToRequest := min(numPending, maxOutstanding-numRequested)
					if docsToRequest > 0 {
						logger.Info.Printf("Requesting %v docs\n", docsToRequest)
						metrics.docRequests.Add(float64(docsToRequest))
						err = self.requestDocuments(mainContext, space, pendingIDs[:docsToRequest])
						if err != nil {
							logger.Error.Printf("Failed to request documents: %v", err)
						} else {
							lastDocumentRequest = time.Now()
							numRequested += docsToRequest
						}
					}

					allServed := numPending == 0 && numRequested == 0

					if allServed {

						err = self.commitFetched(mainContext, space)
						if err != nil {
							logger.Error.Printf("Failed to commit docs: %v", err)
							continue
						}

						err = self.requestNextChunk(mainContext, space)
						if err != nil {
							logger.Error.Printf("Failed to request next chunk: %v", err)
							continue
						}

					} else {
						timeout := cfg.Index.MaxDocumentWait
						if timeout != 0 && time.Now().After(lastDocumentRequest.Add(timeout)) {
							logger.Warning.Printf("Timeout waiting for documents, re-requesting")
							err = db.resetRequested(mainContext, space)
							if err != nil {
								logger.Error.Printf("Failed to reset interest list state: %v", err)
							}
						}
					}
				}
			}

			if totalInterests == 0 {
				cycleThrottle = time.After(cfg.Index.EmptyCycleWait)
			}
			select {
			case <-mainContext.Done():
				logger.Info.Printf("Indexer exiting")
				err = subscription.Drain()
				if err != nil {
					logger.Error.Printf("Failed to drain document subscription: %v", err)
				} else {
					go func() {
						self.waiter.Add(1)
						for {
							messages, _, _ := subscription.Pending()
							if messages == 0 {
								break
							}
							time.Sleep(time.Millisecond * 20)
						}
						self.waiter.Done()
					}()
				}
				cancel()
				close(updates)
				self.waiter.Done()
				return
			case <-cycleThrottle:
				// Loop will never be faster than cfg.CycleWait
			}
		}

	}()

	return self, nil
}

type indexer struct {
	close   context.CancelFunc
	context context.Context
	waiter  sync.WaitGroup

	cfg  Config
	conn *nats.EncodedConn
	db   Database
}

func (idx *indexer) Close() {
	idx.close()
	idx.waiter.Wait()
}

func (idx *indexer) commitFetched(ctx context.Context, space string) error {
	return idx.db.commitInterestList(ctx, space)
}

func (idx *indexer) requestNextChunk(ctx context.Context, space string) error {
	topic := idx.cfg.Nats.Topic + ".index.request"
	state, err := idx.db.getInterestListState(ctx, space)
	if err != nil {
		return fmt.Errorf("Failed to get interest list state: %w", err)
	}
	updateRequest := protocol.IndexUpdateRequest{
		Space:         space,
		FromTime:      state.lastUpdatedTime(),
		AfterDocument: state.LastUpdatedDocID,
		Limit:         idx.cfg.Index.ChunkSize,
	}
	timeout, cancel := context.WithTimeout(ctx, idx.cfg.Index.MaxInterestWait)

	var update protocol.IndexUpdate
	err = idx.conn.RequestWithContext(timeout, topic, updateRequest, &update)
	cancel()

	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	if len(update.Updates) > 0 {
		logger.Info.Printf("Received interest list of %v docs\n", len(update.Updates))
	}
	err = idx.db.setInterestList(ctx, update.Space, update.Updates)

	if err != nil {
		return fmt.Errorf("Failed to set interest list: %w", err)
	}

	return nil
}

func (idx *indexer) requestDocuments(ctx context.Context, space string, wanted []protocol.DocumentID) error {
	topic := idx.cfg.Nats.Topic + ".document.request"
	request := protocol.DocumentRequest{
		Space:  space,
		Wanted: wanted,
	}
	for _, docID := range wanted {
		err := idx.db.setInterestState(ctx, space, docID, requested)
		if err != nil {
			return fmt.Errorf("Failed to update interest state: %w", err)
		}
	}
	err := idx.conn.Publish(topic, request)
	return err
}
