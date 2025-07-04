package errors

type GoCode string

const (
	CodeUnknown        GoCode = "Unknown"
	CodeNotImplemented GoCode = "NotImplemented"
	CodeNoDatabase     GoCode = "NoDatabase"
	CodeUnknownDriver  GoCode = "UnknownDriver"
	CodeNoTableName    GoCode = "NoTableName"
	CodeNoWhereClause  GoCode = "NoWhereClause"

	CodeLastInsertId  GoCode = "LastInsertId"
	CodeFieldNull     GoCode = "FieldNull"
	CodeNilPointer    GoCode = "NilPointer"
	CodeFieldNotFound GoCode = "FieldNotFound"
	CodeTypeMismatch  GoCode = "TypeMismatch"
	CodeValueError    GoCode = "ValueError"

	CodeUnsupportedLookup GoCode = "UnsupportedLookup"
	CodeAlreadyExecuted   GoCode = "AlreadyExecuted"
	CodeNoUniqueKey       GoCode = "NoUniqueKey"
	CodeSaveFailed        GoCode = "SaveFailed"

	CodeNoChanges          GoCode = "NoChanges"
	CodeNoResults          GoCode = "NoResults"
	CodeNoRows             GoCode = "NoRows"
	CodeMultipleRows       GoCode = "MultipleRows"
	CodeUnexpectedRowCount GoCode = "UnexpectedRowCount"
)

type DBCode string

func (c DBCode) String() string {
	return string(c)
}

const (
	CodeInvalid DBCode = "E000"

	// Common Errors Across All
	CodeGenericError         DBCode = "E001"
	CodeConnectionFailed     DBCode = "E002"
	CodeAuthenticationFailed DBCode = "E003"
	CodeProtocolError        DBCode = "E004"
	CodeTimeout              DBCode = "E005"
	CodeInternalError        DBCode = "E006"
	CodeConnectionClosed     DBCode = "E007"
	CodeConnectionLost       DBCode = "E008"

	// DDL/Schema Errors
	CodeExists             DBCode = "E109"
	CodeNotExists          DBCode = "E110"
	CodeInvalidTable       DBCode = "E111"
	CodeTableExists        DBCode = "E112"
	CodeTableNotFound      DBCode = "E113"
	CodeInvalidColumn      DBCode = "E114"
	CodeColumnExists       DBCode = "E115"
	CodeColumnNotFound     DBCode = "E116"
	CodeIndexExists        DBCode = "E117"
	CodeIndexNotFound      DBCode = "E118"
	CodeConstraintExists   DBCode = "E119"
	CodeConstraintNotFound DBCode = "E120"

	// Data & Type Errors
	CodeDBTypeMismatch DBCode = "E201"
	CodeInvalidEnum    DBCode = "E202"
	CodeStringTooLong  DBCode = "E203"
	CodeInvalidDefault DBCode = "E204"
	CodeDateOutOfRange DBCode = "E205"
	CodeEncodingError  DBCode = "E206"
	CodePrecisionLoss  DBCode = "E207"
	CodeDivisionByZero DBCode = "E208"

	// Constraint errors
	CodeConstraintViolation      DBCode = "E301"
	CodeForeignKeyViolation      DBCode = "E302"
	CodeUniqueViolation          DBCode = "E303"
	CodeNotNullViolation         DBCode = "E304"
	CodeCheckConstraintViolation DBCode = "E305"

	// Transaction Errors
	CodeTransactionStarted       DBCode = "E401"
	CodeTransactionDeadlock      DBCode = "E402"
	CodeFailedStartTransaction   DBCode = "E403"
	CodeNoTransaction            DBCode = "E404"
	CodeTransactionNil           DBCode = "E405"
	CodeRollbackFailed           DBCode = "E406"
	CodeCommitFailed             DBCode = "E407"
	CodeSavepointFailed          DBCode = "E408"
	CodeCrossDatabaseTransaction DBCode = "E409"

	// Query / QueryRow / Exec
	CodeSyntaxError          DBCode = "E501"
	CodeDeadlockDetected     DBCode = "E502"
	CodeQueryCanceled        DBCode = "E503"
	CodeQueryTimeout         DBCode = "E504"
	CodeTooManyConnections   DBCode = "E505"
	CodeOutOfMemory          DBCode = "E506"
	CodeIOError              DBCode = "E507"
	CodeDiskFull             DBCode = "E508"
	CodePermissionDenied     DBCode = "E509"
	CodeSerializationFailure DBCode = "E510"

	// Planning / Execution
	CodeAmbiguousColumn        DBCode = "E601"
	CodeUnsupportedFeature     DBCode = "E602"
	CodeFunctionNotImplemented DBCode = "E603"
	CodeInvalidCast            DBCode = "E604"

	// System/Internal
	CodeCorruptedData     DBCode = "E701"
	CodeInconsistentState DBCode = "E702"
	CodeAssertionFailure  DBCode = "E703"
)

var databaseErrors = map[DBCode]DatabaseError{
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
	CodeDBTypeMismatch:           DBTypeMismatch,
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
