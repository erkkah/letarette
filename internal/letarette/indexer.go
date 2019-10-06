package letarette

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/go-nats"

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
		err := db.clearInterestList(space)
		if err != nil {
			return nil, fmt.Errorf("Failed to clear interest list: %w", err)
		}
	}

	ec, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	closer := make(chan bool, 1)
	self := &indexer{
		closer: closer,
		cfg:    cfg,
		conn:   ec,
		db:     db,
	}

	ec.Subscribe(cfg.Nats.Topic+".document.update", func(update *protocol.DocumentUpdate) {
		for _, doc := range update.Documents {
			err := db.addDocumentUpdate(doc)
			if err != nil {
				log.Printf("Failed to add document update: %v", err)
			}
		}
	})

	go func() {
		log.Println("Indexer starting")

		mainContext, cancel := context.WithCancel(context.Background())
		var lastDocumentRequest time.Time
		for {
			for _, space := range cfg.Index.Spaces {
				interests, err := db.getInterestList(space)
				if err != nil {
					log.Printf("Failed to fetch current interest list: %v", err)
				} else {

					numPending := 0
					numRequested := 0
					numServed := 0
					pendingIDs := []protocol.DocumentID{}
					const maxOutstanding = 10

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
						log.Printf("Requesting %v docs\n", docsToRequest)
						err = self.requestDocuments(space, pendingIDs[:docsToRequest])
						if err != nil {
							log.Printf("Failed to request documents: %v", err)
						} else {
							lastDocumentRequest = time.Now()
							numRequested += docsToRequest
						}
					}

					allServed := numPending == 0 && numRequested == 0

					if allServed {

						err = self.commitFetched(space)
						if err != nil {
							log.Printf("Failed to commit docs: %v", err)
							continue
						}

						err = self.requestNextChunk(mainContext, space)
						if err != nil {
							log.Printf("Failed to request next chunk: %v", err)
							continue
						}

					} else {
						timeout := cfg.Index.MaxDocumentWait
						if time.Now().After(lastDocumentRequest.Add(timeout)) {
							log.Printf("Timeout waiting for documents, re-requesting")
							err = db.resetRequested(space)
							if err != nil {
								log.Printf("Failed to reset interest list state: %v", err)
							}
						}
					}
				}
			}

			select {
			case <-closer:
				log.Println("Indexer exiting")
				cancel()
				closer <- true
				return
			case <-time.After(cfg.Index.CycleWait):
			}
		}

	}()

	return self, nil
}

type indexer struct {
	closer chan bool
	cfg    Config
	conn   *nats.EncodedConn
	db     Database
}

func (idx *indexer) Close() {
	assert(idx.closer != nil, "Indexer close channel is not nil")
	idx.closer <- true
	<-idx.closer
}

func (idx *indexer) commitFetched(space string) error {
	return idx.db.commitInterestList(space)
}

func (idx *indexer) requestNextChunk(ctx context.Context, space string) error {
	topic := idx.cfg.Nats.Topic + ".index.request"
	state, err := idx.db.getInterestListState(space)
	if err != nil {
		return err
	}
	updateRequest := protocol.IndexUpdateRequest{
		Space:         space,
		StartTime:     state.lastUpdatedTime(),
		StartDocument: state.LastUpdatedDocID,
		Limit:         idx.cfg.Index.ChunkSize,
	}
	timeout, cancel := context.WithTimeout(ctx, idx.cfg.Index.MaxInterestWait)

	var update protocol.IndexUpdate
	err = idx.conn.RequestWithContext(timeout, topic, updateRequest, &update)
	cancel()

	if err != nil {
		return err
	}

	if len(update.Updates) > 0 {
		log.Printf("Received interest list of %v docs\n", len(update.Updates))
	}
	err = idx.db.setInterestList(update.Space, update.Updates)

	return err
}

func (idx *indexer) requestDocuments(space string, wanted []protocol.DocumentID) error {
	topic := idx.cfg.Nats.Topic + ".document.request"
	request := protocol.DocumentRequest{
		Space:  space,
		Wanted: wanted,
	}
	for _, docID := range wanted {
		err := idx.db.setInterestState(space, docID, requested)
		if err != nil {
			return fmt.Errorf("Failed to update interest state: %w", err)
		}
	}
	err := idx.conn.Publish(topic, request)
	return err
}
