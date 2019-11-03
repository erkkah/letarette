package letarette

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erkkah/letarette/pkg/protocol"
)

// Cache keeps search results for a set duration before they are
// thrown out. There is no max size.
type Cache struct {
	timeout  time.Duration
	mapped   entryMap
	sorted   []cacheEntry
	requests chan cacheRequest
	updates  chan cacheEntry
}

type entryMap map[string]cacheEntry

type cacheRequest chan entryMap

type cacheEntry struct {
	key    string
	result protocol.SearchResult
	stamp  time.Time
}

// NewCache creates cache with a given timeout.
func NewCache(timeout time.Duration) *Cache {
	newCache := &Cache{
		timeout:  timeout,
		mapped:   entryMap{},
		sorted:   []cacheEntry{},
		requests: make(chan cacheRequest, 100),
		updates:  make(chan cacheEntry, 100),
	}
	const cleanupInterval = time.Second * 1
	go func(cache *Cache) {
		cleanup := time.After(cleanupInterval)
		for {
			select {
			case req := <-cache.requests:
				req <- cache.mapped
			case update := <-cache.updates:
				if _, found := cache.mapped[update.key]; found {
					break
				}
				newMap := entryMap{}
				for k, v := range cache.mapped {
					newMap[k] = v
				}
				newMap[update.key] = update
				cache.mapped = newMap
				cache.sorted = append(cache.sorted, update)
			case <-cleanup:
				cut := sort.Search(len(cache.sorted), func(i int) bool {
					return time.Now().Before(cache.sorted[i].stamp.Add(cache.timeout))
				})

				if cut > 0 {
					if cut < len(cache.sorted) {
						newList := cache.sorted[cut:]
						newMap := entryMap{}
						for _, k := range newList {
							newMap[k.key] = cache.mapped[k.key]
						}
						cache.mapped = newMap
						cache.sorted = newList
					} else {
						cache.mapped = entryMap{}
						cache.sorted = []cacheEntry{}
					}
				}
				cleanup = time.After(cleanupInterval)
			}
		}
	}(newCache)
	return newCache
}

func (c *Cache) getEntries() entryMap {
	req := make(cacheRequest, 0)
	c.requests <- req
	return <-req
}

func makeKey(query string, spaces []string, limit uint16, offset uint16) string {
	return fmt.Sprintf("%s-%s-%d-%d", query, strings.Join(spaces, ":"), limit, offset)
}

// Get fetches cached search results
func (c *Cache) Get(query string, spaces []string, limit uint16, offset uint16) (protocol.SearchResult, bool) {
	mapped := c.getEntries()
	entry, found := mapped[makeKey(query, spaces, limit, offset)]
	return entry.result, found
}

// Put stores search results in the cache
func (c *Cache) Put(query string, spaces []string, limit uint16, offset uint16, res protocol.SearchResult) {
	clonedHits := append(res.Hits[:0:0], res.Hits...)

	entry := cacheEntry{
		key:    makeKey(query, spaces, limit, offset),
		result: res,
		stamp:  time.Now(),
	}
	entry.result.Hits = clonedHits
	c.updates <- entry
}
