package client

import (
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
		err = m.conn.Publish(m.topic+".document.update", update)
		if err != nil {
			m.onError(err)
		}
	})
}
