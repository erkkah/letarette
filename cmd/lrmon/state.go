// Copyright 2020 Erik Agsjö
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

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sync/atomic"
	"time"

	"github.com/erkkah/letarette"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

var state atomic.Value
var stateUpdates chan stateUpdater

func init() {
	state.Store(State{
		Version:     letarette.Version(),
		IndexStatus: indexStatus{},
		Metrics:     indexMetrics{},
		Plots:       plotsMap{},
	})

	stateUpdates = make(chan stateUpdater, 10)

	go func() {
	updating:
		for {
			select {
			case update, ok := <-stateUpdates:
				if !ok {
					break updating
				}
				oldState := getState()
				state.Store(update(oldState))
			case <-time.After(time.Second * 10):
				oldState := getState()

				now := time.Now()
				stale := now.Add(-time.Second * 30)
				dead := now.Add(-time.Minute * 10)

				alive := indexStatus{}
				for index, status := range oldState.IndexStatus {
					if status.Updated.Before(dead) {
						continue
					}
					status.Stale = status.Updated.Before(stale)
					alive[index] = status
				}
				oldState.IndexStatus = alive
				state.Store(oldState)
			}
		}
		logger.Info.Printf("Stopping updater")
	}()
}

type metricsMap map[string]float64
type metricsQuote struct {
	Value     metricsMap
	Timestamp time.Time
}
type indexMetrics map[string]metricsQuote

type statusUpdate struct {
	protocol.IndexStatus
	Updated time.Time
	Stale   bool
}

type indexStatus map[string]statusUpdate

// Maps from plot specifier
type plotsMap map[string]*plot

type plot struct {
	index    string
	metric   string
	method   string
	plotType string
	period   time.Duration
	window   time.Duration
	Reload   int
	// Make atomic?
	SVG template.HTML
}

// State is the context passed to HTML/SVG rendering
type State struct {
	Version string
	// IndexStatus by index ID
	IndexStatus indexStatus
	// MetricsMap by index ID
	Metrics indexMetrics
	// Plots by specifier
	Plots plotsMap
}

func getState() State {
	return state.Load().(State)
}

type stateUpdater func(State) State

func cloneStatusWith(source indexStatus, index string, update protocol.IndexStatus) indexStatus {
	result := indexStatus{}
	for k, v := range source {
		result[k] = v
	}
	result[index] = statusUpdate{
		update,
		time.Now(),
		false,
	}
	return result
}

func cloneMetricsWith(source indexMetrics, index string, update metricsQuote) indexMetrics {
	result := indexMetrics{}
	for k, v := range source {
		result[k] = v
	}
	result[index] = update
	return result
}

func clonePlotsWith(source plotsMap, index string, update *plot) plotsMap {
	result := plotsMap{}
	for k, v := range source {
		result[k] = v
	}
	result[index] = update
	return result
}

func clonePlotsWithout(source plotsMap, index string) plotsMap {
	result := plotsMap{}
	for k, v := range source {
		if k != index {
			result[k] = v
		}
	}
	return result
}

func handleStatusUpdate(indexStatus protocol.IndexStatus) {
	stateUpdates <- func(state State) State {
		state.IndexStatus = cloneStatusWith(state.IndexStatus, indexStatus.IndexID, indexStatus)
		return state
	}
}

func handleMetricsUpdate(metrics protocol.Metrics) {
	stateUpdates <- func(state State) State {
		unpacked, err := unpackMetrics(metrics)
		if err != nil {
			logger.Error.Printf("Failed to unpack metrics: %v", err)
		}
		quote := metricsQuote{
			Timestamp: metrics.Updated,
			Value:     unpacked,
		}
		state.Metrics = cloneMetricsWith(state.Metrics, metrics.IndexID, quote)
		updatePlots(state)
		return state
	}
}

func unpackMetrics(metrics protocol.Metrics) (metricsMap, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(metrics.PackedJSON)
	if err != nil {
		return nil, fmt.Errorf("error while base64 unpacking: %w", err)
	}

	compressed := bytes.NewBuffer(jsonBytes)
	reader, err := zlib.NewReader(compressed)
	if err != nil {
		return nil, fmt.Errorf("error creating zlib reader: %w", err)
	}

	uncompressed := new(bytes.Buffer)
	_, err = io.Copy(uncompressed, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress: %w", err)
	}

	var metricsMap metricsMap
	err = json.Unmarshal(uncompressed.Bytes(), &metricsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return metricsMap, nil
}
