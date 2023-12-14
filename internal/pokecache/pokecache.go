package pokecache

import (
	"sync"
	"time"
)

type chacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	entries  map[string]chacheEntry
	mutex    sync.Mutex
	interval time.Duration
}

func (cache *Cache) Add(key string, val []byte) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.entries[key] = chacheEntry{
		val:       val,
		createdAt: time.Now(),
	}
}

func (cache *Cache) Get(key string) ([]byte, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	result, ok := cache.entries[key]
	return result.val, ok
}

func (cache *Cache) reapLoop() {
	ticker := time.NewTicker(cache.interval)
	for {
		now := <-ticker.C
		cache.mutex.Lock()
		for key, entry := range cache.entries {
			if entry.createdAt.Add(cache.interval).Compare(now) <= 0 {
				delete(cache.entries, key)
			}
		}
		cache.mutex.Unlock()
	}
}

func NewCache(interval time.Duration) *Cache {
	result := new(Cache)
	result.entries = make(map[string]chacheEntry)
	result.interval = interval
	go result.reapLoop()

	return result
}
