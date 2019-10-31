package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/protocol"
)

type entry struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	Compressed []byte
	Date       time.Time `json:"date"`
	alive      bool
}

type ixentry struct {
	date time.Time
	id   int
}

var space = ""
var id = 0
var db = map[int]entry{}
var ix = []ixentry{}

func loadDatabase(objFile string) error {
	var fileReader io.Reader

	file, err := os.Open(objFile)
	if err != nil {
		return err
	}
	defer file.Close()
	fileReader = file

	if strings.HasSuffix(objFile, ".gz") {
		if config.Verbose {
			log.Printf("Reading from compressed file...")
		}
		gzipReader, err := gzip.NewReader(fileReader)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		fileReader = gzipReader
	}

	decoder := json.NewDecoder(fileReader)
	rawSize := 0
	compressedSize := 0

	packer := NewPacker()

	report := func() {
		compressedInfo := ""
		if config.Compress {
			compressedInfo = fmt.Sprintf(", %v MB compressed", compressedSize/1024/1024)
		}

		log.Printf("%v docs, %v MB text loaded%s\n", id, rawSize/1024/1024, compressedInfo)
	}

	for config.NumLimit == 0 || id < config.NumLimit {

		var e entry
		readErr := decoder.Decode(&e)
		if readErr == nil {
			rawSize += len([]byte(e.Text))
			rawSize += len([]byte(e.Title))

			if config.Compress {
				e.Compressed, err = packer.Pack(e.Text)
				if err != nil {
					return err
				}
				compressedSize += len(e.Compressed)
				e.Text = ""
			}

			e.alive = true
			if e.Date.IsZero() {
				e.Date = time.Now()
			}
			db[id] = e
			ix = append(ix, ixentry{
				date: e.Date,
				id:   id,
			})
			id++

			if config.Verbose && id%1000 == 0 {
				report()
			}
		} else {
			if readErr != io.EOF {
				return readErr
			}
			break
		}
	}

	if config.Verbose {
		report()
	}

	return nil
}

func sortIndex() {
	sort.Slice(ix, func(i, j int) bool {
		first := ix[i]
		second := ix[j]
		if first.date.Equal(second.date) {
			return first.id < second.id
		}
		return first.date.Before(second.date)
	})
}

func updateRandomDocument(delete bool) {
	updateIx := rand.Intn(len(ix))
	ixEntry := ix[updateIx]
	updateTime := time.Now()

	msg := "Updating"
	if delete {
		msg = "Deleting"
	}
	log.Printf("%s doc %v @ %v\n", msg, ixEntry.id, updateTime)

	ix[updateIx].date = updateTime

	dbEntry := db[ixEntry.id]
	dbEntry.Date = updateTime
	dbEntry.alive = !delete
	db[ixEntry.id] = dbEntry

	sortIndex()
}

func fetchInitial(limit uint16) []protocol.DocumentID {
	result := []protocol.DocumentID{}
	for _, v := range ix[:limit] {
		result = append(result, protocol.DocumentID(strconv.Itoa(v.id)))
	}
	return result
}

