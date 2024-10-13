package cache

import (
	"sync"
	"time"
)

func init() {
	SetDefault(
		NewMemoryCache(5 * time.Second),
	)
}

const Infinity = Duration(1<<63 - 1) // 290 years

type memitem[T any] struct {
	key      string
	value    T
	lifeTime time.Time
}

func (m *memitem[T]) expired() bool {
	return m.lifeTime.Before(time.Now())
}

// A simple in-memory cache implementation based on a map of string[TYPE].
//
// Look at the interface implementation in cache.go for more information on the methods.
type MemoryCache[T any] struct {
	cache           map[string]*memitem[T]
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	closed          chan struct{}
	mu              sync.Mutex
}

// Returns a new in-memory cache.
func NewMemoryCache(interval time.Duration) *MemoryCache[interface{}] {
	var cache = NewGenericMemoryCache[interface{}]()
	cache.Run(interval)
	return cache
}

// Might as well make it generic, right?
func NewGenericMemoryCache[T any]() *MemoryCache[T] {
	return &MemoryCache[T]{
		cache:  make(map[string]*memitem[T]),
		closed: make(chan struct{}),
	}
}

func (c *MemoryCache[T]) Run(interval time.Duration) {
	c.closed = make(chan struct{})
	c.cleanupInterval = interval
	go c.work()
}

func (c *MemoryCache[T]) Set(key string, value T, ttl time.Duration) error {
	if ttl == 0 {
		ttl = Infinity
	}

	var lifeTime = time.Now().Add(ttl)
	var item *memitem[T] = &memitem[T]{
		key:      key,
		value:    value,
		lifeTime: lifeTime,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = item
	return nil
}

func (c *MemoryCache[T]) Get(key string) (value T, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var item, ok = c.cache[key]
	if !ok {
		return value, ErrItemNotFound
	}
	if item.expired() {
		delete(c.cache, key)
		return value, ErrItemNotFound
	}
	return item.value, nil
}

func (c *MemoryCache[T]) GetDefault(key string, defaultValue T) (value T, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var item, ok = c.cache[key]
	if !ok {
		return defaultValue, nil
	}
	if item.expired() {
		delete(c.cache, key)
		return defaultValue, nil
	}
	return item.value, nil
}

func (c *MemoryCache[T]) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var _, ok = c.cache[key]
	if !ok {
		return ErrItemNotFound
	}
	delete(c.cache, key)
	return nil
}

func (c *MemoryCache[T]) Clear() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*memitem[T])
	return nil
}

func (c *MemoryCache[T]) TTL(key string) (ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var item, ok = c.cache[key]
	if !ok {
		return 0
	}
	if item.expired() {
		delete(c.cache, key)
		return 0
	}
	return time.Until(item.lifeTime)
}

func (c *MemoryCache[T]) Keys() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var keys = make([]string, 0, len(c.cache))
	for key, value := range c.cache {
		if value.expired() {
			delete(c.cache, key)
			continue
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (c *MemoryCache[T]) Close() error {
	close(c.closed)
	return nil
}

func (c *MemoryCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cache)
}

func (c *MemoryCache[T]) Has(key string) (exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var item, ok = c.cache[key]
	if !ok {
		return false
	}
	if item.expired() {
		delete(c.cache, key)
		return false
	}
	return true
}

// work is a background goroutine that cleans up the cache.
//
// It runs every cleanupInterval and removes items that have expired.
func (c *MemoryCache[T]) work() {
	c.cleanupTicker = time.NewTicker(c.cleanupInterval)
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mu.Lock()
			for key, item := range c.cache {
				if item.expired() {
					delete(c.cache, key)
				}
			}
			c.mu.Unlock()
		case <-c.closed:
			c.cleanupTicker.Stop()
			return
		}
	}
}
