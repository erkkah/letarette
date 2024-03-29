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
	"errors"
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
func StartIndexer(nc *nats.Conn, db Database, cfg Config, cache *Cache) (Indexer, error) {

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
		indexUpdates:        map[string]chan protocol.IndexUpdate{},
		updateReceived:      make(chan struct{}, 1),
	}

	for _, space := range cfg.Index.Spaces {
		self.indexUpdates[space] = make(chan protocol.IndexUpdate)
		err := self.db.clearInterestList(context.Background(), space)
		if err != nil {
			return nil, fmt.Errorf("failed to clear interest list: %w", err)
		}
	}

	err = self.startIndexFetcher()
	if err != nil {
		return nil, err
	}

	updates := make(chan protocol.DocumentUpdate, 50)

	self.waiter.Add(1)
	go func() {
		for update := range updates {
			self.notifyUpdateReceived()
			err := self.db.addDocumentUpdates(mainContext, update.Space, update.Documents)
			if err != nil {
				logger.Error.Printf("failed to add document update: %v", err)
			}
			for _, doc := range update.Documents {
				cache.Invalidate(doc.ID)
			}
		}
		self.waiter.Done()
	}()

	subscription, err := ec.Subscribe(cfg.Nats.Topic+".document.update", func(update *protocol.DocumentUpdate) {
		filtered := make([]protocol.Document, 0, len(update.Documents))
		for _, doc := range update.Documents {
			index := ShardIndexFromDocumentID(doc.ID, int(cfg.ShardgroupSize))
			if index == int(cfg.ShardIndex) {
				filtered = append(filtered, doc)
			}
		}

		metrics.UpdateQueue.Set(int64(len(updates)))

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
			var drainWaiter sync.WaitGroup
			drainWaiter.Add(1)
			go func() {
				for {
					messages, _, _ := subscription.Pending()
					if messages == 0 {
						break
					}
					time.Sleep(time.Millisecond * 20)
				}
				drainWaiter.Done()
			}()
			drainWaiter.Wait()
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

	indexUpdates map[string]chan protocol.IndexUpdate

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
			logger.Debug.Printf("main loop empty cycle wait")
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

func (idx *indexer) notifyUpdateReceived() {
	select {
	case idx.updateReceived <- struct{}{}:
	default:
	}
}

func (idx *indexer) runUpdateCycle(space string) int {
	interests, err := idx.db.getInterestList(idx.context, space)
	if err != nil {
		logger.Error.Printf("Failed to fetch current interest list: %v", err)
		return 0
	}

	total := len(interests)

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

	metrics.PendingDocs.Set(int64(numPending))
	metrics.ServedDocs.Add(int64(numServed))

	docsToRequest := min(numPending, maxRequestedDocuments-numRequested)
	docsToRequest = min(docsToRequest, int(idx.cfg.Index.ReqSize))
	if docsToRequest > 0 {
		logger.Debug.Printf("Requesting %v docs\n", docsToRequest)
		metrics.DocRequests.Add(int64(docsToRequest))
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
			if !errors.Is(err, context.Canceled) {
				logger.Error.Printf("Failed to commit docs: %v", err)
			}
			return total
		}

		err = idx.db.clearInterestList(idx.context, space)
		if err != nil {
			logger.Error.Printf("Failed to clean interest list: %v", err)
		}

		err = idx.processIndexUpdateQueue(space)
		if err != nil {
			logger.Error.Printf("Failed to request next chunk: %v", err)
			return total
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
				return total
			}

			if now.After(state.createdAtTime().Add(timeout)) {
				logger.Warning.Printf("Waited too long for documents, moving on")
				err = idx.db.fakeServeRequested(idx.context, space)
				if err != nil {
					logger.Error.Printf("Failed to fake request served: %v", err)
				}
			}

			logger.Warning.Printf("Timeout waiting for documents, re-requesting")
			err = idx.db.resetRequested(idx.context, space)
			if err != nil {
				logger.Error.Printf("Failed to reset interest list state: %v", err)
			}
		}
	}

	return total
}

func (idx *indexer) commitFetched(space string) error {
	return idx.db.commitInterestList(idx.context, space)
}

