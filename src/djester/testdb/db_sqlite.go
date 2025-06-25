//go:build (!mysql && !postgres && !mariadb && !mysql_local) || (!mysql && !postgres && !mysql_local && !mariadb && !sqlite)

package testdb

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

const ENGINE = "sqlite3"

func open() (which string, db drivers.Database) {
	var openDSN = "file::memory:?cache=shared&_loc=auto"
	var sqlDB, err = drivers.Open(
		context.Background(),
		ENGINE,
		openDSN,

		// SQLite in-memory databases should only have one connection
		drivers.SQLDBOption(func(driverName string, db *sql.DB) error {
			db.SetMaxOpenConns(1)
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}

	// Create test_pages table
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}

	return ENGINE, sqlDB
}
