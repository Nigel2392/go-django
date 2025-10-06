package drivers

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/mattn/go-sqlite3"
)

type DriverSQLite = sqlite3.SQLiteDriver

const SQLITE3_DRIVER_NAME = "sqlite3"

func init() {
	Register(SQLITE3_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverSQLite{},
		Open: func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(SQLITE3_DRIVER_NAME, drv, dsn, opts...)
		},
		ExplainQuery: func(ctx context.Context, q DB, query string, args []any) (string, error) {
			return explainMySQL(ctx, q, query, args) // generic enough for SQLite
		},
		BuildDatabaseError: func(err error) errors.DatabaseError {
			var sqliteErr sqlite3.Error
			if !errors.As(err, &sqliteErr) {
			}

			switch sqliteErr.ExtendedCode {
			// --- IO / Disk / Memory errors ---
			case sqlite3.ErrIoErrRead,
				sqlite3.ErrIoErrShortRead,
				sqlite3.ErrIoErrWrite,
				sqlite3.ErrIoErrFsync,
				sqlite3.ErrIoErrDirFsync,
				sqlite3.ErrIoErrTruncate,
				sqlite3.ErrIoErrFstat,
				sqlite3.ErrIoErrDelete,
				sqlite3.ErrIoErrClose,
				sqlite3.ErrIoErrDirClose,
				sqlite3.ErrIoErrSHMOpen,
				sqlite3.ErrIoErrSHMSize,
				sqlite3.ErrIoErrSHMLock,
				sqlite3.ErrIoErrSHMMap,
				sqlite3.ErrIoErrSeek,
				sqlite3.ErrIoErrDeleteNoent,
				sqlite3.ErrIoErrMMap,
				sqlite3.ErrIoErrGetTempPath,
				sqlite3.ErrIoErrConvPath:
				return errors.IOError.WithCause(err)

			case sqlite3.ErrIoErrUnlock,
				sqlite3.ErrIoErrRDlock,
				sqlite3.ErrIoErrLock,
				sqlite3.ErrIoErrAccess,
				sqlite3.ErrIoErrCheckReservedLock:
				return errors.PermissionDenied.WithCause(err)

			case sqlite3.ErrIoErrBlocked:
				return errors.QueryTimeout.WithCause(err)

			case sqlite3.ErrIoErrNoMem:
				return errors.OutOfMemory.WithCause(err)

			// --- Locking / Busy / Deadlocks ---
			case sqlite3.ErrLockedSharedCache,
				sqlite3.ErrBusyRecovery,
				sqlite3.ErrBusySnapshot:
				return errors.DeadlockDetected.WithCause(err)

			// --- Cannot open / temp file issues ---
			case sqlite3.ErrCantOpenNoTempDir,
				sqlite3.ErrCantOpenIsDir,
				sqlite3.ErrCantOpenFullPath,
				sqlite3.ErrCantOpenConvPath:
				return errors.InternalError.WithCause(err)

			// --- Corruption / read-only / recoverable ---
			case sqlite3.ErrCorruptVTab:
				return errors.InternalError.WithCause(err)

			case sqlite3.ErrReadonlyRecovery,
				sqlite3.ErrReadonlyCantLock,
				sqlite3.ErrReadonlyRollback,
				sqlite3.ErrReadonlyDbMoved:
				return errors.PermissionDenied.WithCause(err)

			case sqlite3.ErrAbortRollback:
				return errors.RollbackFailed.WithCause(err)

			// --- Constraint Violations ---
			case sqlite3.ErrConstraintCheck:
				return errors.CheckViolation.WithCause(err)

			case sqlite3.ErrConstraintCommitHook:
				return errors.ConstraintViolation.WithCause(err)

			case sqlite3.ErrConstraintForeignKey:
				return errors.ForeignKeyViolation.WithCause(err)

			case sqlite3.ErrConstraintFunction:
				return errors.ConstraintViolation.WithCause(err)

			case sqlite3.ErrConstraintNotNull:
				return errors.NullViolation.WithCause(err)

			case sqlite3.ErrConstraintPrimaryKey,
				sqlite3.ErrConstraintUnique,
				sqlite3.ErrConstraintRowID:
				return errors.UniqueViolation.WithCause(err)

			case sqlite3.ErrConstraintTrigger,
				sqlite3.ErrConstraintVTab:
				return errors.ConstraintViolation.WithCause(err)

			// --- Notices and Warnings ---
			case sqlite3.ErrNoticeRecoverWAL,
				sqlite3.ErrNoticeRecoverRollback:
				return errors.InternalError.WithCause(err)

			case sqlite3.ErrWarningAutoIndex:
				// return errors.InternalError.WithCause(err)
				logger.Warnf("SQLite warning: %s", err.Error())
				return nil
			}

			switch {
			case strings.Contains(err.Error(), "syntax error"):
				return errors.SyntaxError.WithCause(err)
			case strings.Contains(err.Error(), "no such table"), strings.Contains(err.Error(), "table not found"):
				return errors.InvalidTable.WithCause(err)
			case strings.Contains(err.Error(), "no such column"), strings.Contains(err.Error(), "no column named"):
				return errors.InvalidColumn.WithCause(err)
			}

			return errors.InvalidDatabaseError(err)
		},
	})
}
