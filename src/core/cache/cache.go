package cache

import (
	"context"
	"time"
)

const DefaultCache = "default"

type Duration = time.Duration

type CacheConnector interface {
	Connect() error
	HasConnected() bool
}

type TypedCache[T any] interface {
	// Implementation details of the cache interface are written as comments below.

	// Get retrieves a value from the cache.
	//
	// If the key does not exist, Get returns nil and ErrItemNotFound.
	Get(c context.Context, key string) (T, error)

	// GetDefault retrieves a value from the cache.
	//
	// If the key does not exist, GetDefault returns the defaultValue.
	//
	// It may return an error if the key exists but the cache itself returns an error.
	GetDefault(c context.Context, key string, defaultValue T) (T, error)

	// Set sets a value in the cache.
	//
	// The value is stored in the cache with the specified key.
	// The value will expire after the specified ttl.
	//
	// If the TTL is 0, or Infinity, the value will never expire.
	Set(c context.Context, key string, value T, ttl Duration) error

	// TTL returns the time to live for a key.
	//
	// If the key does not exist, TTL returns 0.
	//
	// If any error occurs, TTL returns 0.
	TTL(c context.Context, key string) Duration

	// Has returns true if the key exists in the cache.
	//
	// If any error occurs, Has returns false.
	Has(c context.Context, key string) bool

	// Delete removes a key from the cache.
	//
	// If the key does not exist, Delete should return ErrItemNotFound.
	Delete(c context.Context, key string) error

	// Keys returns all keys in the cache.
	//
	// If any error occurs, Keys returns an empty slice and the error.
	Keys(c context.Context) ([]string, error)

	// Clear removes all keys from the cache.
	//
	// If any error occurs, Clear should return the error.
	Clear(c context.Context) error

	// Close closes the cache.
	//
	// If any error occurs, Close should return the error.
	Close(c context.Context) error
}

type TypedTransactionalCache[T any] interface {
	TypedCache[T]

	// RunInTx executes the given function inside a transaction.
	// The provided txCache should be used for all operations inside the function.
	RunInTx(ctx context.Context, fn func(txCache TypedCache[T]) error) error
}

type (
	Cache              = TypedCache[any]
	TransactionalCache = TypedTransactionalCache[any]
)

type cacheBackend struct {
	backends map[string]TransactionalCache
}

var caches = cacheBackend{
	backends: make(map[string]TransactionalCache),
}

// RegisterCache registers a cache backend with a name.
//
// This can later be used to retrieve the cache backend using GetCache.
func RegisterCache(name string, cache TransactionalCache) {
	caches.backends[name] = cache
}

// RemoveCache removes a cache backend from the cache backend registry.
//
// This should be used when a cache backend is no longer needed.
func RemoveCache(name string) {
	delete(caches.backends, name)
}

// GetCache retrieves a cache backend by name.
//
// If the cache backend does not exist, GetCache returns nil.
func GetCache(names ...string) TransactionalCache {
	if len(names) == 0 {
		names = []string{DefaultCache}
	}
	for _, name := range names {
		if cache, ok := caches.backends[name]; ok {
			var connector, ok = cache.(CacheConnector)

			if ok && !connector.HasConnected() {
				connector.Connect()
			}

			return cache
		}
	}
	return nil
}

// SetDefault sets the default cache backend.
//
// The default cache backend is used by the cache package functions.
func SetDefault(cache TransactionalCache) {
	RegisterCache(DefaultCache, cache)
}

// GetDefault retrieves the default cache backend.
//
// If the default cache backend does not exist, GetDefault returns nil.
func Default() Cache {
	return GetCache(DefaultCache)
}

// Get retrieves a value from the default cache backend.
//
// If the key does not exist, Get returns nil and ErrItemNotFound.
func Get(ctx context.Context, key string) (interface{}, error) {
	return Default().Get(ctx, key)
}

// GetDefault retrieves a value from the default cache backend.
//
// If the key does not exist, GetDefault returns the defaultValue.
//
// It may return an error if the key exists but the cache itself returns an error.
func GetDefault(ctx context.Context, key string, defaultValue interface{}) (interface{}, error) {
	return Default().GetDefault(ctx, key, defaultValue)
}

// Set sets a value in the default cache backend.
//
// The value is stored in the cache with the specified key.
// The value will expire after the specified ttl.
func Set(ctx context.Context, key string, value interface{}, ttl Duration) error {
	return Default().Set(ctx, key, value, ttl)
}

// TTL returns the time to live for a key in the default cache backend.
//
// If the key does not exist, TTL returns 0.
//
// If any error occurs, TTL returns 0.
func TTL(ctx context.Context, key string) Duration {
	return Default().TTL(ctx, key)
}

// Has returns true if the key exists in the default cache backend.
//
// If any error occurs, Has returns false.
func Has(ctx context.Context, key string) bool {
	return Default().Has(ctx, key)
}

// Delete removes a key from the default cache backend.
//
// If the key does not exist, Delete should return ErrItemNotFound.
func Delete(ctx context.Context, key string) error {
	return Default().Delete(ctx, key)
}

// Keys returns all keys in the default cache backend.
//
// If any error occurs, Keys returns an empty slice and the error.
func Keys(ctx context.Context) ([]string, error) {
	return Default().Keys(ctx)
}

// Clear removes all keys from the default cache backend.
//
// If any error occurs, Clear should return the error.
func Clear(ctx context.Context) error {
	return Default().Clear(ctx)
}

// Close closes the default cache backend.
//
// If any error occurs, Close should return the error.
func Close(ctx context.Context) error {
	return Default().Close(ctx)
}
