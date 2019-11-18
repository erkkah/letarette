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
	"fmt"
	"math"
	"testing"

	"github.com/erkkah/letarette/pkg/protocol"
)

func TestShardSpread(t *testing.T) {
	spread := map[int]int{}
	docs := 1076
	shards := 5
	for i := 0; i < docs; i++ {
		docID := fmt.Sprintf("%d", i)
		idx := shardIndexFromDocumentID(protocol.DocumentID(docID), shards)
		count := spread[idx]
		spread[idx] = count + 1
	}
	shardSize := docs / shards
	expected := float64(shardSize) * 2 / 3
	for k, v := range spread {
		t.Logf("%d: %d", k, v)
		if math.Abs(float64(shardSize-v)) > expected {
			t.Errorf("Uneven shard group spread")
		}
	}
}
