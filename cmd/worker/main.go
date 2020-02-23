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

package main

/*
	Letarette main application, the "worker".
	Communicates via "NATS" message bus, maintains the index and responds to queries.
*/

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		letarette.Usage()
		fmt.Printf("Stemmer languages: %v\n", strings.Join(snowball.ListStemmers(), ", "))
		os.Exit(99)
	}

	cfg, err := letarette.LoadConfig()
	if err != nil {
		logger.Error.Printf("Failed to load config: %v", err)
		letarette.Usage()
		os.Exit(1)
	}

	logger.Info.Printf("Starting Letarette %s (%s)", letarette.Tag, letarette.Revision)

	profiler, err := letarette.StartProfiler(cfg)
	if err != nil {
		logger.Error.Printf("Failed to start profiler: %v", err)
		os.Exit(1)
	}
	defer profiler.Close()

	logger.Info.Printf("Connecting to nats server at %q\n", cleanURLs(cfg.Nats.URLS))

	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
	}

	if cfg.Nats.SeedFile != "" {
		option, err := nats.NkeyOptionFromSeed(cfg.Nats.SeedFile)
		if err != nil {
			logger.Error.Printf("Failed to load nats seed file: %v", err)
			os.Exit(1)
		}
		options = append(options, option)
	}

	if len(cfg.Nats.RootCAs) > 0 {
		options = append(options, nats.RootCAs(cfg.Nats.RootCAs...))
	}
	URLS := strings.Join(cfg.Nats.URLS, ",")
	conn, err := nats.Connect(URLS, options...)
	if err != nil {
		logger.Error.Printf("Failed to connect to nats server: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to connect to DB: %v", err)
		os.Exit(1)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error.Printf("Failed to close DB: %v", err)
		}
	}()

	err = letarette.CheckStemmerSettings(db, cfg)
	if err == letarette.ErrStemmerSettingsMismatch {
		logger.Error.Printf("Index and config stemmer settings mismatch. Re-build index or force changes.")
		os.Exit(1)
	}
	if err != nil {
		logger.Error.Printf("Failed to check stemmer config: %w", err)
		os.Exit(1)
	}

	monitor, err := letarette.StartStatusMonitor(conn, db, cfg)
	if err != nil {
		logger.Error.Printf("Failed to start status monitor: %v", err)
		os.Exit(1)
	}

	metrics, err := letarette.StartMetricsCollector(conn, db, cfg)
	if err != nil {
		logger.Error.Printf("Failed to start metrics collector: %v", err)
		os.Exit(1)
	}

	var indexer letarette.Indexer
	if !cfg.Index.Disable {
		indexer, err = letarette.StartIndexer(conn, db, cfg)
		if err != nil {
			logger.Error.Printf("Failed to start indexer: %v", err)
			os.Exit(1)
		}
	}

	var searcher letarette.Searcher
	if !cfg.Search.Disable {
		searcher, err = letarette.StartSearcher(conn, db, cfg)
		if err != nil {
			logger.Error.Printf("Failed to start searcher: %v", err)
			os.Exit(1)
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	select {
	case s := <-signals:
		logger.Info.Printf("Received signal %v\n", s)
		if metrics != nil {
			metrics.Close()
		}
		if monitor != nil {
			monitor.Close()
		}
		if searcher != nil {
			searcher.Close()
		}
		if indexer != nil {
			indexer.Close()
		}
	}
}

func cleanURLs(URLs []string) []string {
	result := make([]string, len(URLs))

	for i, URL := range URLs {
		parsed, err := url.Parse(URL)
		if err != nil {
			result[i] = URL
		} else {
			parsed.User = nil
			result[i] = parsed.String()
		}
	}
	return result
}
