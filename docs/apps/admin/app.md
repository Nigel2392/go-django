# Admin Application (`admin`)

The `admin` app provides a customizable web administration site for managing registered Django apps, database models, search views, modal object choosers, revision comparisons, dashboard widgets, and sidebar navigation.

For registering apps and configuring model CRUD views, forms, panels, and list columns, see the companion pages:

- [Registering Apps in Admin](./apps.md)
- [Admin Models, Forms, Panels, and Columns](./models.md)

---

## Setup and Requirements

Create the admin application with `admin.NewAppConfig()` and register it in `django.Apps(...)`. The `admin` app interacts through the global `admin.AdminSite` singleton (`*AdminApplication`) and declares `Deps = []string{"messages"}` (see [Messages](../messages.md)):

```go
const BASE_KEY = "admin"

var ADMIN_BASE_URL = "admin/"

var AdminSite *AdminApplication
```

```go
type AdminApplication struct {
	*apps.AppConfig
	Ordering []string
	Route    *mux.Route
	Apps     *orderedmap.OrderedMap[string, *AppDefinition]
	// ...
}

func (a *AdminApplication) IsReady() bool
func (a *AdminApplication) AuthLogout(r *http.Request) error
func (a *AdminApplication) AuthLoginForm(r *http.Request, formOpts ...func(forms.Form)) LoginForm
func (a *AdminApplication) AuthLoginHandler() func(w http.ResponseWriter, r *http.Request)
func (a *AdminApplication) RegisterApp(name string, appOptions AppOptions, opts ...ModelOptions) *AppDefinition
func (a *AdminApplication) Check(ctx context.Context, settings django.Settings) []checks.Message
func (a *AdminApplication) SearchableModels(r *http.Request) []*searches.SearchOptions
```

Package-level aliases forward directly to `AdminSite`: `admin.RegisterApp`, `admin.IsReady`, and `admin.ConfigureAuth`.

### Mandatory `admin.ConfigureAuth(...)`

Before `AdminSite.Ready` runs, authentication **must** be configured via `admin.ConfigureAuth(admin.AuthConfig{...})`. If `auth` is not set, if both `GetLoginForm` and `GetLoginHandler` are `nil`, or if `Logout` is `nil`, `AdminSite.Ready` triggers a fatal assertion panic.

When using the built-in [Auth](../auth/app.md) app, `auth.Init` calls `admin.ConfigureAuth` automatically. If you provide a custom authentication app or login handler, call `admin.ConfigureAuth` in your app's `Init`:

```go
package main

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/session"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/forms"
)

func main() {
	// Only required manually if not using contrib/auth (auth calls this in Init).
	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	app := django.App(
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			messages.NewAppConfig,
			admin.NewAppConfig,
		),
	)
	_ = app.Initialize()
}
```

---

## Home Page Settings

Three settings in `contrib/admin/settings.go` customize the admin home page (`/admin/`):

```go
const (
	APPVAR_HOME_PAGE_TITLE     = "APPVAR_HOME_PAGE_TITLE"
	APPVAR_HOME_PAGE_SUBTITLE  = "APPVAR_HOME_PAGE_SUBTITLE"
	APPVAR_HOME_PAGE_LOGO_PATH = "APPVAR_HOME_PAGE_LOGO"
)
```

Do note the exact string key for `APPVAR_HOME_PAGE_LOGO_PATH` is `"APPVAR_HOME_PAGE_LOGO"` (without `_PATH`).

- **`APPVAR_HOME_PAGE_TITLE`**: Accepts `string`, `func(context.Context) string`, or `func(*http.Request) string`. Defaults to `trans.S("Home")`.
- **`APPVAR_HOME_PAGE_SUBTITLE`**: Accepts `string`, `func(context.Context) string`, or `func(*http.Request) string`. Defaults to `"Welcome to the Go-Django admin dashboard, <user>!"`.
- **`APPVAR_HOME_PAGE_LOGO_PATH`**: Accepts `string`, `func() string`, or `func(*http.Request) string`. Defaults to `django.Static("admin/images/home_logo.png")`.

---

## Authentication, `AuthConfig`, and Auth Errors

