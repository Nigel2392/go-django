package cache

import "context"

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

	// Increment atomically increments a numeric key by the given amount.
	// If the key does not exist, it initializes it to the amount with an infinite TTL.
	// It does NOT reset the TTL of an existing key.
	Increment(c context.Context, key string, amount int64) (int64, error)

	// Decrement atomically decrements a numeric key by the given amount.
	// If the key does not exist, it initializes it to -amount with an infinite TTL.
	// It does NOT reset the TTL of an existing key.
	Decrement(c context.Context, key string, amount int64) (int64, error)

	// CounterValue retrieves the counter for the specified key.
	// If the key does not exist in the cache, [ErrItemNotFound] is returned.
	CounterValue(c context.Context, key string) (int64, error)

	// Expire sets the TTL for a given key.
	// If the key does not exist in the cache, [ErrItemNotFound] is returned.
	Expire(c context.Context, key string, ttl Duration) error

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
	RunInTx(ctx context.Context, fn func(ctx context.Context, txCache TypedTransaction[any]) error) error
}

type TypedTransaction[T any] interface {
	TypedCache[T]
	InTransaction() bool
}

type (
	Cache              = TypedCache[any]
	TransactionalCache = TypedTransactionalCache[any]
	Transaction        = TypedTransaction[any]
)
