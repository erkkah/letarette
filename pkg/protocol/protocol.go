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
	StartTime     time.Time
	StartDocument DocumentID
	Limit         uint16
}

// IndexUpdate is a list of updated IDs, sent in response to
// the IndexUpdateRequest above.
type IndexUpdate struct {
	Space   string
	Updates []DocumentID
}

// Document is the representation of a searchable item
type Document struct {
	Space   string
	ID      DocumentID
	Updated time.Time
	Text    string
	Alive   bool
}

// DocumentUpdate is sent in response to DocumentRequest
type DocumentUpdate struct {
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
}

// SearchResult represents one search hit
type SearchResult struct {
	ID         DocumentID
	Snippet    string
	MatchStart int
	MatchEnd   int
	Rank       float32
}

// SearchResponse is sent in response to SearchRequest
type SearchResponse struct {
	Documents []SearchResult
}
