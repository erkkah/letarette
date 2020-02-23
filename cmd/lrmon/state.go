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
	"strings"
	"sync/atomic"
	"time"

	"github.com/erkkah/letarette"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/erkkah/margaid"
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
		for update := range stateUpdates {
			oldState := getState()
			state.Store(update(oldState))
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

type indexStatus map[string]protocol.IndexStatus

// Maps from plot specifier
type plotsMap map[string]*plot
type seriesMap map[string]*margaid.Series

type plot struct {
	index  string
	metric string
	method string
	period time.Duration
	window time.Duration
	Reload int
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
	result[index] = update
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

var plotSeries = map[string]*margaid.Series{}

func updatePlots(state State) {
	for spec, plot := range state.Plots {
		series := plotSeries[spec]
		if series == nil {
			series = margaid.NewSeries(
				margaid.CappedByAge(plot.window, time.Now),
				margaid.AggregatedBy(margaid.Avg, plot.period),
			)
			plotSeries[spec] = series
		}

		var rendered bytes.Buffer

		metrics, indexFound := state.Metrics[plot.index]
		update, metricFound := metrics.Value[plot.metric]
		if !(indexFound && metricFound) {
			m := margaid.New(640, 240, margaid.WithTitleFont("sans-serif", 12))
			m.Title("Unknown metric")
			m.Frame()
			m.Render(&rendered)
		} else {
			series.Add(margaid.MakeValue(margaid.SecondsFromTime(metrics.Timestamp), update))

			m := margaid.New(640, 240,
				margaid.WithTitleFont("sans-serif", 12),
				margaid.WithLabelFont("sans", 10),
				margaid.WithAutorange(margaid.XAxis, series),
				margaid.WithAutorange(margaid.YAxis, series),
			)

			m.Title(fmt.Sprintf("%s: %s", plot.index[:6], plot.metric))
			xTitle := fmt.Sprintf("%s %s / %s", plot.period, plot.method, plot.window)
			m.Axis(series, margaid.XAxis, m.TimeTicker("15:04:05"), true, xTitle)
			m.Axis(series, margaid.YAxis, m.ValueTicker('f', 0, 10), true, "")
			m.Frame()
			m.Line(series)
			m.Render(&rendered)
		}

		state.Plots[spec].SVG = template.HTML(rendered.String())
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
		return nil, fmt.Errorf("Error while base64 unpacking: %w", err)
	}

	compressed := bytes.NewBuffer(jsonBytes)
	reader, err := zlib.NewReader(compressed)
	if err != nil {
		return nil, fmt.Errorf("Error creating zlib reader: %w", err)
	}

	uncompressed := new(bytes.Buffer)
	_, err = io.Copy(uncompressed, reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to uncompress: %w", err)
	}

	var metricsMap metricsMap
	err = json.Unmarshal(uncompressed.Bytes(), &metricsMap)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON: %w", err)
	}
	return metricsMap, nil
}

// Adds a plot.
// Metric identifiers are not allowed to contain colons.
func addPlot(index, metric, method string, period, window time.Duration, plotType string) error {
	if strings.Contains(metric, ":") {
		return fmt.Errorf("Cannot plot metric %q with colon in identifier", metric)
	}

	stateUpdates <- func(state State) State {
		plot := plot{
			index:  index,
			metric: metric,
			method: method,
			period: period,
			window: window,
			Reload: int(period.Milliseconds()),
		}
		specifier := fmt.Sprintf("%s:%s:%s:%v:%v:%s", index, metric, method, period, window, plotType)
		state.Plots = clonePlotsWith(state.Plots, specifier, &plot)
		return state
	}

	return nil
}

func removePlot(id string) error {
	stateUpdates <- func(state State) State {
		state.Plots = clonePlotsWithout(state.Plots, id)
		return state
	}

	return nil
}
