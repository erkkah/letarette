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
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/erkkah/letarette/pkg/protocol"
)

var shardHasher = fnv.New32a()

// ShardIndexFromDocumentID calculated a shard index based on a hash
// of the document ID.
// The hash algorithm is chosen for even distribution in a shard group.
func ShardIndexFromDocumentID(docID protocol.DocumentID, shardGroupSize int) int {
	shardHasher.Reset()
	_, _ = shardHasher.Write([]byte(docID))
	sum := shardHasher.Sum(nil)
	intPart := binary.BigEndian.Uint32(sum)
	return int(intPart % uint32(shardGroupSize))
}

func parseShardString(shardGroup string) (group, size int, err error) {
	parts := strings.SplitN(shardGroup, "/", 2)
	parseError := fmt.Errorf("invalid shard group setting")
	if len(parts) != 2 {
		err = parseError
		return
	}
	group, err = strconv.Atoi(parts[0])
	if err != nil {
		err = parseError
		return
	}
	size, err = strconv.Atoi(parts[1])
	if err != nil {
		err = parseError
		return
	}
	if group > size || group < 1 {
		err = parseError
	}
	return
}
