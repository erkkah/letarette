package letarette

import (
	"log"

	"github.com/nats-io/go-nats"
)

type Indexer interface {
	Close()
}

type indexer struct {
	closer chan bool
}

func (idx *indexer) Close() {
	assert(idx.closer != nil, "Indexer close channel is not nil")
	idx.closer <- true
	<-idx.closer
}

func StartIndexer(nc *nats.Conn, db Database, cfg Config) Indexer {
	ec, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	defer ec.Close()

	closer := make(chan bool, 1)

	ec.Subscribe(cfg.Nats.Topic+".index.update", func(update *DocumentUpdate) {
		// for each updated document:
		//	clear the corresponding interest list entry
		//	update the index table with the document
	})

	go func() {
		// for ever:
		// 	while interest list is not empty, ask for documents and wait
		//	get a new interest list
		log.Println("Indexer starting")
		<-closer
		log.Println("Indexer exiting")
		closer <- true
	}()

	return &indexer{closer}
}
