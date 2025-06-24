//go:build !mysql && !mysql_local && !mariadb && !sqlite && postgres

package pages_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
)

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES ($1)`
	testPageUPDATE       = `UPDATE test_pages SET description = $1 WHERE id = $2`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = $1`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
	 	id  SERIAL PRIMARY KEY,
	 	description  VARCHAR(255) NOT NULL
	)`
)

func init() {
	var (
		// dbEngine = getEnv("DB_ENGINE", "sqlite3")
		// dbURL    = getEnv("DB_URL", "file::memory:?cache=shared")
		// dbURL = getEnv("DB_URL", "test.sqlite3.db")
		// dbEngine = getEnv("DB_ENGINE", "mysql")
		// dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		// dbEngine = getEnv("DB_ENGINE", "mariadb")
		// dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3307)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		dbEngine = getEnv("DB_ENGINE", "postgres") // "sqlite3", "mysql", "postgres"
		dbURL    = getEnv("DB_URL", "postgres://root:my-secret-pw@127.0.0.1:5432/django-pages-test?sslmode=disable&TimeZone=UTC")
	)

	var err error
	sqlDB, err = drivers.Open(context.Background(), dbEngine, dbURL)
	if err != nil {
		panic(err)
	}

	for {
		//  Wait for the database to be ready
		if err := sqlDB.Ping(); err == nil {
			break
		}
		fmt.Println("Waiting for PostgreSQL database to be ready...")
		time.Sleep(5 * time.Second)
	}

	fmt.Println("Using PostgreSQL database for testing")
}