`AuthConfig` and `LoginForm` define how the admin site authenticates and logs out users:

```go
type AuthConfig struct {
	GetLoginForm    func(r *http.Request, formOpts ...func(forms.Form)) LoginForm
	GetLoginHandler func(w http.ResponseWriter, r *http.Request)
	Logout          func(r *http.Request) error
}

type LoginForm interface {
	forms.Form
	SetRequest(req *http.Request)
	Login() error
}
```

- When `GetLoginHandler` is non-nil, it takes precedence over `GetLoginForm` on `admin:login`.
- When `GetLoginHandler` is `nil`, `admin.LoginHandler` renders `"admin/views/auth/login.tmpl"`, creates the form using `GetLoginForm`, invokes `form.Login()`, emits `core.SIGNAL_LOGIN_FAILED` on invalid submissions, and redirects to `?next=` or `admin:home`.
- During `AdminSite.Init`, `admin` registers `autherrors.OnAuthenticationError(RedirectLoginFailedToAdmin)`. Whenever `autherrors.Fail(...)` is raised on a URL matching `AdminSite.Route`, it redirects the user to `admin:login` (preserving `?next=...`) instead of the global `auth:login` route.
- Two helper functions support access failures and redirects:
  - `admin.ReLogin(w, r, nextURL...)`: Redirects (`303 See Other`) to `admin:relogin` with optional `?next=`. Authenticated users without admin permissions see `"admin/views/auth/relogin.tmpl"`; if the user is already an admin, `reloginHandler` redirects to `next` or `admin:home`.
  - `admin.Home(w, r, errorMessage...)`: Flashes an optional `messages.Error(r, ...)` message and redirects (`303 See Other`) to the `"admin"` route.

---

## Middleware and Built-in Routes

### Middleware

Two middleware functions protect and annotate requests under `AdminSite.Route`:

- **`admin.AppMiddleware`**: Applied to all `AdminSite.Route` routes (including login/logout/relogin). Reads `app_name` from `mux.Vars(r)` (defaulting to `"admin"`), looks up the `django.AppConfig` in `django.Global.Apps`, and stores it via `django.ContextWithApp(...)`.
- **`admin.RequiredMiddleware`**: Applied to all authenticated admin routes (everything after `login/`, `logout/`, `relogin/`):
  1. Checks `user := authentication.Retrieve(req)`. If `nil` or `!user.IsAuthenticated()`, raises `autherrors.Fail(http.StatusUnauthorized, ..., req.URL.Path)`.
  2. If `user.IsAdmin()` is `true`, serves the route.
  3. Otherwise checks `permissions.HasPermission(req, "admin:access_admin")`. If `false`, logs a warning and calls `admin.ReLogin(w, req, req.URL.Path)`.

### Route Table

All built-in routes are mounted under `ADMIN_BASE_URL` (`"admin/"`) with route prefix `"admin"`:

| Reverse Name | Methods | URL Pattern (under `/admin/`) | Handler / Notes |
| :--- | :--- | :--- | :--- |
| `admin:login` | `GET`, `POST` | `login/` | `loginHandler` (`AppMiddleware` only) |
| `admin:logout` | `GET` | `logout/` | `logoutHandler` (`AppMiddleware` only) |
| `admin:relogin` | `GET` | `relogin/` | `reloginHandler` (`AppMiddleware` only) |
| `admin:home` | `GET` | `""` | `HomeHandler` (`RequiredMiddleware`) |
| `admin:search` | `GET` | `search/` | `SearchView` (`RequiredMiddleware`) |
| `admin:apps` | `ANY` | `apps/<<app_name>>/` | `AppHandler` (`EnableIndexView` required) |
| `admin:apps:model` | `ANY` | `apps/<<app_name>>/<<model_name>>/` | `ModelListHandler` |
| `admin:apps:model:add` | `ANY` | `apps/<<app_name>>/<<model_name>>/add/` | `ModelAddHandler` |
| `admin:apps:model:edit` | `ANY` | `apps/<<app_name>>/<<model_name>>/edit/<<model_id>>/` | `ModelEditHandler` |
| `admin:apps:model:delete` | `ANY` | `apps/<<app_name>>/<<model_name>>/delete/<<model_id>>/` | `ModelDeleteHandler` |
| `admin:apps:model:bulk_delete` | `ANY` | `apps/<<app_name>>/<<model_name>>/delete/` | `ModelBulkDeleteHandler` (`?pk_list=...`) |
| `admin:apps:model:chooser:list` | `ANY` | `apps/<<app_name>>/<<model_name>>/chooser/<<chooser_key>>/list/` | Registered via `chooser` |
| `admin:apps:model:chooser:create` | `ANY` | `apps/<<app_name>>/<<model_name>>/chooser/<<chooser_key>>/create/` | Registered via `chooser` |
| `admin:ext` | `ANY` | `ext/` | Mounts `AppDefinition.Routing` extension routes |

