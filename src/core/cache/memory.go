package cache

import (
	"context"
	"sync"
	"time"
)

func init() {
	SetDefault(
		NewMemoryCache(5 * time.Second),
	)
}

var _ Cache = (*MemoryCache[any])(nil)
var _ TransactionalCache = (*MemoryCache[any])(nil)

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

func (c *MemoryCache[T]) Set(_ context.Context, key string, value T, ttl time.Duration) error {
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

func (c *MemoryCache[T]) Get(_ context.Context, key string) (value T, err error) {
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

func (c *MemoryCache[T]) GetDefault(_ context.Context, key string, defaultValue T) (value T, err error) {
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

func (c *MemoryCache[T]) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var _, ok = c.cache[key]
	if !ok {
		return ErrItemNotFound
	}
	delete(c.cache, key)
	return nil
}

func (c *MemoryCache[T]) Clear(_ context.Context) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*memitem[T])
	return nil
}

func (c *MemoryCache[T]) TTL(_ context.Context, key string) (ttl time.Duration) {
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

func (c *MemoryCache[T]) Keys(_ context.Context) ([]string, error) {
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

func (c *MemoryCache[T]) Close(_ context.Context) error {
	close(c.closed)
	return nil
}

func (c *MemoryCache[T]) Len(_ context.Context) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cache)
}

func (c *MemoryCache[T]) Has(_ context.Context, key string) (exists bool) {
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

// A local wrapper to track what happens during the transaction
type txItem[T any] struct {
	memitem[T]
	updated bool
	deleted bool
}

// localTx implements TypedCache[T] for the duration of the transaction
type localTx[T any] struct {
	state   map[string]*txItem[T]
	cleared bool
}

func (c *MemoryCache[T]) RunInTx(ctx context.Context, fn func(txCache TypedCache[T]) error) error {
	// copy the cache, but wrap it in our state tracker.
	txMap := make(map[string]*txItem[T])

	// this function is not meant for high throughput production environments
	// or environments with very big caches.
	c.mu.Lock()
	for k, v := range c.cache {
		txMap[k] = &txItem[T]{
			memitem: *v,
			updated: false,
			deleted: false,
		}
	}
	c.mu.Unlock()

	// Execute provided func in transaction
	tx := &localTx[T]{state: txMap}
	if err := fn(tx); err != nil {
		return err // discard the map on error
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// committing, only applying diffs
	c.mu.Lock()
	defer c.mu.Unlock()

	if tx.cleared {
		c.cache = make(map[string]*memitem[T])
	}

	for k, item := range txMap {
		if tx.cleared && item.updated && !item.deleted {
			// If cleared, ONLY apply items that were explicitly Set() AFTER the Clear()
			c.cache[k] = &memitem[T]{
				key:      k,
				value:    item.value,
				lifeTime: item.lifeTime,
			}
		} else if !tx.cleared {
			// Normal diff logic
			if item.deleted {
				delete(c.cache, k)
			} else if item.updated {
				c.cache[k] = &memitem[T]{
					key:      k,
					value:    item.value,
					lifeTime: item.lifeTime,
				}
			}
		}
		// If not cleared, deleted or updated, do nothing.
		// preserves changes made by other goroutines.
	}

	return nil
}

func (tx *localTx[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if ttl == 0 {
		ttl = Infinity
	}
	tx.state[key] = &txItem[T]{
		memitem: memitem[T]{
			key:      key,
			value:    value,
			lifeTime: time.Now().Add(ttl),
		},
		updated: true,
		deleted: false,
	}
	return nil
}

func (tx *localTx[T]) Delete(ctx context.Context, key string) error {
	if item, ok := tx.state[key]; ok {
		item.deleted = true
		item.updated = false
	} else {
		tx.state[key] = &txItem[T]{deleted: true}
	}
	return nil
}

func (tx *localTx[T]) Get(ctx context.Context, key string) (value T, err error) {
	item, ok := tx.state[key]
	// Now we check .expired() so the Tx respects time!
	if !ok || item.deleted || item.expired() {
		return value, ErrItemNotFound
	}
	return item.value, nil
}

func (tx *localTx[T]) Has(ctx context.Context, key string) bool {
	item, ok := tx.state[key]
	return ok && !item.deleted && !item.expired()
}

func (tx *localTx[T]) GetDefault(ctx context.Context, key string, defaultValue T) (T, error) {
	item, ok := tx.state[key]
	if !ok || item.deleted || item.expired() {
		return defaultValue, nil
	}
	return item.value, nil
}

func (tx *localTx[T]) Keys(ctx context.Context) ([]string, error) {
	var keys = make([]string, 0, len(tx.state))
	for k, item := range tx.state {
		// Only return keys that are alive and not deleted
		if !item.deleted && !item.expired() {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (tx *localTx[T]) Clear(ctx context.Context) error {
	tx.cleared = true
	for _, item := range tx.state {
		item.deleted = true
		item.updated = false
	}
	return nil
}

func (tx *localTx[T]) TTL(ctx context.Context, key string) time.Duration {
	item, ok := tx.state[key]
	if !ok || item.deleted || item.expired() {
		return 0
	}
	// Dynamically calculate the remaining time based on the exact moment TTL is called
	return time.Until(item.lifeTime)
}

func (tx *localTx[T]) Close(ctx context.Context) error {
	return nil
}
