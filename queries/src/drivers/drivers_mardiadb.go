package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/go-sql-driver/mysql"
)

type DriverMariaDB struct {
	mysql.MySQLDriver
}

const MARIADB_DRIVER_NAME = "mariadb"

func init() {
	sql.Register(MARIADB_DRIVER_NAME, DriverMariaDB{})

	Register(MARIADB_DRIVER_NAME, Driver{
		SupportsReturning: SupportsReturningColumns,
		Driver:            &DriverMariaDB{},
		Open: func(ctx context.Context, drv *Driver, dsn string, opts ...OpenOption) (Database, error) {
			return OpenSQL(MARIADB_DRIVER_NAME, drv, dsn, opts...)
		},
		BuildDatabaseError: mySQLDatabaseError,
		ExplainQuery: func(ctx context.Context, q DB, query string, args []any) (string, error) {
			return explainMySQL(ctx, q, query, args)
		},
	})
}

type connectorMariaDB struct {
	driver.Connector
}

func (d DriverMariaDB) Open(dsn string) (driver.Conn, error) {
	return d.MySQLDriver.Open(dsn)
}

func (d DriverMariaDB) OpenConnector(dsn string) (driver.Connector, error) {
	connector, err := d.MySQLDriver.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}
	return &connectorMariaDB{
		Connector: connector,
	}, nil
}

func (c *connectorMariaDB) Driver() driver.Driver {
	return &DriverMariaDB{}
}
