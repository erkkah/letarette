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

The most recently updated document in the index is idenfified by document ID
and corresponding timestamp. This is the _index position_.

Each index update will move the _index position_ forward until it points to the
most recently updated document known to the document manager.

Index update requests are done in chunks. Each worker has its own list of documents
that it is currently interested in updating, the _interest list_.

Document managers must keep IDs of deleted documents, or just keep deleted documents.
Reusing IDs is not allowed. For chunking to work, Document Managers must follow strict
document ordering, primarily sorting by timestamp, secondarily by document ID.

Document IDs could be DB row id, uuid or hash.

The indexing algorithm requires cluster clocks to be somewhat in sync.

#### The algorithm

Nodes periodically request updates from the document manager using the following basic algorithm:

1. Set _interest list_ creation time to now
2. Ask the document manager for a limited list of updated documents after the current index position. Save this list as interestList. Note that this is simply a list of document IDs.
	1. If the document manager finds that the document at the index position has been updated since the worker got it, only timestamp will be used and documents >= the timestamp will be retrieved. Note that the worker might have received that update as well, so index position timestamp must be kept separately from the document timestamp.
3. Request updates for the documents on the interest list until all documents
	on the list have been updated.
4. When all documents have been updated, select the most recent update time on the list
	that is less than the listCreationTime and use that document ID and timestamp as the new index position.
5. Repeat

### Searching

Document collections can be sharded by different nodes as long as
strict ordering can be guaranteed.

```mermaid
graph TB
	subgraph apa
	A((Hoho))-->B{koko}
	end

	A.->D

	subgraph gurka
	C>Korv]-->D(Fisk)
	end
```
