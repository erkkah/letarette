# Changelog

## [0.2.0] - 2021-09-01
### Fixed
- Improved sync performance
- Improved cache performance
- Improved error handling
- A bunch of bugs
- Replaced Makefile builds with `bygg` build system to simplify multi-platform builds
### Added
- Indexed documents are optionally zlib-compressed on disk. The index itself is not compressed.
- Preliminary Windows support. Builds and runs! Needs more testing.
- Static build support
- Cloning
  - Index shard worker startup time is greatly reduced by automatically cloning existing shards. If no healthy workers are available for cloning, regular sync will be used.
  - This also simplifies changes in shard configurations.
- Bulk load
  - Index shards can be initialized from (optionally gzipped) JSON - files using the `lrcli` tool
- In-stemmer stop word handling
  - Automatically rejects the most common words in the index.
- Snippet-less, slightly faster, search strategy
- Basic query time synonym handling
  - Synonym lists can be loaded and dumped using `lrcli`
- Metrics + monitoring
  - New `lrmon` tool provides a web interface for monitoring
    clusters, plotting metrics and trying out searches.
- And probably something more!

## [0.1.1] - 2020-01-05
### Fixed
- Version stamping
- Docker build
### Added
- Version logging on startup
- Basic Docker Compose setup

## [0.1.0] - 2020-01-01
### First public release
- Starting off the new decade with making Letarette public

[0.2.0]: https://github.com/erkkah/letarette/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/erkkah/letarette/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/erkkah/letarette/releases/tag/v0.1.0
