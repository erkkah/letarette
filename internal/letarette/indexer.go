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

package letarette

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

// Indexer continuously runs the indexing process, until Close is called.
type Indexer interface {
	Close()
}

// StartIndexer creates and starts an indexer instance. This is really a singleton
// in that only one instance with the same database or config can be run at the
// same time.
func StartIndexer(nc *nats.Conn, db Database, cfg Config) (Indexer, error) {

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	mainContext, cancel := context.WithCancel(context.Background())

	self := &indexer{
		context:             mainContext,
		close:               cancel,
		lastDocumentRequest: map[string]time.Time{},
		cfg:                 cfg,
		conn:                ec,
		db:                  db.(*database),
		updateReceived:      make(chan struct{}, 10),
	}

	for _, space := range cfg.Index.Spaces {
		err := self.db.clearInterestList(context.Background(), space)
		if err != nil {
			return nil, fmt.Errorf("Failed to clear interest list: %w", err)
		}
	}

	updates := make(chan protocol.DocumentUpdate, 10)

	go func() {
		self.waiter.Add(1)
		for update := range updates {
			self.updateReceived <- struct{}{}
			err := self.db.addDocumentUpdates(mainContext, update.Space, update.Documents)
			if err != nil {
				logger.Error.Printf("Failed to add document update: %v", err)
			}
		}
		self.waiter.Done()
	}()

	subscription, err := ec.Subscribe(cfg.Nats.Topic+".document.update", func(update *protocol.DocumentUpdate) {
		filtered := make([]protocol.Document, 0, len(update.Documents))
		for _, doc := range update.Documents {
			index := shardIndexFromDocumentID(doc.ID, int(cfg.ShardgroupSize))
			if index == int(cfg.ShardgroupIndex) {
				filtered = append(filtered, doc)
			}
		}
		updates <- protocol.DocumentUpdate{
			Space:     update.Space,
			Documents: filtered,
		}
	})
	if err != nil {
		return nil, err
	}

	atExit := func() {
		logger.Info.Printf("Indexer exiting")
		err = subscription.Drain()
		if err != nil {
			logger.Error.Printf("Failed to drain document subscription: %v", err)
		} else {
			go func() {
				self.waiter.Add(1)
				for {
					messages, _, _ := subscription.Pending()
					if messages == 0 {
						break
					}
					time.Sleep(time.Millisecond * 20)
				}
				self.waiter.Done()
			}()
		}
		cancel()
		close(updates)
		self.waiter.Done()
	}

	self.waiter.Add(1)
	go self.main(atExit)

	return self, nil
}

type indexer struct {
	close   context.CancelFunc
	context context.Context
	waiter  sync.WaitGroup

	updateReceived chan struct{}

	lastDocumentRequest map[string]time.Time

	cfg  Config
	conn *nats.EncodedConn
	db   *database
}

func (idx *indexer) Close() {
	idx.close()
	idx.waiter.Wait()
}

func (idx *indexer) main(atExit func()) {
	logger.Info.Printf("Indexer starting")

	for {
		cycleThrottle := time.After(idx.cfg.Index.Wait.Cycle)
		totalInterests := 0

		for _, space := range idx.cfg.Index.Spaces {
			totalInterests += idx.runUpdateCycle(space)
		}

		if totalInterests == 0 {
			cycleThrottle = time.After(idx.cfg.Index.Wait.EmptyCycle)
			idx.doHousekeeping()
		}
		select {
		case <-idx.context.Done():
			atExit()
			return
		case <-idx.updateReceived:
			// Trigger cycle if we got an update
		case <-cycleThrottle:
			// Trigger cycle after timeout
		}
	}

}

func (idx *indexer) runUpdateCycle(space string) (total int) {
	interests, err := idx.db.getInterestList(idx.context, space)
	if err != nil {
		logger.Error.Printf("Failed to fetch current interest list: %v", err)
		return
	}

	total = len(interests)

	numPending := 0
	numRequested := 0
	numServed := 0
	pendingDocs := []Interest{}
	maxRequestedDocuments := int(idx.cfg.Index.MaxOutstanding) * int(idx.cfg.Index.ReqSize)

	for _, interest := range interests {
		switch interest.State {
		case served:
			numServed++
		case pending:
			numPending++
			pendingDocs = append(pendingDocs, interest)
		case requested:
			numRequested++
		}
	}

	docsToRequest := min(numPending, maxRequestedDocuments-numRequested)
	docsToRequest = min(docsToRequest, int(idx.cfg.Index.ReqSize))
	if docsToRequest > 0 {
		logger.Debug.Printf("Requesting %v docs\n", docsToRequest)
		metrics.docRequests.Add(float64(docsToRequest))
		err = idx.requestDocuments(space, pendingDocs[:docsToRequest])
		if err != nil {
			logger.Error.Printf("Failed to request documents: %v", err)
		} else {
			idx.lastDocumentRequest[space] = time.Now()
			numRequested += docsToRequest
		}
	}

	allServed := numPending == 0 && numRequested == 0

	if allServed {

		err = idx.commitFetched(space)
		if err != nil {
			logger.Error.Printf("Failed to commit docs: %v", err)
			return
		}

		err = idx.requestNextChunk(space)
		if err != nil {
			logger.Error.Printf("Failed to request next chunk: %v", err)
			return
		}

	} else {
		now := time.Now()
		timeout := idx.cfg.Index.Wait.Document
		refetchInterval := idx.cfg.Index.Wait.Refetch

		lastRequest := idx.lastDocumentRequest[space]
		if now.After(lastRequest.Add(refetchInterval)) {
			state, err := idx.db.getInterestListState(idx.context, space)
			if err != nil {
				logger.Error.Printf("Failed to get interest list state: %v", err)
				return
			}

			if now.After(state.createdAtTime().Add(timeout)) {
				logger.Warning.Printf("Waited too long for documents, moving on")
				err = idx.db.fakeServeRequested(idx.context, space)
			}

			logger.Warning.Printf("Timeout waiting for documents, re-requesting")
			err = idx.db.resetRequested(idx.context, space)
			if err != nil {
				logger.Error.Printf("Failed to reset interest list state: %v", err)
			}
		}
	}

	return
}

