//go:build !sqlite && !postgres && !mysql && !mysql_local && mariadb

package testdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

const ENGINE = "mariadb"

func open() (which string, db drivers.Database) {
	var sqlDB, err = drivers.Open(context.Background(), ENGINE, fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3307)/django-test?parseTime=true&multiStatements=true&interpolateParams=true",
		username, password,
	))
	if err != nil {
		panic(err)
	}

	for i := 0; i < retries; i++ {
		//  Wait for the database to be ready
		if err := sqlDB.Ping(context.Background()); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	_, err = sqlDB.ExecContext(context.Background(), `SET SESSION sql_mode = CONCAT(@@sql_mode, ',STRICT_ALL_TABLES,ERROR_FOR_DIVISION_BY_ZERO')`)
	if err != nil {
		panic(fmt.Errorf("failed to set SQL mode: %w", err))
	}

	return ENGINE, sqlDB
}
