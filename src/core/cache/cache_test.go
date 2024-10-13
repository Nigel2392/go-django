package cache_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/cache"
)

type cacheItem struct {
	key   string
	value []byte
}

var cacheItems [128]*cacheItem

func init() {
	for i := 0; i < len(cacheItems); i++ {
		cacheItems[i] = &cacheItem{
			key:   "key" + strconv.Itoa(i),
			value: []byte("value" + strconv.Itoa(i)),
		}
	}
}

func TestMemoryCache(t *testing.T) {
	var c = cache.NewMemoryCache(1 * time.Second)

	// Test insertion of cache items
	for _, item := range cacheItems {
		err := c.Set(item.key, item.value, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test if keys exist
	keys, err := c.Keys()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range cacheItems {
		found := false
		for _, key := range keys {
			if key == item.key {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("key not found %s", item.key)
		}
	}

	// Test retrieval of cache items
	for _, item := range cacheItems {
		value, err := c.Get(item.key)
		if err != nil {
			t.Fatal(err)
		}
		if string(value.([]byte)) != string(item.value) {
			t.Fatalf("value mismatch %s != %s", string(value.([]byte)), string(item.value))
		}
	}

	// Test TTL (time-to-live).
	for _, item := range cacheItems {
		ttl := c.TTL(item.key)
		if ttl <= 0 {
			t.Fatalf("ttl not positive for key %s %s", item.key, ttl)
		}
	}

	// Test Has function
	for _, item := range cacheItems {
		if !c.Has(item.key) {
			t.Fatalf("item should exist but does not %s", item.key)
		}
	}

	// Test deletion of items
	for _, item := range cacheItems {
		err := c.Delete(item.key)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Ensure items are deleted
	for _, item := range cacheItems {
		_, err := c.Get(item.key)
		if err == nil {
			t.Fatalf("item should be deleted but found %s", item.key)
		}
	}
}

func TestMemoryCacheTTLExpiry(t *testing.T) {
	var c = cache.NewMemoryCache(1 * time.Second)

	// Test that items expire correctly
	err := c.Set("key1", []byte("value1"), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	_, err = c.Get("key1")
	if err == nil {
		t.Fatalf("key1 should have expired but still exists")
	}

	err = c.Set("key2", []byte("value2"), 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)

	_, err = c.Get("key2")
	if err == nil {
		t.Fatalf("key2 should have expired but still exists")
	}
}
