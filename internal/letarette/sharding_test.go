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
