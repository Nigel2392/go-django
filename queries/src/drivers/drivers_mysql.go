package drivers

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/go-sql-driver/mysql"
)

type DriverMySQL = mysql.MySQLDriver

func mySQLDatabaseError(err error) errors.DatabaseError {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return errors.InvalidDatabaseError(err)
	}

	switch mysqlErr.Number {
	// --- Constraint Violations ---
	case 1062: // ER_DUP_ENTRY
		return errors.UniqueViolation.WithCause(err)

	case 1048, // ER_BAD_NULL_ERROR
		1364: // ER_NO_DEFAULT_FOR_FIELD
		return errors.NullViolation.WithCause(err)

	case 1216, // ER_NO_REFERENCED_ROW
		1217, // ER_ROW_IS_REFERENCED
		1451, // ER_ROW_IS_REFERENCED_2
		1452: // ER_NO_REFERENCED_ROW_2
		return errors.ForeignKeyViolation.WithCause(err)

	case 3819: // ER_CHECK_CONSTRAINT_VIOLATED
		return errors.CheckViolation.WithCause(err)

	// --- Syntax & Parse Errors ---
	case 1064: // ER_PARSE_ERROR
		return errors.SyntaxError.WithCause(err)

	case 1054: // ER_BAD_FIELD_ERROR
		return errors.InvalidColumn.WithCause(err)

	case 1146: // ER_NO_SUCH_TABLE
		return errors.InvalidTable.WithCause(err)

	case 1365: // ER_DIVISION_BY_ZERO
		return errors.DivisionByZero.WithCause(err)

	case 1264, // ER_WARN_DATA_OUT_OF_RANGE
		1366: // ER_TRUNCATED_WRONG_VALUE_FOR_FIELD
		return errors.DBTypeMismatch.WithCause(err)

	// --- Transaction/Locking ---
	case 1205: // ER_LOCK_WAIT_TIMEOUT
		return errors.QueryTimeout.WithCause(err)

	case 1213: // ER_LOCK_DEADLOCK
		return errors.DeadlockDetected.WithCause(err)

	case 1637: // ER_GTID_NEXT_TYPE_UNDEFINED_GROUP
		return errors.FailedStartTransaction.WithCause(err)

	// --- Connection / Network ---
	case 2002: // CR_CONNECTION_ERROR (named pipe or TCP)
		return errors.ConnectionFailed.WithCause(err)

	case 2003, // CR_CONN_HOST_ERROR
		2005: // CR_UNKNOWN_HOST
		return errors.ConnectionFailed.WithCause(err)

	case 2013: // CR_SERVER_LOST
		return errors.ConnectionLost.WithCause(err)

	case 2006: // CR_SERVER_GONE_ERROR
		return errors.ConnectionClosed.WithCause(err)

	case 1045: // ER_ACCESS_DENIED_ERROR
		return errors.AuthenticationFailed.WithCause(err)

	// --- Permissions ---
	case 1044: // ER_DBACCESS_DENIED_ERROR
		return errors.PermissionDenied.WithCause(err)

	// --- Resources ---
	case 1040: // ER_CON_COUNT_ERROR
		return errors.TooManyConnections.WithCause(err)

	case 1037, 1038: // ER_OUTOFMEMORY, ER_OUT_OF_SORTMEMORY
		return errors.OutOfMemory.WithCause(err)

	case 1021: // ER_DISK_FULL
		return errors.DiskFull.WithCause(err)
	}

	// --- Fallback: error message parsing (MySQL is inconsistent sometimes) ---
	var lowerMsg = strings.ToLower(mysqlErr.Message)
	switch {
	case strings.Contains(lowerMsg, "syntax"):
		return errors.SyntaxError.WithCause(err)

	case strings.Contains(lowerMsg, "permission denied"):
		return errors.PermissionDenied.WithCause(err)
	}

	return errors.InternalError.WithCause(err)
}

const MYSQL_DRIVER_NAME = "mysql"

func explainMySQL(ctx context.Context, q DB, query string, args []any) (string, error) {
	var explainQuery = "EXPLAIN " + query
	var rows, err = q.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", err
	}

	defer rows.Close()
	var sb strings.Builder
	var tabwriter = tabwriter.NewWriter(&sb, 4, 4, 2, ' ', 0)
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	_, _ = tabwriter.Write([]byte(strings.Join(columns, "\t") + "\n"))
	var values = make([]any, len(columns))
	var valuePtrs = make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}
		var strs = make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				strs[i] = "NULL"
			} else {
				var s string
				switch v := val.(type) {
				case []byte:
					s = string(v)
				case string:
					s = v
				default:
					s = fmt.Sprintf("%v", v)
				}
				strs[i] = s
			}
		}
		_, _ = tabwriter.Write([]byte(strings.Join(strs, "\t") + "\n"))
	}
	tabwriter.Flush()
	return sb.String(), nil
}

func init() {
	Register(MYSQL_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningLastInsertId,
		Driver:            &DriverMySQL{},
		Open: func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(MYSQL_DRIVER_NAME, drv, dsn, opts...)
		},
		BuildDatabaseError: mySQLDatabaseError,
		ExplainQuery: func(ctx context.Context, q DB, query string, args []any) (string, error) {
			return explainMySQL(ctx, q, query, args)
		},
	})
}
