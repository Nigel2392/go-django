# Registering Apps in Admin

Every Django application that exposes models, custom admin views, or dashboard extensions registers an `AppDefinition` with the admin site using `admin.RegisterApp`.

For global site setup and model-level configuration, see:

- [Admin Application (`admin`)](./app.md)
- [Admin Models, Forms, Panels, and Columns](./models.md)

---

## `admin.RegisterApp` and Naming Rules

Register your app during its `Init` callback (before `AdminSite.Ready` runs):

```go
func (a *AdminApplication) RegisterApp(
	name string,
	appOptions AppOptions,
	opts ...ModelOptions,
) *AppDefinition
```

`RegisterApp` panics if `admin.IsReady()` is already `true`. You can register models inline via `opts...` or later (before `Ready`) via `appDef.Register(ModelOptions{...})`.

### App Name Validation and Checks

During `AdminApplication.Check`, the framework verifies every registered `AppDefinition` against two critical checks:

- **`admin.app_name_invalid`**: `app.Name` must match `^[a-zA-Z0-9_]+$` (letters, digits, and underscores only).
- **`admin.app_not_registered`**: `app.Name` must correspond to a registered app name in `django.Global.Apps`.

Matching `name` to your `apps.AppConfig` name also ensures `admin.AppMiddleware` resolves the active app context accurately on `/admin/apps/<app_name>/...` requests.

---

## `AppOptions` Reference

```go
type AppOptions struct {
	RegisterToAdminMenu any
	FullAdminMenu       bool
	EnableIndexView     bool
	AppLabel            func(ctx context.Context) string
	AppDescription      func(ctx context.Context) string
	MenuLabel           func(ctx context.Context) string
	MenuOrder           int
	MenuIcon            func() string
	MediaFn             func() media.Media
	IndexView           func(adminSite *AdminApplication, app *AppDefinition) views.View
}
```

- **`RegisterToAdminMenu`**: Controls where the app appears in the admin sidebar:
  - `bool`: `true` registers the app in the top-level sidebar (`admin.RegisterMenuItemHook`); `false` or `nil` omits it from the sidebar.
  - `string`: Registers the app's submenu under a custom hook name—most commonly another app's menu hook (`"admin:register_menu_item:<parent_app>"`) so your app nests inside that parent app's submenu.
  - Any other non-`nil` type fails with `assert.Fail`.
- **`FullAdminMenu`**: By default (`false`), if an app ends up with only a single visible submenu item (for example, one registered model and `EnableIndexView: false`), `AppDefinition.OnReady` promotes that single item directly to the parent menu rather than rendering a 1-item submenu. Set `FullAdminMenu: true` to always show the collapsible app submenu wrapper.
- **`EnableIndexView`**: Enables `/admin/apps/<app_name>/` (`admin:apps`) and prepends an index link to the app's submenu. When `false`, requests to `/admin/apps/<app_name>/` return `404 Not Found`.
- **`AppLabel`** / **`AppDescription`**: Human-readable title and subtitle returned by `app.Label(ctx)` and `app.Description(ctx)` (displayed on the app index page). `AppLabel` falls back to `app.Name`.
- **`MenuLabel`** / **`MenuOrder`** / **`MenuIcon`**: Sidebar label (defaults to `app.Name`), ascending sort position, and raw HTML icon string (for example `<svg>...</svg>`).
- **`MediaFn`**: Optional callback merged into `RegisterGlobalMediaHook` so the app's CSS/JS assets load on every admin page.
- **`IndexView`**: Optional custom `views.View` builder used instead of the default `"admin/views/apps/index.tmpl"` view when `EnableIndexView` is `true`.

---

## `AppDefinition` and Extension Routing

```go
type AppDefinition struct {
	Name    string
	Options AppOptions
	Models  *orderedmap.OrderedMap[string, *ModelDefinition]
	Routing func(mux.Multiplexer)
	// ...
}

func (a *AppDefinition) Register(opts ModelOptions) *ModelDefinition
func (a *AppDefinition) Label(ctx context.Context) string
func (a *AppDefinition) Description(ctx context.Context) string
func (a *AppDefinition) OnReady(adminSite *AdminApplication)
```

