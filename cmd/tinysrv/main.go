// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
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

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/pennant"
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

func loadDatabase(config Config, objFile string) error {
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
			if !errors.Is(readErr, io.EOF) {
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

func fetchInitial(limit uint16) []protocol.DocumentReference {
	result := []protocol.DocumentReference{}
	if int(limit) > len(ix) {
		limit = uint16(len(ix))
	}
	for _, v := range ix[:limit] {
		result = append(result, protocol.DocumentReference{
			ID:      protocol.DocumentID(strconv.Itoa(v.id)),
			Updated: v.date,
		})
	}
	return result
}

func fetchByTime(startTime time.Time, limit uint16) []protocol.DocumentReference {
	startIndex := sort.Search(len(ix), func(i int) bool {
		return !ix[i].date.Before(startTime)
	})

	result := []protocol.DocumentReference{}
	end := min(len(ix), startIndex+int(limit))
	for _, v := range ix[startIndex:end] {
		result = append(result, protocol.DocumentReference{
			ID:      protocol.DocumentID(strconv.Itoa(v.id)),
			Updated: v.date,
		})
	}
	return result
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func fetchByReference(
	afterDocument protocol.DocumentID, fromTime time.Time, limit uint16,
) []protocol.DocumentReference {

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
		return []protocol.DocumentReference{}
	}

	result := []protocol.DocumentReference{}
	end := min(len(subIndex), docIndex+int(limit))
	for _, v := range subIndex[docIndex:end] {
		result = append(result, protocol.DocumentReference{
			ID:      protocol.DocumentID(strconv.Itoa(v.id)),
			Updated: v.date,
		})
	}
	return result
}

var updateFreq = 0 * time.Second
var lastUpdate = time.Now()

var deleteFreq = 0 * time.Second
var lastDelete = time.Now()

func handleIndexRequest(ctx context.Context, req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	if req.Space != space {
		return protocol.IndexUpdate{}, fmt.Errorf("space %v not in db", req.Space)
	}

	if updateFreq != 0 && time.Now().After(lastUpdate.Add(updateFreq)) {
		lastUpdate = time.Now()
		updateRandomDocument(false)
	}

	if deleteFreq != 0 && time.Now().After(lastDelete.Add(deleteFreq)) {
		lastDelete = time.Now()
		updateRandomDocument(true)
	}

	var updates []protocol.DocumentReference

	if req.AfterDocument == "" {
		// Initial index fetch
		updates = fetchInitial(req.Limit)
	} else {
		log.Printf("Index request, after %v, from %v", req.AfterDocument, req.FromTime)
		entryID, err := strconv.Atoi(string(req.AfterDocument))
		if err != nil {
			return protocol.IndexUpdate{}, fmt.Errorf("invalid document ID: %v", req.AfterDocument)
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

func entryToDocument(id protocol.DocumentID, e entry, compress bool) (protocol.Document, error) {
	doc := protocol.Document{
		ID:      id,
		Updated: e.Date,
		Alive:   e.alive,
	}
	if e.alive {
		var text string
		if compress {
			packer := NewPacker()
			var err error
			text, err = packer.Unpack(e.Compressed)
			if err != nil {
				return doc, err
			}
		} else {
			text = e.Text
		}
		doc.Title = e.Title
		doc.Text = text
	}
	return doc, nil
}

func deadDocument(id protocol.DocumentID) protocol.Document {
	return protocol.Document{
		ID:    id,
		Alive: false,
	}
}

func handleDocumentRequest(
	ctx context.Context, config Config, req protocol.DocumentRequest,
) (protocol.DocumentUpdate, error) {
	if req.Space != space {
		return protocol.DocumentUpdate{}, fmt.Errorf("space %v not in db", req.Space)
	}

	start := time.Now()
	docs := []protocol.Document{}
	for _, v := range req.Wanted {
		entryID, _ := strconv.Atoi(string(v))
		doc, found := db[entryID]

		if config.Verbose {
			log.Printf("Found doc %v from %v\n", entryID, doc.Date.String())
		}

		if found {
			entry, err := entryToDocument(v, doc, config.Compress)
			if err != nil {
				return protocol.DocumentUpdate{}, err
			}
			docs = append(docs, entry)
		} else {
			docs = append(docs, deadDocument(v))
		}
	}
	passed := time.Since(start)
	if config.Verbose {
		log.Printf("Found %v docs in %s", len(docs), passed)
	}
	return protocol.DocumentUpdate{
		Space:     space,
		Documents: docs,
	}, nil
}

// Config holds the commandline config
type Config struct {
	Space      string `arg:"0"`
	DBFile     string `arg:"1"`
	NatsURL    string `name:"n"`
	UpdateFreq int64  `name:"u"`
	DeleteFreq int64  `name:"d"`
	Verbose    bool   `name:"v"`
	Compress   bool   `name:"c"`
	Limit      string `name:"l"`
	NumLimit   int    `name:""`
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
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	var config Config
	pennant.MustParse(&config, os.Args[1:])

	if config.Limit != "unlimited" {
		config.NumLimit, _ = strconv.Atoi(config.Limit)
	}

	space = config.Space
	dbFile := config.DBFile

	if config.UpdateFreq != 0 {
		updateFreq = time.Duration(config.UpdateFreq * int64(time.Second))
		log.Printf("Auto-updating every %v\n", updateFreq)
	}

	if config.DeleteFreq != 0 {
		deleteFreq = time.Duration(config.DeleteFreq * int64(time.Second))
		log.Printf("Auto-deleting every %v\n", deleteFreq)
	}

	log.Println("Loading...")
	err := loadDatabase(config, dbFile)
	if err != nil {
		log.Panicf("Failed to load db: %v", err)
	}

	log.Println("Sorting...")
	sortIndex()

	log.Printf("%v items loaded", len(db))

	ehandler := func(err error) {
		log.Printf("%v\n", err)
	}
	mgr, err := client.StartDocumentManager([]string{config.NatsURL}, client.WithErrorHandler(ehandler))
	if err != nil {
		log.Panicf("Failed to start document manager: %v", err)
	}
	defer mgr.Close()

	_ = mgr.StartIndexRequestHandler(handleIndexRequest)
	_ = mgr.StartDocumentRequestHandler(
		func(ctx context.Context, req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
			return handleDocumentRequest(ctx, config, req)
		})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	s := <-signals
	log.Printf("received signal %v, exiting", s)
}
