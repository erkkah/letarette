package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"time"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/protocol"
)

type entry struct {
	Title string    `json:"title"`
	Text  string    `json:"text"`
	Date  time.Time `json:"date"`
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
	file, err := os.Open(objFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		obj, readErr := reader.ReadString('\n')

		if len(obj) != 0 {
			var e entry
			err = json.Unmarshal([]byte(obj), &e)
			if err != nil {
				return err
			}
			if e.Date.IsZero() {
				e.Date = time.Now()
			}
			db[id] = e
			ix = append(ix, ixentry{
				date: e.Date,
				id:   id,
			})
			id++
		}

		if readErr != nil {
			if readErr != io.EOF {
				return readErr
			}
			return nil
		}
	}
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

func fetchInitial(limit uint16) []protocol.DocumentID {
	result := []protocol.DocumentID{}
	for _, v := range ix[:limit] {
		result = append(result, protocol.DocumentID(strconv.Itoa(v.id)))
	}
	log.Printf("Initial: %v\n", result)
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

func fetchByReference(startDocument protocol.DocumentID, startTime time.Time, limit uint16) []protocol.DocumentID {
	startIndex := sort.Search(len(ix), func(i int) bool {
		return !ix[i].date.Before(startTime)
	})

	subIndex := ix[startIndex:]
	numID, _ := strconv.Atoi(string(startDocument))
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

func handleIndexRequest(req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	log.Println("Received index request")
	if req.Space != space {
		return protocol.IndexUpdate{}, fmt.Errorf("Space %v not in db", req.Space)
	}
	updates := []protocol.DocumentID{}

	if req.StartDocument == "" {
		// Initial index fetch
		updates = fetchInitial(req.Limit)
	} else {
		entryID, err := strconv.Atoi(string(req.StartDocument))
		if err != nil {
			return protocol.IndexUpdate{}, fmt.Errorf("Invalid document ID: %v", req.StartDocument)
		}
		refEntry, found := db[entryID]
		if !found {
			// invalid index state, log
			updates = fetchByTime(req.StartTime, req.Limit)
		} else {
			if refEntry.Date.After(req.StartTime) {
				// entry updated, only use date
				updates = fetchByTime(req.StartTime, req.Limit)
			} else if refEntry.Date.Before(req.StartTime) {
				log.Printf("Unexpected index state: %v, %v\n", refEntry.Date.String(), req.StartTime.String())
				// invalid index state, log and ignore
			} else {
				// use ref entry
				updates = fetchByReference(req.StartDocument, req.StartTime, req.Limit)
			}
		}
	}
	return protocol.IndexUpdate{
		Space:   space,
		Updates: updates,
	}, nil
}

func entryToDocument(id protocol.DocumentID, e entry) protocol.Document {
	return protocol.Document{
		Space:   space,
		ID:      id,
		Updated: e.Date,
		Text:    e.Title + "\n" + e.Text,
		Alive:   true,
	}
}

func deadDocument(id protocol.DocumentID) protocol.Document {
	return protocol.Document{
		Space: space,
		ID:    id,
		Alive: false,
	}
}

func handleDocumentRequest(req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
	log.Println("Received document request")
	if req.Space != space {
		return protocol.DocumentUpdate{}, fmt.Errorf("Space %v not in db", req.Space)
	}

	docs := []protocol.Document{}
	for _, v := range req.Wanted {
		entryID, _ := strconv.Atoi(string(v))
		doc, found := db[entryID]
		log.Printf("Found doc from %v\n", doc.Date.String())
		if found {
			docs = append(docs, entryToDocument(v, doc))
		} else {
			docs = append(docs, deadDocument(v))
		}
	}
	return protocol.DocumentUpdate{
		Documents: docs,
	}, nil
}

func main() {
	space = os.Args[1]
	dbFile := os.Args[2]
	err := loadDatabase(dbFile)
	if err != nil {
		log.Panicf("Failed to load db: %v", err)
	}
	sortIndex()
	log.Printf("%v items loaded", len(db))

	for _, e := range ix[0:100] {
		log.Printf("%v: %v\n", e.id, e.date.String())
	}

	url := "nats://localhost:4222"
	ehandler := func(err error) {
		log.Printf("%v\n", err)
	}
	mgr, err := client.StartDocumentManager(url, client.WithErrorHandler(ehandler))
	if err != nil {
		log.Panicf("Failed to start document manager: %v", err)
	}
	defer mgr.Close()

	mgr.StartIndexRequestHandler(handleIndexRequest)
	mgr.StartDocumentRequestHandler(handleDocumentRequest)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	select {
	case s := <-signals:
		log.Printf("Received signal %v, exiting", s)
	}
}
