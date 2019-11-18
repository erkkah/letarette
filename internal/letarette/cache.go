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
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
)

// Cache keeps search results for a set duration before they are
// thrown out. The cache is also limited in size.
type Cache struct {
	timeout  time.Duration
	mapped   entryMap
	sorted   []cacheEntry
	size     uint64
	maxSize  uint64
	requests chan cacheRequest
	updates  chan cacheEntry
}

type entryMap map[string]cacheEntry

type cacheRequest chan entryMap

type cacheEntry struct {
	key    string
	result protocol.SearchResult
	size   int
	stamp  time.Time
}

// NewCache creates cache with a given timeout and max size.
func NewCache(timeout time.Duration, maxSize uint64) *Cache {
	newCache := &Cache{
		timeout:  timeout,
		mapped:   entryMap{},
		sorted:   []cacheEntry{},
		maxSize:  maxSize,
		requests: make(chan cacheRequest, 100),
		updates:  make(chan cacheEntry, 100),
	}

	go newCache.update()

	return newCache
}

func (cache *Cache) update() {
	const cleanupInterval = time.Second * 1
	cleanup := time.After(cleanupInterval)
	for {
		select {
		case req := <-cache.requests:
			req <- cache.mapped
		case update := <-cache.updates:
			if _, found := cache.mapped[update.key]; found {
				break
			}
			update.size = len(update.key)
			for _, v := range update.result.Hits {
				update.size += len(v.Snippet)
				update.size += len(v.ID)
			}
			cache.size += uint64(update.size)
			newMap := entryMap{}
			for k, v := range cache.mapped {
				newMap[k] = v
			}
			newMap[update.key] = update
			cache.mapped = newMap
			cache.sorted = append(cache.sorted, update)
		case <-cleanup:
			// Find the first entry that has not timed out
			timeCut := sort.Search(len(cache.sorted), func(i int) bool {
				return time.Now().Before(cache.sorted[i].stamp.Add(cache.timeout))
			})

			// Find the cut needed to shrink below accepted size
			sizeCut := 0
			if cache.maxSize > 0 {
				reduced := cache.size
				for i, v := range cache.sorted {
					if reduced > cache.maxSize {
						reduced -= uint64(v.size)
						sizeCut = i + 1
					} else {
						break
					}
				}
			}
			cut := max(timeCut, sizeCut)
			if cut > 0 {
				cache.cutAt(cut)
			}
			cleanup = time.After(cleanupInterval)
		}
	}
}

func (cache *Cache) cutAt(cut int) {
	if cut < len(cache.sorted) {
		newList := cache.sorted[cut:]
		for _, old := range cache.sorted[:cut] {
			cache.size -= uint64(old.size)
		}
		newMap := entryMap{}
		for _, k := range newList {
			newMap[k.key] = cache.mapped[k.key]
		}
		cache.mapped = newMap
		cache.sorted = newList
	} else {
		cache.mapped = entryMap{}
		cache.sorted = []cacheEntry{}
		cache.size = 0
	}
}

func (cache *Cache) getEntries() entryMap {
	req := make(cacheRequest, 0)
	cache.requests <- req
	return <-req
}

func makeKey(query string, spaces []string, limit uint16, offset uint16) string {
	return fmt.Sprintf("%s-%s-%d-%d", query, strings.Join(spaces, ":"), limit, offset)
}

// Get fetches cached search results
func (cache *Cache) Get(query string, spaces []string, limit uint16, offset uint16) (protocol.SearchResult, bool) {
	mapped := cache.getEntries()
	entry, found := mapped[makeKey(query, spaces, limit, offset)]
	return entry.result, found
}

// Put stores search results in the cache
func (cache *Cache) Put(query string, spaces []string, limit uint16, offset uint16, res protocol.SearchResult) {
	clonedHits := append(res.Hits[:0:0], res.Hits...)

	entry := cacheEntry{
		key:    makeKey(query, spaces, limit, offset),
		result: res,
		stamp:  time.Now(),
	}
	entry.result.Hits = clonedHits
	cache.updates <- entry
}
