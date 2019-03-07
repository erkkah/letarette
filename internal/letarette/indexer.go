package letarette

import (
	"log"
	"time"

	"github.com/nats-io/go-nats"
)

type Indexer interface {
	Close()
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

func (idx *indexer) requestNextChunk(space string) error {

	topic := idx.cfg.Nats.Topic + ".index.request"
	state, err := idx.db.getInterestListState(space)
	if err != nil {
		return err
	}
	chunkStart := uint64(0)
	if state.UpdateStart == state.UpdateEnd {
		chunkStart = state.ChunkStart + uint64(state.ChunkSize)
		idx.db.setChunkStart(space, chunkStart)
	}
	updateRequest := IndexUpdateRequest{
		Space:       space,
		UpdateStart: state.UpdateEnd,
		Start:       chunkStart,
		Limit:       idx.cfg.Index.ChunkSize,
	}
	err = idx.conn.Publish(topic, updateRequest)
	return err
}

func StartIndexer(nc *nats.Conn, db Database, cfg Config) Indexer {
	ec, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	defer ec.Close()

	closer := make(chan bool, 1)
	self := &indexer{
		closer: closer,
		cfg:    cfg,
		conn:   ec,
		db:     db,
	}

	ec.Subscribe(cfg.Nats.Topic+".document.update", func(update *DocumentUpdate) {
		// for each updated document:
		//	mark the corresponding interest list entry as served
		//	update the index table with the document
	})

	ec.Subscribe(cfg.Nats.Topic+".index.update", func(update *IndexUpdate) {
		err := db.setInterestList(update.Space, update.Updates)
		if err != nil {
			log.Printf("Failed to set interest list: %v", err)
		}
	})

	go func() {
		// for ever:
		//  while interest list is not served
		//	 request documents and wait
		//   if no updates within given window, re-request
		//
		//	update last update time and chunk index
		//	get a new interest list
		log.Println("Indexer starting")
		for {
			for _, space := range cfg.Index.Spaces {
				interests, err := db.getInterestList(space)
				if err != nil {
					log.Printf("Failed to fetch current interest list: %v", err)
				} else {
					allServed := true
					for _, interest := range interests {
						if !interest.Served {
							allServed = false
							break
						}
					}
					if allServed {
						self.requestNextChunk(space)
					}
				}
			}

			select {
			case <-closer:
				log.Println("Indexer exiting")
				closer <- true
				return
			case <-time.After(10 * time.Second):
			}
		}

	}()

	return self
}
