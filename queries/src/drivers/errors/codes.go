package errors

type Code string

func (c Code) String() string {
	return string(c)
}

const (
	CodeInvalid Code = "E000"

	// Common Errors Across All
	CodeGenericError         Code = "E001"
	CodeConnectionFailed     Code = "E002"
	CodeAuthenticationFailed Code = "E003"
	CodeProtocolError        Code = "E004"
	CodeTimeout              Code = "E005"
	CodeInternalError        Code = "E006"
	CodeConnectionClosed     Code = "E007"
	CodeConnectionLost       Code = "E008"

	// DDL/Schema Errors
	CodeExists             Code = "E109"
	CodeNotExists          Code = "E110"
	CodeInvalidTable       Code = "E111"
	CodeTableExists        Code = "E112"
	CodeTableNotFound      Code = "E113"
	CodeInvalidColumn      Code = "E114"
	CodeColumnExists       Code = "E115"
	CodeColumnNotFound     Code = "E116"
	CodeIndexExists        Code = "E117"
	CodeIndexNotFound      Code = "E118"
	CodeConstraintExists   Code = "E119"
	CodeConstraintNotFound Code = "E120"

	// Data & Type Errors
	CodeTypeMismatch   Code = "E201"
	CodeInvalidEnum    Code = "E202"
	CodeStringTooLong  Code = "E203"
	CodeInvalidDefault Code = "E204"
	CodeDateOutOfRange Code = "E205"
	CodeEncodingError  Code = "E206"
	CodePrecisionLoss  Code = "E207"
	CodeDivisionByZero Code = "E208"

	// Constraint errors
	CodeConstraintViolation      Code = "E301"
	CodeForeignKeyViolation      Code = "E302"
	CodeUniqueViolation          Code = "E303"
	CodeNotNullViolation         Code = "E304"
	CodeCheckConstraintViolation Code = "E305"

	// Transaction Errors
	CodeTransactionStarted       Code = "E401"
	CodeTransactionDeadlock      Code = "E402"
	CodeFailedStartTransaction   Code = "E403"
	CodeNoTransaction            Code = "E404"
	CodeTransactionNil           Code = "E405"
	CodeRollbackFailed           Code = "E406"
	CodeCommitFailed             Code = "E407"
	CodeSavepointFailed          Code = "E408"
	CodeCrossDatabaseTransaction Code = "E409"

	// Query / QueryRow / Exec
	CodeSyntaxError          Code = "E501"
	CodeDeadlockDetected     Code = "E502"
	CodeQueryCanceled        Code = "E503"
	CodeQueryTimeout         Code = "E504"
	CodeTooManyConnections   Code = "E505"
	CodeOutOfMemory          Code = "E506"
	CodeIOError              Code = "E507"
	CodeDiskFull             Code = "E508"
	CodePermissionDenied     Code = "E509"
	CodeSerializationFailure Code = "E510"

	// Planning / Execution
	CodeAmbiguousColumn        Code = "E601"
	CodeUnsupportedFeature     Code = "E602"
	CodeFunctionNotImplemented Code = "E603"
	CodeInvalidCast            Code = "E604"

	// System/Internal
	CodeCorruptedData     Code = "E701"
	CodeInconsistentState Code = "E702"
	CodeAssertionFailure  Code = "E703"
)

var databaseErrors = map[Code]DatabaseError{
	CodeGenericError:             GenericError,
	CodeTransactionStarted:       TransactionStarted,
	CodeFailedStartTransaction:   FailedStartTransaction,
	CodeNoTransaction:            NoTransaction,
	CodeTransactionNil:           TransactionNil,
	CodeRollbackFailed:           RollbackFailed,
	CodeCommitFailed:             CommitFailed,
	CodeCrossDatabaseTransaction: CrossDatabaseTransaction,
	CodeSavepointFailed:          SavepointFailed,
	CodeTransactionDeadlock:      TransactionDeadlock,
	CodeSyntaxError:              SyntaxError,
	CodeConstraintViolation:      ConstraintViolation,
	CodeForeignKeyViolation:      ForeignKeyViolation,
	CodeUniqueViolation:          UniqueViolation,
	CodeNotNullViolation:         NullViolation,
	CodeCheckConstraintViolation: CheckViolation,
	CodeDeadlockDetected:         DeadlockDetected,
	CodeQueryCanceled:            QueryCanceled,
	CodeQueryTimeout:             QueryTimeout,
	CodeDivisionByZero:           DivisionByZero,
	CodeInvalidColumn:            InvalidColumn,
	CodeInvalidTable:             InvalidTable,
	CodeTypeMismatch:             DBTypeMismatch,
	CodeTooManyConnections:       TooManyConnections,
	CodeOutOfMemory:              OutOfMemory,
	CodeIOError:                  IOError,
	CodeDiskFull:                 DiskFull,
	CodePermissionDenied:         PermissionDenied,
	CodeSerializationFailure:     SerializationFailure,
	CodeConnectionClosed:         ConnectionClosed,
	CodeConnectionLost:           ConnectionLost,
	CodeConnectionFailed:         ConnectionFailed,
	CodeAuthenticationFailed:     AuthenticationFailed,
	CodeProtocolError:            ProtocolError,
	CodeTimeout:                  Timeout,
	CodeInternalError:            InternalError,
	CodeTableExists:              TableExists,
	CodeTableNotFound:            TableNotFound,
	CodeColumnExists:             ColumnExists,
	CodeColumnNotFound:           ColumnNotFound,
	CodeIndexExists:              IndexExists,
	CodeIndexNotFound:            IndexNotFound,
	CodeConstraintExists:         ConstraintExists,
	CodeConstraintNotFound:       ConstraintNotFound,
	CodeInvalidEnum:              InvalidEnumValue,
	CodeStringTooLong:            StringTooLong,
	CodeInvalidDefault:           InvalidDefaultValue,
	CodeDateOutOfRange:           DateOutOfRange,
	CodeEncodingError:            EncodingError,
	CodePrecisionLoss:            PrecisionLoss,
	CodeAmbiguousColumn:          AmbiguousColumn,
	CodeUnsupportedFeature:       UnsupportedFeature,
	CodeFunctionNotImplemented:   FunctionNotImplemented,
	CodeInvalidCast:              InvalidCast,
	CodeCorruptedData:            CorruptedData,
	CodeInconsistentState:        InconsistentState,
	CodeAssertionFailure:         AssertionFailure,
}
var _ = databaseErrors
