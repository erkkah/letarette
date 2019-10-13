package main

/*
	Letarette main application, the "worker".
	Communicates via "NATS" message bus, maintains an index and responds to queries.
*/

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/go-nats"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		letarette.Usage()
		os.Exit(99)
	}

	cfg, err := letarette.LoadConfig()
	if err != nil {
		logger.Error.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	letarette.ExposeMetrics(cfg.MetricsPort)

	logger.Info.Printf("Connecting to nats server at %q\n", cfg.Nats.URL)
	conn, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		logger.Error.Printf("Failed to connect to nats server")
		os.Exit(2)
	}
	defer conn.Close()

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to connect to DB: %v", err)
		os.Exit(3)
	}
	defer db.Close()

	indexer, err := letarette.StartIndexer(conn, db, cfg)
	if err != nil {
		logger.Error.Printf("Failed to start indexer: %v", err)
		os.Exit(4)
	}
	searcher, err := letarette.StartSearcher(conn, db, cfg)
	if err != nil {
		logger.Error.Printf("Failed to start searcher: %v", err)
		os.Exit(5)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	go func() {
		for range sighup {
			logger.Debug.Printf("not reloading config...")
		}
	}()

	select {
	case s := <-signals:
		logger.Info.Printf("Received signal %v\n", s)
		indexer.Close()
		searcher.Close()
	}
}
