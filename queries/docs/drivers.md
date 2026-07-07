# Drivers Package (`queries/src/drivers`)

The `drivers` package is the abstraction layer between the `queries` ORM and the underlying database connection. Rather than dealing with raw `*sql.DB` values, the entire `queries` package operates on the `drivers.Database` interface.

---

## Why a custom drivers layer?

Go's standard `database/sql` package works well for basic use cases, but the `queries` package needs:
- A unified `QueryRowContext` interface that returns a scannable row regardless of whether a transaction is active.
- The ability to know whether the underlying driver supports `RETURNING` clauses (PostgreSQL does; SQLite uses last-insert-id; MySQL requires neither).
- A consistent error type system (see `drivers/errors`) so that driver-specific errors can be translated.

---

## Core Interfaces

### `DB`

The lowest-level database interface. Implemented by both `Database` and `Transaction`.

```go
type DB interface {
    Unwrap() any
    QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) SQLRow
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
```

### `Database`

A full database connection, extending `DB` with connection management:

```go
type Database interface {
    DB
    Ping(context.Context) error
    Driver() driver.Driver
    Begin(ctx context.Context) (Transaction, error)
    Close() error
}
```

This is the type that `APPVAR_DATABASE` expects.

### `Transaction`

A database transaction, extending `DB`:

```go
type Transaction interface {
    DB
    Finished() bool
    Commit(context.Context) error
    Rollback(context.Context) error
}
```

### `SQLRow` / `SQLRows`

Thin interfaces over `*sql.Row` / `*sql.Rows`:

```go
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
```

---

## Opening a Database

Use `drivers.Open` — **never** call `sql.Open` directly:

```go
import (
    "context"
    "github.com/Nigel2392/go-django/queries/src/drivers"
    _ "github.com/mattn/go-sqlite3"  // register the driver
)

func main() {
    var db, err = drivers.Open(context.Background(), "sqlite3", "./db.sqlite3")
    if err != nil {
        panic(err)
    }
    defer db.Close()
}
```

Supported built-in driver names:

| Driver name | Package |
|---|---|
| `sqlite3` | `github.com/mattn/go-sqlite3` |
| `mysql` / `mariadb` | `github.com/go-sql-driver/mysql` |
| `pgx` / `postgres` | `github.com/jackc/pgx/v5` |

---

## SupportsReturningType

The `queries` package automatically adapts INSERT queries based on what the driver supports:

| Constant | Meaning |
|---|---|
| `SupportsReturningNone` | No `RETURNING` support (e.g. MySQL — uses two-step query) |
| `SupportsReturningLastInsertId` | Uses `LastInsertId()` after INSERT (e.g. SQLite) |
| `SupportsReturningColumns` | Full `RETURNING` clause support (e.g. PostgreSQL) |

---

## Driver Registration

Built-in drivers are registered automatically via their init functions. If you need a **custom driver**:

```go
drivers.Register("mydriver", drivers.Driver{
    Name:              "mydriver",
    SupportsReturning: drivers.SupportsReturningLastInsertId,
    Driver:            myGoSQLDriver,
    Open: func(ctx context.Context, drv *drivers.Driver, dsn string, opts ...drivers.OpenOption) (drivers.Database, error) {
        // Open and return a drivers.Database implementation
    },
    BuildDatabaseError: func(err error) errors.DatabaseError {
        // Map driver-specific errors to DatabaseError
        return nil
    },
})
```
