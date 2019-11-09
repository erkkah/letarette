package client

import (
	"fmt"
	"sort"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// SearchClient is a letarette cluster searcher
type SearchClient interface {
	Close()
	Search(q string, spaces []string, pageLimit int, pageOffset int) (protocol.SearchResponse, error)
}

// NewSearchClient - SearchClient constructor
func NewSearchClient(url string, options ...Option) (SearchClient, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	client := &searchClient{
		conn:    ec,
		topic:   "leta",
		onError: func(error) {},
	}

	(*state)(client).apply(options)

	return client, nil
}

type searchClient state

func (client *searchClient) Close() {
	client.conn.Close()
}

func (client *searchClient) Search(q string, spaces []string, pageLimit int, pageOffset int) (res protocol.SearchResponse, err error) {
	req := protocol.SearchRequest{
		Spaces:     spaces,
		Query:      q,
		PageLimit:  uint16(pageLimit),
		PageOffset: uint16(pageOffset),
	}
	const numShards = 1
	inbox := client.conn.Conn.NewRespInbox()
	responseCh := make(chan protocol.SearchResponse, numShards)
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
	err = sub.AutoUnsubscribe(numShards)
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
			responseCh = nil
			err = fmt.Errorf("Timeout waiting for search response")
			return
		case response := <-responseCh:
			responses = append(responses, response)
			if len(responses) == numShards {
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
