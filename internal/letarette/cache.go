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
	"sync/atomic"
	"time"

	"github.com/erkkah/immutable"
	"github.com/erkkah/letarette/pkg/protocol"
)

// Cache keeps search results for a set duration before they are
// thrown out. The cache is also limited in size.
type Cache struct {
	timeout time.Duration
	// map[string]cacheEntry
	internalMap   immutable.Map
	sharedMap     atomic.Value
	sortedEntries []cacheEntry
	docToKeys     docKeysMap
	size          uint64
	maxSize       uint64

	updates       chan cacheEntry
	invalidations chan protocol.DocumentID
}

type docKeysMap map[protocol.DocumentID][]string

type cacheEntry struct {
	key    string
	result protocol.SearchResult
	size   int
	stamp  time.Time
}

// NewCache creates cache with a given timeout and max size.
func NewCache(timeout time.Duration, maxSize uint64) *Cache {
	newCache := &Cache{
		timeout:       timeout,
		sortedEntries: []cacheEntry{},
		docToKeys:     docKeysMap{},
		maxSize:       maxSize,
		// ??? Arbitrary chan sizes
		updates:       make(chan cacheEntry, 100),
		invalidations: make(chan protocol.DocumentID, 250),
	}

	newCache.sharedMap.Store(newCache.internalMap)

	go newCache.update()

	return newCache
}

func (cache *Cache) update() {
	const cleanupInterval = time.Second * 10
	cleanup := time.After(cleanupInterval)
	for {
		mappedEntries := cache.internalMap

		select {
		case update := <-cache.updates:
			update.size = len(update.key)
			for _, v := range update.result.Hits {
				update.size += len(v.Snippet)
				update.size += len(v.ID)

				keys := cache.docToKeys[v.ID]
				keys = append(keys, update.key)
				cache.docToKeys[v.ID] = keys
			}
			cache.size += uint64(update.size)
			cache.sortedEntries = append(cache.sortedEntries, update)
			mappedEntries = mappedEntries.Set(update.key, update)

		case document := <-cache.invalidations:
			if keys, ok := cache.docToKeys[document]; ok {
				for _, k := range keys {
					mappedEntries = mappedEntries.Delete(k)
				}
			}

		case <-cleanup:
			// Find the first entry that has not timed out
			timeCut := sort.Search(len(cache.sortedEntries), func(i int) bool {
				return time.Now().Before(cache.sortedEntries[i].stamp.Add(cache.timeout))
			})

			// Find the cut needed to shrink below accepted size
			sizeCut := 0
			if cache.maxSize > 0 {
				reduced := cache.size
				for i, v := range cache.sortedEntries {
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
				mappedEntries, cache.sortedEntries, cache.docToKeys, cache.size = cutAt(
					mappedEntries,
					cache.sortedEntries,
					cache.size,
					cut)
			}
			cleanup = time.After(cleanupInterval)
		}

		cache.internalMap = mappedEntries
		cache.sharedMap.Store(mappedEntries)
	}
}

func cutAt(
	mapped immutable.Map, sorted []cacheEntry, size uint64, cut int,
) (immutable.Map, []cacheEntry, docKeysMap, uint64) {

	if cut < len(sorted) {
		keepList := sorted[cut:]
		for _, old := range sorted[:cut] {
			size -= uint64(old.size)
		}
		var newMap immutable.Map
		docKeys := docKeysMap{}
		for _, k := range keepList {
			val, _ := mapped.Get(k.key)
			newMap = newMap.Set(k.key, val)
			for _, h := range k.result.Hits {
				docKeys[h.ID] = append(docKeys[h.ID], k.key)
			}
		}
		return newMap, keepList, docKeys, size
	}
	return immutable.Map{}, []cacheEntry{}, docKeysMap{}, 0
}

func makeKey(query string, spaces []string, limit uint16, offset uint16) string {
	return fmt.Sprintf("%s-%s-%d-%d", query, strings.Join(spaces, ":"), limit, offset)
}

// Get fetches cached search results
func (cache *Cache) Get(query string, spaces []string, limit uint16, offset uint16) (protocol.SearchResult, bool) {
	mapped := cache.sharedMap.Load().(immutable.Map)
	key := makeKey(query, spaces, limit, offset)
	if entry, found := mapped.Get(key); found {
		return entry.(cacheEntry).result, true
	}
	return protocol.SearchResult{}, false
}

// Put stores search results in the cache
func (cache *Cache) Put(query string, spaces []string, limit uint16, offset uint16, res protocol.SearchResult) {
	if cache.timeout == 0 {
		return
	}
	clonedHits := append(res.Hits[:0:0], res.Hits...)

	entry := cacheEntry{
		key:    makeKey(query, spaces, limit, offset),
		result: res,
		stamp:  time.Now(),
	}
	entry.result.Hits = clonedHits
	cache.updates <- entry
}

// Invalidate marks a document as updated
func (cache *Cache) Invalidate(doc protocol.DocumentID) {
	cache.invalidations <- doc
}
