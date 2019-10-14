all: tinysrv worker lrcli client

worker: generate
	go build -v --tags "fts5" ./cmd/worker

util: generate
	go build -v --tags "fts5" ./cmd/util

tinysrv: client
	go build -v ./cmd/tinysrv

lrcli: client
	go build -v ./cmd/lrcli

client:
	go build -v ./pkg/client

test:
	go test --tags "fts5" github.com/erkkah/letarette/internal/letarette

generate:
	go generate internal/letarette/db.go
	go generate internal/snowball/snowball.go

