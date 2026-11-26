# Auditlogs

The `auditlogs` app (`github.com/Nigel2392/go-django/contrib/reports/audit_logs`) records user and system actions in a dedicated database table and provides an admin report view for inspecting, filtering, and exporting entries.

## Required Parent App: `contrib/reports`

`auditlogs` declares `"reports"` as a hard dependency (`Logs.Deps = []string{"reports"}`). The parent package `github.com/Nigel2392/go-django/contrib/reports` provides the top-level **Reports** submenu in the admin interface:

- `reports.NewAppConfig() django.AppConfig`: Registers the `"reports"` app and hooks into `admin.RegisterMenuItemHook` to add the `"Reports"` submenu.
- `reports.ReportsMenuHook` (`"reports:register_to_menu"`): Goldcrest hook constant invoked when building items inside the **Reports** submenu.
- `reports.ReportsMenuHookFunc`: Hook signature `func(r *http.Request) []menu.MenuItem`.
- `reports.RegisterMenuItem(fn func(r *http.Request) []menu.MenuItem)`: Helper that registers menu items under the **Reports** submenu.

## Installation

Include `reports.NewAppConfig` alongside `auditlogs.NewAppConfig`, `admin.NewAppConfig`, and `django.APPVAR_DATABASE`:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
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
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

`auditlogs.NewAppConfig` returns a `*migrator.MigratorAppConfig` with migrations for `&auditlogs.Entry{}`. When `migrator` is not installed, `auditlogs` automatically creates the `Entry` table during initialization.

## Viewing and Filtering Auditlogs

In the admin interface, open **Reports -> Audit Logs** (mounted at `/admin/auditlogs` and reversible as `"admin:auditlogs"`). Accessing `/admin/auditlogs` and viewing the admin home page events panel both require the `"auditlogs:list"` permission.

The audit logs view provides:

- **Structured sidebar filters**: `type` (log entry type), `level` (`Debug`, `Info`, `Warning`, `Error`), `user`, `content_type`, and `object_id`. There is no free-text search bar.
- **Pagination**: Defaults to 25 rows per page with a per-page selector (`15`, `25`, `50`, `100`) controlled by `page` and `amount` query parameters.
- **CSV export**: Appending `?_export=1` (via the **Download CSV** action button) exports the filtered queryset as CSV.

## Automatic Logging

Once installed, `auditlogs` automatically records:

- **Admin model CRUD actions**: Registers hooks on `admin.AdminModelHookAdd`, `admin.AdminModelHookEdit` (`logger.INF`), and `admin.AdminModelHookDelete` (`logger.WRN`) once `Logs.IsReady()` is true, storing `model_name`, `instance_id`, and `content_type`.
- **Failed logins**: Listens to `core.SIGNAL_LOGIN_FAILED` and logs `"auth.login_failed"` entries at `logger.ERR` containing `form` (masking sensitive fields via `safety.IsSecretField`), `host`, and `ip`, formatted by a built-in definition.

## Models, Interfaces, and Errors

Each stored audit log record uses `&auditlogs.Entry{}` (implementing `LogEntry` and `DBCreateLogEntry`):

```go
type LogEntry interface {
	attrs.Definer
	ID() uuid.UUID
	Type() string
	Level() logger.LogLevel
	Timestamp() time.Time
	User() users.User
	ObjectID() interface{}
	ContentType() contenttypes.ContentType
	Data() map[string]interface{}
}

type DBCreateLogEntry interface {
	LogEntry
	Create(context.Context) error
}

type Entry struct {
	models.Model
	Id    drivers.UUID                         `db:"id" json:"id"`
	Typ   drivers.String                       `db:"type" json:"type"`
	Time  drivers.Timestamp                    `db:"timestamp" json:"timestamp"`
	Usr   users.User                           `db:"user" json:"user"`
	ObjID drivers.JSON[any]                    `db:"object_id" json:"object_id"`
	CType *contenttypes.BaseContentType[any]   `db:"content_type" json:"content_type"`
	Src   drivers.JSON[map[string]interface{}] `db:"data" json:"data"`
	Lvl   logger.LogLevel                      `db:"level" json:"level"`

	Obj interface{} `json:"-"`
}
```

Package variables and sentinel errors:

- `LogUnknownTypes` (`bool`, default `true`): When `false`, `auditlogs.Log(...)` returns an error if no `Definition` is registered for `entryType`.
- `ErrLogsNotReady`: Returned by `Log(...)` if called before the `auditlogs` app is ready.
- `ErrLogEntryNotFound`: Sentinel error indicating a log entry could not be found.

## Customizing Definitions and Actions

Definitions control how log entry types are labeled, formatted, and linked in the admin list view:

