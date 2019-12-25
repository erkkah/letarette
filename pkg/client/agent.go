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

package client

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// SearchAgent is a letarette cluster searcher
type SearchAgent interface {
	Close()
	Search(q string, spaces []string, pageLimit int, pageOffset int) (protocol.SearchResponse, error)
}

// WithShardgroupSize forces shard group size instead of using discovery
func WithShardgroupSize(groupSize int32) Option {
	return func(st *state) {
		sa := st.local.(*searchAgent)
		sa.volatileNumShards = groupSize
	}
}

// WithTimeout sets search request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(st *state) {
		sa := st.local.(*searchAgent)
		sa.timeout = timeout
	}
}

// NewSearchAgent - SearchAgent constructor
func NewSearchAgent(url string, options ...Option) (SearchAgent, error) {
	agent := &searchAgent{
		state: state{
			topic:   "leta",
			onError: func(error) {},
		},
		volatileNumShards: 0,
		timeout:           time.Second * 2,
	}

	agent.local = agent
	agent.apply(options)

	natsOptions := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
		nats.RootCAs(agent.rootCAs...),
	}

	if agent.seedFile != "" {
		option, err := nats.NkeyOptionFromSeed(agent.seedFile)
		if err != nil {
			return nil, err
		}
		natsOptions = append(natsOptions, option)
	}

	nc, err := nats.Connect(url, natsOptions...)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}
	agent.conn = ec

	if agent.volatileNumShards == 0 {
		agent.monitor, err = NewMonitor(
			url,
			func(status protocol.IndexStatus) {
				atomic.SwapInt32(&agent.volatileNumShards, int32(status.ShardgroupSize))
			},
			WithTopic(agent.topic),
			WithSeedFile(agent.seedFile),
			WithRootCAs(agent.rootCAs...),
		)
		if err != nil {
			return nil, err
		}
	}

	return agent, nil
}

type searchAgent struct {
	state
	volatileNumShards int32
	monitor           Monitor
	timeout           time.Duration
}

func (agent *searchAgent) Close() {
	if agent.monitor != nil {
		agent.monitor.Close()
	}

	agent.conn.Close()
}

func (agent *searchAgent) getNumShards() (int32, error) {
	start := time.Now()
	for {
		numShards := atomic.LoadInt32(&agent.volatileNumShards)
		if numShards == 0 {
			if time.Now().After(start.Add(time.Second * 5)) {
				return 0, fmt.Errorf("Timeout waiting for cluster")
			}
			time.Sleep(time.Millisecond * 100)
		} else {
			return numShards, nil
		}
	}
}

func (agent *searchAgent) Search(q string, spaces []string, pageLimit int, pageOffset int) (res protocol.SearchResponse, err error) {
	numShards, err := agent.getNumShards()
	if err != nil {
		return
	}
	shardedLimit := pageLimit / int(numShards)
	if shardedLimit < 1 {
		shardedLimit = 1
	}
	req := protocol.SearchRequest{
		Spaces:     spaces,
		Query:      q,
		PageLimit:  uint16(shardedLimit),
		PageOffset: uint16(pageOffset),
	}

	inbox := agent.conn.Conn.NewRespInbox()
	responseCh := make(chan protocol.SearchResponse, numShards)
	defer func() {
		close(responseCh)
		responseCh = nil
	}()
	sub, err := agent.conn.Subscribe(inbox, func(response *protocol.SearchResponse) {
		if responseCh != nil {
			clone := *response
			clone.Result.Hits = append(clone.Result.Hits[:0:0], clone.Result.Hits...)
			responseCh <- *response
		}
	})
	if err != nil {
		return
	}
	err = sub.AutoUnsubscribe(int(numShards))
	if err != nil {
		return
	}
	err = agent.conn.PublishRequest(agent.topic+".q", inbox, req)
	if err != nil {
		return
	}
	timeout := time.After(agent.timeout)
	var responses []protocol.SearchResponse

waitLoop:
	for {
		select {
		case <-timeout:
			sub.Unsubscribe()
			err = fmt.Errorf("Timeout waiting for search response")
			return
		case response := <-responseCh:
			responses = append(responses, response)
			if len(responses) == int(numShards) {
				break waitLoop
			}
		}
	}

	res = mergeResponses(responses)
	return
}

func mergeResponses(responses []protocol.SearchResponse) protocol.SearchResponse {
	var merged protocol.SearchResponse
	for _, response := range responses {
		if merged.Duration < response.Duration {
			merged.Duration = response.Duration
		}
		if merged.Status < response.Status {
			merged.Status = response.Status
		}
		merged.Result.Capped = merged.Result.Capped || response.Result.Capped
		merged.Result.TotalHits += response.Result.TotalHits
		merged.Result.Hits = append(merged.Result.Hits, response.Result.Hits...)

		// Keep the respelt version with the lowest distance
		if merged.Result.Respelt == "" ||
			(response.Result.RespeltDistance > 0 && merged.Result.RespeltDistance > response.Result.RespeltDistance) {
			merged.Result.Respelt = response.Result.Respelt
			merged.Result.RespeltDistance = response.Result.RespeltDistance
		}
	}
	sort.SliceStable(merged.Result.Hits, func(a, b int) bool {
		return merged.Result.Hits[a].Rank < merged.Result.Hits[b].Rank
	})
	return merged
}
