package main

import (
	"database/sql"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/contrib/session"
	"github.com/Nigel2392/mux/middleware"
	"github.com/Nigel2392/src/core"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         true,
			"HOST":          "127.0.0.1",
			"PORT":          "8080",
			"DATABASE": func() *sql.DB {
				var db, err = sql.Open("sqlite3", "file::memory:?cache=shared")
				if err != nil {
					panic(err)
				}
				return db
			}(),
		}),
		django.AppMiddleware(
			middleware.DefaultLogger.Intercept,
		),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			core.NewAppConfig,
			blocks.NewAppConfig,
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
