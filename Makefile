all: letarette lrcli lrload lrmon

STAMP := $(shell ./stamp.sh github.com/erkkah/letarette)

GO_SQLITE := $(shell go list -f '{{.Dir}}' github.com/mattn/go-sqlite3)

export CGO_CFLAGS := $(shell go env CGO_CFLAGS)
CGO_CFLAGS += -I$(GO_SQLITE)
CGO_CFLAGS += -DSQLITE_OMIT_LOAD_EXTENSION -DSQLITE_OMIT_SHARED_CACHE -DSQLITE_USE_ALLOCA

ifeq ($(OS),Windows_NT)
EXE=.exe
endif

ifdef STATIC
LDFLAGS := -linkmode external -extldflags -static
endif

SQLITE_TAGS := fts5,sqlite_omit_load_extension

letarette: generate snowball
	go build -ldflags="$(STAMP) $(LDFLAGS)" -mod=readonly -v -tags "$(SQLITE_TAGS)" -o letarette$(EXE) ./cmd/worker

lrcli: client snowball
	go build -ldflags="$(STAMP) $(LDFLAGS)" -mod=readonly -v -tags "$(SQLITE_TAGS),dbstats" ./cmd/lrcli

tinysrv: client
	go build -ldflags="$(LDFLAGS)" -mod=readonly -v ./cmd/tinysrv

lrload: client
	go build -ldflags="$(LDFLAGS)" -mod=readonly -v ./cmd/lrload

lrmon: client
	go generate -tags "prod" ./cmd/lrmon
	go build -ldflags="$(STAMP) $(LDFLAGS)" -v -tags "prod" ./cmd/lrmon

client:
	go build -v ./pkg/client

SNOWBALL := internal/snowball/snowball

snowball: $(SNOWBALL)/libstemmer.o

$(SNOWBALL)/libstemmer.o: $(SNOWBALL)/README
	$(MAKE) -C $(SNOWBALL) libstemmer.o

$(SNOWBALL)/README:
	git submodule init && git submodule update --recursive

.PHONY: test
test: generate
	go test -tags "$(SQLITE_TAGS)" ./internal/letarette ./pkg/*

generate:
	go generate internal/letarette/db.go

clean:
	go clean github.com/erkkah/letarette/...

tidy:
	touch $(SNOWBALL)/go.mod
	go mod tidy
	rm $(SNOWBALL)/go.mod