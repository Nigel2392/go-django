package documents

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
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
	"github.com/Nigel2392/goldcrest"
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

	//go:embed migrations/*
	migrationFS embed.FS

	app *AppConfig
)

type documentResult struct {
	*Document
	PreviewHTML string
}

func NewAppConfig(opts *Options) django.AppConfig {
	if app == nil {
		app = &AppConfig{
			DBRequiredAppConfig: apps.NewDBAppConfig(
				"documents",
			),
		}
	}

	if opts == nil {
		opts = &Options{
			MediaBackend: mediafiles.GetDefault(),
			MediaDir:     "documents",
		}
	}

	if opts.MediaDir == "" {
		opts.MediaDir = "documents"
	}

	if opts.MaxByteSize == 0 {
		opts.MaxByteSize = 1024 * 1024 * 128 // 128MB
	}

	if opts.AllowedFileExts == nil {
		opts.AllowedFileExts = []string{
			".pdf", ".docx", ".doc", ".odt",
			".xlsx", ".xls",
			".pptx", ".ppt",
			".txt", ".rtf",
			".csv", ".tsv",
			".zip", ".rar", ".7z",
			".tar", ".gz", ".bz2",
			".md", ".markdown",
		}
	}

	app.ModelObjects = []attrs.Definer{
		&Document{},
	}

	app.Options = opts
	app.Init = func(settings django.Settings, db drivers.Database) error {
		tpl.Add(*tpl.MergeConfig(
			&tpl.Config{
				FS:      filesystem.Sub(assetsFS, "assets/templates"),
				Matches: filesystem.MatchPrefix("documents/"),
			},
			admin.AdminSite.TemplateConfig,
		))
		staticfiles.AddFS(filesystem.Sub(assetsFS, "assets/static"), nil)

		admin.RegisterApp(
			"documents",
			admin.AppOptions{
				RegisterToAdminMenu: true,
				MenuOrder:           15,
				AppLabel:            trans.S("Documents"),
				MenuLabel:           trans.S("Documents"),
			},
			AdminDocumentModelOptions(app),
		)

		chooser.Register(&chooser.ChooserDefinition[*Document]{
			Model: &Document{},
			Title: trans.S("Document Chooser"),
			PreviewString: func(ctx context.Context, instance *Document) string {
				return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`,
					django.Reverse("documents:serve", instance.Path), instance.Title,
				)
			},
			ExtraData: func(ctx context.Context, instance *Document) map[string]any {
				return map[string]any{
					"caption":  instance.Title,
					"filesize": instance.FileSize.Int32,
				}
			},
			ListPage: &chooser.ChooserListPage[*Document]{
				QuerySet: func(r *http.Request, model *Document) (*queries.QuerySet[*Document], error) {
					return queries.GetQuerySet(model).OrderBy("-CreatedAt"), nil
				},
				SearchFields: []admin.SearchField{
					{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
					{Name: "Path", Lookup: expr.LOOKUP_ICONTANS},
				},
			},
			CreatePage: &chooser.ChooserFormPage[*Document]{},
		})

		goldcrest.Register(admin.RegisterHomePageDisplayPanelHook, 2, admin.RegisterHomePageDisplayPanelHookFunc(func(*http.Request, *admin.AdminApplication) []admin.DisplayPanel {
			return []admin.DisplayPanel{{
				IconName: "icon-file-document",
				Title: func(ctx context.Context, count int64) string {
					return trans.P(ctx, "Document", "Documents", count)
				},
				QuerySet: func(r *http.Request) *queries.QuerySet[attrs.Definer] {
					return queries.GetQuerySet[attrs.Definer](&Document{})
				},
				URL: func(r *http.Request) string {
					return django.Reverse("admin:apps:model", "documents", "document")
				},
			}}
		}))

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&Document{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create documents table: %w", err)
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
		var g = m.Any("/documents", nil, "documents")
		g.Get(
			"/<<id>>",
			mux.NewHandler(app.serveDocumentByIDView),
			"serve_id",
		)
		g.Get(
			"/serve/*",
			mux.NewHandler(app.serveDocumentByPathView),
			"serve",
		)
	}

	return &migrator.MigratorAppConfig{
		AppConfig: app,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/documents",
		),
	}
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
