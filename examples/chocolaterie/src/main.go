package main

import (
	"context"
	"os"

	chocolaterie "github.com/Nigel2392/go-django/examples/chocolaterie/src/chocolaterie"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	os.MkdirAll("./.private/chocolaterie", 0755)
	//
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/chocolaterie/db.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         false,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE:      db,
			// django.APPVAR_RECOVERER: false,

			auth.APPVAR_AUTH_EMAIL_LOGIN: true,
		}),

		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			pages.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			chocolaterie.NewAppConfig,
		),
	)

	pages.SetRoutePrefix("/pages")

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(
		logger.DBG,
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
