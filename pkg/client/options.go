package client

import "github.com/nats-io/go-nats"

type state struct {
	conn    *nats.EncodedConn
	topic   string
	onError func(error)
}

func (s *state) apply(options []Option) {
	for _, option := range options {
		option(s)
	}
}

// Option is the option setter interface. See related WithXXX functions.
type Option func(*state)

// WithTopic sets the Nats topic instead of the default
func WithTopic(topic string) Option {
	return func(o *state) {
		o.topic = topic
	}
}

// WithErrorHandler sets an error handler instead of the default silent one
func WithErrorHandler(handler func(error)) Option {
	return func(o *state) {
		o.onError = handler
	}
}
