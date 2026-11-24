# Configuring Your Application

Go-Django stores application configuration, installed apps, routes, logger setup, and commands in a global `*django.Application` singleton (`django.Global`).

## Settings Interface and Helpers

Configuration values are accessed through the `django.Settings` interface:

```go
type Settings interface {
    Set(key string, value interface{})
    Get(key string) (any, bool)
    Bind(app *Application) error
    App() *Application
}
```

- `Set`: Stores a setting value by key.
- `Get`: Retrieves the raw value and a boolean indicating whether the key is present.
- `Bind`: Associates the `Settings` instance with an `*Application` (and sets `app.Settings`).
- `App`: Returns the bound `*Application`.

We provide a default map-backed implementation via `django.Config(map[string]interface{}) Settings`, which you normally pass using the `django.Configure(...)` option. If you implement `django.Settings` yourself, pass it with `django.AppSettings(settings)`.

### Retrieving Settings Values

Retrieve typed settings with `django.ConfigGet` or `django.ConfigGetOK` (or call `app.Config(key)` for an untyped `interface{}`):

```go
var allowedHosts = django.ConfigGet[[]string](
    django.Global.Settings,
    django.APPVAR_ALLOWED_HOSTS,
    []string{"*"},
)

var db, hasDB = django.ConfigGetOK[drivers.Database](
    django.Global.Settings,
    django.APPVAR_DATABASE,
)
```

`ConfigGet[T]` and `ConfigGetOK[T]` accept zero or one default argument:

- If `s == nil` or the key is missing, they return `default_[0]` (if provided) or the zero value of `T`, with `ok = false`.
- If the stored value is non-nil and not a `string`, it is type-asserted to `T` and panics if the type does not match.

> [!WARNING]
> **Empty-string caveat:** If a setting's stored value is an empty string (`""`), `ConfigGetOK` immediately executes `return default_[0], true`. Calling `ConfigGet[T](s, key)` or `ConfigGetOK[T](s, key)` **without** a default value when the key holds `""` will panic with an index-out-of-range error. Always store non-string settings using their actual Go types (such as `bool`, `int`, `[]string`), and provide a default argument when reading optional string settings.

---

## Core Application Settings

All built-in settings keys are exported as `APPVAR_*` constants in package `django` (`github.com/Nigel2392/go-django/src`):

