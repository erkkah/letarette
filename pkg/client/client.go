package client

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// SearchClient is a letarette cluster searcher
type SearchClient interface {
	Close()
	Search(q string, spaces []string, pageLimit int, pageOffset int) (protocol.SearchResponse, error)
}

// WithShardgroupSize forces shard group size instead of using discovery
func WithShardgroupSize(groupSize int32) Option {
	return func(st *state) {
		sc := st.local.(*searchClient)
		sc.volatileNumShards = groupSize
	}
}

// NewSearchClient - SearchClient constructor
func NewSearchClient(url string, options ...Option) (SearchClient, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	client := &searchClient{
		state: state{
			conn:    ec,
			topic:   "leta",
			onError: func(error) {},
		},
		volatileNumShards: 0,
	}

	client.local = client
	client.state.apply(options)

	if client.volatileNumShards == 0 {
		client.monitor, err = NewMonitor(url, func(status protocol.IndexStatus) {
			atomic.SwapInt32(&client.volatileNumShards, int32(status.ShardgroupSize))
		}, WithTopic(client.state.topic))
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

type searchClient struct {
	state
	volatileNumShards int32
	monitor           Monitor
}

func (client *searchClient) Close() {
	if client.monitor != nil {
		client.monitor.Close()
	}

	client.conn.Close()
}

func (client *searchClient) getNumShards() int32 {
	for {
		numShards := atomic.LoadInt32(&client.volatileNumShards)
		if numShards == 0 {
			time.Sleep(time.Millisecond * 100)
		} else {
			return numShards
		}
	}
}

func (client *searchClient) Search(q string, spaces []string, pageLimit int, pageOffset int) (res protocol.SearchResponse, err error) {
	numShards := client.getNumShards()
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

	inbox := client.conn.Conn.NewRespInbox()
	responseCh := make(chan protocol.SearchResponse, numShards)
	defer func() {
		close(responseCh)
		responseCh = nil
	}()
	sub, err := client.conn.Subscribe(inbox, func(response *protocol.SearchResponse) {
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
	err = client.conn.PublishRequest(client.topic+".q", inbox, req)
	if err != nil {
		return
	}
	timeout := time.After(time.Second * 2)
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
	}
	sort.SliceStable(merged.Result.Hits, func(a, b int) bool {
		return merged.Result.Hits[a].Rank < merged.Result.Hits[b].Rank
	})
	return merged
}
