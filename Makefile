all: worker util client

worker: generate
	go build -v --tags "fts5" ./cmd/worker

util: generate
	go build -v --tags "fts5" ./cmd/util

client:
	go build -v ./pkg/client

test:
	go test --tags "fts5" github.com/erkkah/letarette/internal/letarette

generate:
	go generate internal/letarette/db.go

