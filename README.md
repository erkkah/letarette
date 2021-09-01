
[![GitHub release](https://img.shields.io/github/v/release/erkkah/letarette?include_prereleases&style=for-the-badge)](https://github.com/erkkah/letarette/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/erkkah/letarette?style=for-the-badge)](https://goreportcard.com/report/github.com/erkkah/letarette)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/erkkah/letarette?style=for-the-badge)

# Letarette - a modular full-text search system

Letarette provides full-text search capabilities to your project with very little setup. You can start out small with a single local index, and if you need, grow to a fully redundant installation with a sharded index.

There is a [client library](pkg/client) for Go as part of the Letarette main project and a separate Node.js library, [letarette.js] for easy integration.

Read more on the [**Letarette main site**][Letarette].

## Building

Letarette uses the `bygg` build system, just run `go generate` in the project root to build.

Since Letarette uses cgo, gcc is required for building.
On MacOS and Linux, this is usually not an issue.
On Windows, Letarette builds using `mingw` from Chocolatey:
```sh
choco install mingw
```

[Letarette]: https://letarette.io
[letarette.js]: https://github.com/erkkah/letarette.js
