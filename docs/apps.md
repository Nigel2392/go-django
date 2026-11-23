# Apps

Apps are the modular building blocks of a Go-Django application. Each app implements the `django.AppConfig` interface to register models, routes, templates, context processors, CLI commands, and startup checks.

## The `django.AppConfig` Interface

Every application registered with Go-Django must implement `django.AppConfig` (`github.com/Nigel2392/go-django/src`):

```go
type AppConfig interface {
    Name() string
    Dependencies() []string
    Models() []attrs.Definer
    Commands() []command.Command
    OnReady() error
    Check(ctx context.Context, settings Settings) []checks.Message
    BuildRouting(mux mux.Multiplexer)
    Initialize(settings Settings) error
    Processors() []func(ctx.ContextWithRequest)
    Templates() *tpl.Config
}
```

- `Name`: Unique non-empty identifier for the app (registering two apps with the same name panics).
- `Dependencies`: Names of other apps that must already be registered before this app initializes.
- `Models`: Model structs (`attrs.Definer`) owned by this app; registered globally in `djangoapps`, `contenttypes`, and `attrs`.
- `Commands`: CLI subcommands registered during initialization.
- `Initialize`: Runs during App Loop 1 with the application `Settings`.
- `BuildRouting`: Registers routes and app-scoped middleware during App Loop 2.
- `Processors`: Returns request template context processors registered during App Loop 2.
- `Templates`: Returns a `*tpl.Config` registered with `tpl.Add` during App Loop 2.
- `Check`: Runs app-level checks during App Loop 2 (and via the `check` command).
- `OnReady`: Runs during App Loop 3 after all apps are initialized, routed, and checked.

---

## Creating an App with `apps.AppConfig` and `apps.DBRequiredAppConfig`

The `github.com/Nigel2392/go-django/src/apps` package provides two ready-made implementations of `django.AppConfig`:

- `apps.NewAppConfig(name string) *apps.AppConfig`: Standard app configuration struct.
- `apps.NewDBAppConfig(name string) *apps.DBRequiredAppConfig`: Embeds `*apps.AppConfig`, asserts that `django.APPVAR_DATABASE` is present and implements `drivers.Database`, stores it in the `DB` field, runs `AppConfig.Init(settings)` (if non-nil), and then runs `DBRequiredAppConfig.Init(settings, db)` (if non-nil).

```go
package myapp

import (
    "net/http"

    "github.com/Nigel2392/go-django/queries/src/drivers"
    django "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/apps"
    "github.com/Nigel2392/go-django/src/core/attrs"
    "github.com/Nigel2392/go-django/src/core/ctx"
    "github.com/Nigel2392/mux"
)

type CustomApp struct {
    // Or embed *apps.AppConfig if no database is required
    *apps.DBRequiredAppConfig
}

var globalInstance *CustomApp

func NewCustomAppConfig() *CustomApp {
    myCustomApp := &CustomApp{
        DBRequiredAppConfig: apps.NewDBAppConfig("myapp"),
    }

    // Dependencies that must be listed before this app in django.Apps(...)
    myCustomApp.Deps = []string{
        "session",
    }

    // Models registered in djangoapps, contenttypes, and attrs
    myCustomApp.ModelObjects = []attrs.Definer{
        // &MyModel{},
    }

    myCustomApp.Init = func(settings django.Settings, db drivers.Database) error {
        // Setup using settings and db
        return nil
    }

    myCustomApp.Routing = func(m mux.Multiplexer) {
        m.Get("/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Hello from myapp"))
        }), "index")
    }

    myCustomApp.CtxProcessors = []func(ctx.ContextWithRequest){
        func(c ctx.ContextWithRequest) {
            c.Set("AppName", "myapp")
        },
    }

    myCustomApp.Ready = func() error {
        globalInstance = myCustomApp
        return nil
    }

    return myCustomApp
}
```

### Single-Handler Apps

For tiny apps that only expose a single route, use `apps.NewAppConfigForHandler`:

```go
package myapp

import (
    "net/http"

    "github.com/Nigel2392/go-django/src/apps"
    "github.com/Nigel2392/mux"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
}

var app = apps.NewAppConfigForHandler(
    "myapp", mux.GET, "/myapp", mux.NewHandler(myHandler),
)
```

