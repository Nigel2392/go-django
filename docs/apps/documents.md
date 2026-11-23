# Documents

The `documents` app (`github.com/Nigel2392/go-django/contrib/documents`) provides a `Document` model, file upload handling in the admin interface, a media-backend-backed download endpoint, an admin chooser, and a dashboard panel.

## Table of Contents

- [Installation and Options](#installation-and-options)
- [The `Document` Model](#the-document-model)
- [Routes and Serving Documents](#routes-and-serving-documents)
- [Admin, Chooser, and Dashboard Integration](#admin-chooser-and-dashboard-integration)
- [Migrations and Table Creation](#migrations-and-table-creation)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and Options

Install the app by passing `documents.NewAppConfig(opts)` into `django.Apps(...)`. Make sure you configure a `mediafiles` backend (or pass one explicitly via `Options.MediaBackend`), as well as `admin`, `auth`, and `session`:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/documents"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	mediafs "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "./app.db")
	if err != nil {
		panic(err)
	}

	mediaBackend := mediafs.NewBackend("./media", 0)
	mediafiles.RegisterBackend("filesystem", mediaBackend)
	mediafiles.SetDefault("filesystem")

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE: db,
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			documents.NewAppConfig(&documents.Options{
				MediaBackend:          mediaBackend,
				MediaDir:              "documents",
				CheckServePermissions: false,
			}),
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

### `Options` Reference

```go
type Options struct {
	MediaBackend          mediafiles.Backend
	MediaDir              string
	MaxByteSize           uint
	AllowedFileExts       []string
	AllowedMimeTypes      []string
	CheckServePermissions bool
}
```

- `MediaBackend`: Storage backend for uploaded documents. Defaults to `mediafiles.GetDefault()`; if still nil when `AppConfig.MediaBackend()` is called, it falls back to `memory.NewBackend(5)`.
- `MediaDir`: Subdirectory inside the media backend where uploaded documents are stored. Defaults to `"documents"`.
- `MaxByteSize`: Maximum file size in bytes. Defaults to `1024 * 1024 * 128` (128 MB) when zero.
- `AllowedFileExts`: Allowed file extensions for uploads in the admin form (`filefield.FileStorageField`). When `nil`, defaults to:
  `".pdf"`, `".docx"`, `".doc"`, `".odt"`, `".xlsx"`, `".xls"`, `".pptx"`, `".ppt"`, `".txt"`, `".rtf"`, `".csv"`, `".tsv"`, `".zip"`, `".rar"`, `".7z"`, `".tar"`, `".gz"`, `".bz2"`, `".md"`, `".markdown"`.
- `AllowedMimeTypes`: Optional list of allowed MIME types exposed via `AppConfig.AllowedMimeTypes()`.
- `CheckServePermissions`: When `true`, download routes require the `"documents.model:serve"` permission via `permissions.HasPermission(r, "documents.model:serve")` (returning HTTP `403` otherwise).

Passing `nil` to `documents.NewAppConfig(nil)` applies all default options.

---

## The `Document` Model

The `Document` model stores file metadata and links to the underlying file in the configured `MediaBackend`:

```go
type Document struct {
	models.Model
	ID        uint32        `json:"id"`
	Title     string        `json:"title"`
	Path      string        `json:"path"`
	CreatedAt time.Time     `json:"created_at"`
	FileSize  sql.NullInt32 `json:"file_size"`
	FileHash  string        `json:"file_hash"`
}
```

- `BeforeCreate` populates `CreatedAt` with `time.Now()` when unset.
- `Validate` requires `Path` to be non-empty, rejects negative `FileSize.Int32` values, and marks `FileSize.Valid = true`.
- `File() (mediafiles.StoredObject, error)` opens (and caches on the struct) the stored file from `AppConfig.MediaBackend()` using `Path`.

You can reference `*documents.Document` as a foreign key on your own models and pair it with an admin chooser widget:

```go
attrs.NewField(n, "Document", &attrs.Config{
	Null:     true,
	Label:    trans.S("Document"),
	HelpText: trans.S("Attach a document."),
	RelForeignKey: attrs.Relate(&documents.Document{}, "", nil),
	FormWidget: func(fc attrs.Config) widgets.Widget {
		return chooser.NewChooserWidget(fc.RelForeignKey.Model(), fc.WidgetAttrs)
	},
})
```

See [Permissions](../permissions.md) and [Filesystem](../filesystem.md) for permission checks and media backend configuration.

---

## Routes and Serving Documents

`documents.NewAppConfig` mounts two `GET` routes under `/documents`:

| Route Name | Pattern | Behavior |
| --- | --- | --- |
| `documents:serve_id` | `GET /documents/<<id>>` | Looks up `Document` by ID (see caveat below) and serves as an attachment |
| `documents:serve` | `GET /documents/serve/*` | Cleans the wildcard path, looks up `Document` where `Path` matches, and serves as an attachment |

In templates, generate a download link using `documents:serve` with the document's `Path`:

```html
{{ if .Data.page.Document }}
    <a href="{{ url "documents:serve" .Data.page.Document.Path }}" target="_blank" rel="noopener noreferrer">
        {{ .Data.page.Document.Title }}
    </a>
{{ end }}
```

Both serve routes verify `r.Method == http.MethodGet`, optionally enforce `"documents.model:serve"` when `CheckServePermissions` is `true`, set `Content-Disposition: attachment; filename="<basename>"`, and stream content via `http.ServeContent`.

---

## Admin, Chooser, and Dashboard Integration

During `Init`, `documents` automatically registers:

- **Admin App & Model**: Registers the `"documents"` app (`MenuOrder: 15`) and `&Document{}` (`AdminDocumentModelOptions(app)`) with custom upload handling (`DocumentFile` field), add/edit/delete views, bulk delete, `admin/searches` support on `Title` and `Path`, and [`contrib/filters`](./filters.md) specs (`search` and `date_range`).
- **Admin Chooser**: Registers a `*Document` chooser with `github.com/Nigel2392/go-django/contrib/admin/chooser`, supporting modal list search on `Title` and `Path` as well as inline upload via `CreatePage`.
- **Admin Dashboard Panel**: Registers a display panel on `admin.RegisterHomePageDisplayPanelHook` showing the total document count and linking to the admin document list.

See [Admin App](./admin/app.md) and [Admin Models](./admin/models.md) for admin customization details.

---

## Migrations and Table Creation

`documents.NewAppConfig` returns a `*migrator.MigratorAppConfig` wrapping embedded SQL/schema migrations (`migrations/documents`).

- **With `migrator` installed**: Run `makemigrations` / `migrate` CLI commands as usual to manage the `Document` table; see [Migrations](../../queries/docs/README.md) and [CLI](../cli.md).
- **Without `migrator` installed**: `app.Init` automatically creates the `Document` table and its indexes via `schemaEditor.CreateTable(context.Background(), table, true)` and `schemaEditor.AddIndex(..., true)`.

---

## Notes and Caveats

- **`documents:serve_id` reads `r.FormValue("id")`**: Although the route pattern is `/documents/<<id>>`, `serveDocumentByIDView` reads `r.FormValue("id")` rather than `mux.Vars(r).Get("id")`. Passing only a path segment such as `/documents/42` fails with `"no document ID provided"`; include `?id=42` in the query string or prefer `documents:serve` (`/documents/serve/<path>`).
- **Full file buffering on serve**: `serveDocumentFnView` reads `stat.Size()` bytes from the media backend into memory before calling `http.ServeContent`.
- **Upload validation enforcement**: `AdminDocumentModelOptions` enforces `Options.AllowedFileExts` on the upload field; `Options.MaxByteSize` and `Options.AllowedMimeTypes` are stored on `AppConfig` and returned by `MaxByteSize()` and `AllowedMimeTypes()`, rather than enforced automatically in the built-in admin form validator.
- **Template delete/list note**: The default `documents` admin list view uses `admin`'s standard `ListView` configuration, while delete confirmations use `documents/documents_delete.tmpl`.