func (idx *indexer) commitFetched(space string) error {
	return idx.db.commitInterestList(idx.context, space)
}

func (idx *indexer) requestNextChunk(space string) error {
	topic := idx.cfg.Nats.Topic + ".index.request"
	state, err := idx.db.getInterestListState(idx.context, space)
	if err != nil {
		return fmt.Errorf("Failed to get interest list state: %w", err)
	}
	updateRequest := protocol.IndexUpdateRequest{
		Space:         space,
		FromTime:      state.lastUpdatedTime(),
		AfterDocument: state.LastUpdatedDocID,
		Limit:         idx.cfg.Index.ListSize,
	}
	timeout, cancel := context.WithTimeout(idx.context, idx.cfg.Index.Wait.Interest)

	var update protocol.IndexUpdate
	err = idx.conn.RequestWithContext(timeout, topic, updateRequest, &update)
	cancel()

	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	// Ignore documents from the future. We will get there eventually.
	nowish := time.Now().Add(time.Minute * 5)
	filtered := make([]protocol.DocumentReference, 0, len(update.Updates))
	for _, u := range update.Updates {
		if u.Updated.After(nowish) {
			logger.Info.Printf("Ignoring future document: %v (%v)", u.ID, u.Updated)
			continue
		}
		index := shardIndexFromDocumentID(u.ID, int(idx.cfg.ShardgroupSize))
		if index == int(idx.cfg.ShardgroupIndex) {
			filtered = append(filtered, u)
		}
	}

	update.Updates = filtered

	if len(update.Updates) > 0 {
		logger.Debug.Printf("Received interest list of %v docs\n", len(update.Updates))
	}

	err = idx.db.setInterestList(idx.context, update)
	if err != nil {
		return fmt.Errorf("Failed to set interest list: %w", err)
	}

	idx.updateReceived <- struct{}{}

	return nil
}

func (idx *indexer) requestDocuments(space string, wanted []Interest) error {
	var wantedIDs []protocol.DocumentID
	var existingIDs []protocol.DocumentID

	for _, v := range wanted {
		if ok, err := idx.db.hasDocument(idx.context, space, v); ok && err == nil {
			existingIDs = append(existingIDs, v.DocID)
		} else {
			wantedIDs = append(wantedIDs, v.DocID)
		}
	}

	for _, interest := range existingIDs {
		err := idx.db.setInterestState(idx.context, space, interest, served)
		if err != nil {
			return fmt.Errorf("Failed to update interest state: %w", err)
		}
	}

	rand.Shuffle(len(wantedIDs), func(i, j int) {
		wantedIDs[i], wantedIDs[j] = wantedIDs[j], wantedIDs[i]
	})

	for _, interest := range wantedIDs {
		err := idx.db.setInterestState(idx.context, space, interest, requested)
		if err != nil {
			return fmt.Errorf("Failed to update interest state: %w", err)
		}
	}

	topic := idx.cfg.Nats.Topic + ".document.request"

	request := protocol.DocumentRequest{
		Space:  space,
		Wanted: wantedIDs,
	}

	err := idx.conn.Publish(topic, request)
	return err
}

var lastHousekeeping time.Time
var housekeepingInterval = time.Minute * 5

func (idx *indexer) doHousekeeping() {
	if time.Since(lastHousekeeping) < housekeepingInterval {
		return
	}
	lastHousekeeping = time.Now()

	lag, err := GetSpellfixLag(idx.context, idx.db, idx.cfg.Spelling.MinFrequency)
	if err != nil {
		logger.Error.Printf("Failed to get spelling index lag: %v", err)
		return
	}
	if lag < idx.cfg.Spelling.MaxLag {
		return
	}
	start := time.Now()
	logger.Info.Printf("Housekeeping: Updating spelling index")
	err = UpdateSpellfix(idx.context, idx.db, idx.cfg.Spelling.MinFrequency)
	if err != nil {
		logger.Error.Printf("Housekeeping: Failed to update spelling index: %v", err)
		return
	}
	duration := time.Since(start)
	logger.Info.Printf("Housekeeping: Done updating spelling index in %v seconds", duration.Seconds())
}