### Route Handler Wrappers

When adding custom model or instance routes (for example via `RegisterModelsRouteHook`), wrap handlers with `NewModelHandler` or `NewInstanceHandler` so `AdminApplication`, `AppDefinition`, `ModelDefinition`, and resolved model instances are injected automatically:

```go
type ModelHandlerFunc func(func(
	w http.ResponseWriter,
	req *http.Request,
	adminSite *AdminApplication,
	app *AppDefinition,
	model *ModelDefinition,
)) mux.Handler

func NewModelHandler(
	appnameVar, modelVar string,
	handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition),
	fail ...func(w http.ResponseWriter, r *http.Request, msg string),
) mux.Handler

func NewInstanceHandler(
	appnameVar, modelVar, idVar string,
	handler func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer),
) mux.Handler
```

---

## Context, `PageOptions`, and Templates

### `admin.NewContext` and `PageOptions`

Admin views render with an `*adminContext` created by `admin.NewContext(request, site, baseCtx)`. Calling `context.SetPage(PageOptions{...})` binds the page metadata used by `admin/base.tmpl` and `admin/skeleton.tmpl`:

```go
type BreadCrumb struct {
	Title string
	URL   string
}

type Action struct {
	Icon   string
	Target string
	Title  string
	URL    string
}

type PageOptions struct {
	Request       *http.Request
	TitleFn       func(ctx context.Context) string
	SubtitleFn    func(ctx context.Context) string
	MediaFn       func() media.Media
	BreadCrumbs   []BreadCrumb
	Actions       []Action
	HeaderActions []components.ShowableComponent
	SidePanels    *menu.SidePanels
}

func NewContext(request *http.Request, site *AdminApplication, context ctx.Context) *adminContext
```

Inside `adminContext`:

- `"site"` / `"Site"` maps to `*AdminApplication`.
- `"page"` / `"Page"` maps to `*PageOptions`.
- `CsrfToken()` returns `nosurf.Token(request)`.
- `PageOptions.GetBreadCrumbs()` and `PageOptions.GetActions()` append any global breadcrumbs/actions from `RegisterNavBreadCrumbHook` and `RegisterNavActionHook`.
- `PageOptions.Media()` merges `MediaFn()` with `Media()` from all configured `SidePanels`.

### Extending Admin Templates (`tpl.MergeConfig`)

Apps rendering custom admin views under `"admin/base.tmpl"` can merge their own template filesystem with `admin.AdminSite.TemplateConfig` using `tpl.MergeConfig`:

```go
tpl.Add(*tpl.MergeConfig(
	&tpl.Config{
		AppName: "reports",
		FS:      reportsTplFS,
		Matches: filesystem.MatchPrefix("reports/"),
	},
	admin.AdminSite.TemplateConfig,
))
```

Your custom templates can then begin with `{{ template "base" . }}` and inherit the admin sidebar, navigation bar, messages wrapper, icons sprite, and registered template functions.

### Template Blocks

`admin/skeleton.tmpl` and `admin/base.tmpl` define the following overridable blocks:

