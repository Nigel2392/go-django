package drivers_test

import (
	"context"
	"os"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/go-sql-driver/mysql"
	sqlite3 "github.com/mattn/go-sqlite3"
)

func TestSqliteDatabaseError(t *testing.T) {
	tests := []struct {
		name     string
		code     sqlite3.ErrNoExtended
		expected errors.DatabaseError
	}{
		// Disk/IO errors
		{"ErrIoErrRead", sqlite3.ErrIoErrRead, errors.IOError},
		{"ErrIoErrShortRead", sqlite3.ErrIoErrShortRead, errors.IOError},
		{"ErrIoErrWrite", sqlite3.ErrIoErrWrite, errors.IOError},
		{"ErrIoErrFsync", sqlite3.ErrIoErrFsync, errors.IOError},
		{"ErrIoErrDirFsync", sqlite3.ErrIoErrDirFsync, errors.IOError},
		{"ErrIoErrTruncate", sqlite3.ErrIoErrTruncate, errors.IOError},
		{"ErrIoErrFstat", sqlite3.ErrIoErrFstat, errors.IOError},
		{"ErrIoErrUnlock", sqlite3.ErrIoErrUnlock, errors.PermissionDenied},
		{"ErrIoErrRDlock", sqlite3.ErrIoErrRDlock, errors.PermissionDenied},
		{"ErrIoErrDelete", sqlite3.ErrIoErrDelete, errors.IOError},
		{"ErrIoErrBlocked", sqlite3.ErrIoErrBlocked, errors.QueryTimeout},
		{"ErrIoErrNoMem", sqlite3.ErrIoErrNoMem, errors.OutOfMemory},
		{"ErrIoErrAccess", sqlite3.ErrIoErrAccess, errors.PermissionDenied},
		{"ErrIoErrCheckReservedLock", sqlite3.ErrIoErrCheckReservedLock, errors.PermissionDenied},
		{"ErrIoErrLock", sqlite3.ErrIoErrLock, errors.PermissionDenied},
		{"ErrIoErrClose", sqlite3.ErrIoErrClose, errors.IOError},
		{"ErrIoErrDirClose", sqlite3.ErrIoErrDirClose, errors.IOError},
		{"ErrIoErrSHMOpen", sqlite3.ErrIoErrSHMOpen, errors.IOError},
		{"ErrIoErrSHMSize", sqlite3.ErrIoErrSHMSize, errors.IOError},
		{"ErrIoErrSHMLock", sqlite3.ErrIoErrSHMLock, errors.IOError},
		{"ErrIoErrSHMMap", sqlite3.ErrIoErrSHMMap, errors.IOError},
		{"ErrIoErrSeek", sqlite3.ErrIoErrSeek, errors.IOError},
		{"ErrIoErrDeleteNoent", sqlite3.ErrIoErrDeleteNoent, errors.IOError},
		{"ErrIoErrMMap", sqlite3.ErrIoErrMMap, errors.IOError},
		{"ErrIoErrGetTempPath", sqlite3.ErrIoErrGetTempPath, errors.IOError},
		{"ErrIoErrConvPath", sqlite3.ErrIoErrConvPath, errors.IOError},

		// Locking / busy
		{"ErrLockedSharedCache", sqlite3.ErrLockedSharedCache, errors.DeadlockDetected},
		{"ErrBusyRecovery", sqlite3.ErrBusyRecovery, errors.DeadlockDetected},
		{"ErrBusySnapshot", sqlite3.ErrBusySnapshot, errors.DeadlockDetected},

		// Cannot open
		{"ErrCantOpenNoTempDir", sqlite3.ErrCantOpenNoTempDir, errors.InternalError},
		{"ErrCantOpenIsDir", sqlite3.ErrCantOpenIsDir, errors.InternalError},
		{"ErrCantOpenFullPath", sqlite3.ErrCantOpenFullPath, errors.InternalError},
		{"ErrCantOpenConvPath", sqlite3.ErrCantOpenConvPath, errors.InternalError},

		// Corruption / readonly
		{"ErrCorruptVTab", sqlite3.ErrCorruptVTab, errors.InternalError},
		{"ErrReadonlyRecovery", sqlite3.ErrReadonlyRecovery, errors.PermissionDenied},
		{"ErrReadonlyCantLock", sqlite3.ErrReadonlyCantLock, errors.PermissionDenied},
		{"ErrReadonlyRollback", sqlite3.ErrReadonlyRollback, errors.PermissionDenied},
		{"ErrReadonlyDbMoved", sqlite3.ErrReadonlyDbMoved, errors.PermissionDenied},

		// Transaction
		{"ErrAbortRollback", sqlite3.ErrAbortRollback, errors.RollbackFailed},

		// Constraints
		{"ErrConstraintCheck", sqlite3.ErrConstraintCheck, errors.CheckViolation},
		{"ErrConstraintCommitHook", sqlite3.ErrConstraintCommitHook, errors.ConstraintViolation},
		{"ErrConstraintForeignKey", sqlite3.ErrConstraintForeignKey, errors.ForeignKeyViolation},
		{"ErrConstraintFunction", sqlite3.ErrConstraintFunction, errors.ConstraintViolation},
		{"ErrConstraintNotNull", sqlite3.ErrConstraintNotNull, errors.NullViolation},
		{"ErrConstraintPrimaryKey", sqlite3.ErrConstraintPrimaryKey, errors.UniqueViolation},
		{"ErrConstraintTrigger", sqlite3.ErrConstraintTrigger, errors.ConstraintViolation},
		{"ErrConstraintUnique", sqlite3.ErrConstraintUnique, errors.UniqueViolation},
		{"ErrConstraintVTab", sqlite3.ErrConstraintVTab, errors.ConstraintViolation},
		{"ErrConstraintRowID", sqlite3.ErrConstraintRowID, errors.UniqueViolation},

		// Notices
		{"ErrNoticeRecoverWAL", sqlite3.ErrNoticeRecoverWAL, errors.InternalError},
		{"ErrNoticeRecoverRollback", sqlite3.ErrNoticeRecoverRollback, errors.InternalError},
		{"ErrWarningAutoIndex", sqlite3.ErrWarningAutoIndex, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sqliteErr := sqlite3.Error{
				Code:         sqlite3.ErrIoErr, // general category; not used in mapping
				ExtendedCode: test.code,
			}
			var d, _ = drivers.Retrieve("sqlite3")
			var got = d.BuildDatabaseError(sqliteErr)

			if got == nil && test.expected == nil {
				return // both nil, nothing to check
			}

			if got == nil || test.expected == nil {
				t.Errorf("Expected %v, got %v", test.expected, got)
				return
			}

			if got.Code() != test.expected.Code() {
				t.Errorf("Expected %s, got %s", test.expected.Code(), got.Code())
			}
		})
	}
}