---

## Registering Apps and Dependencies

Pass apps to `django.Apps(...)` or `app.Register(...)`:

```go
app := django.App(
    django.Configure(settings),
    django.Apps(
        session.NewAppConfig,
        myapp.NewCustomAppConfig,
    ),
)
```

`django.Apps(...)` and `Register(...)` accept:

- An instantiated `django.AppConfig` value
- `func() django.AppConfig`
- `func() (django.AppConfig, error)`
- Any zero-argument or variadic-only factory function returning a type implementing `django.AppConfig` (or `(T, error)` where `T` implements `django.AppConfig`), such as `func() *CustomApp`

### Dependency Order (`Deps`)

> [!IMPORTANT]
> `Deps` / `Dependencies()` validates that every listed dependency app is registered in `a.Apps` before running `app.Initialize(settings)` (unless `django.FlagSkipDepsCheck` is set). It **does not** reorder apps automatically—you must pass apps to `django.Apps(...)` in dependency order (for example, `session.NewAppConfig` before `auth.NewAppConfig`).

---

## `apps.AppConfig` Fields and Methods

```go
type AppConfig struct {
    AppName        string
    Deps           []string
    Routing        func(mux.Multiplexer)
    Cmd            []command.Command
    Init           func(settings django.Settings) error
    Ready          func() error
    CtxProcessors  []func(ctx.ContextWithRequest)
    TemplateConfig *tpl.Config
    ModelObjects   []attrs.Definer
    // ...
}
```

### Methods

- `AddCommand(c ...command.Command)`: Appends one or more commands to `Cmd`. See [Commands](./commands.md#registering-a-command).
- `IsReady() bool`: Returns `true` after `OnReady()` has finished executing (note: `OnReady` sets `ready = true` even if `Ready` returns a non-nil error).
- `Templates() *tpl.Config`: Returns `TemplateConfig`; if `TemplateConfig != nil` and `TemplateConfig.AppName == ""`, it automatically sets `TemplateConfig.AppName = a.AppName`.
- `Check(ctx context.Context, settings django.Settings) []checks.Message`: Runs model checks across `ModelObjects`:
  - If a model implements `checks.Checker`, calls `model.Check(ctx)`.
  - Otherwise, emits a `model.cant_check` warning and checks each field via `attrs.Define(ctx, model).Fields()` for duplicate primary keys (`model.multiple_primary_keys`), valid `drivers.DBType(field)` (`field.invalid_db_type`), field-level `checks.Checker` implementations, and missing primary keys (`model.no_primary_key`, skipped for through-models).

### Fields

- `Routing`: Function receiving an app-scoped `mux.Multiplexer` to register routes and app middleware; see [Routing](./routing.md#urls).
- `Cmd`: Slice of `command.Command` values registered with `a.Commands`.
- `TemplateConfig`: `*tpl.Config` registered via `tpl.Add`; see [Rendering](./rendering.md#initial-setup).
- `CtxProcessors`: Slice of `func(ctx.ContextWithRequest)` registered with `tpl.RequestProcessors`; see [Context Processors](./rendering.md#context-processors-requestprocessors).
- `ModelObjects`: Models (`attrs.Definer`) registered with `attrs.RegisterModel` and `contenttypes`; see [Models](./models.md) and [Attrs](./attrs/attrs.md).
- `Deps`: List of required app names validated before initialization.

---

## Looking Up Registered Apps

Once apps are registered and instantiated, Go-Django provides helpers in package `django`:

- `django.GetApp[T AppConfig](name string) T`: Looks up the app by name and asserts it implements `T` (panics if missing or wrong type).
- `app.App(name string) AppConfig`: Returns the `AppConfig` for `name` (panics if missing).
- `django.AppInstalled(name string) bool`: Reports whether an app named `name` is registered on `django.Global`.
- `django.GetAppForModel(model attrs.Definer) (AppConfig, bool)`: Returns the app that owns `model` (registered via `ModelObjects`).
- `django.AppFromContext(ctx context.Context) (AppConfig, bool)`: Inside routes built via `AppConfig.BuildRouting`, retrieves the owning `AppConfig` from the request's `context.Context`.
