# Drivers Package (`queries/src/drivers`)

The `drivers` package provides the database abstraction layer between the `queries` ORM and underlying SQL engines. Rather than interacting directly with `*sql.DB` or `*pgxpool.Pool`, `queries` operates on `drivers.Database` and `drivers.Transaction` values opened via `drivers.Open`.

Importing `github.com/Nigel2392/go-django/queries/src/drivers` automatically registers four built-in drivers (`"sqlite3"`, `"mysql"`, `"mariadb"`, and `"postgres"`), along with default type mappings in [`drivers/dbtype`](#dbtype-registry-driversdbtype).

---

## Core Interfaces

```go
type Unwrapper interface {
	Unwrap() any
}

type SQLRow interface {
	Err() error
	Scan(dest ...any) error
}

type SQLRows interface {
	SQLRow
	Close() error
	Next() bool
	Columns() ([]string, error)
	NextResultSet() bool
}

type DB interface {
	Unwrap() any
	Driver() driver.Driver
	QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) SQLRow
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Database interface {
	DB
	Ping(context.Context) error
	Driver() driver.Driver
	Begin(ctx context.Context) (Transaction, error)
	Close() error
}

type Transaction interface {
	DB
	Finished() bool
	Commit(context.Context) error
	Rollback(context.Context) error
}

type DatabaseSpecificTransaction interface {
	Transaction
	DatabaseName() string
}
```

- `DB`: Base interface shared by both `Database` and `Transaction`. `Unwrap()` returns the underlying driver handle (`*sql.DB`, `*sql.Tx`, `*pgxpool.Pool`, `*pgxpool.Conn`, or `pgx.Tx`), and `Driver()` returns the underlying `database/sql/driver.Driver`.
- `Database`: Full connection interface expected by `django.APPVAR_DATABASE`. Calling `Begin(ctx)` starts a `Transaction`.
- `Transaction`: Extends `DB` with `Finished()`, `Commit(ctx)`, and `Rollback(ctx)`. `DatabaseSpecificTransaction` additionally exposes `DatabaseName() string` on QuerySet-managed transactions.

---

## Opening a Database

Always use `drivers.Open` instead of `sql.Open` or `pgxpool.New` directly:

```go
func Open(ctx context.Context, driverName, dsn string, opts ...OpenOption) (Database, error)
```

Example with `OpenOption` configuration:

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/jackc/pgx/v5/pgxpool"
)

func openSQLite(ctx context.Context) (drivers.Database, error) {
	return drivers.Open(
		ctx,
		"sqlite3",
		"./db.sqlite3",
		drivers.SQLDBOption(func(driverName string, db *sql.DB) error {
			db.SetMaxOpenConns(1)
			db.SetConnMaxLifetime(time.Hour)
			return nil
		}),
	)
}

func openPostgres(ctx context.Context) (drivers.Database, error) {
	return drivers.Open(
		ctx,
		"postgres",
		"postgres://user:password@localhost:5432/dbname?sslmode=disable",
		drivers.PGXOption(func(driverName string, pool *pgxpool.Pool) error {
			// Customize the *pgxpool.Pool if needed
			return nil
		}),
	)
}
```

- `drivers.SQLDBOption(func(driverName string, db *sql.DB) error)`: Applies configuration when using `database/sql`-backed drivers (`sqlite3`, `mysql`, `mariadb`). Returns `errors.NotImplemented` on non-`*sql.DB` backends, which `OpenPGX` ignores.
- `drivers.PGXOption(func(driverName string, db *pgxpool.Pool) error)`: Applies configuration when using the `postgres` (`pgx/v5`) driver. Returns `errors.NotImplemented` on `*sql.DB` backends, which `OpenSQL` ignores.

---

## Built-In Drivers and `SupportsReturningType`

The ORM inspects `SupportsReturning` to decide how newly inserted rows retrieve generated primary keys and defaults:

```go
type SupportsReturningType string

