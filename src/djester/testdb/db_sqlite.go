//go:build (!mysql && !postgres && !mariadb && !mysql_local) || (!mysql && !postgres && !mysql_local && !mariadb && !sqlite)

package testdb

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

const ENGINE = "sqlite3"

func open() (which string, db drivers.Database) {
	var sqlDB, err = drivers.Open(context.Background(), ENGINE, "file::memory:?cache=shared&_loc=auto")
	if err != nil {
		panic(err)
	}

	// Create test_pages table
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}

	return ENGINE, sqlDB
}
