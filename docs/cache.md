# Cache

Go-Django delegates caching to the standalone [`github.com/Nigel2392/cache`](https://github.com/Nigel2392/cache) module. It provides typed and untyped cache interfaces, atomic counters, cache transactions, an in-memory backend, and a global backend registry.

---

## Constants and Sentinel Errors

Cache TTLs use the standard library `time.Duration` directly, along with `cache.Infinity` for non-expiring entries:

```go
const (
	DefaultCache = "default"
	Infinity     = internal.Infinity // time.Duration(1<<63 - 1)
)

const (
	OP_UNKNOWN    CacheOperation = ...
	OP_GET                       = ...
	OP_GETDEFAULT                = ...
	OP_SET                       = ...
	OP_INCR                      = ...
	OP_DECR                      = ...
	OP_CVAL                      = ...
	OP_EXPR                      = ...
	OP_TTL                       = ...
	OP_HAS                       = ...
	OP_DEL                       = ...
	OP_KEYS                      = ...
	OP_CLEAR                     = ...
	OP_CLOSE                     = ...
)

var (
	ErrItemNotFound = internal.ErrItemNotFound
	ErrInvalidType  = internal.ErrInvalidType
	ErrNotSupported = internal.ErrNotSupported
)
```

- **`DefaultCache`**: Key name (`"default"`) used for the default backend.
- **`Infinity`**: Maximum `time.Duration` used to mark entries that do not expire. Passing `0` to `Set` also stores items without expiration in `MemoryCache`.
- **`ErrItemNotFound` / `ErrInvalidType` / `ErrNotSupported`**: Sentinel errors returned when keys are missing, stored types do not match expectations (e.g. calling `Increment` on a non-integer value), or an operation is unsupported by a backend.

---

## Interfaces

### `CacheConnector`

Backends that connect to an external cache server (such as Redis or Memcached) can implement `CacheConnector`:

```go
type CacheConnector interface {
	Connect() error
	HasConnected() bool
}
```

When you retrieve a backend via `cache.GetCache(...)`, if it implements `CacheConnector` and `HasConnected()` returns `false`, `GetCache` automatically calls `Connect()`.

### `TypedCache[T]` and `Cache`

`TypedCache[T]` defines the full context-aware cache contract, and `Cache` is its `any`-typed alias:

```go
type Cache = TypedCache[any]

type TypedCache[T any] interface {
	Get(c context.Context, key string) (T, error)
	GetDefault(c context.Context, key string, defaultValue T) (T, error)
	Set(c context.Context, key string, value T, ttl time.Duration) error
	Increment(c context.Context, key string, amount int64) (int64, error)
	Decrement(c context.Context, key string, amount int64) (int64, error)
	CounterValue(c context.Context, key string) (int64, error)
	Expire(c context.Context, key string, ttl time.Duration) error
	TTL(c context.Context, key string) time.Duration
	Has(c context.Context, key string) bool
	Delete(c context.Context, key string) error
	Keys(c context.Context) ([]string, error)
	Clear(c context.Context) error
	Close(c context.Context) error
}
```

- **`Get` / `GetDefault`**: Retrieves a key or returns `ErrItemNotFound` (or `defaultValue` when missing in `GetDefault`).
- **`Set` / `Expire` / `TTL`**: Stores a value with a TTL, updates an existing key's TTL (`ErrItemNotFound` if missing), or reads the remaining TTL (`0` if missing).
- **`Increment` / `Decrement` / `CounterValue`**: Atomically adjusts numeric keys (`Increment`/`Decrement` initialize missing keys to `amount` or `-amount` with infinite TTL and do **not** reset an existing key's TTL).
- **`Has` / `Delete` / `Keys` / `Clear` / `Close`**: Checks membership, deletes a key (`ErrItemNotFound` if missing), lists keys, clears all items, or closes the backend.

### Transactions: `Transaction`, `TransactionalCache`, and Typed Variants

```go
type Transaction = TypedTransaction[any]
type TransactionalCache = TypedTransactionalCache[any]

type TypedTransaction[T any] interface {
	TypedCache[T]
	InTransaction() bool
}

type TypedTransactionalCache[T any] interface {
	TypedCache[T]
	RunInTx(ctx context.Context, fn func(ctx context.Context, txCache TypedTransaction[any]) error) error
}
```

Inside `RunInTx`, operations should use the `txCache` (or the propagated `ctx` if your backend attaches the transaction to context).

---

## In-Memory Backend: `MemoryCache[T]`

`MemoryCache[T]` is a built-in thread-safe in-memory implementation backed by a map:

```go
func NewMemoryCache(interval time.Duration) *MemoryCache[interface{}]
func NewGenericMemoryCache[T any]() *MemoryCache[T]

func (c *MemoryCache[T]) Run(interval time.Duration)
func (c *MemoryCache[T]) Len(_ context.Context) int
func (c *MemoryCache[T]) RunInTx(ctx context.Context, fn func(ctx context.Context, txCache TypedTransaction[T]) error) error
// Plus all TypedCache[T] methods
```

- **`NewMemoryCache(interval)`**: Creates `*MemoryCache[any]` and immediately starts a background cleanup loop running every `interval` (`c.Run(interval)`).
- **`NewGenericMemoryCache[T]()`**: Creates a strongly-typed `*MemoryCache[T]`. Unlike `NewMemoryCache`, it does **not** start the cleanup ticker automatically—call `c.Run(interval)` if periodic eviction of expired entries is desired.

Do note that `*MemoryCache[any]` satisfies `TransactionalCache` (`TypedTransactionalCache[any]`). For any concrete type `T` other than `any`, `*MemoryCache[T]` satisfies `TypedCache[T]`, while its `RunInTx` expects `func(context.Context, TypedTransaction[T]) error`.

---

## Backend Registry and Package Functions

### Managing Backends

- **`RegisterCache(name string, cache TransactionalCache)`**: Registers a named backend.
- **`RemoveCache(name string)`**: Unregisters a backend by name.
- **`GetCache(names ...string) TransactionalCache`**: Returns the first registered backend matching `names` (or `DefaultCache` if no names are given), calling `Connect()` if the backend implements `CacheConnector` and is not yet connected. Returns `nil` if not found.
- **`SetDefault(cache TransactionalCache)`**: Registers `cache` under `DefaultCache` (`"default"`).
- **`Default() TransactionalCache`**: Returns `GetCache(DefaultCache)`, or `nil` if no default backend has been registered.
- **`GetDefaultTTL(provided time.Duration) time.Duration`** / **`SetGetDefaultTTL(fn func(provided time.Duration) time.Duration)`**: Reads or overrides the package helper for computing default TTLs (defaults to `1 * time.Hour` when `provided <= 0`).

### Default Backend Operations and Context Transactions

All top-level cache operation functions operate on the transaction stored in `ctx` if present, or otherwise fall back to `Default()`:

```go
func Get(ctx context.Context, key string) (interface{}, error)
func GetDefault(ctx context.Context, key string, defaultValue interface{}) (interface{}, error)
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
func Increment(ctx context.Context, key string, amount int64) (int64, error)
func Decrement(ctx context.Context, key string, amount int64) (int64, error)
func CounterValue(ctx context.Context, key string) (int64, error)
func Expire(ctx context.Context, key string, ttl time.Duration) error
func TTL(ctx context.Context, key string) time.Duration
func Has(ctx context.Context, key string) bool
func Delete(ctx context.Context, key string) error
func Keys(ctx context.Context) ([]string, error)
func Clear(ctx context.Context) error
func Close(ctx context.Context) error
func RunInTx(ctx context.Context, fn func(ctx context.Context, tx Transaction) error) error

func ContextWithTransaction(ctx context.Context, tx Transaction) context.Context
func TransactionFromContext(ctx context.Context) (t Transaction, ok bool)
func Typed_ContextWithTransaction[T any](ctx context.Context, tx TypedTransaction[T]) context.Context
func Typed_TransactionFromContext[T any](ctx context.Context) (t TypedTransaction[T], ok bool)
```

Do note that `cache` does **not** automatically register a default backend in `init()`. Before calling top-level functions such as `cache.Get` or `cache.Set`, register a default backend via `cache.SetDefault(...)`:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/cache"
)

func main() {
	c := cache.NewMemoryCache(30 * time.Second)
	cache.SetDefault(c)
	defer c.Close(context.Background())

	ctx := context.Background()

	if err := cache.Set(ctx, "greeting", "hello", 5*time.Minute); err != nil {
		panic(err)
	}

	val, err := cache.Get(ctx, "greeting")
	if err != nil {
		panic(err)
	}
	fmt.Println(val) // Output: hello

	count, err := cache.Increment(ctx, "visits", 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(count) // Output: 1
}
```

---

## Additional Subpackages

The `github.com/Nigel2392/cache` module also includes two specialized subpackages:

- **`github.com/Nigel2392/cache/buffer`**: Provides `CacheBuffer[T]`, a staging `TypedTransaction[T]` wrapper created with `NewCacheBuffer(parent)` that buffers writes and exposes `Flush()`.
- **`github.com/Nigel2392/cache/generational`**: Provides generational cache invalidation helpers (`GetGen`, `RollOver`, `GetKey`, `GetItem`, and cache-specific variants).
