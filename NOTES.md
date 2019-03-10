## LETARETTE

Naive search cluster that actually works.

NATS + SQLite, server in Golang.
Clients can use any NATS supported environment.

Simple, limited search syntax.

Indexes documents based on free text chunks.
Documents are identified by unique immutable IDs.

Clients implement two roles, _search handler_ and
_document manager_. Due to the bus based nature
these roles can be duplicated for redundancy /
scalability.

### Search Handler

Posts search queries to the cluster, matches responses
(lists of document IDs) to documents and present these
to application layer together with pagination info.

### Document Manager

Responds to indexing requests from the cluster.
All requests are based on document ID and last
updated timestamp.

### Worker Node

Keeps an index updated by periodically requesting document
updates. These requests will be handled by the Document Manager.

Responds to search requests.

Can perform bulk load of index from other nodes.

### Data types

```
Document {
	ID
	updated_at
	alive
	text
}
```

```
Query {
	query_string
	limit
}
```

```
Response {
	Matches [
		{
			ID
			rank
			timestamp
		}
	]
}
```

### Indexing

All workers maintain their own index and communicate with document masters
for updates.

Indexing can use timestamps only to request changed documents, or use a combination
of timestamps and document ids. The first is a simpler algorithm, but can fail
for cases where many documents have the same update date, that are being updated
during indexing.

The second is more complex, but should cover all cases.

Note that lastUpdateTime, listCreationTime and the interest list are all kept
persistently as part of the index database.

Document manager must keep IDs of deleted documents, or simply keep deleted documents.
Reusing IDs is not allowed. For chunking to work, Document Managers must follow strict
document ordering, primarily sorting by timestamp, secondarily on document ID.

Document IDs could be DB row id, uuid or hash.

This algorithm requires cluster clocks to be somewhat in sync.

#### Time - based indexing

Nodes periodically request updates from the document
manager using the following basic algorithm:

0. Set lastUpdateTime = 0
1. Set listCreationTime = now
2. Ask for a limited list of updated documents since (>=) lastUpdateTime, save
	this list as interestList
3. Request updates for the documents on the interest list until all documents
	on the list have been updated.
4. When all documents have been updated, select the most recent update time
	on the list that is less than the listCreationTime and store as the new
	lastUpdateTime. This handles the case where documents on the list are
	updated after the list was requested.
	If lastUpdateTime is unchanged, the next request will ask for the next
	chunk within the period.

The document manager must return documents primarily sorted on update date.
For cases where many documents are created at the exact same time,
chunking will be used (start, count). The document manager must therefore
use consistent ordering within documents of the same timestamp.

#### Time and ID - based indexing

Nodes periodically request updates from the document
manager using the following basic algorithm:

0. Set lastUpdateTime = 0
1. Set listCreationTime = now
2. Ask for a limited list of updated documents after the most recently updated document.
	This document, the limit document, is identified by document ID and timestamp. Save this list as interestList.
	1. If the document manager finds that the limit document is updated since the worker got it, timestamp will be used and documents >= the timestamp will be retrieved
	1. If the worker has received an update for the limit document
3. Request updates for the documents on the interest list until all documents
	on the list have been updated.
4. When all documents have been updated, select the most recent update time on the list
	that is less than the listCreationTime and use that document ID and timestamp
	in the next request.

### Searching

Document collections can be sharded by different nodes as long as
strict ordering can be guaranteed.
