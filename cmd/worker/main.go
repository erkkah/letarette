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
		log.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	log.Printf("Connecting to nats server at %q\n", cfg.Nats.URL)
	conn, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		log.Printf("Failed to connect to nats server")
		os.Exit(2)
	}
	defer conn.Close()

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		log.Printf("Failed to connect to DB: %v", err)
		os.Exit(3)
	}
	defer db.Close()

	indexer, err := letarette.StartIndexer(conn, db, cfg)
	if err != nil {
		log.Printf("Failed to start indexer: %v", err)
		os.Exit(4)
	}
	searcher := letarette.StartSearcher(conn, db, cfg)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

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
