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
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	lr "github.com/erkkah/letarette"
	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/internal/snowball"
	"github.com/erkkah/letarette/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		letarette.Usage(false)
		fmt.Printf("Stemmer languages: %v\n", strings.Join(snowball.ListStemmers(), ", "))
		os.Exit(99)
	}

	cfg, err := letarette.LoadConfig()
	if err != nil {
		logger.Error.Printf("Failed to load config: %v", err)
		letarette.Usage(false)
		os.Exit(1)
	}

	logger.Info.Printf("Starting Letarette %s", lr.Version())

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

	// Start trapping signals
	var done sync.WaitGroup
	done.Add(1)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		s := <-signals
		logger.Info.Printf("Received signal %v, initiating graceful shutdown...", s)
		done.Done()
	}()

	db, err := letarette.OpenDatabase(cfg)
	if err != nil {
		logger.Error.Printf("Failed to connect to DB: %v", err)
		os.Exit(1)
	}

	closeDB := func() {
		err := db.Close()
		if err != nil {
			logger.Error.Printf("Failed to close DB: %v", err)
		}
	}

	defer closeDB()

	die := func(msg string, args ...interface{}) {
		closeDB()
		logger.Error.Printf(msg, args...)
		os.Exit(1)
	}

	err = letarette.CheckStemmerSettings(db, cfg)
	if err == letarette.ErrStemmerSettingsMismatch {
		die("Index and config stemmer settings mismatch. Re-build index or force changes.")
	}
	if err != nil {
		die("Failed to check stemmer config: %w", err)
	}

	monitor, err := letarette.StartStatusMonitor(conn, db, cfg)
	if err != nil {
		die("Failed to start status monitor: %v", err)
	}

	metrics, err := letarette.StartMetricsCollector(conn, db, cfg)
	if err != nil {
		die("Failed to start metrics collector: %v", err)
	}

	err = letarette.InitializeShard(conn, db, cfg, monitor)
	if err != nil {
		die("Failed to initialize shard: %v", err)
	}

	maxSize := cfg.Search.CacheMaxsizeMB * 1000 * 1000
	cache := letarette.NewCache(cfg.Search.CacheTimeout, maxSize)

	var indexer letarette.Indexer
	if !cfg.Index.Disable {
		indexer, err = letarette.StartIndexer(conn, db, cfg, cache)
		if err != nil {
			die("Failed to start indexer: %v", err)
		}
	}

	var searcher letarette.Searcher
	if !cfg.Search.Disable {
		searcher, err = letarette.StartSearcher(conn, db, cfg, cache)
		if err != nil {
			die("Failed to start searcher: %v", err)
		}
	}

	cloner, err := letarette.StartCloner(conn, db, cfg)
	if err != nil {
		die("Failed to start cloner: %v", err)
	}

	done.Wait()

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
	if cloner != nil {
		_ = cloner.Close()
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