| Template File | Block Name | Purpose |
| :--- | :--- | :--- |
| `admin/skeleton.tmpl` | `title_root` | Wraps `<title>` (contains `title` and `site_name`) |
| `admin/skeleton.tmpl` | `title` | Page title segment |
| `admin/skeleton.tmpl` | `site_name` | Site suffix (`| Adminsite` by default) |
| `admin/skeleton.tmpl` | `extra_css` | Additional `<link>`/`<style>` tags in `<head>` |
| `admin/skeleton.tmpl` | `root` | Body container (overridden by `admin/base.tmpl`) |
| `admin/skeleton.tmpl` | `extra_js` | Additional `<script>` tags after `admin/js/index.js` and global media |
| `admin/base.tmpl` | `sidebar` | Extra sidebar content rendered below `{{ menu .Request }}` |
| `admin/base.tmpl` | `navigation` | Top navigation bar (breadcrumbs, `Actions`, and `SidePanels`) |
| `admin/base.tmpl` | `admin_messages` | Flash message container (`{{ template "messages" . }}`) |
| `admin/base.tmpl` | `content_node` | Outer `#content` wrapper containing `content_root` |
| `admin/base.tmpl` | `content_root` | Contains `header_root` and `#content__body` |
| `admin/base.tmpl` | `header_root` | `#content__header` wrapper containing `header` |
| `admin/base.tmpl` | `header` | Renders `admin.header` from `.Page.Title`, `.Page.Subtitle`, and `.Page.GetHeaderComponents` |
| `admin/base.tmpl` | `content` | Main page body inside `#content__body` |

### Template Functions and Registered Components

`AdminSite.TemplateConfig.Funcs` and `tpl.RequestFuncs` expose these helpers in admin templates:

- `string`: Alias for `attrs.ToString`.
- `icons`: Emits the hidden `<svg><defs>...</defs></svg>` sprite generated by `icons.DefsHTML()`.
- `icon <name> [attrKey attrVal ...]`: Renders an SVG `<use href="#<name>">` reference with optional attribute pairs.
- `menu .Request`: Renders the main sidebar menu populated via `RegisterMenuItemHook`.
- `footer_menu .Request`: Renders the sidebar footer menu populated via `RegisterFooterMenuItemHook`.
- `script_hook_output`: Merges and returns `media.Media` from `RegisterGlobalMediaHook`.
- `getCookie <name>`: Reads a cookie value from the current `*http.Request` (used to persist sidebar collapsed state).
- `templ <componentOrFn>`: Renders a `components.Component` (`templ.Component`) or `func(context.Context, io.Writer) error` into `template.HTML`.

Admin also registers UI components with `src/components` for use via `{{ component "..." ... }}`:

- `"admin.header"`: `Header(level int, headingText, subText string, extra ...any)` (where `extra` accepts `templ.Component` or `[]templ.Component`).
- `"admin.heading"`, `"admin.heading1"` – `"admin.heading6"`: Renders `<div class="heading"><hN class="heading__title">...</hN></div>`.
- `"admin.button"`: Maps to `components.NewButton`.
- `"admin.button.primary"`, `"admin.button.secondary"`, `"admin.button.success"`, `"admin.button.danger"`, `"admin.button.warning"`: Map to `components.ButtonPrimary` through `components.ButtonWarning`.

---

## Hooks

`contrib/admin/hooks.go` exposes [Goldcrest](../../hooks.md) hooks and typed registration helpers:

