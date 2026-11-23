# Images

The `images` app (`github.com/Nigel2392/go-django/contrib/images`) provides an `Image` model, an admin gallery list and search view, an image chooser modal with inline upload, HTTP serving endpoints (including 1-hour ID-based caching), and a home dashboard panel.

## Table of Contents

- [Installation and Options](#installation-and-options)
- [The `Image` Model](#the-image-model)
- [Routes and Template Usage](#routes-and-template-usage)
- [Admin Gallery, Search, Chooser, and Dashboard Panel](#admin-gallery-search-chooser-and-dashboard-panel)
- [Migrations and Table Creation](#migrations-and-table-creation)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and Options

Pass `images.NewAppConfig(opts)` to `django.Apps(...)` alongside `session`, `auth`, `admin`, and a configured `mediafiles` backend:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/images"
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
			images.NewAppConfig(&images.Options{
				MediaBackend:          mediaBackend,
				MediaDir:              "images",
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

- `MediaBackend`: Storage backend for uploaded images. Defaults to `mediafiles.GetDefault()`; `AppConfig.MediaBackend()` falls back to `memory.NewBackend(5)` if none is configured.
- `MediaDir`: Target subdirectory in the media backend. Defaults to `"images"`.
- `MaxByteSize`: Maximum allowed file size in bytes. Defaults to `1024 * 1024 * 128` (128 MB) when zero.
- `AllowedFileExts`: Allowed file extensions enforced by the admin upload field (`filefield.FileStorageField`). When `nil`, defaults to `[".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp"]`.
- `AllowedMimeTypes`: Optional list of MIME types exposed via `AppConfig.AllowedMimeTypes()`.
- `CheckServePermissions`: When `true`, both image serve routes enforce `permissions.HasPermission(r, "images.model:serve")` and fail with HTTP `403` if missing.

Passing `nil` to `images.NewAppConfig(nil)` applies all default options.

---

## The `Image` Model

```go
type Image struct {
	models.Model
	ID        uint32        `json:"id"`
	Title     string        `json:"title"`
	Path      string        `json:"path"`
	CreatedAt time.Time     `json:"created_at"`
	FileSize  sql.NullInt32 `json:"file_size"`
	FileHash  string        `json:"file_hash"`
}
```

- `BeforeCreate` populates `CreatedAt` with `time.Now()` if zero.
- `Validate` returns `images.ErrEmptyPath` when `Path == ""`, rejects negative `FileSize.Int32` values, and marks `FileSize.Valid = true`.
- `File() (mediafiles.StoredObject, error)` opens and caches the stored file from `images.App.MediaBackend()` using `Path`.

To link an `*images.Image` on a model and use the built-in admin chooser widget:

```go
attrs.NewField(n, "Thumbnail", &attrs.Config{
	Null:     true,
	Label:    trans.S("Thumbnail"),
	HelpText: trans.S("Choose a thumbnail image."),
	RelForeignKey: attrs.Relate(&images.Image{}, "", nil),
	FormWidget: func(fc attrs.Config) widgets.Widget {
		return chooser.NewChooserWidget(fc.RelForeignKey.Model(), fc.WidgetAttrs)
	},
})
```

See [Filesystem](../filesystem.md), [Permissions](../permissions.md), and the [Blog App Example](../examples/blog.md) for usage on page models.

---

## Routes and Template Usage

`images.NewAppConfig` registers two `GET` routes under `/images`:

| Route Name | Pattern | Lookup & Caching |
| --- | --- | --- |
| `images:serve_id` | `GET /images/<<id>>` | Checks global cache key `"contrib:images:<id>"`; on miss, queries `Image` by `ID` and caches the `*Image` for **1 hour** (`time.Hour`). |
| `images:serve` | `GET /images/serve/*` | Cleans the wildcard path and serves the matching object directly from `AppConfig.MediaBackend()`. |

Both routes allow only `GET`, check `"images.model:serve"` when `CheckServePermissions` is `true`, and serve bytes inline via `http.ServeContent`.

In HTML templates, reverse `images:serve` with the image's `Path` (or `images:serve_id` with `ID`):

```html
{{ if .Data.page.Thumbnail }}
    <img src="{{ url "images:serve" .Data.page.Thumbnail.Path }}" alt="{{ .Data.page.Thumbnail.Title }}">
{{ end }}
```

See [Routing](../routing.md) and [Rendering](../rendering.md) for template URL reversal.

---

## Admin Gallery, Search, Chooser, and Dashboard Panel

During `Init`, `images` registers:

- **Admin App & Gallery List**: Registers the `"images"` admin app (`MenuOrder: 10`) and `&Image{}` (`AdminImageModelOptions(App)`). Instead of a plain table, `ListView.GetHandler` renders a thumbnail gallery grid (`images/image_list.tmpl`) ordered by `-CreatedAt` (up to 25 items per page).
- **Admin Search**: Configures `searches.SearchOptions` on `Title` and `Path` using `SearchComponent` (`images/image_search_list.tmpl`) to render thumbnail search results.
- **Admin Chooser**: Registers a `contrib/admin/chooser` definition for `*Image` with a custom thumbnail modal grid (`images/images_chooser_list.tmpl`), search on `Title` and `Path`, `ExtraData` (`caption`, `filesize`), and inline image upload (`CreatePage`).
- **Home Dashboard Panel**: Registers a display panel on `admin.RegisterHomePageDisplayPanelHook` displaying the `Image`/`Images` count and linking to the admin image gallery.

See [Admin App](./admin/app.md) and [Admin Models](./admin/models.md) for admin integration details.

---

## Migrations and Table Creation

`images.NewAppConfig` wraps the app in a `*migrator.MigratorAppConfig` with embedded migrations under `migrations/images`.

- **With `migrator` installed**: Manage schema migrations with `makemigrations` and `migrate`; see [CLI](../cli.md).
- **Without `migrator` installed**: `App.Init` automatically runs `schemaEditor.CreateTable(context.Background(), table, true)` and creates table indexes.

---

## Notes and Caveats

- **1-hour cache on `images:serve_id`**: `serveImageByIDView` caches the `*Image` struct under `"contrib:images:<id>"` for `1 hour`. Updating or deleting an `Image` record in the admin does not automatically evict that cache key.
- **Direct backend lookup on `images:serve`**: `serveImageByPathView` constructs an `&Image{Path: path, Title: title}` from the URL path without querying the database table. Any readable object in `MediaBackend` can be accessed by path unless `CheckServePermissions` restricts access.
- **In-memory buffering on serve**: `serveImageFnView` reads `stat.Size()` bytes into a buffer before invoking `http.ServeContent`.
- **Upload validation options**: `AdminImageModelOptions` enforces `Options.AllowedFileExts` on the upload field; `Options.MaxByteSize` and `Options.AllowedMimeTypes` are stored on `AppConfig` and returned by `MaxByteSize()` and `AllowedMimeTypes()`.