const (
	SupportsReturningNone         SupportsReturningType = ""
	SupportsReturningLastInsertId SupportsReturningType = "last_insert_id"
	SupportsReturningColumns      SupportsReturningType = "columns"
)
```

| Driver Name | Underlying Package | `SupportsReturning` | Notes |
|---|---|---|---|
| `"sqlite3"` | `github.com/mattn/go-sqlite3` | `SupportsReturningColumns` | Uses `RETURNING` (`sqlite3` driver via `OpenSQL`) |
| `"mysql"` | `github.com/go-sql-driver/mysql` | `SupportsReturningLastInsertId` | Uses `LastInsertId()`; requires `multiStatements=true&interpolateParams=true` for bulk inserts/updates |
| `"mariadb"` | `DriverMariaDB` (wraps `go-sql-driver/mysql`) | `SupportsReturningColumns` | Custom wrapper in [drivers_mardiadb.go](../src/drivers/drivers_mardiadb.go) supporting `INSERT ... RETURNING`; also requires `multiStatements=true&interpolateParams=true` |
| `"postgres"` | `github.com/jackc/pgx/v5/stdlib` + `pgxpool` | `SupportsReturningColumns` | Note the registered name is `"postgres"` (not `"pgx"`). Opens a `*pgxpool.Pool` via `OpenPGX` |

### Driver Registration and `ExplainQuery`

Each driver is described by `drivers.Driver` and registered via `drivers.Register(name, driver)`:

```go
type Driver struct {
	Name               string
	SupportsReturning  SupportsReturningType
	Driver             driver.Driver
	Open               func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error)
	BuildDatabaseError func(err error) errors.DatabaseError
	ExplainQuery       func(ctx context.Context, q DB, query string, args []any) (string, error)
}
```

- `drivers.Register(name string, driver Driver)`: Registers a custom driver by name and `reflect.TypeOf(driver.Driver)`. Panics if `name`/`driver.Name` conflict or are empty, or if `Driver` or `Open` is nil.
- `drivers.Retrieve(nameOrType any) (*Driver, bool)`: Looks up a driver by name (`string`), `reflect.Type`, `driver.Driver`, `*Driver`, or any type implementing `Driver() driver.Driver` (such as `drivers.DB`).
- `drivers.Change(name string, fn func(driver *Driver))`: Mutates an existing registered driver in place (primarily for tests).
- `Driver.ExplainQuery`: Runs `EXPLAIN` (SQLite, MySQL, MariaDB) or `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)` (Postgres) when invoked through `(*drivers.Query).Explain(ctx, db)`.

---

## Built-In ORM Types and Scanning

The `drivers` package defines database-aware Go types that implement `sql.Scanner`, `driver.Valuer`, and in several cases JSON/form field helpers:

| Type | Underlying / Purpose | Notes |
|---|---|---|
| `Text`, `String`, `Char` | `string` variants (`dbtype.Text`, `dbtype.String`, `dbtype.Char`) | Differentiate DDL column types in migrations |
| `Int`, `Uint`, `Float`, `Bool` | `int64`, `uint64`, `float64`, `bool` | Include `.String()` and `.Scan(...)` helpers |
| `Bytes`, `BLOB` | `[]byte` variants (`dbtype.Bytes`, `dbtype.BLOB`) | `Scan` accepts `string` or `[]byte` |
| `JSON[T]` | `struct { Data T; Null bool }` | Marshals/unmarshals `Data` to/from JSON; `DBType()` returns `dbtype.JSON` |
| `UUID` | `uuid.UUID` (`NewUUID()`) | Handles `[16]byte` from `pgx` as well as standard UUID scan formats |
| `ULID` | `ulid.ULID` (`NewULID()`) | Stores as 26-char string; scans string or 16/26-byte slices |
| `Email` | `mail.Address` (`MustParseEmail(addr)`) | Maps to `dbtype.String`; parses RFC 5322 addresses on scan |
| `Timestamp` | `time.Time` (`CurrentTimestamp()`) | UTC truncated to `time.Millisecond` |
| `LocalTime` | `time.Time` (`CurrentLocalTime()`) | Local time truncated to `time.Second` |
| `DateTime` | `time.Time` (`CurrentDateTime()`) | UTC truncated to `time.Second` |
| `Value[T]` | Wrapper `{ V T }` for polymorphic/typed JSON serialization | Serializes `{"go_type": ..., "value": ...}` via `TypeFromString` / `StringForType` |

### `ScanTo` and `ScanFlag`

`drivers.ScanTo` delegates to `github.com/Nigel2392/versatile/scan` to scan arbitrary database values into typed pointers without unnecessary reflection:

```go
type ScanFlag = scan.ScanFlag

const (
	SF_NONE        = scan.SF_NONE
	SF_SQL_SCANNER = scan.SF_SQL_SCANNER
	SF_STRCONV     = scan.SF_STRCONV
	SF_REFLECTCONV = scan.SF_REFLECTCONV
	SF_CONVS       = scan.SF_CONVS
	SF_DEFAULT     = scan.SF_DEFAULT
)

