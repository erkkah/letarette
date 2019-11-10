package letarette

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/erkkah/letarette/pkg/protocol"
)

var shardHasher = fnv.New32a()

func shardIndexFromDocumentID(docID protocol.DocumentID, shardGroupSize int) int {
	shardHasher.Reset()
	shardHasher.Write([]byte(docID))
	sum := shardHasher.Sum(nil)
	intPart := binary.BigEndian.Uint32(sum)
	return int(intPart % uint32(shardGroupSize))
}
