package query_errors

import "github.com/Nigel2392/go-django/src/core/errs"

const (
	ErrNotImplemented    errs.Error = "Not implemented"
	ErrNoDatabase        errs.Error = "No database connection"
	ErrUnknownDriver     errs.Error = "Unknown driver"
	ErrNoTableName       errs.Error = "No table name"
	ErrNoWhereClause     errs.Error = "No where clause in query"
	ErrFieldNull         errs.Error = "Field cannot be null"
	ErrLastInsertId      errs.Error = "Last insert id is not valid"
	ErrUnsupportedLookup errs.Error = "Unsupported lookup type"

	ErrNoResults    errs.Error = "No results found"
	ErrNoRows       errs.Error = "No rows in result set"
	ErrMultipleRows errs.Error = "Multiple rows in result set"

	ErrTransactionStarted       errs.Error = "Transaction already started"
	ErrFailedStartTransaction   errs.Error = "Failed to start transaction"
	ErrNoTransaction            errs.Error = "Transaction was not started"
	ErrTransactionNil           errs.Error = "Transaction is nil"
	ErrCrossDatabaseTransaction errs.Error = "Cross-database transaction is not allowed"

	// Returned when a Query[T] is executed twice, the result should
	// be cached, and the second call should return the cached result,
	// along with this error.
	ErrAlreadyExecuted errs.Error = "Query already executed"

	ErrTypeMismatch  errs.Error = "received type does not match expected type"
	ErrNilPointer    errs.Error = "received nil pointer, expected a pointer to initialized value"
	ErrFieldNotFound errs.Error = "field not found in model definition"
	ErrNoUniqueKey   errs.Error = "could not find unique key for model"
	ErrSaveFailed    errs.Error = "failed to save model"
)
