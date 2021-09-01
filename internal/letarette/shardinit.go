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

package letarette

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// InitializeShard tries to locate healthier shards to clone from, to cut down
// start-up times and reduce load on the DocumentManager.
func InitializeShard(ctx context.Context, conn *nats.Conn, db Database, cfg Config, monitor StatusMonitor) error {
	logger.Info.Printf("Looking for healthy shards to clone")
	shardInfo := monitor.GetHealthyShards()
	defer monitor.ShardInitDone()

	var cloneSources []ShardInfo
	var smallestGroupSize uint16

	for _, info := range shardInfo {
		// Prefer cloning the same shard
		if info.ShardgroupSize == cfg.ShardgroupSize && info.ShardIndex == cfg.ShardIndex {
			cloneSources = []ShardInfo{info}
			break
		}

		if (smallestGroupSize == 0 || smallestGroupSize > info.ShardgroupSize) &&
			// No point in cloning from other shards in same group
			info.ShardgroupSize != cfg.ShardgroupSize {
			smallestGroupSize = info.ShardgroupSize
		}
	}

	// No same-shaped shard found, try to clone from smallest shard group
	if len(cloneSources) == 0 {
		indices := map[uint16]bool{}

		for _, info := range shardInfo {
			if info.ShardgroupSize == smallestGroupSize {
				if _, found := indices[info.ShardIndex]; !found {
					cloneSources = append(cloneSources, info)
					indices[info.ShardIndex] = true
				}
			}
		}
	}

	if len(cloneSources) == 0 {
		logger.Info.Printf("Found no valid cloning sources, continuing normal startup")
		return nil
	}

	sourceTotal := uint64(0)
	for _, source := range cloneSources {
		sourceTotal += source.DocCount
	}

	sql := db.(*database)
	count, err := sql.getDocumentCount(ctx)
	if err != nil {
		return err
	}

	// Arbitrary limit for initiating cloning
	const limit = 0.8

	if float32(count) > limit*float32(sourceTotal) {
		logger.Info.Printf("In sync enough to skip cloning, continuing normal startup")
		return nil
	}

	ec, err := nats.NewEncodedConn(conn, nats.JSON_ENCODER)
	if err != nil {
		return err
	}

	for _, source := range cloneSources {
		sourceShard := fmt.Sprintf("%d/%d", source.ShardIndex+1, source.ShardgroupSize)

		logger.Info.Printf("Requesting clone from shard %s", sourceShard)

		req := protocol.CloneRequest{
			SourceShard: sourceShard,
			TargetShard: cfg.Shard,
		}

		var res protocol.CloneStream
		err = ec.Request(cfg.Nats.Topic+".clone", req, &res, time.Second*2)
		if err != nil {
			return err
		}

		httpReq, err := http.NewRequestWithContext(ctx, "GET", res.URL, bytes.NewBuffer([]byte{}))
		if err != nil {
			return err
		}

		httpResponse, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return err
		}
		defer httpResponse.Body.Close()

		logger.Info.Printf("Loading clone")
		err = LoadShardClone(ctx, db, httpResponse.Body)
		if err != nil {
			return err
		}
	}
	return nil
}
