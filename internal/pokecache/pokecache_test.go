package pokecache

import (
	"testing"
	"time"
)

func TestReapLoop(t *testing.T) {
	interval := 10 * time.Millisecond
	cache := NewCache(interval)

	testKey, testVal := "test", []byte("test value")

	cache.Add(testKey, testVal)
	_, ok := cache.Get(testKey)
	if !ok {
		t.Errorf("Expected key '%s' in cache", testKey)
	}

	time.Sleep(interval)

	_, ok = cache.Get(testKey)
	if ok {
		t.Errorf("Key '%s' still in cache after reap duration", testKey)
	}
}