func (idx *indexer) startIndexFetcher() error {

	for _, space := range idx.cfg.Index.Spaces {
		state, err := idx.db.getInterestListState(idx.context, space)
		if err != nil {
			return fmt.Errorf("failed to get interest list state: %w", err)
		}
		go func(space string, state InterestListState) {
			fromTime := state.lastUpdatedTime()
			afterDocument := state.LastUpdatedDocID

			idx.waiter.Add(1)

		fetchLoop:
			for {
				cycleThrottle := idx.cfg.Index.Wait.Cycle

				logger.Debug.Printf("Requesting index update (%v, %v, %v)", space, fromTime, afterDocument)
				update, err := idx.requestIndexUpdate(space, fromTime, afterDocument)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						break fetchLoop
					}

					if errors.Is(err, nats.ErrNoResponders) {
						logger.Info.Printf("No Document Manager available for space %q", space)
						cycleThrottle = idx.cfg.Index.Wait.EmptyCycle * 4
					} else {
						logger.Info.Printf("index update request failed: %v", err)
						cycleThrottle = idx.cfg.Index.Wait.EmptyCycle
					}

				} else {
					numUpdates := len(update.Updates)
					if numUpdates > 0 {
						last := update.Updates[numUpdates-1]
						fromTime = last.Updated
						afterDocument = last.ID
					}
					select {
					case idx.indexUpdates[space] <- update:
						// Update written to channel
					case <-idx.context.Done():
						close(idx.indexUpdates[space])
						break fetchLoop
					}

					if numUpdates == 0 {
						logger.Debug.Printf("Indexer loop empty cycle wait")
						cycleThrottle = idx.cfg.Index.Wait.EmptyCycle
					}
				}

				select {
				case <-time.After(cycleThrottle):
				case <-idx.context.Done():
					close(idx.indexUpdates[space])
					break fetchLoop
				}
			}

			idx.waiter.Done()
		}(space, state)
	}

	return nil
}

func (idx *indexer) processIndexUpdateQueue(space string) error {
	channel := idx.indexUpdates[space]
	select {
	case <-idx.context.Done():
		return nil

	case update := <-channel:
		if len(update.Updates) > 0 {
			logger.Debug.Printf("Received interest list of %v docs\n", len(update.Updates))
			idx.notifyUpdateReceived()

			err := idx.db.setInterestList(idx.context, update)
			if err != nil {
				return fmt.Errorf("failed to set interest list: %w", err)
			}
		}

	case <-time.After(idx.cfg.Index.Wait.Interest):
		// timeout
	}

	return nil
}

func (idx *indexer) requestIndexUpdate(
	space string, fromTime time.Time, afterDocument protocol.DocumentID,
) (protocol.IndexUpdate, error) {

	topic := idx.cfg.Nats.Topic + ".index.request"
	updateRequest := protocol.IndexUpdateRequest{
		Space:         space,
		FromTime:      fromTime,
		AfterDocument: afterDocument,
		Limit:         idx.cfg.Index.ListSize,
	}
	timeout, cancel := context.WithTimeout(idx.context, idx.cfg.Index.Wait.Interest)

	var update protocol.IndexUpdate
	err := idx.conn.RequestWithContext(timeout, topic, updateRequest, &update)
	cancel()

	if err != nil {
		return protocol.IndexUpdate{}, fmt.Errorf("NATS request failed: %w", err)
	}

	// Ignore documents from the future. We will get there eventually.
	nowish := time.Now().Add(time.Minute * 5)
	filtered := make([]protocol.DocumentReference, 0, len(update.Updates))
	for _, u := range update.Updates {
		if u.Updated.After(nowish) {
			logger.Info.Printf("Ignoring future document: %v (%v)", u.ID, u.Updated)
			continue
		}
		index := ShardIndexFromDocumentID(u.ID, int(idx.cfg.ShardgroupSize))
		if index == int(idx.cfg.ShardIndex) {
			filtered = append(filtered, u)
		}
	}

	logger.Debug.Printf("Keeping %v out of %v updates for shard %v", len(filtered), len(update.Updates), idx.cfg.Shard)

	update.Updates = filtered

	return update, nil
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
			return fmt.Errorf("failed to update interest state: %w", err)
		}
	}

	rand.Shuffle(len(wantedIDs), func(i, j int) {
		wantedIDs[i], wantedIDs[j] = wantedIDs[j], wantedIDs[i]
	})

	for _, interest := range wantedIDs {
		err := idx.db.setInterestState(idx.context, space, interest, requested)
		if err != nil {
			return fmt.Errorf("failed to update interest state: %w", err)
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

	idx.updateSpelling()
	idx.updateStopwords()
}

func (idx *indexer) updateSpelling() {
	lag, err := GetSpellfixLag(idx.context, idx.db, idx.cfg.Spelling.MinFrequency)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error.Printf("Failed to get spelling index lag: %v", err)
		}
		return
	}
	if lag < idx.cfg.Spelling.MaxLag {
		return
	}
	start := time.Now()
	logger.Info.Printf("Housekeeping: Updating spelling index")
	err = UpdateSpellfix(idx.context, idx.db, idx.cfg.Spelling.MinFrequency)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error.Printf("Housekeeping: Failed to update spelling index: %v", err)
		}
		return
	}
	duration := time.Since(start)
	logger.Info.Printf("Housekeeping: Done updating spelling index in %v seconds", duration.Seconds())
}

func (idx *indexer) updateStopwords() {
	logger.Debug.Printf("Updating stopwords...")
	stopwordPercentageCutoff := idx.cfg.Stemmer.StopwordCutoff
	err := idx.db.updateStopwords(idx.context, stopwordPercentageCutoff)
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error.Printf("Failed to update stop words: %v", err)
	}
	logger.Debug.Printf("Done updating stopwords")
}