| Constant | Key String | Type | Default / Notes |
| --- | --- | --- | --- |
| `django.APPVAR_DEBUG` | `"DEBUG"` | `bool` | Enables debug-level behavior (e.g., route logging fallback). |
| `django.APPVAR_ALLOWED_HOSTS` | `"ALLOWED_HOSTS"` | `[]string` | **Required non-empty** by the startup `TagSettings` check. Requests whose `Host` header is not in this list are rejected by `middleware.AllowedHosts`. |
| `django.APPVAR_RECOVERER` | `"RECOVERER"` | `bool` | Default `true`. Wraps routes in `middleware.Recoverer(a.ServerError)` to recover from panics. |
| `django.APPVAR_HOST` | `"HOST"` | `string` | Default `"localhost"`. Bind host for HTTP/HTTPS servers. |
| `django.APPVAR_PORT` | `"PORT"` | `string` | Default `"8080"`. HTTP listen port. Set to `""` or `"0"` to disable plain HTTP. |
| `django.APPVAR_TLS_PORT` | `"TLS_PORT"` | `string` | Default `""`. HTTPS listen port (requires `APPVAR_TLS_CERT` and `APPVAR_TLS_KEY`). |
| `django.APPVAR_TLS_CERT` | `"TLS_CERT"` | `string` | Path to the TLS certificate file. |
| `django.APPVAR_TLS_KEY` | `"TLS_KEY"` | `string` | Path to the TLS private key file. |
| `django.APPVAR_TLS_CONFIG` | `"TLS_CONFIG"` | `*tls.Config` | Custom `*tls.Config` applied to both `http.Server` instances. |
| `django.APPVAR_STATIC_URL` | `"STATIC_URL"` | `string` | Default `"/static/"`. URL prefix for serving [staticfiles](./staticfiles.md). |
| `django.APPVAR_MEDIA_URL` | `"MEDIA_URL"` | `string` | Default `"/media/"`. URL prefix for serving [mediafiles](./filesystem.md#media-files). |
| `django.APPVAR_MEDIA_BACKEND` | `"MEDIA_BACKEND"` | `string` | Default `""` (uses the default registered media backend). |
| `django.APPVAR_MAX_FORM_MEMORY` | `"APPVAR_MAX_FORM_MEMORY"` | `int` | Max memory passed to `ParseMultipartForm` (`django.DEFAULT_MAX_FORM_MEMORY` is `1024 * 1024 * 64`, 64 MiB). |
| `django.APPVAR_DATABASE` | `"DATABASE"` | `drivers.Database` | Shared database connection used by apps and `sql` command. |
| `django.APPVAR_CONTINUE_AFTER_COMMANDS` | `"CONTINUE_AFTER_COMMAND"` | `bool` | Default `false`. When `true`, `Initialize()` continues after a CLI command runs instead of returning `command.ErrShouldExit`. Note the singular key string `"CONTINUE_AFTER_COMMAND"`. |
| `django.APPVAR_ROUTE_LOGGING_ENABLED` | `"ROUTE_LOGGING_ENABLED"` | `bool` | Defaults to `APPVAR_DEBUG` (or `true` when `APPVAR_DEBUG` is unset). |
| `django.APPVAR_STATIC_ROUTE_LOGGING_ENABLED` | `"STATIC_ROUTE_LOGGING_ENABLED"` | `bool` | Default `false`. Static routes only log (at `logger.DBG`) when both `APPVAR_DEBUG` and `APPVAR_STATIC_ROUTE_LOGGING_ENABLED` are `true`. |
| `django.APPVAR_STATIC_ROUTE_CACHING_MAXAGE` | `"STATIC_ROUTE_CACHING_MAXAGE"` | `int` | Default `3600` seconds. Set to `0` (or negative) to disable static route cache headers. |
| `django.APPVAR_REQUESTS_PROXIED` | `"REQUESTS_PROXIED"` | `bool` | Default `false`. Controls proxy header inspection in `django.GetIP` and request logging. |
| `django.APPVAR_SESSION_MANAGER` | `"SESSION_MANAGER"` | `*scs.SessionManager` | Session manager stored by the `session` app or provided manually. |
| `django.APPVAR_DISABLE_NOSURF` | `"DISABLE_NOSURF"` | `bool` | Default `false`. Disables automatic `nosurf` CSRF protection middleware when `true`. |
| `django.APPVAR_TRANSLATIONS_DEFAULT_LOCALE` | `"TRANSLATIONS_DEFAULT_LOCALE"` | See caveat | Sets `trans.TRANSLATIONS_DEFAULT_LOCALE` during `Initialize()`. |
| `django.APPVAR_ErrorCode(code int)` | `"HandleErrorCode%d"` | `func(http.ResponseWriter, *http.Request, except.ServerError)` | Custom HTTP error handler function for a specific status code (see [Errors](./errors.md)). |

> [!WARNING]
> **Caveat on `APPVAR_TRANSLATIONS_DEFAULT_LOCALE`:** The built-in `TagSettings` startup check validates `APPVAR_TRANSLATIONS_DEFAULT_LOCALE` as a `string`, whereas `Application.Initialize()` reads it using `django.ConfigGet(a.Settings, APPVAR_TRANSLATIONS_DEFAULT_LOCALE, language.English)`, which expects a `language.Tag`. If you set a `string`, `Initialize()` will panic; if you set a `language.Tag`, the startup check will report a critical error unless checks are skipped (`django.Flag(django.FlagSkipChecks)`) or silenced (`checks.Shutup("settings.matches.TRANSLATIONS_DEFAULT_LOCALE", true)`).

---

## App-Specific Settings

Core packages and contrib apps define additional `APPVAR_*` settings keys:

- **`auth`** (`github.com/Nigel2392/go-django/contrib/auth`):
  - `auth.APPVAR_AUTH_EMAIL_LOGIN` (`bool`): Enable login via email.
  - `auth.APPVAR_REGISTER_AUTH_URLS` (`bool`): Automatically register auth login/logout/register URLs.
  - `auth.APPVAR_ALLOW_USER_REGISTER` (`bool`): Enable the user registration view.
  - `auth.APPVAR_LOGIN_REDIRECT_URL` (`string` or `func(*http.Request) string`): Redirect target after successful login (defaults to `"/"`).
  - `auth.APPVAR_LOGIN_VIEW_REVERSE_URL` (`string`): Reverse route name for the login view.
- **`admin`** (`github.com/Nigel2392/go-django/contrib/admin`):
  - `admin.APPVAR_HOME_PAGE_TITLE` (`string`): Title shown on the admin dashboard.
  - `admin.APPVAR_HOME_PAGE_SUBTITLE` (`string`): Subtitle shown on the admin dashboard.
  - `admin.APPVAR_HOME_PAGE_LOGO_PATH` (`string`): Path to a custom logo for the admin dashboard.
- **`editor`** (`github.com/Nigel2392/go-django/contrib/editor`):
  - `editor.APPVAR_WRAP_RENDERED_BLOCKS` (`bool`): Wrap rendered editor blocks in container HTML.
  - `editor.APPVAR_ENABLE_INLINE_TOOLBAR` (`bool`): Enable the inline toolbar.
- **`messages`** (`github.com/Nigel2392/go-django/contrib/messages`):
  - `messages.APPVAR_COOKIE_KEY` (`http.Cookie`): Base cookie value for the cookie-based message backend. **Must be an `http.Cookie` struct value, NOT `*http.Cookie`.**
- **`pages`** (`github.com/Nigel2392/go-django/contrib/pages`):
  - `pages.APPVAR_OUTDATED_AFTER_DURATION` (`time.Duration`): Duration after which a page is considered outdated.
  - `pages.APPVAR_OUTDATED_LIST_DISPLAY` (`[]string`): Fields displayed when listing outdated pages.
- **`secrets`** (`github.com/Nigel2392/go-django/src/core/secrets`):
  - `secrets.APPVAR_SECRET_KEY` (`string`): Application secret key (**required** by a `TagSettings` check whenever `secrets` is imported).
  - `secrets.APPVAR_SECRET_KEY_FALLBACKS` (`[]secrets.SecretKey`): Fallback keys checked during decryption/unsigning after key rotation.
  - `secrets.APPVAR_SIGNER_BACKEND` (`signing.Signer`): Custom cryptographic signer backend.
  - `secrets.APPVAR_ENCRYPTION_BACKEND` (`encryption.Encryption`): Custom encryption backend.
- **`search`** (`github.com/Nigel2392/go-django/contrib/search`):
  - `search.APPVAR_SEARCH_BACKEND` (`string`): Name of the default registered search backend.
- **`translations`** (`github.com/Nigel2392/go-django/contrib/translations`):
  - `translations.APPVAR_TRANSLATIONS_DIR` (`[]string`): Source directories scanned by `make-translations` (note: documented in source comments as `string` but read as `[]string` in `translations.go`).
  - `translations.APPVAR_TRANSLATIONS_LOCALES` (`[]string` / `[]language.Tag`): Configured locale list.
  - `translations.APPVAR_TRANSLATIONS_PACKAGES` (`[]string`): Package aliases scanned for translation function calls.
  - `translations.APPVAR_TRANSLATIONS_EXTENSIONS` (`[]string`): Template extensions scanned (defaults to `tmpl`, `html`, `txt`).
  - `translations.APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE` (`[]language.Tag`): Supported locales matched against `Accept-Language` and the `lang` cookie.
- **`migrator`** (`github.com/Nigel2392/go-django/queries/src/migrator`):
  - `migrator.APPVAR_MIGRATION_DIR` (`string`): Directory where migration files are stored.

---

## Creating and Configuring the Application

Configure the application using `django.App(opts ...Option) *Application`. The first call allocates `django.Global` if it is `nil`, and **every call** executes the supplied `Option` functions against `django.Global`.

```go
type Option func(*Application) error
```

### Built-in Option Functions

- `django.Configure(m map[string]interface{})`: Wraps `m` in default map-backed `Settings` and binds it to the application.
- `django.AppSettings(settings Settings)`: Binds a custom `Settings` implementation. Returns an error if `app.Settings` is already set.
- `django.Apps(apps ...any)`: Appends apps or app factories to be instantiated during `Initialize()`.
- `django.AppLogger(w logger.Log)`: Assigns `app.Log = w` and immediately calls `logger.Setup(w)` so logging works before `Initialize()`.
- `django.Flag(flags ...AppFlag)`: Enables one or more bitmask flags:
  - `django.FlagSkipDepsCheck`: Skip validating `AppConfig.Dependencies()`.
  - `django.FlagSkipCmds`: Skip CLI command parsing and execution during `Initialize()`.
  - `django.FlagSkipChecks`: Skip startup, command, and application checks during `Initialize()`.

### Example

```go
package main

import (
    "context"
    "os"

    "github.com/Nigel2392/go-django/contrib/session"
    "github.com/Nigel2392/go-django/queries/src/drivers"
    django "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/core/logger"
)

func main() {
    db, err := drivers.Open(context.Background(), "sqlite3", "./.private/db.sqlite3")
    if err != nil {
        panic(err)
    }

    app := django.App(
        django.Configure(map[string]interface{}{
            django.APPVAR_ALLOWED_HOSTS: []string{"*"},
            django.APPVAR_HOST:          "127.0.0.1",
            django.APPVAR_PORT:          "8080",
            django.APPVAR_DEBUG:         false,
            django.APPVAR_DATABASE:      db,
            django.APPVAR_STATIC_URL:    "/static/",
        }),
        django.AppLogger(&logger.Logger{
            Level:       logger.INF,
            OutputTime:  true,
            Prefix:      "myapp",
            OutputDebug: os.Stdout,
            WrapPrefix:  logger.ColoredLogWrapper,
        }),
        django.Apps(
            session.NewAppConfig,
            NewCustomAppConfig,
        ),
    )

    if err := app.Initialize(); err != nil {
        panic(err)
    }

    if err := app.Serve(); err != nil {
        panic(err)
    }
}
```

---

## Initialization Lifecycle

`django.App(...)` returns an uninitialized `*django.Application`. Calling `app.Initialize()` executes the following steps in exact order:

1. **Already-initialized guard:** If `a.initialized.Load()` is `true`, return `nil` immediately.
2. **Logger setup:** If `a.Log == nil`, create a default `*logger.Logger` at `logger.GetLevel()`, with `OutputTime: true`, prefix `"django"`, `ColoredLogWrapper`, and `OutputAll` directed to `os.Stdout`, then call `logger.Setup(a.Log)`.
3. **Settings assertion:** Assert `a.Settings != nil`.
4. **Startup checks:** Unless `FlagSkipChecks` is set, run `checks.TagSettings` and `checks.TagSecurity` checks and fail if any serious errors occur.
5. **Mux, default middleware, and static/media routes:** Set `a.Mux.NotFoundHandler` (delegating 404s to `a.ServerError`), register `AllowedHosts`, `loggerMiddleware`, `ContextDataStoreMiddleware`, optional `Recoverer`, and `BufferMiddleware` on `a.Mux`, and mount static (`/static/*`) and media (`/media/*`, when media backends are registered) routes with `MarkStaticRouteMiddleware` via `route.Preprocess`.
6. **Global template configuration:** Register global `tpl.Processors` (injecting `"Application"` and `"Settings"` on `ctx.Context`), global `tpl.Funcs`, and global `tpl.RequestFuncs`.
7. **Built-in commands:** Register the `sql` and `check` commands on `a.Commands` (alongside `help` from `command.NewRegistry`).
8. **App instantiation:** Invoke each registered app factory in registration order and store the resulting `AppConfig` in `a.Apps` (asserting unique non-empty names).
9. **Model and content-type registration:** Lock `dbtype.Lock()`, emit `core.BeforeModelsReady`, register every app's `Models()` with `djangoapps.RegisterModel` and `contenttypes.Register`, call `attrs.RegisterModel` on each model, emit `core.OnModelsReady`, and emit `attrs.ResetDefinitions`.
10. **App Loop 1 (Dependencies, `Initialize`, `Commands`):** In registration order, verify each app's `Dependencies()` exist in `a.Apps` (unless `FlagSkipDepsCheck`), call `app.Initialize(a.Settings)`, and register `app.Commands()` on `a.Commands`.
11. **Command checks:** Unless `FlagSkipChecks` is set, run `checks.TagCommands` checks against all registered commands.
12. **CLI command execution:** Unless `FlagSkipCmds` is set, normalize `help`/`--help`/`-h`, parse the global `-v` flag (`logger.DBG`), and execute the CLI command **before App Loop 2 and `OnReady()` run**. If a command ran and `APPVAR_CONTINUE_AFTER_COMMANDS` is `false`, `Initialize()` returns `command.ErrShouldExit` (or exits directly for `flag.ErrHelp` / `ErrUnknownCommand`).
13. **App Loop 2 (Routing, Processors, Templates, Check collection):** For each app, call `app.BuildRouting` inside an app-scoped `mux.Namespace` (attaching the app via `ContextWithApp` on matched requests), register `app.Processors()` with `tpl.RequestProcessors`, register `app.Templates()` with `tpl.Add` (if non-nil), and collect `app.Check(ctx, a.Settings)` messages (unless `FlagSkipChecks`).
14. **Enforce application checks:** Unless `FlagSkipChecks` is set, log collected application checks and return an error if serious errors were found.
15. **App Loop 3 (`OnReady`):** Call `app.OnReady()` for every registered app.
16. **Finalize:** Assign `trans.TRANSLATIONS_DEFAULT_LOCALE`, store `a.initialized.Store(true)`, and emit `core.OnDjangoReady`.

> [!NOTE]
> Because CLI commands execute at **step 12**, `app.BuildRouting`, app context processors, app templates, application checks, and `app.OnReady()` have **not** run when a command executes (unless `APPVAR_CONTINUE_AFTER_COMMANDS` is `true`, in which case steps 13–16 run afterwards).

---

## Serving, Shutdown, Tasks, and Direct Routing

### `Serve()`

Call `app.Serve()` to start listening for incoming requests:

- If `Initialize()` has not run yet, `Serve()` calls `app.Initialize()` first.
- Wraps `a.Mux` with `RequestSignalMiddleware` and `nosurf` (unless `APPVAR_DISABLE_NOSURF` is `true`). `nosurf` automatically exempts requests whose `X-CSRF-Token` header passes `nosurf.VerifyToken(nosurf.Token(r), header)` and invokes `django.HOOK_SETUP_NOSURF` hooks.
- Runs `django.HOOK_SERVER_STARTUP` hooks before starting servers.
- Starts HTTP on `HOST:PORT` (when `PORT` is non-empty and not `"0"`) and/or HTTPS on `HOST:TLS_PORT` (when `TLS_CERT`, `TLS_KEY`, and `TLS_PORT` are all non-empty). Both servers receive `TLSConfig` if configured.

### `Quit()`

Call `app.Quit()` to gracefully shut down active HTTP/HTTPS servers and run all `django.HOOK_SERVER_SHUTDOWN` hooks (joining any errors).

### `Task()`

Use `app.Task(description, fn)` (or `django.Task` after `django.App(...)`) to run a maintenance or startup task with automatic duration logging under the `"Task"` logger namespace:

```go
err := app.Task("warm caches", func(a *django.Application) error {
    // Perform task work
    return nil
})
```

### Direct Routing via `app.Mux`

You can also register routes or middleware directly on `app.Mux` without creating a separate `AppConfig`:

```go
app.Mux.Get("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}), "healthz")
```

For larger applications, organizing routes into dedicated apps is recommended; see [Apps](./apps.md) and [Routing](./routing.md).