func ScanTo[T any](dst *T, src any, opts ScanFlag) (wasSet bool, err error)
```

For preparing query parameters before passing them to `database/sql` or `pgx`, `drivers.GetValue(arg any) (driver.Value, error)` unwraps `driver.Valuer`, extracts primary keys from `attrs.Definer`, and normalizes Go integer/float/slice types.

---

## SQL Logging and Query Observability

### Logging SQL Queries

Query execution in both `OpenSQL` and `OpenPGX` passes through `drivers.LogSQL`:

- `drivers.LOG_SQL_QUERIES` (default `true`): Global fallback toggle for SQL query logging under namespace `LOG_SQL_NAMESPACE` (`"SQL"`).
- `drivers.SetLogSQLContext(ctx context.Context, log bool) context.Context`: Overrides SQL logging for operations using the returned context.
- `drivers.LogSQL(ctx context.Context, from string, err error, query string, args ...any) bool`: Logs queries at debug level (or error level when `err != nil`), prefixing `TX.` if marked as part of a transaction.
- `drivers.LogSQLScope(ctx context.Context, log bool, fn func(context.Context) error) (context.Context, error, bool)`: Executes `fn` with a scoped logging context and returns `(newCtx, err, logged)`.

### Capturing Query Metrics in Context

You can attach a database handle or a query collector to a `context.Context`:

- `drivers.ContextWithDatabase(ctx context.Context, db DB) context.Context` and `drivers.ContextDatabase(ctx context.Context, defaultDb ...DB) DB` store and retrieve a `drivers.DB` on a context.
- `drivers.ContextWithQueryInfo(ctx context.Context) (context.Context, *QueryInformation)` attaches a `*QueryInformation` recorder to `ctx`; `drivers.ContextQueryInfo(ctx)` retrieves it.

```go
ctx, info := drivers.ContextWithQueryInfo(context.Background())

// Execute queries using ctx...
// Each driver call records a *drivers.Query entry into info.Queries.

total := info.TotalTime()
wall := info.TotalExecutionTime()
avg := info.AverageTime()
if slowest := info.Slowest(); slowest != nil {
	plan, err := slowest.Explain(ctx, db)
	_ = plan
	_ = err
}
```

Each recorded `*drivers.Query` contains `Driver`, `Flags` (`Q_QUERY`, `Q_QUERYROW`, `Q_EXEC`, `Q_PING`, `Q_TSTART`, `Q_TCOMMIT`, `Q_TROLLBACK`, `Q_MULTIPLE`), `Query`, `Args`, `Error`, `Start`, and `TimeTaken`.

---

## `dbtype` Registry (`drivers/dbtype`)

The `queries/src/drivers/dbtype` package maps Go `reflect.Type` and `reflect.Kind` values to abstract SQL column types (`dbtype.Type`):

```go
const (
	Invalid Type = iota
	Text
	String
	Char
	Int
	Uint
	Float
	Decimal
	Bool
	UUID
	ULID
	Bytes
	JSON
	BLOB
	Timestamp
	LocalTime
	DateTime

	DEFAULT = Text
)
```

Custom Go types can declare their column type in two ways:

1. Implement `dbtype.CanDBType` (`DBType() dbtype.Type`) or `dbtype.CanDBTypeString` (`DBType() string`) on the value type or on the field itself.
2. Register the type before initialization completes using `dbtype.Add(srcTyp, dbType, forceKind...)`.

Key registry functions and `drivers` type lookup helpers:

- `dbtype.Add(srcTyp any, dbType Type, forceKind ...bool) bool`: Registers a type or `reflect.Kind`. Returns `false` if the registry is already locked.
- `dbtype.For(typ reflect.Type) (Type, bool)`: Resolves a registered `dbtype.Type` (falling back to `dbtype.DEFAULT` / `Text` with `exists=false`).
- `dbtype.NewFromString(s string) (Type, bool)`: Parses uppercase type names such as `"TEXT"`, `"STRING"`, `"INT"`, `"JSON"`, `"DATETIME"`.
- `dbtype.Lock()` / `dbtype.IsLocked()`: Locks `dbtype.TYPES` after emitting its `Locked` signal. Calling `dbtype.Lock()` a second time panics.
- `drivers.RegisterGoType(typ any, typObj ...any)` and `drivers.TypeFromString(typeString string) (reflect.Type, bool)`: Map `"<pkgpath>.<TypeName>"` strings to `reflect.Type` (used by `Value[T]` and migration defaults). Note that `drivers.TypeFromString` panics if `dbtype.Lock()` has not been called yet.

In a normal Go-Django application, `app.Initialize()` calls `dbtype.Lock()` automatically right before `BeforeModelsReady` and model registration. In standalone tests or scripts that invoke `drivers.TypeFromString` or the migration engine without `django.App`, call `dbtype.Lock()` explicitly once before use.

---

## Encrypted and Hashed Columns (`drivers/securedb`)

Package `github.com/Nigel2392/go-django/queries/src/drivers/securedb` provides two `sql.Scanner` / `driver.Valuer` / `dbtype.CanDBType` types for storing sensitive strings (`dbtype.String`):

### `EncryptedString`

`EncryptedString` encrypts plaintext via `src/core/secrets` on save and base64-encodes the ciphertext in the database:

```go
type UserSecret struct {
	models.Model
	ID       int64
	APIToken *securedb.EncryptedString
}