func fetchByTime(startTime time.Time, limit uint16) []protocol.DocumentID {
	startIndex := sort.Search(len(ix), func(i int) bool {
		return !ix[i].date.Before(startTime)
	})

	result := []protocol.DocumentID{}
	end := min(len(ix), startIndex+int(limit))
	for _, v := range ix[startIndex:end] {
		result = append(result, protocol.DocumentID(strconv.Itoa(v.id)))
	}
	return result
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func fetchByReference(afterDocument protocol.DocumentID, fromTime time.Time, limit uint16) []protocol.DocumentID {
	startIndex := sort.Search(len(ix), func(i int) bool {
		return !ix[i].date.Before(fromTime)
	})

	subIndex := ix[startIndex:]
	numID, _ := strconv.Atoi(string(afterDocument))
	docIndex := -1
	for i, v := range subIndex {
		if v.id == numID {
			docIndex = i + 1
			break
		}
	}
	if docIndex == -1 {
		log.Printf("Could not find entry % v in fetchByRerefence\n", numID)
		return []protocol.DocumentID{}
	}

	result := []protocol.DocumentID{}
	end := min(len(subIndex), docIndex+int(limit))
	for _, v := range subIndex[docIndex:end] {
		result = append(result, protocol.DocumentID(strconv.Itoa(v.id)))
	}
	return result
}

var updateFreq = 0 * time.Second
var lastUpdate = time.Now()

var deleteFreq = 0 * time.Second
var lastDelete = time.Now()

func handleIndexRequest(req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	if req.Space != space {
		return protocol.IndexUpdate{}, fmt.Errorf("Space %v not in db", req.Space)
	}

	if updateFreq != 0 && time.Now().After(lastUpdate.Add(updateFreq)) {
		lastUpdate = time.Now()
		updateRandomDocument(false)
	}

	if deleteFreq != 0 && time.Now().After(lastDelete.Add(deleteFreq)) {
		lastDelete = time.Now()
		updateRandomDocument(true)
	}

	updates := []protocol.DocumentID{}

	if req.AfterDocument == "" {
		// Initial index fetch
		updates = fetchInitial(req.Limit)
	} else {
		log.Printf("Index request, after %v, from %v", req.AfterDocument, req.FromTime)
		entryID, err := strconv.Atoi(string(req.AfterDocument))
		if err != nil {
			return protocol.IndexUpdate{}, fmt.Errorf("Invalid document ID: %v", req.AfterDocument)
		}
		refEntry, found := db[entryID]
		if !found {
			// invalid index state, log
			log.Printf("Unexpected index state, doc %v not found\n", req.AfterDocument)
			updates = fetchByTime(req.FromTime, req.Limit)
		} else {
			if refEntry.Date.After(req.FromTime) {
				// entry updated, only use date
				updates = fetchByTime(req.FromTime, req.Limit)
			} else if refEntry.Date.Before(req.FromTime) {
				log.Printf("Unexpected index state doc %v@%v has index time %v\n",
					req.AfterDocument, refEntry.Date.String(), req.FromTime.String())
				updates = fetchByTime(req.FromTime, req.Limit)
			} else {
				// use ref entry
				updates = fetchByReference(req.AfterDocument, req.FromTime, req.Limit)
			}
		}
	}
	return protocol.IndexUpdate{
		Space:   space,
		Updates: updates,
	}, nil
}

func entryToDocument(id protocol.DocumentID, e entry) (protocol.Document, error) {
	doc := protocol.Document{
		ID:      id,
		Updated: e.Date,
		Alive:   e.alive,
	}
	if e.alive {
		var text string
		if config.Compress {
			packer := NewPacker()
			var err error
			text, err = packer.Unpack(e.Compressed)
			if err != nil {
				return doc, err
			}
		} else {
			text = e.Text
		}
		doc.Text = e.Title + "\n" + text
	}
	return doc, nil
}

func deadDocument(id protocol.DocumentID) protocol.Document {
	return protocol.Document{
		ID:    id,
		Alive: false,
	}
}

func handleDocumentRequest(req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
	if req.Space != space {
		return protocol.DocumentUpdate{}, fmt.Errorf("Space %v not in db", req.Space)
	}

	docs := []protocol.Document{}
	for _, v := range req.Wanted {
		entryID, _ := strconv.Atoi(string(v))
		doc, found := db[entryID]

		if config.Verbose {
			log.Printf("Found doc %v from %v\n", entryID, doc.Date.String())
		}

		if found {
			entry, err := entryToDocument(v, doc)
			if err != nil {
				return protocol.DocumentUpdate{}, err
			}
			docs = append(docs, entry)
		} else {
			docs = append(docs, deadDocument(v))
		}
	}
	return protocol.DocumentUpdate{
		Space:     space,
		Documents: docs,
	}, nil
}

var config struct {
	Space      string        `docopt:"<space>"`
	DBFile     string        `docopt:"<dbfile>"`
	NatsURL    string        `docopt:"-n"`
	UpdateFreq time.Duration `docopt:"-u"`
	DeleteFreq time.Duration `docopt:"-d"`
	Verbose    bool          `docopt:"-v"`
	Compress   bool          `docopt:"-c"`
	Limit      string        `docopt:"-l"`
	NumLimit   int
}

func main() {
	usage := `Tiny JSON document server.

Usage:
    tinysrv [-n <url>] [-l <limit>] [-u <secs>] [-d <secs>] [-c] [-v] <space> <dbfile>

Options:
    -n <url>    NATS url to connect to [default: nats://localhost:4222]
    -l <limit>  Max number of documents to load [default: unlimited]
    -u <secs>   Auto-update random documents every SECS second
    -d <secs>   Auto-delete random documents every SECS second
    -c          Compress text in memory
    -v          Verbose
`

	args, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Panicf("Failed to parse args: %v", err)
	}
	err = args.Bind(&config)
	if err != nil {
		log.Panicf("Failed to bind args: %v", err)
	}

	if config.Limit != "unlimited" {
		config.NumLimit, _ = strconv.Atoi(config.Limit)
	}

	space = config.Space
	dbFile := config.DBFile

	if config.UpdateFreq != 0 {
		updateFreq = config.UpdateFreq * time.Second
		log.Printf("Auto-updating every %v\n", updateFreq)
	}

	if config.DeleteFreq != 0 {
		deleteFreq = config.DeleteFreq * time.Second
		log.Printf("Auto-deleting every %v\n", deleteFreq)
	}

	log.Println("Loading...")
	err = loadDatabase(dbFile)
	if err != nil {
		log.Panicf("Failed to load db: %v", err)
	}

	log.Println("Sorting...")
	sortIndex()

	log.Printf("%v items loaded", len(db))

	ehandler := func(err error) {
		log.Printf("%v\n", err)
	}
	mgr, err := client.StartDocumentManager(config.NatsURL, client.WithErrorHandler(ehandler))
	if err != nil {
		log.Panicf("Failed to start document manager: %v", err)
	}
	defer mgr.Close()

	mgr.StartIndexRequestHandler(handleIndexRequest)
	mgr.StartDocumentRequestHandler(handleDocumentRequest)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	select {
	case s := <-signals:
		log.Printf("Received signal %v, exiting", s)
	}
}
