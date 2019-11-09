package protocol

import (
	"time"
)

// DocumentID is just a string, could be uuid, hash, numeric, et.c.
type DocumentID string

// IndexStatus is sent in response to "index.status" requests
type IndexStatus struct {
	DocCount uint64
	// Out of sync info here?
	LastUpdate time.Time
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
	Spaces []string
	Query  string
	Limit  uint16
	Offset uint16
}

// SearchResult is a collection of search hits.
// When Capped is true, the search was truncated at Config.Search.Cap.
// Capped results are only locally sorted by rank.
type SearchResult struct {
	Hits      []SearchHit
	Capped    bool
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
	SearchStatusIndexHit SearchStatusCode = iota + 42
	SearchStatusCacheHit
	SearchStatusNoHit
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
	return strings[ssc]
}

// SearchResponse is sent in response to SearchRequest
type SearchResponse struct {
	Result   SearchResult
	Duration float32
	Status   SearchStatusCode
}
