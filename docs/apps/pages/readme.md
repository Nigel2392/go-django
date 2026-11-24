# Pages Application

The `pages` app (`github.com/Nigel2392/go-django/contrib/pages`) provides a hierarchical CMS page tree built on materialized paths. Each page is backed by a `PageNode` record storing tree structure, slugs, URL paths, and publication flags, paired with custom Go page types registered via `PageDefinition`.

## Table of Contents

1. [Installation and Dependencies](#installation-and-dependencies)
2. [Settings, Checks, and CLI Commands](#settings-checks-and-cli-commands)
3. [Routing, URLs, and Template Helpers](#routing-urls-and-template-helpers)
4. [Sites and URL Validation](#sites-and-url-validation)
5. [Serving Pages and View Interfaces](#serving-pages-and-view-interfaces)
6. [Revisions, Previews, and Integrations](#revisions-previews-and-integrations)
7. [Queries and Tree Operations](./queries.md)
8. [Page Definitions and Registration](./contenttypes.md)
9. [PageNode Model and Status Flags](./model_page.md)
10. [Page Signals](./signals.md)
11. [Example Blog App](../../examples/blog.md)

---

## Installation and Dependencies

The Pages app exposes `PageAppConfig`, `NewAppConfig()`, and `App()`:

```go
type PageAppConfig struct {
	*apps.DBRequiredAppConfig
	routePrefix string
}

func NewAppConfig() django.AppConfig
func App() *PageAppConfig
```

`pages.NewAppConfig()` registers `Deps: []string{"admin", "settings"}` and embeds migrations for `&pages.Site{}` and `&pages.PageNode{}` via `*migrator.MigratorAppConfig` (if `migrator` is not installed, `Init` auto-creates the `PageNode` table only).

### Dependency and Prefix Requirements

Before calling `app.Initialize()` (or `app.Serve()`), note three strict setup rules:

- **`admin` is required in practice**: Even if dependency checks are disabled with `django.FlagSkipDepsCheck`, `pages` checks `django.AppInstalled("admin")` during app registration, `Init`, and `Ready`. When `"admin"` is absent, `pages` skips registering all templates, static assets, admin routes, and public page-serving routes.
- **`contrib/pages/settings` circular dependency and `contrib/settings` conflict**: `pages` depends on `"settings"` (`contrib/pages/settings`), and `contrib/pages/settings` depends on `"admin"` and `"pages"`. This circular dependency works because Go-Django registers all apps before checking dependencies. However, both `github.com/Nigel2392/go-django/contrib/pages/settings` and `github.com/Nigel2392/go-django/contrib/settings` ([Settings](../settings.md)) register under the app name `"settings"`, so they **cannot** be registered together in the same `django.Apps(...)` call (`App settings already registered`).
- **`pages.SetRoutePrefix(...)` is mandatory**: You **must** call `pages.SetRoutePrefix(prefix)` before `app.Initialize()`, even when serving pages at `"/"`. Leaving the prefix unset causes `Ready` to terminate fatally (`Route prefix was not set before calling django.App.Initialize().`).

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/pages"
	pagessettings "github.com/Nigel2392/go-django/contrib/pages/settings"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "example.db")
	if err != nil {
		panic(err)
	}

	pages.SetRoutePrefix("/")

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DATABASE:      db,
		}),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			reports.NewAppConfig,
			auditlogs.NewAppConfig,
			revisions.NewAppConfig,
			pagessettings.NewAppConfig,
			pages.NewAppConfig,
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

## Settings, Checks, and CLI Commands

### App Settings

The `pages` package defines two configuration variables for the admin outdated pages report (**Reports -> Outdated Pages**, `admin:pages:outdated`):

- `pages.APPVAR_OUTDATED_AFTER_DURATION` (`time.Duration`, default `24 * time.Hour`): Age threshold after which pages appear in the outdated pages list.
- `pages.APPVAR_OUTDATED_LIST_DISPLAY` (`[]string`, default `[]string{"Title", "UrlPath", "ContentType", "Live", "UpdatedAt"}`): Columns displayed in the outdated pages report.

### System Checks (`PageAppConfig.Check`)

During startup [checks](../../checks.md), `PageAppConfig.Check` inspects every registered `PageDefinition` and verifies that `ID` and `Reference` are **not** promoted methods from an embedded `*PageNode`. If either method is promoted rather than defined directly on your concrete page struct, `Check` emits a `checks.Critical("pages.method_promoted", ...)` error.

### CLI Commands

`pages` registers the `fix-tree` management command, which calls `pages.FixTree(context.Background())` to repair `Path`, `Depth`, `Numchild`, and `UrlPath` inconsistencies in the database:

```bash
go run main.go fix-tree
```

## Routing, URLs, and Template Helpers

When `"admin"` is installed and `pages.SetRoutePrefix` has been called, `Ready` mounts:

- Public page serving under `<routePrefix>/` (`"page"`) and `<routePrefix>/*` (`"pages"` via `pageRouteResolver`), supporting `GET`, `POST`, `PUT`, and `DELETE`.
- Redirect helper `/__pages__/redirect/<<page_id>>` (`"pages_redirect"`), which looks up a `PageNode` by ID and redirects to `pages.URLPath(page)`.
- Admin page management, tree API (`/admin/pages/api/menu`), and revision/preview routes under `/admin/pages`.

### Helper Functions and Template Funcs

- `pages.SetRoutePrefix(prefix string)`: Sets the URL prefix for serving live pages (normalized with a leading `/` and no trailing `/`, except `""` -> `"/"`).
- `pages.URLPath(page Page) string`: Returns `path.Join(routePrefix, page.Reference().UrlPath)`.
- `django.Reverse("pages:page", page)` or `django.Reverse("pages:page", page.ID())`: Resolves the relative page path by walking parent slugs via `pageRouteResolver.Reverse`.
- Template functions (registered via `tpl.RequestFuncs` when `"admin"` is installed):
  - `site_for_request`: `func() (*Site, error)` returning the `Site` resolved for the active request.
  - `PageURL`: Alias for `pages.URLPath`.

## Sites and URL Validation

Each `Site` binds a domain and port to a root `*PageNode`:

```go
type Site struct {
	models.Model `table:"sites"`
	ID           int64
	Name         string
	Domain       string
	Port         int
	Default      bool
	Root         *PageNode
}
```

- `(*Site).URL() string`: Formats `Domain`, appending `:<Port>` when `Port` is neither `80` nor `443`.
- `pages.SiteForRequest(requestOrContext any, fn ...func(*queries.QuerySet[*Site]) *queries.QuerySet[*Site]) (context.Context, *Site, error)`: Resolves the `Site` matching the request's host (`hosts.GetHost`) or falling back to `Default: true`. Results are cached per request context in `django.ContextDataStore`.

### `contrib/pages/settings`

`github.com/Nigel2392/go-django/contrib/pages/settings` provides:

- Admin registration for `&pages.Site{}` under **Settings -> Sites** (and automatically creates a default `"Default Site"` at `localhost:80` during `Ready` if no `Site` rows exist).
- `SettingsMenuHook` (`"settings:register_to_menu"`) and `SettingsMenuHookFunc` (`func() []menu.MenuItem`) for extending the **Settings** admin submenu.
- `Setting` interface (`attrs.Definer`).

### `contrib/pages/validators`

`github.com/Nigel2392/go-django/contrib/pages/validators` exports:

```go
func ValidateSettingsURL(value interface{}) error
```

> [!NOTE]
> `ValidateSettingsURL` only exempts `""`, `"localhost"`, and `"127.0.0.1"`. For any other value, it calls `url.Parse` and requires `u.Host != ""` and `u.Port() == ""`. In Go's `net/url`, bare domain names such as `"example.com"` parse as a path with an empty `Host`, so they fail `ValidateSettingsURL` unless given a scheme (e.g. `"https://example.com"`) or `"//example.com"`.

## Serving Pages and View Interfaces

`pages.Serve(allowedMethods ...string) http.Handler` enforces the allowed HTTP methods and delegates to `pages.PageServeView`.

For each request, `PageServeView` resolves the active `Site`, starts routing at `site.Root` (or falls back to published root pages if `site.Root == nil`), calls `rootPage.Route(req, pathParts)` (honoring custom [`RoutablePage`](./model_page.md#tree-queries-and-routing) implementations), verifies the target node is published and neither hidden nor deleted, loads the concrete page via `pages.Specific(r.Context(), page, true)`, and renders it.

Your specific `Page` model can handle rendering in either of two ways:

1. **Implement `http.Handler` directly** on your page struct (`ServeHTTP(w http.ResponseWriter, r *http.Request)`). If implemented, `servePageView` delegates directly to `handler.ServeHTTP(w, req)`.
2. **Return a `PageView` from `PageDefinition.ServePage`**: `servePageView` populates `"Object"` (the specific `Page`), `"View"` (the `PageView`), and `"Node"` (the `*PageNode`) on the context and invokes the view interfaces below:

```go
type PageViewHandler interface {
	views.MethodsView
	views.ControlledView
	View() PageView
	CurrentPage() Page
}

type PageView interface {
	ServePage()
}

type PageContextGetter interface {
	GetContext(req *http.Request, page Page) (ctx.Context, error)
}

type PageBaseKeyGetter interface {
	GetBaseKey(req *http.Request, page Page) string
}

type PageTemplateGetter interface {
	GetTemplate(req *http.Request, page Page) string
}

type PageTemplateView interface {
	PageContextGetter
	PageTemplateGetter
}

type PageRenderView interface {
	Render(w http.ResponseWriter, req *http.Request, page Page, context ctx.Context) error
}

type PageTemplateRenderView interface {
	PageTemplateView
	Render(w http.ResponseWriter, req *http.Request, template string, page Page, context ctx.Context) error
}

type PageComponentView interface {
	PageContextGetter
	Render(req *http.Request, page Page) templ.Component
}
```

Note that `servePageView` currently dispatches `PageContextGetter`, `PageTemplateGetter`, `PageBaseKeyGetter`, `PageTemplateRenderView`, and `PageRenderView` (falling back from `PageTemplateRenderView` to `PageRenderView` if both `baseKey` and `template` are empty).

## Revisions, Previews, and Integrations

`pages` provides preview handlers and integrates with several contrib apps:

- **Previews and revisions**:
  - `pages.IsPreviewContext(ctx context.Context) bool`: Reports whether the request context was created for a page or revision preview.
  - `pages.PagePreviewHandler` and `pages.PageRevisionPreviewHandler` (instances of `*pages.GenericPreviewHandler[Page]`): Store transient `POST` form data in the user's session for 10 minutes and replay a preview request through `servePageView` with `CommitContext(ctx, false)`.
  - Integrates with [`revisions`](../revisions.md) for draft edits, revision comparison, and restoration.
- **Auditlogs**: Registers definitions for `"pages:add"`, `"pages:add_child"`, `"pages:edit"`, `"pages:move"`, `"pages:publish"`, `"pages:unpublish"`, and `"pages:delete"` (see [Auditlogs](../auditlogs.md)).
- **Chooser**: Registers choosers for `&PageNode{}`, root pages (`pages.CHOOSER_ROOT_PAGES_KEY`), and page revisions (`pages.CHOOSER_PAGE_REVISIONS_KEY`) (see [Chooser](../chooser.md)).
- **Search**: Implements `search.SearchableModel` on `PageNode` and cross-model page searching via `PageQuerySet.Search()` (see [Search](../search.md)).
