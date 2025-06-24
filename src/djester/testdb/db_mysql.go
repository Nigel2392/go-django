//go:build !sqlite && !postgres && !mariadb && !mysql_local && mysql

package testdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
)

const ENGINE = "mysql"

func open() (which string, db drivers.Database) {
	var sqlDB, err = drivers.Open(context.Background(), ENGINE, fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3306)/django-test?parseTime=true&multiStatements=true&interpolateParams=true",
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
