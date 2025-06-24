//go:build !mysql && !mysql_local && !mariadb && !sqlite && postgres

package testdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
)

const ENGINE = "postgres"

func open() (which string, db drivers.Database) {
	var sqlDB, err = drivers.Open(context.Background(), ENGINE, fmt.Sprintf(
		"postgres://%s:%s@127.0.0.1:5432/django-test?sslmode=disable&TimeZone=UTC",
		username, password,
	))
	if err != nil {
		panic(err)
	}

	for i := 0; i < retries; i++ {
		//  Wait for the database to be ready
		if err := sqlDB.Ping(); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	return ENGINE, sqlDB
}
