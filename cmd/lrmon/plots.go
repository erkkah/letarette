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
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/erkkah/margaid"
)

type seriesMap map[string]*margaid.Series

var plotSeries = seriesMap{}

func updatePlots(state State) {
	updated := seriesMap{}

	for spec, plot := range state.Plots {

		var rendered bytes.Buffer

		metrics, indexFound := state.Metrics[plot.index]
		update, metricFound := metrics.Value[plot.metric]
		if !(indexFound && metricFound) {
			m := margaid.New(640, 240, margaid.WithTitleFont("sans-serif", 12))
			m.Title("Unknown metric")
			m.Frame()
			_ = m.Render(&rendered)
		} else {
			// Hacky way of removing plot type from spec
			plotSpec := strings.Join(strings.Split(spec, ":")[:5], ":")

			series := plotSeries[plotSpec]

			if series == nil {
				var aggregator margaid.Aggregator

				switch plot.method {
				case "sum":
					aggregator = margaid.Sum
				case "delta":
					aggregator = margaid.Delta
				case "avg":
					fallthrough
				default:
					aggregator = margaid.Avg
				}

				series = margaid.NewSeries(
					margaid.CappedByAge(plot.window, time.Now),
					margaid.AggregatedBy(aggregator, plot.period),
				)
				plotSeries[plotSpec] = series
			}

			if updated[plotSpec] == nil {
				series.Add(margaid.MakeValue(margaid.SecondsFromTime(metrics.Timestamp), update))
				updated[plotSpec] = series
			}

			padding := 0.0
			if plot.plotType == "bar" {
				padding = 5
			}
			m := margaid.New(640, 240,
				margaid.WithTitleFont("sans-serif", 12),
				margaid.WithLabelFont("sans", 10),
				margaid.WithAutorange(margaid.XAxis, series),
				margaid.WithAutorange(margaid.YAxis, series),
				margaid.WithPadding(padding),
			)

			m.Title(fmt.Sprintf("%s: %s", plot.index[:6], plot.metric))

			switch plot.plotType {
			case "line":
				m.Line(series)
			case "smooth":
				m.Smooth(series)
			case "bar":
				m.Bar([]*margaid.Series{series})
			}

			xTitle := fmt.Sprintf("%s %s / %s", plot.period, plot.method, plot.window)
			m.Axis(series, margaid.XAxis, m.TimeTicker("15:04:05"), true, xTitle)
			m.Axis(series, margaid.YAxis, m.ValueTicker('f', 0, 10), true, "")
			m.Frame()

			_ = m.Render(&rendered)
		}

		state.Plots[spec].SVG = template.HTML(rendered.String())
	}
}

// Adds a plot.
// Metric identifiers are not allowed to contain colons.
func addPlot(index, metric, method string, period, window time.Duration, plotType string) error {
	if strings.Contains(metric, ":") {
		return fmt.Errorf("cannot plot metric %q with colon in identifier", metric)
	}

	stateUpdates <- func(state State) State {
		plot := plot{
			index:    index,
			metric:   metric,
			method:   method,
			period:   period,
			window:   window,
			plotType: plotType,
			Reload:   int(period.Milliseconds()),
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
