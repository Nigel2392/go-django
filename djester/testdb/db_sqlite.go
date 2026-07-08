//go:build (!mysql && !postgres && !mariadb && !mysql_local) || (!mysql && !postgres && !mysql_local && !mariadb && !sqlite)

package testdb

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
)

const ENGINE = "sqlite3"

func open() (which string, db drivers.Database) {
	// var openDSN = "file::memory:?cache=shared&_loc=auto"
	var openDSN = "file::memory:?cache=shared&_loc=auto&mode=memory"
	// var openDSN = "./test.sqlite?_loc=auto&mode=memory&cache=shared"
	var sqlDB, err = drivers.Open(
		context.Background(),
		ENGINE,
		openDSN,

		//// SQLite in-memory databases should only have one connection
		//drivers.SQLDBOption(func(driverName string, db *sql.DB) error {
		//	// db.SetMaxOpenConns(1)
		//	// db.SetMaxIdleConns(1)
		//	// db.SetConnMaxLifetime(30 * time.Second) // 30 seconds
		//	// db.SetConnMaxIdleTime(30) // 30 seconds
		//	return nil
		//}),
	)
	if err != nil {
		panic(err)
	}

	// Create test_pages table
	if err := sqlDB.Ping(context.Background()); err != nil {
		panic(err)
	}

	// set to false to avoid database locking issues in tests
	queries.QUERYSET_CREATE_IMPLICIT_TRANSACTION = false

	return ENGINE, sqlDB
}
