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
2:	Save the list as interestList and set lastUpdateTime to the date
	of the most recently updated document in the list, as long as that date
	is less than listCreationTime. This handles documents that were updated
	since the list request was made.
3:	Request updates for the documents on the interest list until all documents
	on the list have been updated. Note that other workers might ask for updates
	we are waiting for, thus limiting the amount of duplicate requests.
	All incoming document updates are handled, even if they were requested
	from other workers.
4:	When the interest list has been served, go to 1

Note that lastUpdateTime, listCreationTime and the interest list are all kept
persistently as part of the index database.

This algorithm requires cluster clocks to be somewhat in sync.

The document provider must return documents sorted on update date.

*/

// IndexStatus is sent in response to "index.status" requests
type IndexStatus struct {
	DocCount   int64
	LastUpdate time.Time
}

// DocumentUpdateRequest is a request for available updates.
// Returns up to 'Limit' document IDs, updated at or later than
// the specified time.
type DocumentUpdateRequest struct {
	UpdateStart time.Time
	Limit       int16
}

// DocumentUpdate is a list of updated IDs
type DocumentUpdate struct {
	Updates []int64
	// LastUpdate is the timestamp of the last update in the returned ID range
	LastUpdate time.Time
}

// Document is the representation of a searchable item
type Document struct {
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
// Returned documents are broadcast to all workers.
type DocumentRequest struct {
	Wanted []int64
}
