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

// Package protocol describes the wire protocol between workers,
// document managers and search clients.
// The actual NATS messages are JSON-encoded versions of the
// types defined here.
package protocol

import (
	"fmt"
	"time"
)

// Version of the wire protocol
var Version = Semver{0, 5, 0}

// DocumentID is just a string, could be uuid, hash, numeric, et.c.
type DocumentID string

// IndexStatusCode is what is says
type IndexStatusCode uint8

// Codes returned in index status updates.
// There is only one status level at any given time, with higher
// valued codes overriding lower valued codes.
const (
	IndexStatusInSync IndexStatusCode = iota + 72
	IndexStatusStartingUp
	IndexStatusSyncing
	IndexStatusIncompleteShardgroup
	IndexStatusIncompatible
)

func (isc IndexStatusCode) String() string {
	strings := map[IndexStatusCode]string{
		IndexStatusInSync:               "in sync",
		IndexStatusStartingUp:           "starting up",
		IndexStatusSyncing:              "syncing",
		IndexStatusIncompleteShardgroup: "incomplete shard group",
		IndexStatusIncompatible:         "incompatible protocol versions",
	}
	str, found := strings[isc]
	if !found {
		return fmt.Sprintf("unknown (%d)", isc)
	}
	return str
}

// IndexStatus is regularly broadcast from all workers
type IndexStatus struct {
	IndexID        string
	Version        string
	DocCount       uint64
	LastUpdate     time.Time
	ShardgroupSize uint16
	Shardgroup     uint16
	Status         IndexStatusCode
}

func (status IndexStatus) String() string {
	return fmt.Sprintf("Index@%s(%d/%d): %d docs, last update: %v, status: %v",
		status.IndexID, status.Shardgroup+1, status.ShardgroupSize,
		status.DocCount, status.LastUpdate, status.Status)
}

// IndexUpdateRequest is a request for available updates.
// Returns up to 'Limit' document IDs, updated at or later than
// the specified document or timestamp.
type IndexUpdateRequest struct {
	Space         string
	FromTime      time.Time
	AfterDocument DocumentID
	Limit         uint16
}

// DocumentReference corresponds to one document at one point in time
type DocumentReference struct {
	ID      DocumentID
	Updated time.Time
}

// IndexUpdate is a list of updated documents, sent in response to
// the IndexUpdateRequest above.
type IndexUpdate struct {
	Space   string
	Updates []DocumentReference
}

// Document is the representation of a searchable item
type Document struct {
	ID      DocumentID
	Updated time.Time
	Title   string
	Text    string
	Alive   bool
}

// DocumentUpdate is sent in response to DocumentRequest
type DocumentUpdate struct {
	Space     string
	Documents []Document
}

// DocumentRequest is a request for a list of documents.
// Returned documents are broadcasted to all workers.
type DocumentRequest struct {
	Space  string
	Wanted []DocumentID
}

// SearchRequest is sent from a search handler to search the index.
type SearchRequest struct {
	// Spaces to search
	Spaces []string
	// Query string in letarette syntax
	Query string
	// Maximum number of hits returned in one page
	PageLimit uint16
	// Zero-indexed page of hits to retrieve
	PageOffset uint16
	// When true, spelling mistakes are "fixed"
	// and the resulting query is automatically performed.
	// In either case, spell-fixed queries are returned
	// in the SearchResult Respelt field.
	Autocorrect bool
}

// SearchResult is a collection of search hits
type SearchResult struct {
	Hits []SearchHit
	// When true, the search was truncated
	// Capped results are only locally sorted by rank
	Capped bool
	// When not empty, the original query had no matches,
	// and this is a respelt version of the query
	Respelt string
	// The total number of hits to the given query
	TotalHits int
}

// SearchHit represents one search hit
type SearchHit struct {
	Space   string
	ID      DocumentID
	Snippet string
	Rank    float32
}

// SearchStatusCode is what is says
type SearchStatusCode uint8

// Codes returned in search responses
const (
	SearchStatusNoHit SearchStatusCode = iota + 42
	SearchStatusCacheHit
	SearchStatusIndexHit
	SearchStatusTimeout
	SearchStatusQueryError
	SearchStatusServerError
)

func (ssc SearchStatusCode) String() string {
	strings := map[SearchStatusCode]string{
		SearchStatusIndexHit:    "found in index",
		SearchStatusCacheHit:    "found in cache",
		SearchStatusNoHit:       "not found",
		SearchStatusTimeout:     "timeout",
		SearchStatusQueryError:  "query format error",
		SearchStatusServerError: "server error",
	}
	str, found := strings[ssc]
	if !found {
		return fmt.Sprintf("unknown (%d)", ssc)
	}
	return str
}

// SearchResponse is sent in response to SearchRequest
type SearchResponse struct {
	Result   SearchResult
	Duration float32
	Status   SearchStatusCode
}
