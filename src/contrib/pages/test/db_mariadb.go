//go:build !sqlite && !postgres && !mysql && !mysql_local && mariadb

package pages_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/go-django-queries/src/drivers"
)

const (
	testPageINSERT       = `INSERT INTO test_pages (description) VALUES (?)`
	testPageUPDATE       = `UPDATE test_pages SET description = ? WHERE id = ?`
	testPageByID         = `SELECT id, description FROM test_pages WHERE id = ?`
	testPageCREATE_TABLE = `CREATE TABLE IF NOT EXISTS test_pages (
		id INT AUTO_INCREMENT PRIMARY KEY,
		description VARCHAR(255) NOT NULL
	)`
)

func init() {
	var (
		// dbEngine = getEnv("DB_ENGINE", "sqlite3")
		// dbURL    = getEnv("DB_URL", "file::memory:?cache=shared")
		// dbURL = getEnv("DB_URL", "test.sqlite3.db")
		// dbEngine = getEnv("DB_ENGINE", "mysql")
		// dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		dbEngine = getEnv("DB_ENGINE", "mariadb")
		dbURL    = getEnv("DB_URL", "root:my-secret-pw@tcp(127.0.0.1:3307)/django-pages-test?parseTime=true&multiStatements=true&interpolateParams=true")
		// dbEngine = getEnv("DB_ENGINE", "postgres") // "sqlite3", "mysql", "postgres"
		// dbURL    = getEnv("DB_URL", "postgres://root:my-secret-pw@127.0.0.1:5432/django-pages-test?sslmode=disable&TimeZone=UTC")

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
		fmt.Println("Waiting for MariaDB database to be ready...")
		time.Sleep(5 * time.Second)
	}

	fmt.Println("Using MariaDB database for testing")
}
