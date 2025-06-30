package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/Nigel2392/go-django/examples/blogapp/blog"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/contrib/editor/features/images"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"

	// auditlogs_mysql "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/audit_logs_mysql"
	"github.com/Nigel2392/go-django/src/contrib/revisions"

	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/db.blogapp.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS:   []string{"*"},
			django.APPVAR_DEBUG:           true,
			django.APPVAR_HOST:            "127.0.0.1",
			django.APPVAR_PORT:            "8080",
			django.APPVAR_DATABASE:        db,
			django.APPVAR_RECOVERER:       false,
			auth.APP_AUTH_EMAIL_LOGIN:     true,
			migrator.APPVAR_MIGRATION_DIR: "./migrations-blogapp",
		}),
		// django.AppMiddleware(
		// middleware.DefaultLogger.Intercept,
		// ),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			messages.NewAppConfig,
			pages.NewAppConfig,
			revisions.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			editor.NewAppConfig,
			blog.NewAppConfig,
			images.NewAppConfig(&images.Options{
				MediaBackend: mediafiles.GetDefault(),
				MediaDir:     "images-blogapp",
			}),
			migrator.NewAppConfig,
		),
	)

	mediafiles.SetDefault("filesystem")

	pages.SetRoutePrefix("/pages")
	app.Mux.Any("/pages/*", http.StripPrefix("/pages", pages.Serve(
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	)), "pages")

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(
		logger.DBG,
	)

	if len(os.Args) == 1 {
		blogPages, err := queries.GetQuerySet(&blog.BlogPage{}).All()
		if err != nil {
			panic(fmt.Errorf("failed to get blog pages: %w", err))
		}
		fmt.Println("Blog pages:", len(blogPages))
		for page := range blogPages.Objects() {
			fmt.Printf(" - %s (ID: %d, %d)\n", page.Title, page.ID(), page.PageNode.PageID)
		}

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
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
