package errors

import (
	"database/sql"
)

/*

Package errors provides functions better work with errors across different backends
and drivers, such as SQLite, MySQL, PostgreSQL, and others.

Drivers may return errors that are specific to their implementation, and
this package provides a unified way to handle these errors. It includes
error types for common scenarios, such as NoRows, MultipleRows, and
NotImplemented, which can be used to check for specific error conditions.

It also provides functions to manipulate errors in a way that is compatible
with the standard library's errors package, while also providing additional functionality
for wrapping and unwrapping errors from `github.com/pkg/errors`.

*/

var (
	NotImplemented     Error = New(CodeNotImplemented, "Not implemented")
	NoDatabase         Error = New(CodeNoDatabase, "No database connection")
	UnknownDriver      Error = New(CodeUnknownDriver, "Unknown driver")
	NoTableName        Error = New(CodeNoTableName, "No table name")
	NoWhereClause      Error = New(CodeNoWhereClause, "No where clause in query")
	FieldNull          Error = New(CodeFieldNull, "Field cannot be null")
	LastInsertId       Error = New(CodeLastInsertId, "Last insert id is not valid")
	UnsupportedLookup  Error = New(CodeUnsupportedLookup, "Unsupported lookup type")
	AlreadyExecuted    Error = New(CodeAlreadyExecuted, "Query has already been executed")
	CheckFailed        Error = New(CodeCheckFailed, "Check failed")
	InvalidContentType Error = New(CodeContentTypeNotFound, "Content type not found")

	TypeMismatch  Error = New(CodeTypeMismatch, "received type does not match expected type")
	NilPointer    Error = New(CodeNilPointer, "received nil pointer, expected a pointer to initialized value")
	FieldNotFound Error = New(CodeFieldNotFound, "field not found in model definition")
	ValueError    Error = New(CodeValueError, "error retrieving value")
	NoUniqueKey   Error = New(CodeNoUniqueKey, "could not find unique key for model")
	SaveFailed    Error = New(CodeSaveFailed, "failed to save model")

	NoChanges          Error = New(CodeNoChanges, "No changes were made", sql.ErrNoRows)
	NoResults          Error = New(CodeNoResults, "No results found", sql.ErrNoRows)
	NoRows             Error = New(CodeNoRows, "No rows in result set", sql.ErrNoRows)
	MultipleRows       Error = New(CodeMultipleRows, "Multiple rows in result set")
	UnexpectedRowCount Error = New(CodeUnexpectedRowCount, "Unexpected row count in result set", sql.ErrNoRows)

	// Common Errors Across All
	GenericError         DatabaseError = dbError(CodeGenericError, "Error")
	ConnectionFailed     DatabaseError = dbError(CodeConnectionFailed, "Failed to connect to database")
	AuthenticationFailed DatabaseError = dbError(CodeAuthenticationFailed, "Invalid username or password")
	ProtocolError        DatabaseError = dbError(CodeProtocolError, "Database protocol error")
	Timeout              DatabaseError = dbError(CodeTimeout, "Operation timed out")
	InternalError        DatabaseError = dbError(CodeInternalError, "Internal database error")
	ConnectionClosed     DatabaseError = dbError(CodeConnectionClosed, "Connection already closed")
	ConnectionLost       DatabaseError = dbError(CodeConnectionLost, "Lost connection to database server")

	// DDL/Schema Errors
	Exists             DatabaseError = &databaseError{code: CodeExists}
	NotExists          DatabaseError = &databaseError{code: CodeNotExists}
	InvalidTable       DatabaseError = dbError(CodeInvalidTable, "Invalid or unknown table", NoTableName, TableNotFound)
	TableExists        DatabaseError = dbError(CodeTableExists, "Table already exists", Exists)
	TableNotFound      DatabaseError = dbError(CodeTableNotFound, "Table does not exist", NotExists)
	InvalidColumn      DatabaseError = dbError(CodeInvalidColumn, "Invalid or unknown column", FieldNotFound)
	ColumnExists       DatabaseError = dbError(CodeColumnExists, "Column already exists", Exists)
	ColumnNotFound     DatabaseError = dbError(CodeColumnNotFound, "Column does not exist", NotExists)
	IndexExists        DatabaseError = dbError(CodeIndexExists, "Index already exists", Exists)
	IndexNotFound      DatabaseError = dbError(CodeIndexNotFound, "Index does not exist", NotExists)
	ConstraintExists   DatabaseError = dbError(CodeConstraintExists, "Constraint already exists", Exists)
	ConstraintNotFound DatabaseError = dbError(CodeConstraintNotFound, "Constraint not found", NotExists)

	// Data & Type Errors
	DBTypeMismatch      DatabaseError = dbError(CodeDBTypeMismatch, "Type mismatch or invalid cast", TypeMismatch)
	InvalidEnumValue    DatabaseError = dbError(CodeInvalidEnum, "Invalid enum value")
	StringTooLong       DatabaseError = dbError(CodeStringTooLong, "String value is too long")
	InvalidDefaultValue DatabaseError = dbError(CodeInvalidDefault, "Invalid default value")
	DateOutOfRange      DatabaseError = dbError(CodeDateOutOfRange, "Date/time value out of range")
	EncodingError       DatabaseError = dbError(CodeEncodingError, "Invalid character encoding")
	PrecisionLoss       DatabaseError = dbError(CodePrecisionLoss, "Value lost precision during conversion")
	DivisionByZero      DatabaseError = dbError(CodeDivisionByZero, "Division by zero")

	// Constraint errors
	ConstraintViolation DatabaseError = dbError(CodeConstraintViolation, "Constraint violation")
	ForeignKeyViolation DatabaseError = dbError(CodeForeignKeyViolation, "Foreign key constraint failed", ConstraintViolation)
	UniqueViolation     DatabaseError = dbError(CodeUniqueViolation, "Unique constraint failed", ConstraintViolation)
	NullViolation       DatabaseError = dbError(CodeNotNullViolation, "NULL value in non-null column", FieldNull)
	CheckViolation      DatabaseError = dbError(CodeCheckConstraintViolation, "Check constraint failed", ConstraintViolation)

	// Transaction Errors
	TransactionStarted       DatabaseError = dbError(CodeTransactionStarted, "Transaction already started")
	TransactionDeadlock      DatabaseError = dbError(CodeTransactionDeadlock, "Transaction deadlock detected")
	FailedStartTransaction   DatabaseError = dbError(CodeFailedStartTransaction, "Failed to start transaction")
	NoTransaction            DatabaseError = dbError(CodeNoTransaction, "Transaction was not started")
	TransactionNil           DatabaseError = dbError(CodeTransactionNil, "Transaction is nil", NoTransaction)
	RollbackFailed           DatabaseError = dbError(CodeRollbackFailed, "Failed to rollback transaction", sql.ErrTxDone)
	CommitFailed             DatabaseError = dbError(CodeCommitFailed, "Failed to commit transaction", sql.ErrTxDone)
	SavepointFailed          DatabaseError = dbError(CodeSavepointFailed, "Failed to create savepoint in transaction")
	CrossDatabaseTransaction DatabaseError = dbError(CodeCrossDatabaseTransaction, "Cross-database transaction is not allowed")

	// Query / QueryRow / Exec
	SyntaxError          DatabaseError = dbError(CodeSyntaxError, "Syntax error in SQL statement")
	DeadlockDetected     DatabaseError = dbError(CodeDeadlockDetected, "Deadlock detected")
	QueryCanceled        DatabaseError = dbError(CodeQueryCanceled, "Query was canceled")
	QueryTimeout         DatabaseError = dbError(CodeQueryTimeout, "Query execution timed out")
	TooManyConnections   DatabaseError = dbError(CodeTooManyConnections, "Too many connections", IOError)
	OutOfMemory          DatabaseError = dbError(CodeOutOfMemory, "Database out of memory", IOError)
	IOError              DatabaseError = dbError(CodeIOError, "I/O error occurred during operation")
	DiskFull             DatabaseError = dbError(CodeDiskFull, "Disk full or write failure", IOError)
	PermissionDenied     DatabaseError = dbError(CodePermissionDenied, "Permission denied for operation")
	SerializationFailure DatabaseError = dbError(CodeSerializationFailure, "Could not serialize access due to concurrent update")

	// Planning / Execution
	AmbiguousColumn        DatabaseError = dbError(CodeAmbiguousColumn, "Ambiguous column reference")
	UnsupportedFeature     DatabaseError = dbError(CodeUnsupportedFeature, "Feature not supported by this database")
	FunctionNotImplemented DatabaseError = dbError(CodeFunctionNotImplemented, "Function not implemented by this database")
	InvalidCast            DatabaseError = dbError(CodeInvalidCast, "Invalid type cast", DBTypeMismatch)

	// System/Internal
	CorruptedData     DatabaseError = dbError(CodeCorruptedData, "Database file/data corrupted")
	InconsistentState DatabaseError = dbError(CodeInconsistentState, "Internal database state is inconsistent")
	AssertionFailure  DatabaseError = dbError(CodeAssertionFailure, "Internal assertion failed")
)
