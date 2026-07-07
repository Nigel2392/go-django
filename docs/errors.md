# Errors

Error handling is an important part of the framework. We utilize structured errors, wrapped errors, and database-specific errors to provide rich context throughout the application lifecycle.

## Basic Errors

For basic, constant errors, we use a string-based error type: `errs.Error`.  
This is similar to how constant errors are sometimes defined in Go, allowing for reliable `errors.Is` checks.

```go
package myapp

import "github.com/Nigel2392/go-django/src/core/errs"

const ErrInvalidInput errs.Error = "invalid input provided"
```

## The `github.com/Nigel2392/errors` Package

For more complex errors that require stack traces, formatting, or wrapping, the framework integrates with `github.com/Nigel2392/errors`.

This package extends the standard library error capabilities with error codes, causes, and structured data.

```go
import "github.com/Nigel2392/errors"

// Creating a new error with an error code
var ErrNotFound = errors.New(404, "resource not found")

// Formatting an error
var err = errors.Errorf(500, "failed to process item %d", itemID)
```

## Database Errors

Database interactions are prone to a wide variety of errors (constraints, timeouts, connections). To standardize these across different SQL drivers (SQLite, PostgreSQL, MySQL), we provide the `DatabaseError` interface located in `github.com/Nigel2392/go-django/queries/src/drivers/errors`.

```go
type DatabaseError interface {
    Error() string
    Code() DBCode
    Reason() error
    WithCause(otherErr error) DatabaseError
    Wrap(message string) DatabaseError
    Wrapf(format string, args ...any) DatabaseError
}
```

### Pre-defined Database Errors

The framework pre-defines numerous database errors that map to standard SQL errors, no matter the underlying driver.

Examples include:

- `errors.NoRows`
- `errors.MultipleRows`
- `errors.UniqueViolation`
- `errors.ForeignKeyViolation`
- `errors.TableNotFound`

These can be used with `errors.Is` to safely check the type of a database failure:

```go
import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

func GetUser(id int) (*User, error) {
    user, err := db.QueryUser(id)
    if err != nil {
        if errors.Is(err, errors.NoRows) {
            return nil, nil // Return nil gracefully if not found
        }
        return nil, err
    }
    return user, nil
}
```

### Handling Database Errors

Database errors wrap the original driver error inside `Reason()`. You can add more context to a database error while keeping the original error code by using `.Wrap()` or `.WithCause()`:

```go
if err != nil {
    // If err is a DatabaseError, it retains its DBCode but adds the context message
    return errors.InvalidDatabaseError(err).Wrap("failed to fetch user profile")
}
```

## Summary

- Use `errs.Error` for simple, package-level constant errors.
- Use `github.com/Nigel2392/errors` for application-level errors requiring codes or formatted context.
- Use the `DatabaseError` interface and predefined variables in `queries/src/drivers/errors` to handle SQL database errors uniformly across drivers.
