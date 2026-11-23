# Errors Package (`queries/src/drivers/errors`)

Package `github.com/Nigel2392/go-django/queries/src/drivers/errors` provides unified Go-level ORM errors (`Error`), structured SQL backend errors (`DatabaseError`), and standard `errors.Is` / `errors.As` / wrapping helpers.

See [Errors](../../docs/errors.md) for Go-Django's core HTTP and error-handling utilities.

---

## Types and Constructors

```go
type GoCode = errors.GoCode
type Error = errors.AbstractError[errors.Error]

func New(code GoCode, message string, related ...error) Error
func Errorf(code GoCode, format string, args ...interface{}) Error
```

`Error` is used for ORM-level failures (such as no rows, multiple rows, missing fields, or missing where clauses).

```go
type DBCode string

func (c DBCode) String() string

type DatabaseError interface {
	Error() string
	Code() DBCode
	Reason() error
	WithCause(otherErr error) DatabaseError
	Wrap(message string) DatabaseError
	Wrapf(format string, args ...any) DatabaseError
}

func InvalidDatabaseError(err error) DatabaseError
func UnknownDatabaseError(code DBCode, message string, related ...error) DatabaseError
```

`DatabaseError` represents errors translated by a driver's `BuildDatabaseError` hook:

- `InvalidDatabaseError(err)`: Wraps an unrecognized driver error with `CodeInvalid` (`"E000"`) and `err` as its `Reason()`.
- `UnknownDatabaseError(code, message, related...)`: Constructs a custom `DatabaseError` sentinel with a `DBCode` and optional related errors matched by `errors.Is`.
- `(*databaseError).Is(chk error) bool`: Matches another `*databaseError` with the same `DBCode` (or original sentinel pointer), matches the wrapped `Reason()`, and matches any `related` error (for example, `errors.Is(err, errors.ConstraintViolation)` is `true` for `errors.UniqueViolation`).

---

## Wrapping and Inspection Helpers

The package re-exports standard library and `github.com/pkg/errors` helpers so callers do not need separate error imports:

- `Is(err, target error) bool`
- `As(err error, target interface{}) bool`
- `Unwrap(err error) error`
- `Wrap(err error, message string) error`
- `Wrapf(err error, format string, args ...interface{}) error`
- `WithMessage(err error, message string) error`
- `WithMessagef(err error, format string, args ...interface{}) error`
- `Cause(err error) error`
- `Join(errs ...error) error`

```go
user, err := queries.GetQuerySet(&User{}).Filter("Email", email).Get()
if err != nil {
	if errors.Is(err, errors.NoRows) {
		// Also matches sql.ErrNoRows and errors.NoResults
		return nil
	}
	if errors.Is(err, errors.MultipleRows) {
		return fmt.Errorf("duplicate accounts for %s: %w", email, err)
	}
	return err
}
```

---

## ORM `Error` Sentinels and `GoCode`s

| Sentinel | `GoCode` | Related Errors | Meaning |
|---|---|---|---|
| `Unknown` | `CodeUnknown` (`"Unknown"`) | — | Generic unknown ORM error |
| `NotImplemented` | `CodeNotImplemented` (`"NotImplemented"`) | — | Operation or option not supported by target backend |
| `NoDatabase` | `CodeNoDatabase` (`"NoDatabase"`) | — | No `drivers.Database` configured for model/settings key |
| `UnknownDriver` | `CodeUnknownDriver` (`"UnknownDriver"`) | — | Driver name or `driver.Driver` is not registered |
| `NoTableName` | `CodeNoTableName` (`"NoTableName"`) | — | Model metadata has no table name |
| `NoWhereClause` | `CodeNoWhereClause` (`"NoWhereClause"`) | — | Update/delete attempted without a `WHERE` clause or unique key |
| `FieldNull` | `CodeFieldNull` (`"FieldNull"`) | — | Non-nullable field received `nil` |
| `LastInsertId` | `CodeLastInsertId` (`"LastInsertId"`) | — | Invalid or unsupported `LastInsertId()` |
| `UnsupportedLookup` | `CodeUnsupportedLookup` (`"UnsupportedLookup"`) | — | Lookup name is not registered |
| `AlreadyExecuted` | `CodeAlreadyExecuted` (`"AlreadyExecuted"`) | — | Compiled query or iterator was already executed |
| `CheckFailed` | `CodeCheckFailed` (`"CheckFailed"`) | — | Validation or integrity check failed |
| `InvalidContentType` | `CodeContentTypeNotFound` (`"ContentTypeNotFound"`) | — | Content type could not be resolved |
| `TypeMismatch` | `CodeTypeMismatch` (`"TypeMismatch"`) | — | Scanned or converted Go value type mismatch |
| `NilPointer` | `CodeNilPointer` (`"NilPointer"`) | — | Received a nil pointer where an initialized pointer was expected |
| `FieldNotFound` | `CodeFieldNotFound` (`"FieldNotFound"`) | — | Referenced field does not exist on the model |
| `ValueError` | `CodeValueError` (`"ValueError"`) | — | Error reading or encoding a field value |
| `NoUniqueKey` | `CodeNoUniqueKey` (`"NoUniqueKey"`) | — | Model lacks a primary key or unique field combination |
| `SaveFailed` | `CodeSaveFailed` (`"SaveFailed"`) | — | Saving a model or relation failed |
| `NoChanges` | `CodeNoChanges` (`"NoChanges"`) | `sql.ErrNoRows` | Save or update resulted in no database rows modified |
| `NoResults` | `CodeNoResults` (`"NoResults"`) | `sql.ErrNoRows` | Query returned zero results |
| `NoRows` | `CodeNoRows` (`"NoRows"`) | `sql.ErrNoRows`, `NoResults` | `Get()`, `First()`, `Last()`, or row scan found no rows |
| `MultipleRows` | `CodeMultipleRows` (`"MultipleRows"`) | — | `Get()` returned more than one row |
| `UnexpectedRowCount` | `CodeUnexpectedRowCount` (`"UnexpectedRowCount"`) | `sql.ErrNoRows` | Affected or returned row count did not match expectations |

