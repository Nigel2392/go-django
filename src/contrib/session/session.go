package session

import (
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
)

func NewAppConfig() django.AppConfig {
	var app = apps.NewDBAppConfig("session")

	var sessionManager = scs.New()

	app.Init = func(settings django.Settings, db *sql.DB) error {

		settings.Set(
			django.APPVAR_SESSION_MANAGER, sessionManager,
		)

		switch db.Driver().(type) {
		case *mysql.MySQLDriver:

			_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
					token CHAR(43) PRIMARY KEY,
					data BLOB NOT NULL,
					expiry TIMESTAMP(6) NOT NULL
				);`)
			assert.Err(err)

			rows, err := db.Query(`SHOW INDEX FROM sessions WHERE Key_name = 'sessions_expiry_idx'`)
			assert.Err(err)

			if !rows.Next() {
				logger.Info("Creating index sessions_expiry_idx on sessions table for MySQL")
				_, err = db.Exec(`CREATE INDEX sessions_expiry_idx ON sessions(expiry)`)
				assert.Err(err)
			}

			logger.Info("Using mysqlstore for session storage")
			sessionManager.Store = mysqlstore.New(db)

		case *sqlite3.SQLiteDriver:

			_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	data BLOB NOT NULL,
	expiry REAL NOT NULL
);`)
			assert.Err(err)

			logger.Info("Using sqlite3store for session storage")
			sessionManager.Store = sqlite3store.New(db)

		case *stdlib.Driver:

			_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	data BYTEA NOT NULL,
	expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);`)
			assert.Err(err)

			logger.Info("Using postgresstore for session storage")
			sessionManager.Store = postgresstore.New(db)

		default:

			logger.Info("Using memstore for session storage")
			sessionManager.Store = memstore.New()
		}

		return nil
	}

	app.Routing = func(m django.Mux) {
		m.Use(
			sessions.SessionMiddleware(sessionManager),
		)
	}

	return app
}
