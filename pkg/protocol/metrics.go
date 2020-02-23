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

package protocol

import "time"

// MetricsRequest asks for a metrics dump from all service instances
type MetricsRequest struct {
	RequestID string
}

// Metrics holds a metrics response from one service instance.
// The RequestID field contains the ID provided in the corresponding request.
// The PackedJSON field contains Base64 encoded, zlib compressed JSON of all exported metrics.
type Metrics struct {
	RequestID  string
	IndexID    string
	Updated    time.Time
	PackedJSON string
}