func TestMySQLDatabaseError(t *testing.T) {
	tests := []struct {
		name     string
		mysqlErr *mysql.MySQLError
		expected errors.DatabaseError
	}{
		{"DuplicateEntry", &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, errors.UniqueViolation},
		{"BadNull", &mysql.MySQLError{Number: 1048, Message: "Column cannot be null"}, errors.NullViolation},
		{"NoReferencedRow", &mysql.MySQLError{Number: 1216}, errors.ForeignKeyViolation},
		{"RowIsReferenced", &mysql.MySQLError{Number: 1451}, errors.ForeignKeyViolation},
		{"CheckViolation", &mysql.MySQLError{Number: 3819}, errors.CheckViolation},
		{"SyntaxError", &mysql.MySQLError{Number: 1064}, errors.SyntaxError},
		{"InvalidColumn", &mysql.MySQLError{Number: 1054}, errors.InvalidColumn},
		{"InvalidTable", &mysql.MySQLError{Number: 1146}, errors.InvalidTable},
		{"DivisionByZero", &mysql.MySQLError{Number: 1365}, errors.DivisionByZero},
		{"TypeMismatch", &mysql.MySQLError{Number: 1264}, errors.DBTypeMismatch},
		{"QueryTimeout", &mysql.MySQLError{Number: 1205}, errors.QueryTimeout},
		{"Deadlock", &mysql.MySQLError{Number: 1213}, errors.DeadlockDetected},
		{"ConnectionFailed", &mysql.MySQLError{Number: 2002}, errors.ConnectionFailed},
		{"ConnectionLost", &mysql.MySQLError{Number: 2013}, errors.ConnectionLost},
		{"ConnectionClosed", &mysql.MySQLError{Number: 2006}, errors.ConnectionClosed},
		{"AuthFailed", &mysql.MySQLError{Number: 1045}, errors.AuthenticationFailed},
		{"PermissionDenied", &mysql.MySQLError{Number: 1044}, errors.PermissionDenied},
		{"TooManyConnections", &mysql.MySQLError{Number: 1040}, errors.TooManyConnections},
		{"OutOfMemory", &mysql.MySQLError{Number: 1037}, errors.OutOfMemory},
		{"DiskFull", &mysql.MySQLError{Number: 1021}, errors.DiskFull},

		// Fallback case: unknown code but "syntax" in message
		{"Fallback_Syntax", &mysql.MySQLError{Number: 9999, Message: "some syntax error"}, errors.SyntaxError},
		// Unknown error: should return InternalError
		{"UnknownError", &mysql.MySQLError{Number: 9999, Message: "some unknown error"}, errors.InternalError},
	}

	d, _ := drivers.Retrieve("mysql")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := d.BuildDatabaseError(test.mysqlErr)

			if got == nil {
				t.Errorf("Expected %v, got nil", test.expected)
				return
			}
			if got.Code() != test.expected.Code() {
				t.Errorf("Expected code %s, got %s", test.expected.Code(), got.Code())
			}
		})
	}
}

