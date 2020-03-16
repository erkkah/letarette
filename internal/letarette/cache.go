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
	"container/list"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/erkkah/immutable"
	"github.com/erkkah/letarette/pkg/protocol"
)

// Cache keeps search results for a set duration or until the cache
// max size is reached.
type Cache struct {
	timeout time.Duration

	// map[string]cacheEntry
	internalMap   immutable.Map
	sharedMap     atomic.Value
	sortedEntries list.List

	docToElements docElementsMap

	size    uint64
	maxSize uint64

	updates       chan cacheEntry
	invalidations chan protocol.DocumentID
}

type docElementsMap map[protocol.DocumentID][]*list.Element

type cacheEntry struct {
	key    string
	result protocol.SearchResult
	size   int
	stamp  time.Time
}

// NewCache creates cache with a given max size.
func NewCache(timeout time.Duration, maxSize uint64) *Cache {
	newCache := &Cache{
		timeout:       timeout,
		docToElements: docElementsMap{},
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

			element := cache.sortedEntries.PushBack(update)
			update.size = len(update.key)

			for _, v := range update.result.Hits {
				update.size += len(v.Snippet)
				update.size += len(v.ID)

				elements := cache.docToElements[v.ID]
				elements = append(elements, element)
				cache.docToElements[v.ID] = elements
			}

			element.Value = update
			cache.size += uint64(update.size)

			mappedEntries = mappedEntries.Set(update.key, update)

		case document := <-cache.invalidations:
			if elements, ok := cache.docToElements[document]; ok {
				for _, element := range elements {
					entry := element.Value.(cacheEntry)
					mappedEntries = mappedEntries.Delete(entry.key)
					cache.sortedEntries.Remove(element)
				}
				delete(cache.docToElements, document)
			}

		case <-cleanup:

			reduced := cache.size
			limit := time.Now().Add(-cache.timeout)

			keepShrinking := func(e *list.Element) bool {
				if e == nil {
					return false
				}
				entry := e.Value.(cacheEntry)
				return entry.stamp.Before(limit) || (reduced > cache.maxSize)
			}

			// Shrink to below accepted size
			for e := cache.sortedEntries.Front(); keepShrinking(e); e = cache.sortedEntries.Front() {
				entry := cache.sortedEntries.Remove(e).(cacheEntry)
				reduced -= uint64(entry.size)
				mappedEntries = mappedEntries.Delete(entry.key)
			}

			if reduced == cache.size {
				break
			}

			docToElements := docElementsMap{}
			for e := cache.sortedEntries.Front(); e != nil; e = e.Next() {
				entry := e.Value.(cacheEntry)
				for _, h := range entry.result.Hits {
					docToElements[h.ID] = append(docToElements[h.ID], e)
				}
			}
			cache.docToElements = docToElements
			cache.size = reduced

			cleanup = time.After(cleanupInterval)
		}

		cache.internalMap = mappedEntries
		cache.sharedMap.Store(mappedEntries)
	}
}

func makeKey(query string, spaces []string, limit uint16, offset uint16) string {
	return fmt.Sprintf("%s-%s-%d-%d", query, strings.Join(spaces, ":"), limit, offset)
}

// Get fetches cached search results
func (cache *Cache) Get(query string, spaces []string, limit uint16, offset uint16) (protocol.SearchResult, bool) {
	mapped := cache.sharedMap.Load().(immutable.Map)
	key := makeKey(query, spaces, limit, offset)
	if e, found := mapped.Get(key); found {
		entry := e.(cacheEntry)
		return entry.result, true
	}
	return protocol.SearchResult{}, false
}

// Put stores search results in the cache
func (cache *Cache) Put(query string, spaces []string, limit uint16, offset uint16, res protocol.SearchResult) {
	if cache.maxSize == 0 {
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