// Create with plaintext (encrypted lazily in Value() or eagerly via SetValue):
secret := &UserSecret{
	APIToken: securedb.NewEncryptedString("tok_live_123"),
}

// Update an existing secret:
if err := secret.APIToken.SetValue("tok_live_456"); err != nil {
	return err
}
```

- `NewEncryptedString(raw string) *EncryptedString`: Wraps plaintext for insertion. Passing `""` stores SQL `NULL` when saved.
- `(*EncryptedString).SetValue(raw string) error`: Immediately calls `secrets.Encrypt` and updates `Raw`, internal state, and `Ciphertext`.
- `(*EncryptedString).String() string`: Returns `"<nil>"` when empty or `"********"` when set so plaintext/ciphertext never leaks in logs.

### `StringHash[HASHER]`

`StringHash[HASHER]` stores one-way hashes (such as bcrypt passwords) parameterized by a `HasherType`:

```go
type HasherType interface {
	HashValue(b string) (string, error)
	CheckValueHash(hashedPassword, value string) error
}

type BcryptHasher struct{}
```

- `securedb.NewString[HASHER](raw string) *StringHash[HASHER]`: Stores raw text to be hashed on `Value()` or `Check()`.
- `securedb.NewStringHash[HASHER](hash string) *StringHash[HASHER]`: Wraps an already-hashed string.
- `(*StringHash[H]).SetValue(raw string) error`: Hashes `raw` immediately and updates `Hash` and `Raw`.
- `(*StringHash[H]).Check(value string) error`: Verifies `value` against `Hash` (hashing `Raw` first if `Hash` is empty), returning `securedb.ErrSecureValue` on mismatch (with `BcryptHasher`).

### `securedb` Caveats

- **Always use pointer fields (`*securedb.EncryptedString`, `*securedb.StringHash[H]`):** `Value()`, `Scan()`, `DBType()`, `IsZero()`, `String()`, `SetValue()`, and `Check()` all have pointer receivers. Non-pointer struct fields will not match `driver.Valuer` or `dbtype.CanDBType` checks when passed by value.
- **Always call `SetValue()` to update `StringHash[H]`:** `StringHash[H].Value()` checks `if p.Hash != "" { return p.Hash, nil }` before checking `p.Raw`. If you load a row from the database (`p.Hash` is non-empty) and mutate `p.Raw` directly without calling `p.SetValue(newRaw)`, the old hash will be written back to the database.
- **Do not call `Value()` directly on a nil `*StringHash[H]`:** Unlike `EncryptedString.Value()`, `StringHash.Value()` does not guard against a nil receiver (though `drivers.GetValue` skips nil pointers before calling `Value()`).
- **`secrets` configuration is required at runtime:** `EncryptedString.SetValue`, `Scan`, and `Value` call `secrets.Encrypt` / `secrets.Decrypt`. Your application must configure `secrets.APPVAR_SECRET_KEY` (or `secrets.APPVAR_ENCRYPTION_BACKEND`) in `django.Global.Settings` (see [Secrets](../../docs/secrets.md)); otherwise `secrets.SECRET_KEY()` fails with an assertion panic.

---

## Caveats

- **Postgres `BuildDatabaseError` does not translate `pgconn.PgError` codes:** In [drivers_postgres.go](../src/drivers/drivers_postgres.go), `Driver.BuildDatabaseError` is wired directly to `errors.InvalidDatabaseError`. While SQLite and MySQL/MariaDB map constraint and schema errors to typed sentinels (`errors.UniqueViolation`, `errors.ForeignKeyViolation`, etc.), Postgres driver errors wrap the underlying error with code `errors.CodeInvalid` (`E000`) instead of `errors.UniqueViolation` or similar sentinels. Inspect the unwrapped `pgconn.PgError` with `errors.As` when handling Postgres-specific error codes.
- **MySQL / MariaDB bulk DSN parameters:** Bulk inserts and updates require `multiStatements=true&interpolateParams=true` in the DSN.
