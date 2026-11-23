# Settings

The `settings` app (`github.com/Nigel2392/go-django/contrib/settings`) provides database-backed, groupable runtime settings (`RegisteredSetting`) that administrators can view and edit in the admin interface and application handlers can inspect per request.

> [!WARNING]
> **Do NOT install `contrib/settings` alongside `contrib/pages/settings` (or `contrib/pages`).**
>
> `github.com/Nigel2392/go-django/contrib/settings` (dynamic key/value settings) and `github.com/Nigel2392/go-django/contrib/pages/settings` (multi-site `pages.Site` settings required by [Pages](./pages/readme.md)) are two distinct packages that share both the app name `"settings"` and the Goldcrest hook name `SettingsMenuHook = "settings:register_to_menu"` (with incompatible callback signatures). Registering both in the same Go-Django application panics on duplicate app registration and causes menu hook type collisions.

## Table of Contents

- [Installation and Dependencies](#installation-and-dependencies)
- [Defining and Registering Settings](#defining-and-registering-settings)
  - [Core Interfaces and Types](#core-interfaces-and-types)
  - [Registration Example](#registration-example)
- [Sync Command and System Check](#sync-command-and-system-check)
- [Middleware and Reading Settings per Request](#middleware-and-reading-settings-per-request)
- [Admin Routes, Permissions, Menu Hook, and Audit Logs](#admin-routes-permissions-menu-hook-and-audit-logs)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and Dependencies

Register `settings.NewAppConfig` together with `session`, `auth`, `messages`, [`blocks`](./blocks.md) (used by the settings views to render grouped `StructBlock` forms), `admin`, and `migrator`:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/messages"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := drivers.Open(context.Background(), "sqlite3", "./app.db")
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE:                          db,
			migrator.APPVAR_MIGRATION_DIR:                   "./migrations",
			"contrib.settings.SettingsForRequestMiddleware": true, // default: true
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			messages.NewAppConfig,
			blocks.NewAppConfig,
			admin.NewAppConfig,
			auditlogs.NewAppConfig,
			settings.NewAppConfig,
			migrator.NewAppConfig,
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

`settings.NewAppConfig()` wraps a `*migrator.MigratorAppConfig` with embedded migrations (`migrations/settings`) for the `RegisteredSetting` model. Run `makemigrations` and `migrate` to create the table before starting the server.

---

## Defining and Registering Settings

### Core Interfaces and Types

```go
const (
	DEFAULT_GROUP    = "Default"
	SettingsMenuHook = "settings:register_to_menu"
)

type SettingsMenuHookFunc = func(*http.Request) []menu.MenuItem

type SettingsGroup struct {
	Icon        func(context.Context) template.HTML
	Label       func(context.Context) string
	Description func(context.Context) string
}

type Setting interface {
	DBType() dbtype.Type
	Permissions() []string
	Icon(ctx context.Context) template.HTML
	Label(ctx context.Context) string
	Description(ctx context.Context) string
	FormField(ctx context.Context) forms.Field
	DefaultValue() any
	ValueFromDB(context.Context, json.RawMessage) (any, error)
}

type BoundSetting interface {
	Icon() template.HTML
	Name() string
	Label() string
	Description() string
	Value() (any, error)
	DBType() dbtype.Type
	DefaultValue() any
	FormField() forms.Field
	Permissions() []string
}

type BoundSettingGroup struct {
	Settings []*boundSetting
	// ...
}

func (b BoundSettingGroup) Icon() template.HTML
func (b BoundSettingGroup) Label() string
func (b BoundSettingGroup) Description() string

type RegisteredSetting struct {
	ID           uint64
	InternalName string
	Data         json.RawMessage
}

func RegisterGroup(ordering int, groupName string, group SettingsGroup)
func RegisterSetting(ordering int, group, name string, setting Setting)
```

- `RegisterGroup` creates a named group sorted by `(ordering, groupName)`.
- `RegisterSetting` adds a `Setting` implementation under an existing `group` sorted by `(ordering, name)`. Note that `DEFAULT_GROUP` (`"Default"`) is defined as a constant name, so you must still call `RegisterGroup(..., settings.DEFAULT_GROUP, ...)` before registering settings under it.
- **Timing requirement**: Both `RegisterGroup` and `RegisterSetting` **must** be called before `settings` finishes `Ready()` (for example, in `init()` or inside an app's `Init` callback). Calling either after `Ready()` locks the registry panics with `"settings registry is locked, no settings may be registered anymore"`.

### Registration Example

```go
package mysettings

import (
	"context"
	"encoding/json"
	"html/template"
	"reflect"

	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type SiteSetting struct {
	Type        dbtype.Type
	Default     any
	Perms       []string
	LabelFn     func(context.Context) string
	DescFn      func(context.Context) string
	FormFieldFn func(context.Context) forms.Field
}

func (s SiteSetting) DBType() dbtype.Type                      { return s.Type }
func (s SiteSetting) Permissions() []string                  { return s.Perms }
func (s SiteSetting) Icon(context.Context) template.HTML     { return "" }
func (s SiteSetting) Label(ctx context.Context) string       { return s.LabelFn(ctx) }
func (s SiteSetting) Description(ctx context.Context) string { return s.DescFn(ctx) }
func (s SiteSetting) FormField(ctx context.Context) forms.Field {
	return s.FormFieldFn(ctx)
}
func (s SiteSetting) DefaultValue() any { return s.Default }
func (s SiteSetting) ValueFromDB(_ context.Context, raw json.RawMessage) (any, error) {
	v := reflect.New(drivers.DBToDefaultGoType(s.Type))
	err := json.Unmarshal(raw, v.Interface())
	return v.Elem().Interface(), err
}

func init() {
	settings.RegisterGroup(1, "general", settings.SettingsGroup{
		Label:       trans.S("General"),
		Description: trans.S("General site configuration"),
	})

	settings.RegisterSetting(1, "general", "general.site_name", SiteSetting{
		Type:    dbtype.String,
		Default: "My Go-Django Site",
		LabelFn: trans.S("Site Name"),
		DescFn:  trans.S("Displayed in page headers."),
		FormFieldFn: func(ctx context.Context) forms.Field {
			return fields.CharField()
		},
	})
}
```

---

## Sync Command and System Check

Each registered setting requires a row in `RegisteredSetting` (keyed by `InternalName`) before values can be edited in the admin:

- **CLI command (`sync-settings`)**: `SyncSettingsCmd` (`ID: "sync-settings"`) creates any missing `RegisteredSetting` rows in a transaction and registers `"settings:view"`, `"settings:edit"`, plus any custom permissions declared by `setting.Permissions()`:

  ```bash
  go run ./... sync-settings
  ```

- **System check (`contrib.settings.SyncRequired`)**: During startup checks, `SettingsApp.Check` emits a critical check message (`"contrib.settings.SyncRequired"`) for every registered setting whose database row does not yet exist (`setting.instance.ID == 0`).

See [CLI](../cli.md) and [Checks](../checks.md) for running commands and handling startup checks.

---

## Middleware and Reading Settings per Request

During `Init`, if `"contrib.settings.SettingsForRequestMiddleware"` is `true` (the default), `settings` attaches `SettingsForRequestMiddleware` to `django.Global.Mux`. On every non-static request, it calls `RequestWithSettings(r)` to query and cache the user's permitted settings in `r.Context()`.

```go
func RequestWithSettings(r *http.Request) (*http.Request, error)
func SettingsForRequest(r *http.Request, refresh ...bool) (*orderedmap.OrderedMap[string, BoundSettingGroup], error)
```

- `SettingsForRequest(r)` returns the cached map from `r.Context()`, or queries the database if not cached (or if `refresh` is `true`).
- Any individual setting that specifies non-empty `Permissions()` is omitted unless `permissions.HasPermission(r, perms...)` succeeds.

Reading a setting in a handler:

```go
groups, err := settings.SettingsForRequest(r)
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

if group, ok := groups.Get("general"); ok {
	for _, s := range group.Settings {
		if s.Name() == "general.site_name" {
			val, err := s.Value()
			if err == nil && val == nil {
				val = s.DefaultValue()
			}
			_ = val
		}
	}
}
```

---

## Admin Routes, Permissions, Menu Hook, and Audit Logs

When `Ready()` runs, `settings` registers two routes under `admin.AdminSite.Route`:

| Route Name | Path | Permissions Checked |
| --- | --- | --- |
| `admin:settings` | `GET /admin/settings` | `settings:view` |
| `admin:settings:edit` | `GET`, `POST /admin/settings/change` | `settings:edit` |

- **Permissions**: `"settings:view"` controls visibility of the Settings menu item and read-only detail view; `"settings:edit"` controls the `"Change"` button and `/settings/change` form submission. Individual settings can also specify custom permissions via `Setting.Permissions()`. See [Permissions](../permissions.md).
- **Menu customization (`SettingsMenuHook`)**: `settings` hooks `admin.RegisterMenuItemHook` and invokes every callback registered on `settings.SettingsMenuHook` (`"settings:register_to_menu"`). If one menu item is returned, it is appended directly to the admin menu; if multiple items are returned, they are grouped into a `"db_settings"` submenu labeled `"Settings"`. See [Hooks](../hooks.md) and [Admin App](./admin/app.md).
- **Audit logs & messages**: When settings are saved successfully in `admin:settings:edit`, `viewSettingsEdit` logs an entry via `auditlogs.Log(r.Context(), "settings:changed", logger.WRN, nil, map[string]interface{}{"settings": settings})` and adds flash messages via [`contrib/messages`](./messages.md). See [Auditlogs](./auditlogs.md).

---

## Notes and Caveats

- **Requires `migrator` to create its table**: Unlike `documents`, `images`, and `revisions`, `contrib/settings` does not auto-create `RegisteredSetting` in `Init` when `migrator` is absent; include `migrator.NewAppConfig` and run `makemigrations` + `migrate`.
- **Requires `blocks` and `messages` for the admin UI**: `viewSettings` and `viewSettingsEdit` build forms with `contrib/blocks` (`NewStructBlock` / `BlockField`) and flash status via `contrib/messages`. Include both apps when mounting the admin UI.
- **Unreturned `autherrors.Fail` on missing permission**: In `viewSettings` and `viewSettingsEdit`, failing `permissions.HasPermission` calls `autherrors.Fail(http.StatusInternalServerError, "user has no permissions")` without an explicit `return` statement immediately following it (relying on `autherrors.Fail` panicking for the error-handling middleware). Ensure `django.APPVAR_RECOVERER` or the auth/admin error middleware is active in production.
- **Non-existent group panics**: Calling `RegisterSetting` for a group name that has not been registered with `RegisterGroup` panics.
