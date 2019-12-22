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

import "github.com/nats-io/nats.go"

type state struct {
	conn     *nats.EncodedConn
	seedFile string
	topic    string
	onError  func(error)
	local    interface{}
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

// WithSeedFile specifies a seed file for Nkey authentication
func WithSeedFile(seedFile string) Option {
	return func(o *state) {
		if seedFile != "" {
			o.seedFile = seedFile
		}
	}
}
