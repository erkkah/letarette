# Letarette

If all you need is a scalable, simple, understandable full text search engine - this it it!
Letarette is easy to set up and integrate with your data.

If you need customizable stemming, suggestions, structured documents, et.c. then Letarette might not be for you.
There are several well-known alternatives there, like Elasticsearch, Solr and friends.

## Overview

Letarette is a distributed search engine that uses [NATS][NATS] for messaging and [SQLite FTS5][FTS5] for the search index.
The Letarette service is a single binary written in Go. Clients can be written in any language that has NATS
support, currently there is a client library for Go as part of the Letarette main project.

## Getting started

### NATS server
Since NATS is the core messaging component, a NATS server needs to be running for Letarette to function. If you haven't used NATS before, it is super lightweight and requires no setup to get started: [NATS Server Installation][NATS Installation].

### Worker
The Letarette work horse is called the "worker", and handles both indexing and search requests.
The worker is configured by environment variables, with reasonable defaults. A Letarette search cluster needs at least one worker instance up and running.

The worker maintains the index database, which is a local SQLite database. Workers can be launched on multiple nodes within a search cluster, and they will all maintain their own copy of the index.

> Each Letarette cluster divides documents into different user defined spaces. An indexed movie database could have one space for "actors" and another for "movies".

To launch the worker, you just need to set the spaces it should index:

```sh
LETARETTE_INDEX_SPACES=fruits worker
```

### Document Manager
The "Document Manager" is the component that provides searchable documents to the worker. It listens to index requests from the cluster workers and provides the actual document text that will be indexed for searching.

> For Documents to be indexable in Letarette, they must have a unique `ID`, a `Text` field for indexing and an `Updated` timestamp field. The `ID` must be unique within the document space and never be reused. Deleted documents are marked as such instead of being removed.

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
