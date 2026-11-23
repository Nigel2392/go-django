# Errors

Go-Django combines lightweight sentinel errors, structured HTTP errors, assertion helpers, and uniform database error wrappers so failures remain consistent across HTTP handlers, forms, and SQL drivers.

---

## `src/core/errs`

The `github.com/Nigel2392/go-django/src/core/errs` package provides Go-Django's core error primitives, validation errors, and multi-error wrappers.

### `Error` and Sentinel Constants

`errs.Error` is a string-based error type suited for constant sentinel errors that work reliably with `errors.Is`:

```go
package errs

type Error string

func (e Error) Error() string
func (e Error) Is(other error) bool

const (
	ErrNotImplemented Error = "This operation is not implemented"
	ErrFieldRequired  Error = "Required field cannot be empty"
	ErrInvalidSyntax  Error = "Invalid syntax for value"
	ErrInvalidType    Error = "Invalid type provided"
	ErrInvalidValue   Error = "Invalid value provided"
	ErrLengthMin      Error = "Minimum length not met"
	ErrLengthMax      Error = "Maximum length exceeded"
	ErrUnknown        Error = "Unknown error occurred"
)
```

Define your own constant sentinel errors the same way:

```go
package myapp

import "github.com/Nigel2392/go-django/src/core/errs"

const ErrInvalidInput errs.Error = "invalid input provided"
```

### `DjangoError` and `ValidationError[T]`

`ValidationError[T]` pairs a comparable field/key identifier `Name` with an underlying error and implements `DjangoError` and `json.Marshaler`:

```go
type DjangoError interface {
	error
	Is(error) bool
	DjangoError() // marker method
}

type ValidationError[T comparable] struct {
	Name T
	Err  error
}

func NewValidationError[T comparable](name T, err any) ValidationError[T]
func Errors[T comparable](m []ValidationError[T]) []error
```

- **`NewValidationError`**: Accepts an `error`, `string`, or arbitrary value (`fmt.Errorf("%v", e)`) and binds it to `name`.
- **`ValidationError.Is`**: Matches another `ValidationError[T]` (or `*ValidationError[T]`) with the same `Name` and matching `Err`, or matches `Err` directly against an `errs.Error` or wrapped error target.
- **`Errors`**: Converts `[]ValidationError[T]` into `[]error`.

### `MultiError`, `WithMessage`, and `Convert`

```go
type MultiError struct {
	Errors []error
}

func WrapErrors(errs ...error) error
func NewMultiError(errs ...error) *MultiError
func (m *MultiError) Append(err error)
func (m *MultiError) Is(target error) bool
func (m *MultiError) As(target interface{}) bool
func (m *MultiError) Error() string
func (m *MultiError) MarshalJSON() ([]byte, error)
func (m *MultiError) Unwrap() []error
func (m *MultiError) Len() int

type WithMessage struct {
	// unexported fields
}

func Wrap(err error, message string) error
func (w *WithMessage) Error() string
func (w *WithMessage) Is(other error) bool
func (w *WithMessage) Unwrap() error

func Convert(fmtOrErr any, default_ any, args ...any) error
```

- **`WrapErrors`**: Discards `nil` errors, returns `nil` if empty, returns the single error unchanged if only one is non-nil, or returns a `*MultiError` if multiple errors are present.
- **`Wrap`**: Wraps `err` in `*WithMessage` formatted as `"<message>: <err>"` while preserving `errors.Is` and `errors.Unwrap` (`nil` if `err == nil`).
- **`Convert`**: Converts an `error`, `string` (formatted with `args...` if provided), or arbitrary value into an `error`, falling back to `default_` when `fmtOrErr == nil`.

---

## `github.com/Nigel2392/errors`

For application errors with string error codes, causes, and related errors, Go-Django integrates with `github.com/Nigel2392/errors`.

In `github.com/Nigel2392/errors`, `GoCode` is a `string` type (not an `int`):

