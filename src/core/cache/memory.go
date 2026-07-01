package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Nigel2392/go-django/src/core/errs"
)

func init() {
	SetDefault(
		NewMemoryCache(5 * time.Second),
	)
}

var _ Cache = (*MemoryCache[any])(nil)
var _ TransactionalCache = (*MemoryCache[any])(nil)
var _ Transaction = (*localTx[any])(nil)

const Infinity = Duration(1<<63 - 1) // 290 years

type memitem struct {
	key      string
	value    interface{}
	lifeTime time.Time
}

func (m *memitem) expired() bool {
	return m.lifeTime.Before(time.Now())
}

// A simple in-memory cache implementation based on a map of string[TYPE].
//
// Look at the interface implementation in cache.go for more information on the methods.
type MemoryCache[T any] struct {
	cache           map[string]*memitem
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
		cache:  make(map[string]*memitem),
		closed: make(chan struct{}),
	}
}

func (c *MemoryCache[T]) Run(interval time.Duration) {
	c.closed = make(chan struct{})
	c.cleanupInterval = interval
	go c.work()
}

// Increment atomically increments a numeric key by the given amount.
// If the key does not exist, it initializes it to the amount with an infinite TTL.
// It does NOT reset the TTL of an existing key.
func (c *MemoryCache[T]) Increment(_ context.Context, key string, amount int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key = fmt.Sprintf("cache.counter.%s", key)
	item, ok := c.cache[key]

	if !ok || item.expired() {
		c.cache[key] = &memitem{
			key:      key,
			value:    amount, // Drops in strictly as an int64
			lifeTime: time.Now().Add(Infinity),
		}
		return amount, nil
	}

	var currentVal = item.value.(int64)
	newVal := currentVal + amount
	item.value = newVal

	return newVal, nil
}

// Decrement atomically decrements a numeric key by the given amount.
// If the key does not exist, it initializes it to -amount with an infinite TTL.
// It does NOT reset the TTL of an existing key.
func (c *MemoryCache[T]) Decrement(ctx context.Context, key string, amount int64) (int64, error) {
	return c.Increment(ctx, key, -amount)
}

func (c *MemoryCache[T]) CounterValue(ctx context.Context, key string) (int64, error) {
	key = fmt.Sprintf("cache.counter.%s", key)

	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.cache[key]
	if !ok || item.expired() {
		return 0, ErrItemNotFound
	}

	currentVal, ok := item.value.(int64)
	if !ok {
		return 0, errs.ErrInvalidType
	}

	return currentVal, nil
}

