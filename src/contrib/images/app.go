package images

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux"
)

type (
	Options struct {
		MediaBackend          mediafiles.Backend
		MediaDir              string
		MaxByteSize           uint
		AllowedFileExts       []string
		AllowedMimeTypes      []string
		CheckServePermissions bool
	}

	AppConfig struct {
		*apps.DBRequiredAppConfig
		Options *Options
	}
)

var (
	//go:embed assets/*
	assetsFS embed.FS

	app *AppConfig
)

type imageResult struct {
	*Image
	PreviewHTML string
}

func NewAppConfig(opts *Options) *AppConfig {
	if app == nil {
		app = &AppConfig{
			DBRequiredAppConfig: apps.NewDBAppConfig(
				"images",
			),
		}
	}

	if opts == nil {
		opts = &Options{
			MediaBackend: mediafiles.GetDefault(),
			MediaDir:     "images",
		}
	}

	if opts.MediaDir == "" {
		opts.MediaDir = "images"
	}

	if opts.MaxByteSize == 0 {
		opts.MaxByteSize = 1024 * 1024 * 128 // 128MB
	}

	if opts.AllowedFileExts == nil {
		opts.AllowedFileExts = []string{
			".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
		}
	}

	app.ModelObjects = []attrs.Definer{
		&Image{},
	}

	app.Options = opts
	app.Init = func(settings django.Settings, db drivers.Database) error {
		tpl.Add(*tpl.MergeConfig(
			&tpl.Config{
				FS:      filesystem.Sub(assetsFS, "assets/templates"),
				Matches: filesystem.MatchPrefix("images/"),
			},
			admin.AdminSite.TemplateConfig,
		))
		staticfiles.AddFS(filesystem.Sub(assetsFS, "assets/static"), nil)

		admin.RegisterApp(
			"images",
			admin.AppOptions{
				RegisterToAdminMenu: true,
				AppLabel:            trans.S("Images"),
				MenuLabel:           trans.S("Images"),
			},
			AdminImageModelOptions(app),
		)

		chooser.Register(&chooser.ChooserDefinition[*Image]{
			Model: &Image{},
			Title: trans.S("Image Chooser"),
			PreviewString: func(ctx context.Context, instance *Image) string {
				return fmt.Sprintf(`<img src="%s" alt="%s">`,
					django.Reverse("images:serve", instance.Path), instance.Title,
				)
			},
			ListPage: &chooser.ChooserListPage[*Image]{
				Template: "images/images_chooser_list.tmpl",
				SearchFields: []chooser.SearchField[*Image]{
					{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
					{Name: "Path", Lookup: expr.LOOKUP_ICONTANS},
				},
				NewList: func(req *http.Request, results []*Image) any {
					var resultList = make([]imageResult, len(results))
					for i, img := range results {
						resultList[i] = imageResult{
							Image: img,
							PreviewHTML: fmt.Sprintf(`<img src="%s" alt="%s">`,
								django.Reverse("images:serve", img.Path), img.Title,
							),
						}
					}
					return resultList
				},
			},
			CreatePage: &chooser.ChooserFormPage[*Image]{},
		})

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&Image{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create images table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		return nil
	}
	app.Routing = func(m mux.Multiplexer) {
		var g = m.Any("/images", nil, "images")
		g.Get(
			"/<<id>>",
			mux.NewHandler(app.serveImageByIDView),
			"serve_id",
		)
		g.Get(
			"/serve/*",
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
