//go:build !sqlite && !postgres && !mariadb && !mysql_local && mysql

package testdb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

const ENGINE = "mysql"

const mysqlTestDSN = "%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true&interpolateParams=true"

type mysqlTestDatabase struct {
	drivers.Database
	name     string
	once     sync.Once
	closeErr error
}

func (db *mysqlTestDatabase) Close() error {
	db.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := db.Database.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+db.name+"`")
		db.closeErr = errors.Join(err, db.Database.Close())
	})

	return db.closeErr
}

func open() (which string, db drivers.Database) {
	var sqlDB, err = drivers.Open(context.Background(), ENGINE, fmt.Sprintf(
		mysqlTestDSN, username, password, "django-test",
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

	// Packages define different models with the same table names. Keep their
	// schemas separate so foreign keys cannot refer to another package's fixtures.
	name := "django_test_" + drivers.NewULID().String()
	if _, err = sqlDB.ExecContext(context.Background(), "CREATE DATABASE `"+name+"`"); err != nil {
		sqlDB.Close()
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	isolated, err := drivers.Open(context.Background(), ENGINE, fmt.Sprintf(
		mysqlTestDSN, username, password, name,
	))
	if err != nil {
		sqlDB.ExecContext(context.Background(), "DROP DATABASE `"+name+"`")
		sqlDB.Close()
		panic(err)
	}

	sqlDB.Close()
	database := &mysqlTestDatabase{Database: isolated, name: name}
	_, err = database.ExecContext(context.Background(), `SET SESSION sql_mode = CONCAT(@@sql_mode, ',STRICT_ALL_TABLES,ERROR_FOR_DIVISION_BY_ZERO')`)
	if err != nil {
		database.Close()
		panic(fmt.Errorf("failed to set SQL mode: %w", err))
	}

	return ENGINE, database
}
