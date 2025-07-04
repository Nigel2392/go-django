package drivers

import (
	"context"

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
	})
}