func (c *MemoryCache[T]) Set(_ context.Context, key string, value T, ttl time.Duration) error {
	if ttl == 0 {
		ttl = Infinity
	}

	var lifeTime = time.Now().Add(ttl)
	var item *memitem = &memitem{
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
	t, ok := item.value.(T)
	if !ok {
		return *new(T), errs.ErrInvalidType
	}
	return t, nil
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
	t, ok := item.value.(T)
	if !ok {
		return *new(T), errs.ErrInvalidType
	}
	return t, nil
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
	c.cache = make(map[string]*memitem)
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

//
//	func (c *MemoryCache[T]) Scan(ctx context.Context, pattern string) iter.Seq2[string, error] {
//		return func(yield func(string, error) bool) {
//			// 1. Gather matches safely
//			var matched []string
//
//			c.mu.Lock()
//			for k, v := range c.cache {
//				if v.expired() {
//					continue
//				}
//
//				// filepath.Match handles redis- like globbing like "user:*" natively
//				if ok, _ := filepath.Match(pattern, k); ok {
//					matched = append(matched, k)
//				}
//			}
//			c.mu.Unlock()
//
//			for _, k := range matched {
//				// respect context cancellation
//				if err := ctx.Err(); err != nil {
//					yield("", err)
//					return
//				}
//
//				if !yield(k, nil) {
//					return
//				}
//			}
//		}
//	}

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
	memitem
	updated bool
	deleted bool
}

// localTx implements TypedCache[T] for the duration of the transaction
type localTx[T any] struct {
	state         map[string]*txItem[T]
	cleared       bool
	inTransaction bool
}

func (c *MemoryCache[T]) RunInTx(ctx context.Context, fn func(ctx context.Context, txCache TypedTransaction[T]) error) error {
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
	tx := &localTx[T]{state: txMap, inTransaction: true}
	if err := fn(ctx, tx); err != nil {
		return err // discard the map on error
	}
	tx.inTransaction = false

	if err := ctx.Err(); err != nil {
		return err
	}

	// committing, only applying diffs
	c.mu.Lock()
	defer c.mu.Unlock()

	if tx.cleared {
		c.cache = make(map[string]*memitem)
	}

	for k, item := range txMap {
		if tx.cleared && item.updated && !item.deleted {
			// If cleared, ONLY apply items that were explicitly Set() AFTER the Clear()
			c.cache[k] = &memitem{
				key:      k,
				value:    item.value,
				lifeTime: item.lifeTime,
			}
		} else if !tx.cleared {
			// Normal diff logic
			if item.deleted {
				delete(c.cache, k)
			} else if item.updated {
				c.cache[k] = &memitem{
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

func (tx *localTx[T]) InTransaction() bool {
	return tx.inTransaction
}

func (tx *localTx[T]) Increment(_ context.Context, key string, amount int64) (int64, error) {
	key = fmt.Sprintf("cache.counter.%s", key)

	item, ok := tx.state[key]

	if !ok || item.deleted || item.expired() {
		tx.state[key] = &txItem[T]{
			memitem: memitem{
				key:      key,
				value:    amount,
				lifeTime: time.Now().Add(Infinity),
			},
			updated: true,
			deleted: false,
		}
		return amount, nil
	}

	// 2. Extract current value safely to avoid panics
	currentVal, castOk := item.value.(int64)
	if !castOk {
		return 0, fmt.Errorf("cannot increment non-int64 type")
	}

	// 3. Do the math and update state flags
	newVal := currentVal + amount
	item.value = newVal
	item.updated = true
	item.deleted = false

	return newVal, nil
}

func (tx *localTx[T]) Decrement(ctx context.Context, key string, amount int64) (int64, error) {
	return tx.Increment(ctx, key, -amount)
}

func (tx *localTx[T]) CounterValue(ctx context.Context, key string) (int64, error) {
	key = fmt.Sprintf("cache.counter.%s", key)

	item, ok := tx.state[key]
	if !ok || item.expired() {
		return 0, ErrItemNotFound
	}

	currentVal, ok := item.value.(int64)
	if !ok {
		return 0, errs.ErrInvalidType
	}

	return currentVal, nil
}

func (tx *localTx[T]) Set(_ context.Context, key string, value T, ttl time.Duration) error {
	if ttl == 0 {
		ttl = Infinity
	}
	tx.state[key] = &txItem[T]{
		memitem: memitem{
			key:      key,
			value:    value,
			lifeTime: time.Now().Add(ttl),
		},
		updated: true,
		deleted: false,
	}
	return nil
}

func (tx *localTx[T]) Delete(_ context.Context, key string) error {
	if item, ok := tx.state[key]; ok {
		item.deleted = true
		item.updated = false
	} else {
		tx.state[key] = &txItem[T]{deleted: true}
	}
	return nil
}

func (tx *localTx[T]) Get(_ context.Context, key string) (value T, err error) {
	item, ok := tx.state[key]
	// Now we check .expired() so the Tx respects time!
	if !ok || item.deleted || item.expired() {
		return value, ErrItemNotFound
	}
	t, ok := item.value.(T)
	if !ok {
		return *new(T), errs.ErrInvalidType
	}
	return t, nil
}

func (tx *localTx[T]) Has(_ context.Context, key string) bool {
	item, ok := tx.state[key]
	return ok && !item.deleted && !item.expired()
}

func (tx *localTx[T]) GetDefault(_ context.Context, key string, defaultValue T) (T, error) {
	item, ok := tx.state[key]
	if !ok || item.deleted || item.expired() {
		return defaultValue, nil
	}
	t, ok := item.value.(T)
	if !ok {
		return *new(T), errs.ErrInvalidType
	}
	return t, nil
}

func (tx *localTx[T]) Keys(_ context.Context) ([]string, error) {
	var keys = make([]string, 0, len(tx.state))
	for k, item := range tx.state {
		// Only return keys that are alive and not deleted
		if !item.deleted && !item.expired() {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (tx *localTx[T]) Clear(_ context.Context) error {
	tx.cleared = true
	for _, item := range tx.state {
		item.deleted = true
		item.updated = false
	}
	return nil
}

func (tx *localTx[T]) TTL(_ context.Context, key string) time.Duration {
	item, ok := tx.state[key]
	if !ok || item.deleted || item.expired() {
		return 0
	}
	// Dynamically calculate the remaining time based on the exact moment TTL is called
	return time.Until(item.lifeTime)
}

func (tx *localTx[T]) Close(_ context.Context) error {
	return nil
}
