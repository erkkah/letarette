# Letarette

If all you need is a scalable, simple, understandable full text search engine - this it it!
Letarette is easy to set up and integrate with your data.

If you need scriptable stemming, suggestions, structured documents, et.c. then Letarette might not be for you.
There are several well-known alternatives there, like Elasticsearch, Solr and friends.

## Overview

Letarette is a distributed search engine that uses [NATS][NATS] for messaging and [SQLite FTS5][FTS5] for the search index.
The Letarette service is a single binary written in Go. Clients can be written in any language that has NATS
support. Currently there is a client library for Go as part of the Letarette main project.

## Getting started

### NATS server
Since NATS is the core messaging component, a NATS server needs to be running for Letarette to function. If you haven't used NATS before, it is super lightweight and requires no setup to get started: [NATS Server Installation][NATS Installation].

### Worker
The Letarette main service, the "worker", handles indexing and search requests.
The worker is configured by environment variables, with reasonable defaults. A Letarette search cluster needs at least one worker instance up and running.

The worker maintains the index database, which is a local SQLite database. Workers can be launched on multiple nodes within a search cluster, and they will all maintain their own copy of the index.

> Each Letarette cluster divides documents into different user defined spaces. An indexed movie database could have one space for "actors" and another for "movies".

To launch the worker using the default space "docs" and reasonable default, just run `./worker`. 

> Run `./worker -h` to get help for all environment variables and see default values.

### Document Manager
The "Document Manager" is the component that provides searchable documents to the worker. It listens to index requests from the cluster workers and provides the actual document text that will be indexed for searching.

> For Documents to be indexable in Letarette, they must have a unique `ID`, a `Text` field for indexing and an `Updated` timestamp field. The `ID` must be unique within the document space and never be reused. Deleted documents are marked as such instead of being removed.

Document managers respond to two different types of requests, _index update requests_ and _document requests_.

Index update requests asks for a list of documents updated at or after a given document or timestamp. That document is the most recent document in the requesting worker's index. If that document has been updated since the worker got it, the document manager must use the timestamp instead.

By using a strict ordering of update timestamps primarily and document ID secondarily, this update scheme makes sure that the distributed indexes can be kept up to date even in situations of communication disruption or downtime.

Some overfetching might occur.

After a worker has received an index update response, it will start to request the documents on the list. Document updates are broadcast to all workers, meaning that workers often will have documents that they need based on their last index update response.

### Search client
Searching in Letarette is easiest done by using the search client library:

```go
c, err := client.NewSearchClient(config.NatsURL)
if err != nil {
    ...
}
defer c.Close()

spaces := []string{"fruits"}
limit := 10
offset := 0

res, err := c.Search("apple", spaces, limit, offset)
if err != nil {
    ...
}

for _, doc := range res.Documents {
    fmt.Println(doc.Snippet)
}
```

Letarette search results contain the document `ID`, rank and a snippet including the matching phrases. Dressing up the result is up to the search client implementation.

## Test setup

_Describe tinysrv test setup here!_

### Query language

TBD

## Details

All messages are JSON-encoded and uses the following topics.
The top NATS topic is configurable and defaults to "leta".

- leta.q:
    >Query request. Sent from search client and is distributed within a configured queue group.
- leta.status:
    >Worker status request. All workers respond with their individual status.
- leta.index.request:
    >Index update request. Sent from worker to document manager to get a list of updates.
- leta.document.request:
    >Document request. Sent from worker to get documents.
- leta.document.update:
    >Sent in response to document requests.

See [protocol.go](pkg/protocol/protocol.go) for messaging details.

[NATS]: https://nats.io
[FTS5]: https://www.sqlite.org/fts5.html
[NATS Installation]: https://nats-io.github.io/docs/nats_server/installation.html
