package letarette

import (
	"time"
)

/*

All workers maintain their own index and communicate with document masters
for updates.

The update algorithm follows this pattern:

0:	Set lastUpdateTime = 0, listCreationTime = now
1:	Ask for a limited list of updated documents since (>=) lastUpdateTime
2:	Save this list as interestList
3:	Request updates for the documents on the interest list until all documents
	on the list have been updated.
4:	When all documents have been updated, select the most recent update time
	on the list that is less than the listCreationTime and store as the new
	lastUpdateTime. This handles the case where documents on the list are
	updated after the list was requested.
	If lastUpdateTime is unchanged, the next request will ask for the next
	chunk within the period.
	Note that other workers might ask for updates we are waiting for, thus
	limiting the amount of duplicate requests.
	All incoming document updates are handled, even if they were requested
	from other workers.
5:	When the interest list has been served, go to 1

Note that lastUpdateTime, listCreationTime and the interest list are all kept
persistently as part of the index database.

This algorithm requires cluster clocks to be somewhat in sync.

The document provider must return documents primarily sorted on update date.
For cases where many documents are created at the exact same time,
chunking will be used (start, count). The document provider must therefore
use consistent ordering within documents of the same timestamp.
*/

// IndexStatus is sent in response to "index.status" requests
type IndexStatus struct {
	DocCount int64
	// Out of sync info here?
	LastUpdate time.Time
}

// DocumentUpdateRequest is a request for available updates.
// Returns up to 'Limit' document IDs, updated at or later than
// the specified time.
type DocumentUpdateRequest struct {
	Space       string
	UpdateStart time.Time
	Start       int64
	Limit       int16
}

// DocumentUpdate is a list of updated IDs
type DocumentUpdate struct {
	Space   string
	Updates []int64
	// LastUpdate is the timestamp of the last update in the returned ID range
	LastUpdate time.Time
}

// Document is the representation of a searchable item
type Document struct {
	Space   string
	ID      int64
	Updated time.Time
	Text    string
	Alive   bool
}

// DocumentResponse is sent in response to DocumentRequest
type DocumentResponse struct {
	Documents []Document
}

// DocumentRequest is a request for a list of documents.
// Returned documents are broadcasted to all workers.
type DocumentRequest struct {
	Space  string
	Wanted []int64
}
