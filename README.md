# Letarette - a full-text search thingy

Letarette provides full-text search capabilities to your project with very little setup. You can start out small with a single local index, and if you need, grow to a fully redundant installation with a sharded index.

Letarette is a distributed search system that uses [NATS][NATS] for messaging and [SQLite FTS5][FTS5] for the search index.

There is a [client library](pkg/client) for Go as part of the Letarette main project and a separate Typescript/Javascript library, [letarette.js][letarette.js].

Letarette is released under the Apache v2 license.

[**Letarette main site**][Letarette]

## Getting started

### NATS server

Since NATS is the core messaging component, a NATS server needs to be running for Letarette to function.
NATS it is super lightweight and requires no setup to get started: [NATS Server Installation][NATS Installation].

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

Read more on the [Letarette main site][Letarette].

[Letarette]: https://letarette.io
[NATS]: https://nats.io
[FTS5]: https://www.sqlite.org/fts5.html
[NATS Installation]: https://nats-io.github.io/docs/nats_server/installation.html
[letarette.js]: https://github.com/erkkah/letarette.js
