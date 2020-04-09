// Copyright 2020 Erik Agsjö
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

import (
	"fmt"
	"os"
	"time"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

/*
	Letarette web-based monitor.
*/

func main() {
	if len(os.Args) > 1 {
		Usage()
		os.Exit(0)
	}

	cfg, err := LoadConfig()
	if err != nil {
		logger.Error.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	monitor, err := client.NewMonitor(
		cfg.Nats.URLS, listener, client.WithMetricsCollector(collector, time.Second),
		client.WithTopic(cfg.Nats.Topic),
		client.WithRootCAs(cfg.Nats.RootCAs...),
		client.WithSeedFile(cfg.Nats.SeedFile),
	)
	if err != nil {
		logger.Error.Printf("Failed to create monitor: %v", err)
		return
	}
	defer monitor.Close()

	err = startSearchClient(
		cfg.Nats.URLS,
		cfg.Nats.SeedFile,
		cfg.Nats.RootCAs,
	)
	if err != nil {
		logger.Error.Printf("Failed to start search client: %v", err)
		return
	}

	server := &server{
		lookupTemplate: lookupTemplate,
		serveRaw:       serveRaw,
	}
	server.run(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
}

func listener(status protocol.IndexStatus) {
	handleStatusUpdate(status)
}

func collector(metrics protocol.Metrics) {
	handleMetricsUpdate(metrics)
}
