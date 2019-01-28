package main

import (
	"database/sql"

	"github.com/nats-io/go-nats"
)

func startIndexer(nc *nats.Conn, db *sql.DB, cfg config) {
	nc.Subscribe(cfg.Nats.Topic+".index.update", func(m *nats.Msg) {
		// for each updated document:
		//	clear the corresponding interest list entry
		//	update the index table with the document
	})

	go func() {
		// for ever:
		// 	while interest list is not empty, ask for documents and wait
		//	get a new interest list
	}()

}