```go
type GoCode string

const CodeUnknown GoCode = "Unknown"

type Error struct {
	Code    GoCode
	Reason  error
	Message string
	Related []error
}

type AbstractError[ERROR AbstractError[ERROR]] interface {
	error
	Cause() error
	Is(chk error) bool
	WithCause(reason error) AbstractError[ERROR]
	Unwrap() error
	Wrap(message string) ERROR
	Wrapf(format string, args ...any) ERROR
}

func New(code GoCode, message string, related ...error) AbstractError[Error]
func Errorf(code GoCode, format string, args ...interface{}) AbstractError[Error]
func WithMessage(err error, message string) error
func WithMessagef(err error, format string, args ...interface{}) error
func Wrap(err error, message string) error
func Wrapf(err error, format string, args ...interface{}) error
func Cause(err error) error
func AsType[ERROR error](err error) (ERROR, bool)
func Exec[ERROR AbstractError[ERROR], T1 any, T2 error](_err ERROR, exec func() (T1, error)) (T1, AbstractError[ERROR])
```

Example usage:

```go
package myapp

import "github.com/Nigel2392/errors"

var ErrNotFound = errors.New("NOT_FOUND", "resource not found")

func loadItem(itemID int) error {
	return errors.Errorf("INTERNAL_ERROR", "failed to process item %d", itemID)
}
```

---

## `src/core/except`

The `github.com/Nigel2392/go-django/src/core/except` package defines HTTP-oriented server errors and panic-based request assertions:

```go
type ServerError interface {
	error
	StatusCode() int
	UserMessage() string
}

type HttpError struct {
	Message error
	Code    int
}

func NewServerError(code int, msg any, args ...any) ServerError
func GetServerError(err error) ServerError
```

### Constructing vs. Raising `ServerError`

`except` provides two distinct categories of helpers:

1. **Non-panicking constructors** (`NewServerError` and `Raise*`): These **only construct and return** a `ServerError` value; they do **not** panic:
   ```go
   func RaiseBadRequest(msg any, args ...any) ServerError
   func RaiseUnauthorized(msg any, args ...any) ServerError
   func RaiseForbidden(msg any, args ...any) ServerError
   func RaiseNotFound(msg any, args ...any) ServerError
   func RaiseMethodNotAllowed(msg any, args ...any) ServerError
   func RaiseInternalServerError(msg any, args ...any) ServerError
   ```
2. **Panicking assertions** (`Fail`, `Assert`, `AssertNil`, `AssertNotNil`, `AssertEqual`, `AssertContains`, `AssertContainsFunc`): When a condition fails, these build `&HttpError{Code: code, Message: ...}` and call `assert.Fail(err)`, which **panics** so the request recoverer middleware can handle it:
   ```go
   func Fail(code int, msg any, args ...interface{}) error
   func Assert(cond interface{}, code int, msg any, args ...interface{}) error
   func AssertNil(v any, code int, msg any, args ...interface{}) error
   func AssertNotNil(v any, code int, msg any, args ...interface{}) error
   func AssertEqual[T comparable](a, b T, code int, msg any, args ...interface{}) error
   func AssertContains[T comparable](v T, a []T, code int, msg any, args ...interface{}) error
   func AssertContainsFunc[T any](a []T, f func(T) (in bool), code int, msg any, args ...interface{}) error
   ```

```go
func ItemHandler(w http.ResponseWriter, r *http.Request) {
	item, err := fetchItem(r.Context())
	except.Assert(err == nil, http.StatusInternalServerError, "failed to load item")
	except.AssertNotNil(item, http.StatusNotFound, "item not found")
}
```

### Caveat: `except.Assert(nil, ...)`

Do not pass an untyped `nil` directly to `except.Assert`. Internally, `Assert` calls `reflect.TypeOf(cond)` and immediately inspects `rTyp.Kind()`, which panics with a nil-pointer dereference if `cond` is untyped `nil` instead of constructing an `HttpError`. Use a `bool` expression, a `func() bool`, a typed pointer, or `AssertNotNil` / `AssertNil`.

---

## `src/core/assert`

The `github.com/Nigel2392/go-django/src/core/assert` package powers internal invariants and `except.Fail`:

