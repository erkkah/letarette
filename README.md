# Letarette - a full-text search thingy

Letarette provides full-text search capabilities to your project with very little setup. You can start out small with a single local index, and if you need, grow to a fully redundant installation with sharded index.

Letarette is a distributed search system that uses [NATS][NATS] for messaging and [SQLite FTS5][FTS5] for the search index.

There is a [client library](pkg/client) for Go as part of the Letarette main project and a separate Typescript/Javascript library, [letarette.js][letarette.js].

Letarette is released under the Apache v2 license.

[Letarette main site](https://letarette.io)

## Getting started

### NATS server

Since NATS is the core messaging component, a NATS server needs to be running for Letarette to function.
If you haven't used NATS before, it is super lightweight and requires no setup to get started: [NATS Server Installation][NATS Installation].

### Letarette service

The Letarette main service handles indexing and search requests.
The service is configured by environment variables, with reasonable defaults.
A Letarette search cluster needs at least one service instance up and running.

Each service instance maintains the index database, which is a local SQLite database.
Services can be launched on multiple nodes within a search cluster, and they will all maintain their own copy of the index.

> Each Letarette cluster divides documents into user defined spaces. An indexed movie database could have one space for "actors" and another for "movies".

To launch a Letarette service using the default space "docs" and reasonable defaults,
just run `./letarette`. 

> Run `./letarette -h` to get help for all environment variables and see default values.

### Document Manager

The "Document Manager" is the component that provides searchable documents to the Letarette service cluster. It listens to index requests and provides the actual documents that will be indexed for searching.

*This is the main integration point for connecting your data to Letarette.*

> For Documents to be indexable by Letarette, they must have `Title` and `Text body` fields for indexing and an `Updated` timestamp field.
Documents should also have an `ID` field that must be unique within the document space and never be reused.

### Search Agent

Searching in Letarette is easy using the client library:

```go
agent, err := client.NewSearchAgent(config.NatsURL)
if err != nil {
    ...
}
defer agent.Close()

spaces := []string{"fruits"}
limit := 10
offset := 0

res, err := agent.Search("apple", spaces, limit, offset)
if err != nil {
    ...
}

for _, doc := range res.Documents {
    fmt.Println(doc.Snippet)
}
```

Letarette search results contain the document ID, rank and a short snippet of the matching text.
Dressing up the result, if needed, is up to the search client implementation.

Read more on the [Letarette main site](https://letarette.io).

[NATS]: https://nats.io
[FTS5]: https://www.sqlite.org/fts5.html
[NATS Installation]: https://nats-io.github.io/docs/nats_server/installation.html
[letarette.js]: https://github.com/erkkah/letarette.js
