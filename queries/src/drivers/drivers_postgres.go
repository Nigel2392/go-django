package drivers

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	pg_stdlib "github.com/jackc/pgx/v5/stdlib"
)

type DriverPostgres = pg_stdlib.Driver

const POSTGRES_DRIVER_NAME = "postgres"

func init() {
	Register(POSTGRES_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverPostgres{},
		Open: func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error) {
			return OpenPGX(ctx, drv, dsn, opts...)
		},
		BuildDatabaseError: errors.InvalidDatabaseError,
		ExplainQuery: func(ctx context.Context, q DB, query string, args []any) (string, error) {
			query = "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + query
			var rows, err = q.QueryContext(ctx, query, args...)
			if err != nil {
				return "", err
			}
			defer rows.Close()
			var sb strings.Builder
			if rows.Next() {
				var result string
				if err := rows.Scan(&result); err != nil {
					return "", err
				}
				sb.WriteString(result)
			}
			return sb.String(), nil
		},
	})
}
