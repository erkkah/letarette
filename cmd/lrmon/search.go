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
	"time"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/protocol"
)

type searchRequest struct {
	query    string
	spaces   []string
	limit    int
	response chan searchResponse
}

type searchResponse struct {
	Error    error
	Response protocol.SearchResponse
}

var searchRequests chan searchRequest

func startSearchClient(URLS []string, seedFile string, rootCAs []string) error {
	searchRequests = make(chan searchRequest)
	agent, err := client.NewSearchAgent(
		URLS,
		client.WithSeedFile(seedFile),
		client.WithRootCAs(rootCAs...),
		client.WithTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	go func() {
		for req := range searchRequests {
			response, err := agent.Search(req.query, req.spaces, req.limit, 0)
			req.response <- searchResponse{
				Error:    err,
				Response: response,
			}
		}
		agent.Close()
	}()

	return nil
}

func search(query string, spaces []string, limit int) searchResponse {
	req := searchRequest{
		query:    query,
		spaces:   spaces,
		limit:    limit,
		response: make(chan searchResponse),
	}
	searchRequests <- req
	return <-req.response
}