type testDatabaseError struct {
	execStr         string
	exec            func(ctx context.Context, which string, db drivers.Database) (any, error)
	expected        errors.DatabaseError
	whichExclusions []string
}

func whichIs(which string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if exclusion == which {
			return true
		}
	}
	return false
}

func TestDatabaseErrors(t *testing.T) {
	var which, db = testdb.Open()
	if which == "postgres" || which == "mysql_local" {
		// postgres is unsupported FOR NOW
		t.Skip("Skipping database error tests for PostgreSQL (NYI) and MySQL Local (unsupported)")
		return
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
	})

	var tests = []testDatabaseError{
		{
			execStr:  "SELECT ABC FROM DELETE SELECT WHERE 1 = AND 2 = 3",
			expected: errors.SyntaxError,
		},
		{execStr: "SELECT * FROM non_existent_table", expected: errors.InvalidTable},
		{
			exec: func(ctx context.Context, which string, db drivers.Database) (any, error) {
				if whichIs(which, []string{"mysql", "mariadb"}) {
					return db.ExecContext(ctx, "CREATE TEMPORARY TABLE tmp_test (id INT NOT NULL); INSERT INTO tmp_test (id) VALUES (1 / 0)")
				}
				return db.ExecContext(ctx, "SELECT 1 / 0")
			},
			expected:        errors.DivisionByZero,
			whichExclusions: []string{"sqlite3"},
		},
	}

	var ctx = context.Background()
	for _, test := range tests {
		var err error
		var skip = false
		for _, exclusion := range test.whichExclusions {
			if exclusion == which {
				skip = true
				break
			}
		}

		if skip {
			t.Skipf("Skipping test for %s (%v)", which, test.expected.Code())
			continue
		}

		t.Run(string(test.expected.Code()), func(t *testing.T) {

			if test.execStr != "" {
				_, err = db.ExecContext(ctx, test.execStr)
			} else {
				_, err = test.exec(ctx, which, db)
			}
			if !errors.Is(err, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, err)
				return
			}

			t.Log(err)
		})
	}
}
