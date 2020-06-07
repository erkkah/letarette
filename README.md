
[![GitHub release](https://img.shields.io/github/release/erkkah/letarette.svg)](https://github.com/erkkah/letarette/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/erkkah/letarette)](https://goreportcard.com/report/github.com/erkkah/letarette)

# Letarette - a full-text search thingy

Letarette provides full-text search capabilities to your project with very little setup. You can start out small with a single local index, and if you need, grow to a fully redundant installation with a sharded index.

There is a [client library](pkg/client) for Go as part of the Letarette main project and a separate Node.js library, [letarette.js][letarette.js] for easy integration.

Read more on the [**Letarette main site**][Letarette].


## Building

Since Letarette uses cgo, gcc is required for building.
On MacOS and Linux, this is usually not an issue.
On Windows, Letarette test builds use `mingw` from Chocolatey:
```sh
choco install mingw
```

Letarette requires go 1.13 or above to build.
Letarette uses the `bygg` build system, just run `go generate` in the project root.

[Letarette]: https://letarette.io
[NATS]: https://nats.io
[letarette.js]: https://github.com/erkkah/letarette.js
