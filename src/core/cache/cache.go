package cache

import (
	"context"
	"time"

	django "github.com/Nigel2392/go-django/src"
)

const (
	DefaultCache = "default"

	DEFAULT_TTL_KEY = "CACHE.DEFAULT_TTL"
)

type Duration = time.Duration

func TTL_GetOrDefault(ttl Duration) Duration {
	if ttl == 0 {
		if django.Global == nil || django.Global.Settings == nil {
			return Infinity
		}

		return django.ConfigGet(
			django.Global.Settings,
			DEFAULT_TTL_KEY,
			Infinity,
		)
	}

	return ttl
}

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

// GetCache retrieves the first cache backend it can find by name.
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
func Default() TransactionalCache {
	return GetCache(DefaultCache)
}

// Get retrieves a value from the default cache backend.
//
// If the key does not exist, Get returns nil and ErrItemNotFound.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Get(ctx context.Context, key string) (interface{}, error) {
	return transactionOrDefault(ctx).Get(ctx, key)
}

// GetDefault retrieves a value from the default cache backend.
//
// If the key does not exist, GetDefault returns the defaultValue.
//
// It may return an error if the key exists but the cache itself returns an error.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func GetDefault(ctx context.Context, key string, defaultValue interface{}) (interface{}, error) {
	return transactionOrDefault(ctx).GetDefault(ctx, key, defaultValue)
}

// Set sets a value in the default cache backend.
//
// The value is stored in the cache with the specified key.
// The value will expire after the specified ttl.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Set(ctx context.Context, key string, value interface{}, ttl Duration) error {
	return transactionOrDefault(ctx).Set(ctx, key, value, ttl)
}

// Increment atomically increments a numeric key by the given amount.
// If the key does not exist, it initializes it to the amount with an infinite TTL.
// It does NOT reset the TTL of an existing key.
func Increment(ctx context.Context, key string, amount int64) (int64, error) {
	return transactionOrDefault(ctx).Increment(ctx, key, amount)
}

// Decrement atomically decrements a numeric key by the given amount.
// If the key does not exist, it initializes it to -amount with an infinite TTL.
// It does NOT reset the TTL of an existing key.
func Decrement(ctx context.Context, key string, amount int64) (int64, error) {
	return transactionOrDefault(ctx).Decrement(ctx, key, amount)
}

// Expire sets the TTL for a given key.
// If the key does not exist in the cache, [ErrItemNotFound] is returned.
func Expire(ctx context.Context, key string, ttl Duration) error {
	return transactionOrDefault(ctx).Expire(ctx, key, ttl)
}

// CounterValue retrieves the counter for the specified key.
// If the key does not exist in the cache, an error is returned.
func CounterValue(ctx context.Context, key string) (int64, error) {
	return transactionOrDefault(ctx).CounterValue(ctx, key)
}

// TTL returns the time to live for a key in the default cache backend.
//
// If the key does not exist, TTL returns 0.
//
// If any error occurs, TTL returns 0.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func TTL(ctx context.Context, key string) Duration {
	return transactionOrDefault(ctx).TTL(ctx, key)
}

// Has returns true if the key exists in the default cache backend.
//
// If any error occurs, Has returns false.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Has(ctx context.Context, key string) bool {
	return transactionOrDefault(ctx).Has(ctx, key)
}

// Delete removes a key from the default cache backend.
//
// If the key does not exist, Delete should return ErrItemNotFound.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Delete(ctx context.Context, key string) error {
	return transactionOrDefault(ctx).Delete(ctx, key)
}

// Keys returns all keys in the default cache backend.
//
// If any error occurs, Keys returns an empty slice and the error.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Keys(ctx context.Context) ([]string, error) {
	return transactionOrDefault(ctx).Keys(ctx)
}

// Clear removes all keys from the default cache backend.
//
// If any error occurs, Clear should return the error.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Clear(ctx context.Context) error {
	return transactionOrDefault(ctx).Clear(ctx)
}

// Close closes the default cache backend.
//
// If any error occurs, Close should return the error.
//
// If a transaction is active in the context, it will be called on the transaction instead.
func Close(ctx context.Context) error {
	return Default().Close(ctx)
}

// RunInTx executes the given function inside a transaction.
// The provided txCache should be used for all operations inside the function.
func RunInTx(ctx context.Context, fn func(ctx context.Context, tx Transaction) error) error {
	return Default().RunInTx(ctx, fn)
}