Because `NoRows` relates to both `sql.ErrNoRows` and `NoResults`, `errors.Is(err, errors.NoRows)`, `errors.Is(err, sql.ErrNoRows)`, and `errors.Is(err, errors.NoResults)` all match when `errors.NoRows` is returned.

---

## `DatabaseError` Sentinels and `DBCode`s

| Category | Sentinels (`DBCode` in parentheses) | Notable Related Errors |
|---|---|---|
| Common | `GenericError` (`E001`), `ConnectionFailed` (`E002`), `AuthenticationFailed` (`E003`), `ProtocolError` (`E004`), `Timeout` (`E005`), `InternalError` (`E006`), `ConnectionClosed` (`E007`), `ConnectionLost` (`E008`) | — |
| DDL / Schema | `Exists` (`E109`), `NotExists` (`E110`), `InvalidTable` (`E111`), `TableExists` (`E112`), `TableNotFound` (`E113`), `InvalidColumn` (`E114`), `ColumnExists` (`E115`), `ColumnNotFound` (`E116`), `IndexExists` (`E117`), `IndexNotFound` (`E118`), `ConstraintExists` (`E119`), `ConstraintNotFound` (`E120`) | `InvalidTable` -> `NoTableName`, `TableNotFound`; `InvalidColumn` -> `FieldNotFound`; `*Exists` -> `Exists`; `*NotFound` -> `NotExists` |
| Data & Types | `DBTypeMismatch` (`E201`), `InvalidEnumValue` (`E202`), `StringTooLong` (`E203`), `InvalidDefaultValue` (`E204`), `DateOutOfRange` (`E205`), `EncodingError` (`E206`), `PrecisionLoss` (`E207`), `DivisionByZero` (`E208`) | `DBTypeMismatch` -> `TypeMismatch` |
| Constraints | `ConstraintViolation` (`E301`), `ForeignKeyViolation` (`E302`), `UniqueViolation` (`E303`), `NullViolation` (`E304`), `CheckViolation` (`E305`) | `ForeignKeyViolation`, `UniqueViolation`, `CheckViolation` -> `ConstraintViolation`; `NullViolation` -> `FieldNull` |
| Transactions | `TransactionStarted` (`E401`), `TransactionDeadlock` (`E402`), `FailedStartTransaction` (`E403`), `NoTransaction` (`E404`), `TransactionNil` (`E405`), `RollbackFailed` (`E406`), `CommitFailed` (`E407`), `SavepointFailed` (`E408`), `CrossDatabaseTransaction` (`E409`) | `TransactionNil` -> `NoTransaction`; `RollbackFailed` & `CommitFailed` -> `sql.ErrTxDone` |
| Query / Runtime | `SyntaxError` (`E501`), `DeadlockDetected` (`E502`), `QueryCanceled` (`E503`), `QueryTimeout` (`E504`), `TooManyConnections` (`E505`), `OutOfMemory` (`E506`), `IOError` (`E507`), `DiskFull` (`E508`), `PermissionDenied` (`E509`), `SerializationFailure` (`E510`) | `TooManyConnections`, `OutOfMemory`, `DiskFull` -> `IOError` |
| Planning | `AmbiguousColumn` (`E601`), `UnsupportedFeature` (`E602`), `FunctionNotImplemented` (`E603`), `InvalidCast` (`E604`) | `InvalidCast` -> `DBTypeMismatch` |
| System / Internal | `CorruptedData` (`E701`), `InconsistentState` (`E702`), `AssertionFailure` (`E703`) | — |

Note that `NoTransaction` is a `DatabaseError` sentinel (`DBCode` `"E404"`), not an `Error` sentinel.

---

## Caveats

- **Postgres error translation:** While SQLite (`drivers_sqlite.go`) and MySQL/MariaDB (`drivers_mysql.go`) translate native driver error numbers and extended codes into sentinels such as `UniqueViolation`, `ForeignKeyViolation`, `NullViolation`, `CheckViolation`, and `InvalidTable`, the `"postgres"` driver currently sets `BuildDatabaseError: errors.InvalidDatabaseError`. Postgres errors therefore come back wrapped with `CodeInvalid` (`"E000"`) rather than `UniqueViolation` or `ForeignKeyViolation`. Use `errors.As(err, &pgErr)` with `*pgconn.PgError` when checking Postgres SQLSTATE codes.

For general Go-Django error types outside the ORM, see [Errors](../../docs/errors.md).
