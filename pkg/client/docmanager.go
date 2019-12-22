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
	"context"
	"fmt"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// IndexRequestHandler processes index update requests from the letarette cluster
// and returns index updates.
type IndexRequestHandler func(ctx context.Context, req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error)

// DocumentRequestHandler processes document requests from the letarette cluster
// and returns document updates.
type DocumentRequestHandler func(ctx context.Context, req protocol.DocumentRequest) (protocol.DocumentUpdate, error)

// DocumentManager connects to the letarette cluster and processes indexing requests
type DocumentManager interface {
	Close()
	StartIndexRequestHandler(handler IndexRequestHandler)
	StartDocumentRequestHandler(handler DocumentRequestHandler)
}

type manager struct {
	state
	ctx    context.Context
	cancel context.CancelFunc
}

// StartDocumentManager creates a DocumentManager and connects to Nats daemon
func StartDocumentManager(url string, options ...Option) (DocumentManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &manager{
		state: state{
			topic:   "leta",
			onError: func(error) {},
		},
		ctx:    ctx,
		cancel: cancel,
	}

	mgr.local = mgr
	mgr.apply(options)

	natsOptions := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
		nats.RootCAs(mgr.rootCAs...),
	}

	if mgr.seedFile != "" {
		option, err := nats.NkeyOptionFromSeed(mgr.seedFile)
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
	mgr.conn = ec

	return mgr, nil
}

func (m *manager) Close() {
	m.cancel()
	m.conn.Close()
}

func (m *manager) StartIndexRequestHandler(handler IndexRequestHandler) {
	m.conn.Subscribe(m.topic+".index.request", func(sub, reply string, req *protocol.IndexUpdateRequest) {
		update, err := handler(m.ctx, *req)
		if err != nil {
			m.onError(err)
			return
		}
		err = m.conn.Publish(reply, update)
		if err != nil {
			m.onError(err)
		}
	})
}

func (m *manager) StartDocumentRequestHandler(handler DocumentRequestHandler) {
	m.conn.Subscribe(m.topic+".document.request", func(req *protocol.DocumentRequest) {
		update, err := handler(m.ctx, *req)
		if err != nil {
			m.onError(err)
			return
		}
		updates := []protocol.DocumentUpdate{update}

		for len(updates) > 0 {
			current := updates[len(updates)-1]
			updates = updates[:len(updates)-1]

			err = m.conn.Publish(m.topic+".document.update", current)
			if err != nil {
				if err == nats.ErrMaxPayload {
					length := len(current.Documents)
					if length > 1 {
						mid := length / 2
						updates = append(updates,
							protocol.DocumentUpdate{
								Space:     current.Space,
								Documents: current.Documents[:mid],
							},
							protocol.DocumentUpdate{
								Space:     current.Space,
								Documents: current.Documents[mid:],
							},
						)
						m.onError(fmt.Errorf("Document list too large, splitting"))
					} else {
						doc := current.Documents[0]
						doc.Text = truncateString(doc.Text, int(m.conn.Conn.MaxPayload()/2))
						updates = append(updates,
							protocol.DocumentUpdate{
								Space: current.Space,
								Documents: []protocol.Document{
									doc,
								},
							},
						)
						m.onError(fmt.Errorf("Document %v too large, truncating", doc.ID))
					}
				} else {
					m.onError(err)
				}
			}
		}
	})
}

func truncateString(long string, max int) string {
	result := long
	// i indexes in bytes, but steps in runes
	for i := range long {
		if i >= max {
			result = long[:i] + "\u2026" // ellipsis
		}
	}
	return result
}
