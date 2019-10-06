package client

import (
	"fmt"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/go-nats"
)

// IndexRequestHandler processes index update requests from the letarette cluster
// and returns index updates.
type IndexRequestHandler func(req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error)

// DocumentRequestHandler processes document requests from the letarette cluster
// and returns document updates.
type DocumentRequestHandler func(req protocol.DocumentRequest) (protocol.DocumentUpdate, error)

// DocumentManager connects to the letarette cluster and processes indexing requests
type DocumentManager interface {
	Close()
	StartIndexRequestHandler(handler IndexRequestHandler)
	StartDocumentRequestHandler(handler DocumentRequestHandler)
}

type manager struct {
	conn    *nats.EncodedConn
	topic   string
	onError func(error)
}

// Option is the option setter interface. See related WithXXX functions.
type Option func(*manager)

// WithTopic sets the Nats topic instead of the default
func WithTopic(topic string) Option {
	return func(m *manager) {
		m.topic = topic
	}
}

// WithErrorHandler sets an error handler instead of the default silent one
func WithErrorHandler(handler func(error)) Option {
	return func(m *manager) {
		m.onError = handler
	}
}

// StartDocumentManager creates a DocumentManager and connects to Nats daemon
func StartDocumentManager(url string, options ...Option) (DocumentManager, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	mgr := &manager{
		conn:    ec,
		topic:   "leta",
		onError: func(error) {},
	}

	for _, option := range options {
		option(mgr)
	}

	return mgr, nil
}

func (m *manager) Close() {
	m.conn.Close()
}

func (m *manager) StartIndexRequestHandler(handler IndexRequestHandler) {
	m.conn.Subscribe(m.topic+".index.request", func(sub, reply string, req *protocol.IndexUpdateRequest) {
		update, err := handler(*req)
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
		update, err := handler(*req)
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
								Documents: current.Documents[:mid],
							},
							protocol.DocumentUpdate{
								Documents: current.Documents[mid:],
							},
						)
						m.onError(fmt.Errorf("Document list too large, splitting"))
					} else {
						doc := current.Documents[0]
						doc.Text = truncateString(doc.Text, int(m.conn.Conn.MaxPayload()/2))
						updates = append(updates,
							protocol.DocumentUpdate{
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
	chars := 0
	result := long
	for i := range long {
		if chars >= max {
			result = long[:i] + "\u2026" // ellipsis
		}
		chars++
	}
	return result
}
