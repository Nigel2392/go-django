package images

import (
	"database/sql"
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/mux"
	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"

	_ "embed"
)

type (
	Options struct {
		MediaBackend     mediafiles.Backend
		MediaDir         string
		MaxByteSize      uint
		AllowedFileExts  []string
		AllowedMimeTypes []string
	}

	AppConfig struct {
		*apps.DBRequiredAppConfig
		Options *Options
	}
)

var (
	app *AppConfig

	//go:embed sql/schema.mysql.sql
	schemaMySQL string

	//go:embed sql/schema.sqlite.sql
	schemaSqlite string
)

func NewAppConfig(opts *Options) *AppConfig {
	if app == nil {
		app = &AppConfig{
			DBRequiredAppConfig: apps.NewDBAppConfig(
				"images",
			),
		}
	}

	if opts.MediaDir == "" {
		opts.MediaDir = "images"
	}

	if opts.MaxByteSize == 0 {
		opts.MaxByteSize = 1024 * 1024 * 128 // 128MB
	}

	app.Options = opts
	app.Init = func(settings django.Settings, db *sql.DB) error {
		admin.RegisterApp(
			"images",
			admin.AppOptions{},
			AdminImageModelOptions(),
		)

		var err error
		switch db.Driver().(type) {
		case mysql.MySQLDriver:
			_, err = db.Exec(schemaMySQL)
		case *sqlite3.SQLiteDriver:
			_, err = db.Exec(schemaSqlite)
		default:
			err = fmt.Errorf("unsupported database driver for app images: %T", db.Driver())
		}
		return err
	}
	app.Routing = func(m django.Mux) {
		var g = m.Any("/images", nil, "images")
		g.Get(
			"/<<id>>",
			mux.NewHandler(app.serveImageByIDView),
			"serve",
		)
		g.Post(
			"/upload",
			mux.NewHandler(app.serveImageUpload),
			"upload",
		)
		g.Get(
			"/list",
			mux.NewHandler(app.serveImageList),
			"list",
		)
		g.Post(
			"/<<id>>/delete",
			mux.NewHandler(app.serveImageDeletion),
			"delete",
		)
		g.Get(
			"/*",
			mux.NewHandler(app.serveImageByPathView),
			"serve",
		)
	}

	return app
}

func (c *AppConfig) MediaBackend() mediafiles.Backend {
	if c.Options.MediaBackend == nil {
		c.Options.MediaBackend = mediafiles.GetDefault()
	}
	if c.Options.MediaBackend == nil {
		c.Options.MediaBackend = memory.NewBackend(5)
	}
	return c.Options.MediaBackend
}

func (c *AppConfig) MediaDir() string {
	return c.Options.MediaDir
}

func (c *AppConfig) MaxByteSize() uint {
	return c.Options.MaxByteSize
}

func (c *AppConfig) AllowedFileExts() []string {
	return c.Options.AllowedFileExts
}

func (c *AppConfig) AllowedMimeTypes() []string {
	return c.Options.AllowedMimeTypes
}
