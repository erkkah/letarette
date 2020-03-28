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

package letarette

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"expvar"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// All exported metrics
var metrics = struct {
	DocRequests expvar.Int
	UpdateQueue expvar.Int
	PendingDocs expvar.Int
	ServedDocs  expvar.Int
	QueryQueue  expvar.Int
}{}

type jsonExpvar struct {
	expvar.Var
}

var metricsByName = map[string]jsonExpvar{}

func (v jsonExpvar) MarshalJSON() ([]byte, error) {
	return []byte(v.String()), nil
}

func init() {
	mType := reflect.TypeOf(metrics)
	mValue := reflect.ValueOf(&metrics).Elem()

	for i := 0; i < mType.NumField(); i++ {
		field := mType.Field(i)
		value := mValue.Field(i)
		metricName := strings.ToLower(field.Name)
		metricsByName[metricName] = jsonExpvar{value.Addr().Interface().(expvar.Var)}
	}
}

func getMetricsJSON() ([]byte, error) {
	json, err := json.Marshal(&metricsByName)
	return json, err
}

func getPackedMetrics() (string, error) {
	json, err := getMetricsJSON()
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err = w.Write(json)
	_ = w.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// MetricsCollector listens and responds to metrics requests
type MetricsCollector interface {
	Close()
}

type metricsCollector nats.Subscription

func (mc *metricsCollector) Close() {
	sub := (*nats.Subscription)(mc)
	_ = sub.Unsubscribe()
}

// StartMetricsCollector creates a new metrics collector, and starts responding to requests
func StartMetricsCollector(nc *nats.Conn, db Database, cfg Config) (MetricsCollector, error) {
	privateDB := db.(*database)
	indexID, err := privateDB.getIndexID()
	if err != nil {
		return nil, fmt.Errorf("failed to read index ID: %w", err)
	}

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	sub, err := ec.Subscribe(cfg.Nats.Topic+".metrics.request", func(req *protocol.MetricsRequest) {
		packed, _ := getPackedMetrics()
		reply := protocol.Metrics{
			RequestID:  req.RequestID,
			IndexID:    indexID,
			Updated:    time.Now(),
			PackedJSON: packed,
		}
		_ = ec.Publish(cfg.Nats.Topic+".metrics.reply", &reply)
	})
	if err != nil {
		return nil, err
	}

	return (*metricsCollector)(sub), nil
}
