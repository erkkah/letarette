all: letarette lrcli lrload

STAMP := $(shell ./stamp.sh github.com/erkkah/letarette/internal/letarette)

ifdef STATIC
LDFLAGS := -linkmode external -extldflags -static
endif

export CGO_CFLAGS := -DSQLITE_OMIT_LOAD_EXTENSION -DSQLITE_OMIT_SHARED_CACHE -DSQLITE_USE_ALLOCA

letarette: generate snowball
	go build -ldflags="$(STAMP) $(LDFLAGS)" -v -tags "fts5,sqlite_omit_load_extension" -o letarette ./cmd/worker

lrcli: client snowball
	go build -ldflags="$(STAMP) $(LDFLAGS)" -v -tags "fts5,dbstats,sqlite_omit_load_extension" ./cmd/lrcli

tinysrv: client
	go build -ldflags="$(LDFLAGS)" -v ./cmd/tinysrv

lrload: client
	go build -ldflags="$(LDFLAGS)" -v ./cmd/lrload

client:
	go build -v ./pkg/client

SNOWBALL := internal/snowball/snowball

snowball: $(SNOWBALL)/libstemmer.o

$(SNOWBALL)/libstemmer.o: $(SNOWBALL)/README
	$(MAKE) -C $(SNOWBALL) libstemmer.o

$(SNOWBALL)/README:
	git submodule init && git submodule update --recursive

test:
	go test -tags "fts5" ./internal/letarette ./pkg/*

generate:
	go generate internal/letarette/db.go
