package client

import (
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/go-nats"
)

type SearchClient interface {
	Close()
	Search(q string, spaces []string, limit uint16) (protocol.SearchResponse, error)
}

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

func (client *searchClient) Search(q string, spaces []string, limit uint16) (res protocol.SearchResponse, err error) {
	req := protocol.SearchRequest{
		Spaces: spaces,
		Query:  q,
		Limit:  limit,
	}
	err = client.conn.Request(client.topic+".q", req, &res, time.Second*2)
	return
}
