all: worker util

worker: generate
	go build -v --tags "fts5" ./cmd/worker

util: generate
	go build -v --tags "fts5" ./cmd/util

test:
	go test --tags "fts5" github.com/erkkah/letarette/internal/letarette

generate:
	go generate internal/letarette/db.go

