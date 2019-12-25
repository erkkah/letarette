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
	"github.com/erkkah/letarette/pkg/protocol"
)

// Monitor listens to status broadcasts from a letarette cluster
type Monitor interface {
	Close()
}

// MonitorListener is a callback function receiving status broadcasts
type MonitorListener func(protocol.IndexStatus)

// NewMonitor - Monitor constructor
func NewMonitor(url string, listener MonitorListener, options ...Option) (Monitor, error) {
	client := &monitor{
		state: state{
			topic:   "leta",
			onError: func(error) {},
		},
		listener: listener,
	}

	client.apply(options)

	ec, err := connect(url, client.state)
	if err != nil {
		return nil, err
	}

	client.conn = ec

	_, err = client.conn.Subscribe(client.topic+".status", func(status *protocol.IndexStatus) {
		client.listener(*status)
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

type monitor struct {
	state
	listener MonitorListener
}

func (m *monitor) Close() {
	m.conn.Close()
}