| Hook Constant | Hook String | Helper Function | Callback Signature |
| :--- | :--- | :--- | :--- |
| `RegisterMenuItemHook` | `"admin:register_menu_item"` | `RegisterGlobalMenuItem` | `func(r *http.Request, adminSite *AdminApplication, items components.Items[menu.MenuItem])` |
| — | `"admin:register_menu_item:<app_name>"` | `RegisterAppMenuItem(appName, fn)` | `func(r *http.Request, adminSite *AdminApplication, app *AppDefinition) []menu.MenuItem` |
| `RegisterFooterMenuItemHook` | `"admin:register_footer_menu_item"` | `RegisterGlobalFooterMenuItem` | `func(r *http.Request, adminSite *AdminApplication, items components.Items[menu.MenuItem])` |
| `RegisterGlobalMediaHook` | `"admin:register_global_media"` | `RegisterGlobalMedia` | `func(adminSite *AdminApplication) media.Media` |
| `RegisterNavBreadCrumbHook` | `"admin:register_breadcrumb"` | `RegisterGlobalNavBreadCrumb` | `func(r *http.Request, adminSite *AdminApplication) []BreadCrumb` |
| `RegisterNavActionHook` | `"admin:register_nav_action"` | `RegisterGlobalNavAction` | `func(r *http.Request, adminSite *AdminApplication) []Action` |
| `RegisterRouteHook` | `"admin:register_route"` | `RegisterRoute` | `func(adminSite *AdminApplication, route mux.Multiplexer)` |
| `RegisterHomePageBreadcrumbHook` | `"admin:home:register_breadcrumb"` | `RegisterHomePageBreadcrumb` | `func(*http.Request, *AdminApplication, []BreadCrumb) []BreadCrumb` |
| `RegisterHomePageActionHook` | `"admin:home:register_action"` | `RegisterHomePageAction` | `func(*http.Request, *AdminApplication, []Action) []Action` |
| `RegisterHomePageComponentHook` | `"admin:home:register_component"` | `RegisterHomePageComponent` | `func(*http.Request, *AdminApplication) AdminPageComponent` |
| `RegisterHomePageDisplayPanelHook` | `"admin:home:register_display_panel"` | *(register via `goldcrest.Register`)* | `func(*http.Request, *AdminApplication) []DisplayPanel` |
| `RegisterAdminPageComponentHook(app)` | `"admin:<app_name>:register_component"` | `RegisterAdminAppPageComponent(app, fn)` | `func(r *http.Request, adminSite *AdminApplication, app *AppDefinition) AdminPageComponent` |
| `AdminModelHookAdd` | `"admin:model:add"` | `RegisterModelAddHook` | `func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instance attrs.Definer)` |
| `AdminModelHookEdit` | `"admin:model:edit"` | `RegisterModelEditHook` | `func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instance attrs.Definer)` |
| `AdminModelHookDelete` | `"admin:model:delete"` | `RegisterModelDeleteHook` | `func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instances []attrs.Definer)` |
| `AdminModelHookRegisterRoute` | `"admin:model:register_route"` | `RegisterModelsRouteHook` | `func(adminSite *AdminApplication, route mux.Multiplexer)` |