```go
type LogEntryAction interface {
	Icon() string
	Label() string
	URL() string
}

type BaseAction struct {
	DisplayLabel string
	ActionIcon   string
	ActionURL    string
}

type Definition interface {
	TypeLabel(r *http.Request, typeName string) string
	GetLabel(r *http.Request, logEntry LogEntry) string
	FormatMessage(r *http.Request, logEntry LogEntry) any // string | template.HTML
	GetActions(r *http.Request, logEntry LogEntry) []LogEntryAction
}

type BoundDefinition struct {
	Request    *http.Request
	Definition Definition
	LogEntry
}
```

Register and resolve definitions using:

- `RegisterDefinition(typ string, definition Definition) Definition`: Associates `definition` with `typ`.
- `DefinitionForType(typ string) Definition`: Looks up the registered `Definition` for `typ` (returns `nil` if unregistered).
- `SimpleDefinition() Definition`: Returns the default fallback `Definition` used when no custom definition is registered.
- `Define(r *http.Request, l LogEntry) *BoundDefinition`: Binds `l` to its registered `Definition` (or `SimpleDefinition()`). `BoundDefinition` exposes `Label()`, `Message()`, and `Actions()`, which appends quick-filter links (`?filters-type=...`, `?filters-content_type=...`, and `?filters-object_id=...&filters-content_type=...`) to any custom actions.

```go
func init() {
	auditlogs.RegisterDefinition("example.delete_model", &YourCustomAuditlogDefinition{})
}
```

## Filtering and Transforming Log Entries

You can intercept entries before they are persisted by implementing `EntryFilter` or `EntryHandler`:

```go
type EntryFilter interface {
	Type() string
	EntryFilter(message LogEntry) bool
}

type EntryHandler interface {
	Type() string
	Handle(message DBCreateLogEntry) (DBCreateLogEntry, error)
}
```

- `RegisterFilter(filter EntryFilter)` / `RegisterHandler(handler EntryHandler)`: Registers a filter or handler for all entries of `Type()`.
- `RegisterFilterForObject(filter EntryFilter, contentType contenttypes.ContentType)` / `RegisterHandlerForObject(handler EntryHandler, contentType contenttypes.ContentType)`: Scopes a filter or handler to a specific object `contentType` package path as well as `Type()`.

If any matching `EntryFilter` returns `false`, `Log(...)` skips saving the entry and returns `uuid.Nil, nil`. Matching `EntryHandler`s execute sequentially and may modify or replace the `DBCreateLogEntry` before `Create(ctx)` runs.

## Logging Custom Events

Call `auditlogs.Log` with a context, entry type, log level, optional target `attrs.Definer`, and optional data map:

```go
package main

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
)

func logDelete(ctx context.Context, actor users.User, obj attrs.Definer, title string) error {
	ctxWithUser := authentication.ContextWithUser(ctx, actor)

	_, err := auditlogs.Log(
		ctxWithUser,            // context (honors request/ORM transactions and user)
		"example.delete_model", // log entry type
		logger.INF,             // log level
		obj,                    // related object (may be nil)
		map[string]any{         // additional JSON payload
			"title": title,
		},
	)
	return err
}
```

`Log` reads the acting user from `authentication.UserFromContext(ctx)` (ignoring unauthenticated users). Outside HTTP requests, attach the actor via `authentication.ContextWithUser(ctx, user)`. Because `Entry.Create(ctx)` uses the `queries` package, any active database transaction on `ctx` is respected.

## Query Filter Helpers

`contrib/reports/audit_logs` provides expression builders for querying `&auditlogs.Entry{}` with `queries`:

- `FilterType(values ...string) expr.Expression`
- `FilterLevel(values ...logger.LogLevel) expr.Expression`
- `FilterLevelGT(value logger.LogLevel) expr.Expression`
- `FilterLevelLT(value logger.LogLevel) expr.Expression`
- `FilterTimestamp(values ...time.Time) expr.Expression`
- `FilterUserID(values ...interface{}) expr.Expression`
- `FilterObjectID(values ...interface{}) expr.Expression`
- `FilterContentType(values ...contenttypes.ContentType) expr.Expression`
- `FilterData(values ...interface{}) expr.Expression`

### Caveats

- **`FilterUserID` field-name mismatch**: `FilterUserID` builds `expr.Q("UserID", ...)` / `expr.Q("UserID__in", ...)`, whereas `Entry.Fields` registers its user foreign key under `NameOverride: "User"`. When querying `&auditlogs.Entry{}`, filter on `"User"` directly (for example `expr.Q("User", userID)`) instead of calling `FilterUserID`.
- **Zero-argument panic**: All variadic `Filter*` helpers (`FilterType`, `FilterLevel`, `FilterTimestamp`, `FilterUserID`, `FilterObjectID`, `FilterContentType`, and `FilterData`) index `values[0]` / `v[0]` unconditionally when fewer than two arguments are passed, so calling any of them with 0 arguments panics.
