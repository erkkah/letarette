all: letarette lrcli lrload

STAMP := $(shell ./stamp.sh github.com/erkkah/letarette/internal/letarette)

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

letarette: generate snowball sqlite.a
	go build -ldflags="$(STAMP) $(LDFLAGS)" -v -tags "$(SQLITE_TAGS)" -o letarette$(EXE) ./cmd/worker

lrcli: client snowball
	go build -ldflags="$(STAMP) $(LDFLAGS)" -v -tags "$(SQLITE_TAGS),dbstats" ./cmd/lrcli

tinysrv: client
	go build -ldflags="$(LDFLAGS)" -v ./cmd/tinysrv

lrload: client
	go build -ldflags="$(LDFLAGS)" -v ./cmd/lrload

client:
	go build -v ./pkg/client

sqlite.a: go.mod
	go build -buildmode archive -o sqlite.a -tags "$(SQLITE_TAGS)" github.com/mattn/go-sqlite3
	ranlib sqlite.a

SNOWBALL := internal/snowball/snowball

snowball: $(SNOWBALL)/libstemmer.o

$(SNOWBALL)/libstemmer.o: $(SNOWBALL)/README
	$(MAKE) -C $(SNOWBALL) libstemmer.o

$(SNOWBALL)/README:
	git submodule init && git submodule update --recursive

test:
	go test -tags "$(SQLITE_TAGS)" ./internal/letarette ./pkg/*

generate:
	go generate internal/letarette/db.go

clean:
	go clean -i -r github.com/erkkah/letarette/... && rm sqlite.a
