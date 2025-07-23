package main

import (
	"context"

	formsapp "github.com/Nigel2392/go-django/examples/formsapp/src/formsapp"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "./db.sqlite3")
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

		}),

		django.Apps(
			formsapp.NewAppConfig,
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
