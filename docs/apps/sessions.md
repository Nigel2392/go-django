# Sessions

The `session` app provides session middleware and persistent database-backed storage for Go-Django, wrapping [alexedwards/scs](https://github.com/alexedwards/scs/v2) and [Nigel2392/mux/middleware/sessions](https://github.com/Nigel2392/mux/v2). By default, sessions expire after 7 days (`time.Hour * 24 * 7`).

## Installation and Storage

Register `session.NewAppConfig` in `django.Apps(...)` to enable sessions across your application:

```go
package main

import (
	"context"

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
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

### Storage Backends and Migrations

During initialization, `session` stores the created `*scs.SessionManager` under `django.APPVAR_SESSION_MANAGER` in your application [settings](../configuring.md), making it accessible elsewhere in your project.

The [`django.APPVAR_DATABASE`](../configuring.md) setting is **optional**:

- **Without `APPVAR_DATABASE`**: `session` automatically falls back to an in-memory store (`memstore.New()`).
- **With `APPVAR_DATABASE`**: `session` configures `QueryStore` (`session.NewQueryStore(db)`), supporting `sqlite3`, `mysql`, `mariadb`, and `postgres` out of the box, as well as any custom driver registered with `drivers.Retrieve`.

`session.NewAppConfig` returns a `*migrator.MigratorAppConfig` containing migrations for the `sessions` table. If the `migrator` app is **not** installed, `session` automatically creates the `sessions` table and its indexes (`sessions_expiry_idx` and `sessions_token_idx`) during initialization for the built-in drivers (`sqlite3`, `mysql`, `mariadb`, and `postgres`).

### Routing Middleware

During routing setup, `session` installs `sessions.SessionMiddleware(sessionManager)` wrapped with `django.NonStaticMiddleware`. This skips session loading and committing for requests routed through Go-Django's static file handler.

## Session Interface vs. Session Model

Go-Django distinguishes between the request-scoped `sessions.Session` interface and the database-backed `session.Session` model:

- **`sessions.Session`** (`github.com/Nigel2392/mux/v2/middleware/sessions`) is the interface stored on the HTTP request context for reading and mutating session state.
- **`session.Session`** (`github.com/Nigel2392/go-django/contrib/session`) is the ORM model mapped to the `sessions` database table.

### `sessions.Session` Interface

```go
type Session interface {
	Token() string
	Set(key string, value interface{})
	Get(key string) interface{}
	Exists(key string) bool
	Delete(key string)
	Destroy() error
	RenewToken() error
	AddFinalizer(finalizer func(r *http.Request, ctx context.Context) (context.Context, error))
}
```

- `Token`: Returns the current session token string.
- `Set`: Stores a key-value pair in the session, overwriting any existing value for `key`.
- `Get`: Retrieves the value stored under `key`, or `nil` if not present.
- `Exists`: Reports whether `key` exists in the session.
- `Delete`: Removes `key` from the session.
- `Destroy`: Deletes the session from the store and expires the client cookie.
- `RenewToken`: Generates a new session token while preserving existing session data (useful after authentication changes).
- `AddFinalizer`: Registers a callback invoked before the session middleware commits the response.

### `session.Session` Database Model

```go
type Session struct {
	Token  drivers.Char
	Data   drivers.Bytes
	Expiry int64 // Unix timestamp in nanoseconds
}
```

The `Session` model maps to the `"sessions"` table and defines unique and expiry indexes via `DatabaseIndexes`.

## Retrieving the Session

The `github.com/Nigel2392/mux/v2/middleware/sessions` package provides two functions for retrieving the active session:

```go
func Retrieve(r *http.Request) sessions.Session
func SessionFromContext(ctx context.Context) sessions.Session
```

- `sessions.Retrieve(r)` retrieves the `sessions.Session` from an `*http.Request`.
- `sessions.SessionFromContext(ctx)` retrieves the `sessions.Session` directly from a `context.Context`.

## Example Usage

The following handlers demonstrate reading and writing session values alongside template rendering (see the [rendering](../rendering.md) and [context](../context.md) documentation for template setup details):

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
)

func Index(w http.ResponseWriter, r *http.Request) {
	sess := sessions.Retrieve(r)
	fmt.Println(sess.Get("page_key"))
	sess.Set("page_key", "Last visited the index page")

	requestContext := ctx.RequestContext(r)
	if err := tpl.FRender(w, requestContext, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func About(w http.ResponseWriter, r *http.Request) {
	sess := sessions.Retrieve(r)
	fmt.Println(sess.Get("page_key"))
	sess.Set("page_key", "Last visited the about page")

	requestContext := ctx.RequestContext(r)
	html, err := tpl.Render(requestContext, "mycustomapp", "mycustomapp/about.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(html))
}
```

On the first visit to `Index`, `sess.Get("page_key")` prints `<nil>`. On a subsequent visit to `About`, it prints `Last visited the index page`, and then updates the stored value to `Last visited the about page`.

## `QueryStore`

When using a database, `contrib/session` uses `QueryStore`, which implements `scs.CtxStore` via Go-Django's ORM:

- `session.NewQueryStore(db drivers.Database) *QueryStore`: Creates a `QueryStore` with a background cleanup goroutine running every 30 minutes to delete expired sessions.
- `session.NewQueryStoreWithCleanupInterval(db drivers.Database, cleanupInterval time.Duration) *QueryStore`: Creates a `QueryStore` with a custom cleanup interval; passing `0` (or negative) disables the background cleanup goroutine.
- `Find` / `FindCtx`, `Commit` / `CommitCtx`, `Delete` / `DeleteCtx`, `All` / `AllCtx`: Standard `scs.Store` and `scs.CtxStore` methods.
- `StopCleanup()`: Terminates the background cleanup goroutine.

## Serialization and Caveats

- **`encoding/gob` registration**: By default, `scs.SessionManager` serializes session payloads with `encoding/gob`. Primitive Go types work automatically, while any custom structs or concrete interface implementations stored via `session.Set(...)` must be registered with `gob.Register(...)` (typically in `init()`). See the [alexedwards/scs](https://github.com/alexedwards/scs/v2) documentation for details.
- **`QueryStore.StopCleanup` usage**: When `cleanupInterval > 0`, `QueryStore` initializes an unbuffered `stopCleanup` channel inside the cleanup goroutine. Calling `StopCleanup()` more than once will block indefinitely because the goroutine exits after receiving the first signal, and calling it immediately before the goroutine assigns the channel may be a no-op. For short-lived instances or tests, prefer `NewQueryStoreWithCleanupInterval(db, 0)`.
- **Custom registered drivers**: Drivers outside the four built-in types (`sqlite3`, `mysql`, `mariadb`, `postgres`) that are registered with `drivers.Retrieve` use `QueryStore` with a warning log, but do not automatically create the `sessions` table when `migrator` is disabled. Ensure the table is migrated or created beforehand.
