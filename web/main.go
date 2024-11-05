package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	auditlogs_sqlite "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/audit_logs_sqlite"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/web/blog"
	"github.com/Nigel2392/go-django/web/core"
	"github.com/Nigel2392/go-django/web/todos"

	_ "github.com/Nigel2392/go-django/src/contrib/pages/backend-sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         false,
			"HOST":          "127.0.0.1",
			"PORT":          "8080",
			"DATABASE": func() *sql.DB {
				// var db, err = sql.Open("mysql", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true")
				var db, err = sql.Open("sqlite3", "./.private/db.sqlite3")
				if err != nil {
					panic(err)
				}
				auditlogs.RegisterBackend(
					auditlogs_sqlite.NewSQLiteStorageBackend(db),
				)
				return db
			}(),
			// django.APPVAR_RECOVERER: false,

			"AUTH_EMAIL_LOGIN": true,
		}),
		// django.AppMiddleware(
		// middleware.DefaultLogger.Intercept,
		// ),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			pages.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			core.NewAppConfig,
			blocks.NewAppConfig,
			blog.NewAppConfig,
			todos.NewAppConfig,
		),
	)

	pages.SetPrefix("/pages")
	app.Mux.Any("/pages/*", http.StripPrefix("/pages", pages.Serve(
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	)), "pages")

	var err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(
		logger.DBG,
	)

	err = staticfiles.Collect(func(path string, f fs.File) error {
		var stat, err = f.Stat()
		if err != nil {
			return err
		}
		fmt.Println("Collected", path, stat.Size())
		return nil
	})
	if err != nil {
		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
