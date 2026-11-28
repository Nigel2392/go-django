package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/contrib/shop"
	"github.com/Nigel2392/go-django/examples/commerce/tutorial/store"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/goldcrest"
	"github.com/justinas/nosurf"
	"golang.org/x/text/currency"
)

func main() {
	port := flag.String("port", "8081", "port for the local tutorial")
	flag.Parse()
	if err := run(*port); err != nil {
		log.Fatal(err)
	}
}

func run(port string) error {
	drivers.LOG_SQL_QUERIES = false
	db, err := drivers.Open(context.Background(), "sqlite3", "file:commerce_tutorial?mode=memory&cache=shared&_foreign_keys=on",
		drivers.SQLDBOption(func(_ string, db *sql.DB) error {
			// Serialize requests to the in-memory SQLite database.
			db.SetMaxOpenConns(1)
			return nil
		}),
	)
	if err != nil {
		return err
	}
	defer db.Close()

	mediafiles.RegisterBackend("memory", memory.NewBackend(1))
	mediafiles.SetDefault("memory")
	// Match the CSRF origin check to the scheme of the local request.
	goldcrest.Register(django.HOOK_SETUP_NOSURF, 0, django.NosurfSetupHook(func(_ *django.Application, h *nosurf.CSRFHandler) {
		h.SetIsTLSFunc(func(r *http.Request) bool { return r.TLS != nil })
	}))

	app := django.App(
		django.AppLogger(&logger.Logger{Level: logger.ERR}),
		django.Configure(map[string]any{
			django.APPVAR_DATABASE:      db,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          port,
			django.APPVAR_ALLOWED_HOSTS: []string{"127.0.0.1", "localhost"},
			shop.APPVAR_SHOP_CURRENCY:   currency.EUR,
		}),
		django.Flag(django.FlagSkipChecks, django.FlagSkipCmds),
		django.Apps(
			session.NewAppConfig, messages.NewAppConfig, auth.NewAppConfig,
			admin.NewAppConfig, reports.NewAppConfig, auditlogs.NewAppConfig,
			editor.NewAppConfig, blocks.NewAppConfig, revisions.NewAppConfig,
			settings.NewAppConfig, shop.NewAppConfig, store.NewAppConfig,
		),
	)
	if err := app.Initialize(); err != nil {
		return err
	}
	fmt.Printf("Commerce tutorial at http://127.0.0.1:%s\n", port)
	return app.Serve()
}