### Mounting Extension Routes under `/admin/ext/`

Set `appDef.Routing` to mount custom admin endpoints under the `/admin/ext/` (`"admin:ext"`) route group. Routes added inside `appDef.Routing` automatically inherit both `admin.AppMiddleware` and `admin.RequiredMiddleware`:

```go
package reports

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux/v2"
)

func NewAppConfig() django.AppConfig {
	app := apps.NewAppConfig("reports")

	app.Init = func(settings django.Settings) error {
		adminApp := admin.RegisterApp(
			"reports",
			admin.AppOptions{
				RegisterToAdminMenu: true,
				EnableIndexView:     true,
				FullAdminMenu:       true,
				AppLabel:            trans.S("Reports"),
				AppDescription:      trans.S("Analytics and operational exports."),
				MenuLabel:           trans.S("Reports"),
				MenuOrder:           50,
			},
			admin.ModelOptions{
				Name:                "report_entry",
				Model:               &ReportEntry{},
				RegisterToAdminMenu: true,
			},
		)

		// Mounted at /admin/ext/reports/summary/ -> reverse name "admin:ext:reports:summary"
		adminApp.Routing = func(ext mux.Multiplexer) {
			g := ext.Any("/reports", nil, "reports")
			g.Get("/summary/", http.HandlerFunc(summaryHandler), "summary")
		}

		// Add custom submenu link alongside registered models
		admin.RegisterAppMenuItem("reports", func(r *http.Request, site *admin.AdminApplication, def *admin.AppDefinition) []menu.MenuItem {
			return []menu.MenuItem{
				&menu.Item{
					BaseItem: menu.BaseItem{
						ItemName: "reports_summary",
						Label:    trans.T(r.Context(), "Summary"),
						Ordering: 10,
					},
					Link: func() string {
						return django.Reverse("admin:ext:reports:summary")
					},
				},
			}
		})

		return nil
	}

	return app
}
```

### App Index Components and Menu Hooks

- **`admin.RegisterAdminAppPageComponent(appName, fn)`**: Registers a `func(r *http.Request, adminSite *AdminApplication, app *AppDefinition) AdminPageComponent` on `RegisterAdminPageComponentHook(appName)` (`"admin:<app_name>:register_component"`). Returned components are sorted by `Ordering()` and rendered on `/admin/apps/<app_name>/` when `EnableIndexView` is `true` and `IndexView` is `nil`.
- **`admin.RegisterAppMenuItem(appName, fn)`**: Appends extra `menu.MenuItem` entries to the named app's submenu (`"admin:register_menu_item:<app_name>"`). Note that the target app must have `RegisterToAdminMenu` set to `true` or a parent hook string; otherwise `AppDefinition.OnReady` returns early and never reads the app's submenu hooks.

### Nesting One App Inside Another

To place an app's menu items inside another app's submenu (for example, placing `translations` or `reports` tools under `settings`), pass the parent app's submenu hook string to `RegisterToAdminMenu`:

```go
admin.RegisterApp(
	"audit_reports",
	admin.AppOptions{
		RegisterToAdminMenu: "admin:register_menu_item:reports",
		MenuLabel:           trans.S("Audit Reports"),
	},
	admin.ModelOptions{
		Name:                "audit_log",
		Model:               &AuditLog{},
		RegisterToAdminMenu: true,
	},
)
```

---

## Permissions (`admin:view_app:<app_name>`)

When accessing `/admin/apps/<app_name>/` (`AppHandler`), admin enforces:

```go
permissions.HasPermission(r, fmt.Sprintf("admin:view_app:%s", app.Name))
```

If the user lacks that permission (and `IsAdmin()` is `false`), `AppHandler` redirects to `admin.ReLogin`.

The same `"admin:view_app:<app_name>"` permission also controls `Hidden` on the app's `menu.SubmenuItem` in the sidebar—except when a single-item app collapses to a promoted `menu.Item` (`len(items) == 1 && !FullAdminMenu`), in which case the promoted item's visibility is determined by that item's own `Hidden` field (such as `"admin:view_list"` for a model).
