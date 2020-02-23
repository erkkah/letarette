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
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
)

// Monitor listens to status broadcasts from a letarette cluster
type Monitor interface {
	Close()
}

// MonitorListener is a callback function receiving status broadcasts
type MonitorListener func(protocol.IndexStatus)

// NewMonitor - Monitor constructor
func NewMonitor(URLs []string, listener MonitorListener, options ...Option) (Monitor, error) {
	client := &monitor{
		state: state{
			topic:   "leta",
			onError: func(error) {},
		},
		listener: listener,
	}

	client.local = client
	client.apply(options)

	ec, err := connect(URLs, client.state)
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

	if client.metricsCollector != nil {
		client.startMetricsCollector()
	}

	return client, nil
}

// MetricsCollector is a callback function receiving metrics updates
type MetricsCollector func(metrics protocol.Metrics)

// WithMetricsCollector makes the monitor periodically request metrics
// from the cluster.
func WithMetricsCollector(collector MetricsCollector, interval time.Duration) Option {
	return func(st *state) {
		m := st.local.(*monitor)
		m.metricsCollector = collector
		m.metricsInterval = interval
		m.metricsDone = make(chan struct{})
	}
}

type monitor struct {
	state
	listener MonitorListener

	metricsCollector MetricsCollector
	metricsInterval  time.Duration
	metricsDone      chan struct{}
}

func (m *monitor) Close() {
	m.conn.Close()
	if m.metricsDone != nil {
		close(m.metricsDone)
		m.metricsDone = nil
	}
}

func (m *monitor) startMetricsCollector() error {
	sub, err := m.conn.Subscribe(m.topic+".metrics.reply", func(metrics *protocol.Metrics) {
		m.metricsCollector(*metrics)
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-time.After(m.metricsInterval):
				m.requestMetrics()
			case <-m.metricsDone:
				sub.Unsubscribe()
				return
			}
		}
	}()
	return nil
}

func (m *monitor) requestMetrics() error {
	req := protocol.MetricsRequest{
		RequestID: time.Now().String(),
	}
	err := m.conn.Publish(m.topic+".metrics.request", &req)
	return err
}
