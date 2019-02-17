package main

/*
	Letarette main application, the "worker".
	Communicates via "nats" message bus, maintains an index and responds to queries.
*/

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/erkkah/letarette/internal/letarette"

	"github.com/nats-io/go-nats"
)

func main() {
	conf := flag.String("conf", "letarette.toml", "Configuration TOML file")
	flag.Parse()

	cfg, err := letarette.LoadConfig(*conf)
	if err != nil {
		log.Panic("Failed to load config:", err)
	}

	log.Printf("Connecting to nats server at %q\n", cfg.Nats.URL)
	conn, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		log.Panicf("Failed to connect to nats server")
	}
	defer conn.Close()

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	indexer := letarette.StartIndexer(conn, db, cfg)
	searcher := letarette.StartSearcher(conn, db, cfg)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	signal.Reset(syscall.SIGHUP)
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	go func() {
		for range sighup {
			log.Println("not reloading config...")
		}
	}()

	select {
	case s := <-signals:
		log.Printf("Received signal %v\n", s)
		indexer.Close()
		searcher.Close()
	}
}