**Note on `RegisterSearchHook`**: The constant `RegisterSearchHook` (`"admin:register_search"`) and `RegisterSearchHookFunc` are defined in `hooks.go`, but `RegisterSearchHook` is unused at runtime. To include a model in global admin search, configure `ListViewOptions.Search` on your model registration (see [Global Search](#global-search)).

---

## Home Page Components and Display Panels

Custom dashboard blocks on `/admin/` (and on app index pages via `RegisterAdminAppPageComponent`) implement `admin.AdminPageComponent`:

```go
type AdminPageComponent interface {
	HTML() template.HTML
	Ordering() int
	Media() media.Media
}
```

Components are sorted by `Ordering()` in ascending order, and `HomeHandler` wraps each home component in panic recovery when calling `HTML()`.

For summary counter cards on the home dashboard, register `DisplayPanel` items on `RegisterHomePageDisplayPanelHook`:

```go
type DisplayPanel struct {
	IconName string
	Title    func(ctx context.Context, count int64) string
	IsShown  func(r *http.Request) bool
	QuerySet func(r *http.Request) *queries.QuerySet[attrs.Definer]
	URL      func(r *http.Request) string
}
```

```go
goldcrest.Register(
	admin.RegisterHomePageDisplayPanelHook, 0,
	admin.RegisterHomePageDisplayPanelHookFunc(func(r *http.Request, a *admin.AdminApplication) []admin.DisplayPanel {
		return []admin.DisplayPanel{
			{
				IconName: "icon-file-document",
				Title: func(ctx context.Context, count int64) string {
					return trans.P(ctx, "Article", "Articles", int(count))
				},
				QuerySet: func(r *http.Request) *queries.QuerySet[attrs.Definer] {
					return queries.GetQuerySet[attrs.Definer](&blog.Article{})
				},
				URL: func(r *http.Request) string {
					return django.Reverse("admin:apps:model", "blog", "article")
				},
			},
		}
	}),
)
```

---

## Menus, Side Panels, and UI Components

### Menus and Side Panels (`contrib/admin/components/menu`)

The `github.com/Nigel2392/go-django/contrib/admin/components/menu` package defines sidebar menus, dropdown actions, and slide-out side panels:

- **`MenuItem` interface**: `Order() int`, `IsShown() bool`, `Component() templ.Component`, `Name() string`.
- **`BaseItem`**: `ID`, `ItemName`, `Hidden`, `Classes`, `Ordering`, `Logo templ.Component`, `Label`. If `Logo` is `nil`, renders a circular badge with the first letter of `Label`/`ItemName`/`ID`.
- **`Item`**: Embeds `BaseItem` and adds `Link func() string`, `LinkID`, `LinkClasses`.
- **`SubmenuItem`**: Embeds `BaseItem` and renders a flyout submenu via `Menu *Menu` (hidden automatically if no child items are shown).
- **`DropdownItem`**: Type over `SubmenuItem` rendered as an accordion toggle inside the sidebar (used for the built-in `"Account Details"` footer menu).
- **`Menu`**: `ID`, `Classes`, `ItemClasses`, `Items []MenuItem` with `MenuItems()` (filters hidden items and stable-sorts by `Order()`) and `HTML() string`.
- **`Action` / `BaseAction` / `Actions`**: Dropdown row/header action menus (`Show`, `Text`, `URL`, `ClassName`).
- **`SidePanel` / `SidePanels` / `BaseSidePanel`**: Slide-out panels in the top navigation bar. `BaseSidePanel` renders any template (`TemplateName`) with `"panel"` in context.
- **`admin.SidePanelFilters(request, filters, pageObject, hidden...)`**: Built-in `SidePanel` (`ID: "filters"`, `Ordering: 100`, `PanelIcon: "icon-funnel"`) used by `ModelListHandler` to render [Filters](../filters.md).

### UI Components (`contrib/admin/components`)

The `github.com/Nigel2392/go-django/contrib/admin/components` package provides buttons, links, headings, headers, and ordered component lists:

- **`ClassType`**: Bitmask flags `ClassTypePrimary`, `ClassTypeSecondary`, `ClassTypeSuccess`, `ClassTypeInfo`, `ClassTypeWarning`, `ClassTypeDanger`, `ClassTypeHollow`.
- **`Button(ButtonConfig)` / `Link(ButtonConfig, href func() string)`**: Styled `<button>` or `<a>` components.
- **`NewButton` and `ButtonPrimary` / `ButtonSecondary` / `ButtonSuccess` / `ButtonDanger` / `ButtonWarning`**: Convenience constructors registered globally under `admin.button*`.
- **`Heading` / `Heading1`–`Heading6` / `Header`**: Section headings and page headers registered under `admin.heading*` and `admin.header`.
- **`ShowableComponent` / `NewShowableComponent(req, isShown, component)`**: Wraps a `templ.Component` with a request-scoped `IsShown() bool` guard (used for `PageOptions.HeaderActions`).
- **`Item` / `Items[T]` / `NewItems[T]()`**: Ordered, deduplicated-by-name item collection sorted by `Order()` (used in menu hooks).

---

## Icons (`contrib/admin/icons`)

The `github.com/Nigel2392/go-django/contrib/admin/icons` package manages the SVG symbol sprite rendered into admin pages.

```go
func Register(fs fs.FS, path ...string) error
func Icon(name string, attrs map[string]string) components.Component
func DefsHTML() template.HTML
func Component(name string) func(c context.Context, w io.Writer) error
func Icons() []IconInterface
func Load(name string) template.HTML
```

### Strict `id="icon-<name>"` Rule

When calling `icons.Register(fs, "path/to/my-icon.svg")`, the SVG markup **must** contain an `id` attribute matching `"icon-<filename-stem>"` (pattern `id=["']icon-([a-z0-9-]+)["']`):

- File `my-example.svg` must contain `id="icon-my-example"`.
- Passing mismatched IDs or missing `id` attributes returns an error from `icons.Register`.
- Register custom icons before `AdminSite.Ready` runs (for example in your app's `Init` function), because `AdminSite.Ready` calls `icons.DefsHTML()` once and caches the sprite returned by `{{ icons }}`.
- Reference icons in templates with `{{ icon "icon-my-example" }}` or in Go/templ via `icons.Icon("icon-my-example", map[string]string{"class": "..."})`.

### Built-in Icons

`admin` registers 25 built-in icons from `admin/icons/*.svg`:

`icon-view`, `icon-funnel`, `icon-history`, `icon-download`, `icon-image`, `icon-pulse`, `icon-plus`, `icon-trash`, `icon-link`, `icon-arrow-right`, `icon-arrow-left`, `icon-arrow-up`, `icon-arrow-down`, `icon-grip-horizontal`, `icon-file-earmark`, `icon-file-document`, `icon-no-view`, `icon-graph-up`, `icon-save`, `icon-graph-down`, `icon-cash`, `icon-calendar`, `icon-cake`, `icon-list`, `icon-toggles`.

---

## Global Search

The `github.com/Nigel2392/go-django/contrib/admin/searches` package powers both the sidebar global search view (`/admin/search/`) and model list search options (see also [Search](../search.md)):

```go
type SearchField struct {
	Name            string
	Lookup          string // defaults to expr.LOOKUP_EXACT
	BuildExpression func(sf SearchField, value any) expr.Expression
}

func NewSearchField(nameWithlookup string) SearchField

type Prefetch struct {
	SelectRelated   []string
	PrefetchRelated []any
}

type SearchOptions struct {
	Order          int
	PerPage        int
	Fields         []SearchField
	ListFields     []string
	Model          attrs.Definer
	Prefetch       Prefetch
	PluralLabel    func(ctx context.Context) string
	GetEditLink    func(req *http.Request, id any) string
	Searchable     func(req *http.Request) bool
	AllowEdit      func(req *http.Request) bool
	QuerySet       func(req *http.Request) *queries.QuerySet[attrs.Definer]
	SearchQuerySet func(req *http.Request, qs *queries.QuerySet[attrs.Definer], query string) *queries.QuerySet[attrs.Definer]
	GetList        func(b *BoundSearchView, list []attrs.Definer, columns []list.ListColumn[attrs.Definer]) (list.StringRenderer, error)
	GetColumn      func(ctx context.Context, opts *SearchOptions, field string) list.ListColumn[attrs.Definer]
	// ...
}
```

Any model registered with `ListViewOptions.Search != nil` (where `CanSearch(r)` is `true`) automatically appears in the sidebar search bar and `/admin/search/`. `AdminApplication.SearchableModels` fills unset `Model`, `PerPage`, `ListFields`, `Order`, `GetEditLink`, `PluralLabel`, `QuerySet`, `GetList`, `GetColumn`, `Prefetch`, and `AllowEdit` values from the model's `ModelDefinition`.

`/admin/search/` accepts these query parameters:

- `global-search`: Search string matched via `expr.Or(...)` across `SearchOptions.Fields` (and passed to `SearchQuerySet`).
- `content_type`: Short content-type name selecting which searchable model tab to display (defaults to the first searchable model sorted by `Order`).
- `page`: 1-based pagination index.

---

## Modal Chooser

The `github.com/Nigel2392/go-django/contrib/admin/chooser` package provides modal pickers and optional inline creation forms for foreign keys and editor tools (see [Chooser](../chooser.md) for widget usage details):

```go
const DEFAULT_KEY = "default"

func Register(chsr chooser, key ...string)
func NewChooserWidget(model attrs.Definer, widgetAttrs map[string]string, chooserKey ...string) *ChooserWidget
```

Define a chooser with `chooser.ChooserDefinition[T]`:

```go
type ChooserDefinition[T attrs.Definer] struct {
	ChooserKey    string
	Model         T
	Title         any // string or func(ctx context.Context) string
	Labels        map[string]func(ctx context.Context) string
	PreviewString func(ctx context.Context, instance T) string
	ExtraData     func(ctx context.Context, instance T) map[string]any

	ListPage   *ChooserListPage[T]
	CreatePage *ChooserFormPage[T]

	DjangoApp  django.AppConfig
	AdminApp   *admin.AppDefinition
	AdminModel *admin.ModelDefinition
	MediaFn    func() media.Media
}
```

- **`ChooserListPage[T]`**: Configures `Fields`, `Labels`, `PerPage` (default `20`), `Columns`, `Format`, `QuerySet`, `Filter`, `SearchFields`, `SearchQueryVar` (default `"search"`), `NewList`, and `BoundView`. Checks `"admin:view_list"` object permission.
- **`ChooserFormPage[T]`**: When non-nil, enables an inline creation tab inside the chooser modal (`CanCreate() == true`) using `Options admin.FormViewOptions` (falling back to the model's `AddView`). Checks `"admin:add"` object permission and fires `AdminModelHookAdd` on save.
- **`NewChooserWidget(model, widgetAttrs, chooserKey...)`**: Builds a form widget bound to the registered chooser (defaulting to `chooser.DEFAULT_KEY`). During `core.OnDjangoReady`, each registered chooser resolves its `DjangoApp`, `AdminApp`, and `AdminModel` (so the model must be registered in both a Django app and `admin`).

---

## Revision Comparison

The `github.com/Nigel2392/go-django/contrib/admin/compare` package provides field- and panel-level diffing used by [Revisions](../revisions.md) and [Editor](../editor/editor.md):

```go
type Comparison interface {
	Label() string
	HTMLDiff() (template.HTML, error)
	HasChanged() (bool, error)
}

type ComparisonWrapper interface {
	Comparison
	Unwrap() []Comparison
}

type ComparisonFactory func(
	ctx context.Context,
	label func(context.Context) string,
	fieldName string,
	modelMeta attrs.ModelMeta,
	old, new attrs.Definer,
) (Comparison, error)

var ErrSkip error = errors.New(errors.CodeNotImplemented, "skip step")

func RegisterComparisonType(typ reflect.Type, factory ComparisonFactory)
func RegisterComparisonKind(kind reflect.Kind, factory ComparisonFactory)
func RegisterDefaultComparison(factory ComparisonFactory) (replaced bool)
func GetComparison(ctx context.Context, typ any, label any, fieldname string, oldInstance, newInstance attrs.Definer) (Comparison, error)
func DiffText(a, b string) *TextDiff
```

- `GetComparison` resolves factories in priority order: exact `reflect.Type` registry -> `reflect.Kind` registry -> default factory (`FieldComparison`).
- `DiffText(a, b)` tokenizes alphanumeric runs and punctuation/whitespace, computes an LCS diff, merges adjacent runs into `DIFF_EQUALS`, `DIFF_ADDED`, and `DIFF_REMOVED` segments, and renders `<span class="diff-added">` / `<span class="diff-removed">` HTML via `TextDiff.HTML()`.
- `admin.PanelComparison(ctx, panels, oldInstance, newInstance, mustWrap...)` recursively traverses a slice of admin panels and returns a combined `compare.MultipleComparison` (rendering `<dl><dt>label</dt><dd>diff</dd></dl>`).

---

## Admin Permission Strings

When `permissions.Tester` is enabled (see [Permissions Backend and RBAC](../auth/permissions.md)), admin checks the following permission strings:

| Permission String | Check Type | Checked Where |
| :--- | :--- | :--- |
| `admin:access_admin` | `HasPermission` | `RequiredMiddleware` (if `!user.IsAdmin()`), `loginHandler` |
| `admin:view_app:<app_name>` | `HasPermission` | `AppHandler` (app index view), app sidebar submenu visibility |
| `admin:view_list` | `HasObjectPermission` | `ModelListHandler`, model sidebar item visibility, `BoundChooserListPage` |
| `admin:add` | `HasObjectPermission` | `ModelAddHandler`, list header "Add" button, `BoundChooserFormPage` |
| `admin:edit` | `HasObjectPermission` / `HasPermission` | `ModelEditHandler`, inline list edit + "Change" button + `TitleFieldColumn` link, footer "Edit Account", `searches.BoundSearchView` edit link |
| `admin:delete` | `HasObjectPermission` / `HasPermission` | `AdminDeleteView.Setup` (single and bulk delete), `BulkActionDelete` |
| `admin:export` | `HasObjectPermission` | `ModelListHandler` (`ListExportMixin` and Export nav action) |
| `admin:bulk_actions` | `HasPermission` | `ModelListHandler` (enables row-select checkboxes and bulk action execution) |
