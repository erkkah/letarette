// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/erkkah/letarette/internal/letarette"
	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

type searchOptions struct {
	Space       string   `arg:"0"`
	Phrases     []string `args:"1"`
	Limit       int      `name:"l" default:"10"`
	Offset      int      `name:"p" default:"0"`
	GroupSize   int32    `name:"g"`
	Interactive bool     `name:"i"`
}

func doSearch(cfg letarette.Config, options searchOptions) {
	if len(options.Space) == 0 {
		fmt.Println("Expected <space> arg")
		return
	}
	fmt.Printf("Searching space %q\n", options.Space)
	a, err := client.NewSearchAgent(
		cfg.Nats.URLS,
		client.WithSeedFile(cfg.Nats.SeedFile),
		client.WithShardgroupSize(options.GroupSize),
		client.WithRootCAs(cfg.Nats.RootCAs...),
		client.WithTimeout(10*time.Second),
	)
	if err != nil {
		logger.Error.Printf("Failed to create search agent: %v", err)
		return
	}
	defer a.Close()

	if options.Interactive {
		scanner := bufio.NewScanner(os.Stdin)
		const prompt = "search>"
		_, _ = os.Stdout.WriteString(prompt)
		for scanner.Scan() {
			searchPhrase(scanner.Text(), a, options)
			_, _ = os.Stdout.WriteString(prompt)
		}
	} else {
		searchPhrase(strings.Join(options.Phrases, " "), a, options)
	}
}

func searchPhrase(phrase string, agent client.SearchAgent, options searchOptions) {
	res, err := agent.Search(
		phrase,
		[]string{options.Space},
		options.Limit,
		options.Offset,
	)
	if err != nil {
		logger.Error.Printf("Failed to perform search: %v", err)
		return
	}

	fmt.Printf("Query executed in %v seconds with status %q\n", res.Duration, res.Status.String())
	fmt.Printf("Returning %v of %v total hits, capped: %v\n",
		len(res.Result.Hits), res.Result.TotalHits, res.Result.Capped)
	if res.Status == protocol.SearchStatusNoHit && res.Result.Respelt != "" {
		fmt.Printf("Did you mean %s?\n", res.Result.Respelt)
	}
	fmt.Println()
	for _, doc := range res.Result.Hits {
		fmt.Printf("[%v] %s\n", doc.ID, doc.Snippet)
	}
}
