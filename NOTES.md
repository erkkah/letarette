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

Nodes periodically request updates from the document
manager using the following basic algorithm:

1. Set event horizon to now
2. Request documents up until horizon ordered by timestamp
3. Chunk by last received document ID and timestamp.
4. When last document received is same as the most
	recent we had before chunk - we are done!


Document manager must keep IDs of deleted documents, or
simply keep deleted documents.
Reusing IDs is not allowed.
For chunking to work, Document Managers must
follow strict document ordering.

Document IDs could be DB row id, uuid or hash.

Node: If document manager gets chunk document ID for a
document that is updated after horizon - it must use
timestamp instead.

This is why chunking must provide both document ID _and_
timestamp! Some overfetching will occur.

Since index requests are time bound - other nodes can
respond, possibly offloading the document manager.


### Searching
Document collections can be sharded by different nodes as long as
strict ordering can be guaranteed.
