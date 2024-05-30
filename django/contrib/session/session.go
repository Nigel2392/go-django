package session

import (
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
)

// schemaMySQL
var schemaMySQL = `CREATE TABLE IF NOT EXISTS sessions (
	token CHAR(43) PRIMARY KEY,
	data BLOB NOT NULL,
	expiry TIMESTAMP(6) NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);`

// schemaSQLite
var schemaSQLite = `CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	data BLOB NOT NULL,
	expiry REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);`

func NewAppConfig() django.AppConfig {
	var app = apps.NewDBAppConfig("session")

	var sessionManager = scs.New()

	app.Init = func(settings django.Settings, db *sql.DB) error {

		settings.Set("SESSION_MANAGER", sessionManager)

		switch db.Driver().(type) {
		case *mysql.MySQLDriver:
			fmt.Println("Using mysqlstore for session storage")
			sessionManager.Store = mysqlstore.New(db)

			_, err := db.Exec(schemaMySQL)
			assert.Err(err)

		case *sqlite3.SQLiteDriver:
			fmt.Println("Using sqlite3store for session storage")
			sessionManager.Store = sqlite3store.New(db)

			_, err := db.Exec(schemaSQLite)
			assert.Err(err)
		default:
			fmt.Println("Using memstore for session storage")
			sessionManager.Store = memstore.New()
		}

		return nil
	}

	app.AddMiddleware(
		sessions.SessionMiddleware(sessionManager),
	)

	return app
}