```go
type AssertError = errs.Error

var (
	Panic                func(error) = func(err error) { panic(err) }
	LogOnError           func(error)
	AssertionFailedError AssertError = "assertion failed"
)

func FailFunc(failFn func(err error) error, msg any, args ...interface{}) error
func Fail(msg any, args ...interface{}) error
func Assert(cond bool, msg any, args ...interface{}) error
func AssertFunc(fail func(err error) error, cond bool, msg any, args ...interface{}) error
func Equal[T comparable](a, b T, msg any, args ...interface{}) error
func True(cond bool, msg any, args ...interface{}) error
func False(cond bool, msg any, args ...interface{}) error
func Truthy(v any, msg any, args ...interface{}) error
func Falsy(v any, msg any, args ...interface{}) error
func Err(err error) error
func ErrNil(err error) error
func Gt[T any](a []T, min int, msg any, args ...interface{}) error
func Lt[T any](a []T, max int, msg any, args ...interface{}) error
func Gte[T any](a []T, min int, msg any, args ...interface{}) error
func Lte[T any](a []T, max int, msg any, args ...interface{}) error
func Contains[T comparable](v T, a []T, msg any, args ...interface{}) error
func ContainsFunc[T any](a []T, f func(T) (in bool), msg any, args ...interface{}) error
```

All failing assertions build an error wrapped with `AssertionFailedError`, invoke `LogOnError` if set, and call `Panic(err)` (which panics by default).

### Caveat: Inverted `assert.Err` and `assert.ErrNil` Semantics

In `src/core/assert`, `Err` and `ErrNil` currently have inverted checks relative to their names:

- `assert.Err(err)` checks `err == nil` and **panics if `err != nil`** (printing `"expected non-nil error: ..."`).
- `assert.ErrNil(err)` checks `err != nil` and **panics if `err == nil`** (printing `"expected nil error"`).

To keep call sites unambiguous, prefer `assert.True(err == nil, ...)` or `assert.True(err != nil, ...)` instead of `assert.Err` or `assert.ErrNil`.

---

## `Application.ServerError` Integration

When `django.APPVAR_RECOVERER` is enabled (it defaults to `true`), `django.Application` wraps HTTP routing with `middleware.Recoverer(a.ServerError)`. Any panic carrying a `except.ServerError` (such as from `except.Assert` or `except.Fail`) is extracted via `except.GetServerError(err)` and routed through `a.ServerError`:

1. If `except.GetServerError(err)` returns `nil`, `Application` logs the unexpected error and returns a plain HTTP `500` (`"An unexpected error occurred"`).
2. Otherwise, `Application` logs `StatusCode()`, the truncated URL, and `UserMessage()`, then checks `a.Settings.Get(django.APPVAR_ErrorCode(code))` (key format `"HandleErrorCode<code>"`). If a handler of type `func(http.ResponseWriter, *http.Request, except.ServerError)` is configured, it is invoked and handling stops.
3. If no status-specific setting handler is registered, `Application` executes all registered [`django.HOOK_SERVER_ERROR`](./hooks.md) hooks (`ServerErrorHook`).
4. If none of the hooks wrote to the response writer, `Application` falls back to `http.Error(w, message, code)`.

Note that `StatusCode()` must be non-zero; `handleErrorCodePure` asserts `code != 0`.

```go
app := django.App(
	django.Configure(map[string]any{
		django.APPVAR_ErrorCode(http.StatusNotFound): func(
			w http.ResponseWriter,
			r *http.Request,
			err except.ServerError,
		) {
			http.Error(w, "Custom 404: "+err.UserMessage(), http.StatusNotFound)
		},
	}),
)
```

---

## Database Errors

To standardize SQL failures across SQLite, PostgreSQL, MySQL, and MariaDB, `github.com/Nigel2392/go-django/queries/src/drivers/errors` defines `DatabaseError`, `DBCode`, and shared sentinel errors such as `NoRows`, `MultipleRows`, `UniqueViolation`, `ForeignKeyViolation`, and `TableNotFound`:

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

Use `errors.Is` to match driver-independent failure conditions, or `errors.InvalidDatabaseError(err)` to wrap unknown driver errors into a `DatabaseError`:

```go
import (
	"context"
	"errors"

	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

func getUserOrNil(ctx context.Context, id int64) (*User, error) {
	row, err := queries.GetQuerySet(&User{}).WithContext(ctx).Filter("ID", id).Get()
	if err != nil {
		if errors.Is(err, dberrors.NoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.Object, nil
}
```

For the full list of error codes, driver mappings, and `GoCode` / `DBCode` types, see the [queries errors documentation](../queries/docs/errors.md).
