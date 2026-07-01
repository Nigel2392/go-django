package cache_test

import (
	"context"
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
		err := c.Set(context.Background(), item.key, item.value, 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test if keys exist
	keys, err := c.Keys(context.Background())
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
		value, err := c.Get(context.Background(), item.key)
		if err != nil {
			t.Fatal(err)
		}
		if string(value.([]byte)) != string(item.value) {
			t.Fatalf("value mismatch %s != %s", string(value.([]byte)), string(item.value))
		}
	}

	// Test TTL (time-to-live).
	for _, item := range cacheItems {
		ttl := c.TTL(context.Background(), item.key)
		if ttl <= 0 {
			t.Fatalf("ttl not positive for key %s %s", item.key, ttl)
		}
	}

	// Test Has function
	for _, item := range cacheItems {
		if !c.Has(context.Background(), item.key) {
			t.Fatalf("item should exist but does not %s", item.key)
		}
	}

	// Test deletion of items
	for _, item := range cacheItems {
		err := c.Delete(context.Background(), item.key)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Ensure items are deleted
	for _, item := range cacheItems {
		_, err := c.Get(context.Background(), item.key)
		if err == nil {
			t.Fatalf("item should be deleted but found %s", item.key)
		}
	}
}

func TestMemoryCacheTTLExpiry(t *testing.T) {
	var c = cache.NewMemoryCache(1 * time.Second)
	// Test that items expire correctly
	err := c.Set(context.Background(), "key1", []byte("value1"), 100*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Expire(context.Background(), "key1", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	_, err = c.Get(context.Background(), "key1")
	if err == nil {
		t.Fatalf("key1 should have expired but still exists")
	}

	err = c.Set(context.Background(), "key2", []byte("value2"), 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)

	_, err = c.Get(context.Background(), "key2")
	if err == nil {
		t.Fatalf("key2 should have expired but still exists")
	}
}

func TestMemoryCacheCounters(t *testing.T) {
	ctx := context.Background()
	// Using [any] so we can test mixing string keys and integer counters
	c := cache.NewGenericMemoryCache[any]()
	c.Run(1 * time.Second)

	t.Run("Initial Increment Creates Key", func(t *testing.T) {
		val, err := c.Increment(ctx, "counter_inc", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != 10 {
			t.Fatalf("expected 10, got %v", val)
		}
	})

	t.Run("Sequential Increment Math", func(t *testing.T) {
		val, err := c.Increment(ctx, "counter_inc", 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != 15 {
			t.Fatalf("expected 15, got %v", val)
		}
	})

	t.Run("Initial Decrement Creates Key", func(t *testing.T) {
		val, err := c.Decrement(ctx, "counter_dec", 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != -5 {
			t.Fatalf("expected -5, got %v", val)
		}
	})

	t.Run("Sequential Decrement Math", func(t *testing.T) {
		val, err := c.Decrement(ctx, "counter_dec", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != -15 {
			t.Fatalf("expected -15, got %v", val)
		}

		val, err = c.CounterValue(ctx, "counter_dec")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != -15 {
			t.Fatalf("expected -15, got %v", val)
		}
	})

	t.Run("Preserves TTL on Increment", func(t *testing.T) {
		_ = c.Set(ctx, "ttl_counter", int64(100), 10*time.Minute)

		ttlBefore := c.TTL(ctx, "ttl_counter")
		_, _ = c.Increment(ctx, "ttl_counter", 1)
		ttlAfter := c.TTL(ctx, "ttl_counter")

		// Ensure the TTL wasn't reset to Infinity or 0
		if ttlAfter > 10*time.Minute || ttlAfter < 9*time.Minute {
			t.Fatalf("TTL was altered! Before: %v, After: %v", ttlBefore, ttlAfter)
		}
	})

	t.Run("Errors on Non-Numeric Types", func(t *testing.T) {
		_ = c.Set(ctx, "string_key", "im_a_string", 0)

		_, err := c.Increment(ctx, "string_key", 1)
		if err == nil {
			t.Fatalf("expected error when incrementing a string, got nil")
		}
	})
}
