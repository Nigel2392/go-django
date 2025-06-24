//go:build (!mysql && !mysql_local && !postgres && !mariadb) || (!mysql && !mysql_local && !postgres && !mariadb && !sqlite)

package pages_test

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django-queries/src/drivers"
)

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES (?)`
	testPageUPDATE       = `UPDATE test_pages SET description = ? WHERE id = ?`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = ?`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT
	)`
)

func init() {
	var (
		dbEngine = getEnv("DB_ENGINE", "sqlite3")
		dbURL    = getEnv("DB_URL", "file::memory:?cache=shared")
		// dbURL = getEnv("DB_URL", "test.sqlite3.db")
		// dbEngine = getEnv("DB_ENGINE", "mysql")
		// dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		// dbEngine = getEnv("DB_ENGINE", "mariadb")
		// dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3307)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		// dbEngine = getEnv("DB_ENGINE", "postgres") // "sqlite3", "mysql", "postgres"
		// dbURL    = getEnv("DB_URL", "postgres://root:my-secret-pw@127.0.0.1:5432/django-pages-test?sslmode=disable&TimeZone=UTC")
	)

	var err error
	sqlDB, err = drivers.Open(context.Background(), dbEngine, dbURL)
	if err != nil {
		panic(err)
	}

	// Create test_pages table
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}

	fmt.Println("Using SQLite3 in-memory database for testing")
}
