all: tinysrv worker lrcli client

worker: generate snowball
	go build -v -tags "fts5" ./cmd/worker

tinysrv: client
	go build -v ./cmd/tinysrv

lrcli: client snowball
	go build -v -tags "fts5,dbstats" ./cmd/lrcli

client:
	go build -v ./pkg/client

SNOWBALL := internal/snowball/snowball

snowball: $(SNOWBALL)/libstemmer.o

$(SNOWBALL)/libstemmer.o: $(SNOWBALL)/README
	$(MAKE) -C $(SNOWBALL)

$(SNOWBALL)/README:
	git submodule init && git submodule update --recursive

test:
	go test -tags "fts5" ./internal/letarette ./pkg/*

generate:
	go generate internal/letarette/db.go
