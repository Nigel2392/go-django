# Blog Application Example

The `blogapp` example (`examples/blogapp`) shows how to build a full CMS-style blog using Go-Django's `pages` app, proxy page models (`models.Model` + `*pages.PageNode`), rich-text `editor` fields, structured `blocks`, admin `chooser` widgets, inline `ModelFormPanel` relations (`BlogImage` with `OrderableMixin`), `images`, `documents`, `search`, `translations`, and `migrator`.

## Table of Contents

- [Project Structure](#project-structure)
- [Running the Example](#running-the-example)
- [Models (`blog/models.go`)](#models-blogmodelsgo)
  - [`BlogPage` and Proxy `PageNode`](#blogpage-and-proxy-pagenode)
  - [`BlogImage` and `OrderableMixin`](#blogimage-and-orderablemixin)
- [App Configuration, Admin Panels, and `pages.Register` (`blog/apps.go`)](#app-configuration-admin-panels-and-pagesregister-blogappsgo)
- [Application Entrypoint (`main.go`)](#application-entrypoint-maingo)
- [Templates and Static Files](#templates-and-static-files)

---

## Project Structure

```text
examples/blogapp/
├── go.mod
├── main.go
└── blog/
    ├── apps.go
    ├── models.go
    └── assets/
        ├── static/
        │   └── blog/
        │       └── css/
        │           └── blog.css
        └── templates/
            └── blog/
                ├── base.tmpl
                ├── page.tmpl
                └── blocks/
                    └── heading.tmpl
```

---

## Running the Example

Create the local state directory if needed, generate and apply migrations, optionally create a superuser, and run the server from the repository root:

```bash
mkdir -p .private/blogapp
go run ./examples/blogapp makemigrations
go run ./examples/blogapp migrate
go run ./examples/blogapp createuser -s
go run ./examples/blogapp
```

On startup, `main.go` also creates a default demo admin account (`admin@localhost` / `Administrator123!`) if it does not already exist.

- **Admin interface**: Open `http://127.0.0.1:8080/admin/login/` to manage pages, upload images/documents, and edit `BlogPage` entries.
- **Published pages**: Pages created under the root site are served under `/pages/...` (configured via `pages.SetRoutePrefix("/pages")`).

See [CLI](../cli.md) and [Pages](../apps/pages/readme.md) for managing migrations and the page tree.

---

## Models (`blog/models.go`)

### `BlogPage` and Proxy `PageNode`

`BlogPage` embeds `models.Model` (`table:"blog_pages"`) and embeds `*pages.PageNode` with the `proxy:"true"` struct tag rather than inheriting from legacy `page_models` or database-specific backends:

```go
type BlogPage struct {
	models.Model `table:"blog_pages"`
	Page         *pages.PageNode `proxy:"true"`
	Image        *mediafiles.SimpleStoredObject
	Editor       *editor.EditorJSBlockData
	Content      *blocks.StreamBlockValue
	Thumbnail    *images.Image
	Document     *documents.Document
	User         users.User
}

func (b *BlogPage) ID() int64 {
	return b.Page.PageID
}

func (b *BlogPage) Reference() *pages.PageNode {
	return b.Page
}
```

Key integrations on `BlogPage`:

- **Proxy node fields**: `FieldDefs(ctx)` initializes `n.Page` when `nil`, binds `PageID` (`Primary: true, Column: "id"`), and exposes embedded fields (`Title`, `UrlPath`, `Slug`, `CreatedAt`, `UpdatedAt`) using `IsEmbedded: true`.
- **Rich text (`editor.NewField`)**: `Editor` is defined using `editor.NewField(n, "Editor", editor.FieldConfig{...})`. See [Editor](../apps/editor/editor.md).
- **StreamBlock (`blocks.StreamBlockValue`)**: `Content` is paired with `GetContentBlock() *blocks.StreamBlock`, combining a custom-templated `heading` `StructBlock` (`blog/blocks/heading.tmpl`) and an EditorJS-backed `text` block (`blocks.FormFieldBlock(editor.EditorJSField(nil), &editor.EditorJSBlockData{}, ...)`). See [Blocks](../apps/blocks.md).
- **Admin Chooser widgets**: `Thumbnail`, `Document`, and `User` (`attrs.RelatedDeferred(attrs.RelManyToOne, users.MODEL_KEY, "", nil)`) use `github.com/Nigel2392/go-django/contrib/admin/chooser`'s `chooser.NewChooserWidget(fc.RelForeignKey.Model(), fc.WidgetAttrs)`.
- **Search integration**: Implements `search.SearchableModel` via `SearchableFields()`:

  ```go
  func (n *BlogPage) SearchableFields() []search.SearchField {
  	return []search.SearchField{
  		search.NewSearchField(4, "Editor", expr.LOOKUP_ICONTANS),
  	}
  }
  ```

  See [Search](../apps/search.md) and [Model Page](../apps/pages/model_page.md).
- **HTTP rendering**: `ServeHTTP(w, r)` wraps the request context in `BlogContext`, sets `"blog_page"`, and renders `"blog/page.tmpl"` via `tpl.FRender`.

### `BlogImage` and `OrderableMixin`

`blog/models.go` also defines a reusable `OrderableMixin[T attrs.Definer]` (`attrs.Embedded`) that injects an `Ordering` column, and applies it to `BlogImage`:

```go
type BlogImage struct {
	models.Model `table:"blog_images"`
	OrderableMixin[*BlogImage]
	ID       int64
	Image    *images.Image
	BlogPage *BlogPage
	Content  *blocks.ListBlockValue
}

func (b *BlogImage) UniqueTogether() [][]string {
	return [][]string{
		{"BlogPage", "Image"},
	}
}
```

`BlogImage.GetContentBlock()` returns a `*blocks.ListBlock` (with `Min: 1`, `Max: 9`) whose items are `StructBlock`s containing `Caption`, `Attribution`, and a nested `StreamBlock` (`"Stream"`) with `paragraph` and `Data` blocks.

---

## App Configuration, Admin Panels, and `pages.Register` (`blog/apps.go`)

`blog.NewAppConfig()` registers both `&BlogPage{}` and `&BlogImage{}` in `ModelObjects`:

- In `Init`: mounts embedded static assets (`blog/` CSS/JS/images), embedded templates (`AppName: "blog"`, base `"blog/base.tmpl"`), and registers `blog` models in `admin.RegisterApp` (`RegisterToAdminMenu: false` so pages are managed primarily through the `pages` tree).
- In `Ready`: calls `pages.Register(&pages.PageDefinition{...})` to configure:
  - `Prefetch`: `SelectRelated` (`"User"`, `"Thumbnail"`, `"Document"`) and `PrefetchRelated` (`"BlogImageSet"`).
  - `ContentTypeDefinition`: label, description, `&BlogPage{}`, and historical alias `"github.com/Nigel2392/go-django/example/blogapp/core.BlogPage"`.
  - `AddPanels`: `TitlePanel` (slug auto-synced via `.WithOutputFields("Slug")`), `RowPanel`, `LabeledRowPanel` with `AlertPanel` and `PanelGroup`, `FieldPanel`s, and `&admin.ModelFormPanel[*BlogImage, modelforms.ModelForm[*BlogImage]]` for editing inline `BlogImageSet` items (`MinNum: 3`, `MaxNum: 9`).
  - `EditPanels`: organizes editing into `TabbedPanel` / `PanelTab` tabs (`"Content"` and `"Media"` with nested chooser tabs) plus read-only `CreatedAt` and `UpdatedAt` panels.
  - `ParentPageTypes`: restricts valid parent page types (`"github.com/Nigel2392/go-django/examples/blogapp/blog.BlogPage"`).

See [Page Content Types](../apps/pages/contenttypes.md) and [Admin Models](../apps/admin/models.md).

---

## Application Entrypoint (`main.go`)

`examples/blogapp/main.go` ties together the database, media storage, search backend, page route prefix, and all required contrib apps:

- Blank-imports editor features (`contrib/editor/features/...`) and the ORM search backend `_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"`.
- Registers a filesystem media backend (`mediafs.NewBackend("./.private/blogapp/media", 0)`) as the `"filesystem"` default and passes it to `images.NewAppConfig` and `documents.NewAppConfig`.
- Installs **`contrib/pages/settings`** (`"github.com/Nigel2392/go-django/contrib/pages/settings"`), which `pages` requires for its `"settings"` app dependency and `Site` configuration (do **not** install `contrib/settings` alongside `pages`).
- Installs `session`, `auth`, `admin`, `messages`, `pages`, `pages/settings`, `revisions`, `auditlogs`, `reports`, `editor`, `blocks`, `blog`, `images`, `documents`, `migrator`, and `translations`.
- Calls `pages.SetRoutePrefix("/pages")` so all published blog pages are routed under `/pages`.
- Demonstrates querying pages with `pages.NewPageQuerySet().WithContext(ctx).Select("*").Specific().Search("test").Types(&blog.BlogPage{}).All()`.

See [Pages Queries](../apps/pages/queries.md), [Images](../apps/images.md), and [Documents](../apps/documents.md).

---

## Templates and Static Files

`blog/assets/templates/blog/page.tmpl` renders the page title, thumbnail URL, author email, and block stream:

```html
{{ define "title" }}{{ .Page.Page.Title }}{{ end }}

{{ define "extra_css" }}
    <link rel="stylesheet" href="{{ static "blog/css/blog.css" }}">
{{ end }}

{{ define "content" }}
    <div class="blog-wrapper">
        <h1>{{ .Data.blog_page.Page.Title }}</h1>
        <h2>Hello, {{ user.Username }}!</h2>

        {{ if .Data.blog_page.Thumbnail }}
        <img src="{{ url "images:serve" .Data.blog_page.Thumbnail.Path }}" alt="{{ .Data.blog_page.Thumbnail.Title }}">
        {{ end }}

        {{ if and .Data.blog_page.User .Data.blog_page.User.Email }}
        <p>By {{ .Data.blog_page.User.Email | toString }}</p>
        {{ end }}

        {{ render_block . .Data.blog_page.Content }}

        {{ $ctx := . }}
        {{ range $value := .Data.blog_page.Content.V }}
            {{ render_block $ctx $value.Data }}
        {{ end }}
    </div>
{{ end }}
```

`blog/assets/templates/blog/blocks/heading.tmpl` renders the `heading` `StructBlock` using `block_value`:

```html
{{ define "heading" }}
    <h1>{{ block_value . "title" }}</h1>
    <h2>{{ block_value . "subtitle" }}</h2>
{{ end }}
```

See [Rendering](../rendering.md) and [Static Files](../staticfiles.md) for template inheritance and static asset serving.
