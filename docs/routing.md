# Routing

Routing maps HTTP methods and URL patterns to handlers and middleware using `github.com/Nigel2392/mux`. Routes for an app are typically registered inside its `Routing` function (`AppConfig.BuildRouting`).

## The `mux.Multiplexer` Interface

Both `app.Mux` and the app-scoped multiplexer passed to `AppConfig.Routing` implement `mux.Multiplexer`:

```go
type Multiplexer interface {
    Use(middleware ...Middleware)
    Handle(method string, path string, handler Handler, name ...string) *Route
    HandleFunc(method string, path string, handler func(w http.ResponseWriter, r *http.Request), name ...string) *Route
    AddRoute(route *Route)

    Any(path string, handler Handler, name ...string) *Route
    Get(path string, handler Handler, name ...string) *Route
    Post(path string, handler Handler, name ...string) *Route
    Put(path string, handler Handler, name ...string) *Route
    Patch(path string, handler Handler, name ...string) *Route
    Delete(path string, handler Handler, name ...string) *Route
}
```

Any `*mux.Route` returned by `Handle`, `HandleFunc`, `Any`, `Get`, `Post`, `Put`, `Patch`, or `Delete` also implements `mux.Multiplexer` (plus `Head` and `Options`), allowing you to nest child routes and group middleware under a parent route.

---

## URLs

### HTTP Method Helpers

- `Any`: Matches any HTTP method (`mux.ANY`).
- `Get`: Matches `GET` (`mux.GET`).
- `Post`: Matches `POST` (`mux.POST`).
- `Put`: Matches `PUT` (`mux.PUT`).
- `Patch`: Matches `PATCH` (`mux.PATCH`).
- `Delete`: Matches `DELETE` (`mux.DELETE`).

```go
myCustomApp.Routing = func(m mux.Multiplexer) {
    m.Handle(mux.GET, "/", mux.NewHandler(Index), "index")
    m.Get("/about", mux.NewHandler(About), "about")
    m.HandleFunc(mux.POST, "/contact", Contact, "contact")
}
```

### Path Variables

Use `<<name>>` segments in URL patterns to capture dynamic path parameters:

```go
m.Get("/user/<<id>>", mux.NewHandler(User), "user")
```

Retrieve captured path parameters with `mux.Vars(r)` (`mux.Variables`):

```go
type Variables map[string][]string

func (v Variables) Get(key string) string
func (v Variables) GetAll(key string) []string
func (v Variables) GetInt(key string) int
```

```go
func User(w http.ResponseWriter, r *http.Request) {
    pathVars := mux.Vars(r)
    id := pathVars.Get("id")
    // Or as an integer:
    userID := pathVars.GetInt("id")
    _, _ = id, userID
}
```

### Wildcards

Use `*` in a URL pattern to capture trailing path segments under the `"*"` variable key:

```go
m.Get("/files/*", mux.NewHandler(Files), "files")
```

```go
func Files(w http.ResponseWriter, r *http.Request) {
    pathVars := mux.Vars(r)
    pathParts := pathVars.GetAll("*")
    _ = pathParts
}
```

### Named Routes, Nesting, and URL Reversing

Pass an optional route name as the final argument to `Handle` or method helpers. Adding routes onto a parent `*mux.Route` creates nested route names separated by `:` (`mux.NAME_SEPARATOR`):

```go
myCustomApp.Routing = func(m mux.Multiplexer) {
    blog := m.Any("/blog", mux.NewHandler(BlogIndex), "blog")
    blog.Get("/post/<<slug>>", mux.NewHandler(BlogPost), "detail")
}
```

Resolve a named route back to a URL path using `django.Reverse(name, args...)` (or `app.Mux.Reverse(name, args...)` to handle errors manually):

```go
url := django.Reverse("blog:detail", "hello-world") // "/blog/post/hello-world"
```

In HTML templates, use the built-in `url` template function:

```html
<a href="{{ url "blog:detail" "hello-world" }}">Read post</a>
```

---

## Middleware

Register middleware on a `mux.Multiplexer` or `*mux.Route` with `Use(...)`:

```go
type Handler = http.Handler
type Middleware = func(Handler) Handler
```

```go
myCustomApp.Routing = func(m mux.Multiplexer) {
    m.Use(func(next mux.Handler) mux.Handler {
        return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
            // Logic before the handler
            next.ServeHTTP(w, req)
            // Logic after the handler
        })
    })
}
```

### Per-Route Preprocessing and Disabling Middleware

- `route.Preprocess(middleware...)`: Registers `PreMiddleware` on a `*mux.Route` that executes **before** `app.Mux` global middleware (used internally by `MarkStaticRouteMiddleware` for static and media routes).
- `route.RunsMiddleware(false)`: Sets `route.DisabledMiddleware = true` so `PreMiddleware`, `app.Mux` middleware, and `route.Middleware` are skipped for that route. (Outer server wrappers added in `app.Serve()`, such as `RequestSignalMiddleware` and `nosurf`, still wrap `a.Mux`.)

```go
route := m.Get("/health", mux.NewHandler(Health), "health")
route.RunsMiddleware(false)
```

---

## Default Middleware Stack

Go-Django automatically wires built-in middleware during `app.Initialize()` and `app.Serve()`:

1. **Outer wrappers applied in `app.Serve()` (`middlewareBuiltins()`):**
   - `nosurf` CSRF protection (when `APPVAR_DISABLE_NOSURF` is `false`; exempts requests with a valid `X-CSRF-Token` header and runs `django.HOOK_SETUP_NOSURF` hooks).
   - `django.RequestSignalMiddleware` (emits `core.SIGNAL_BEFORE_REQUEST` and `core.SIGNAL_AFTER_REQUEST`).
   *(Note: `Serve()` loops over `[RequestSignalMiddleware, nosurf]` wrapping `a.Mux` forward, so `nosurf` is the outermost layer when enabled, followed by `RequestSignalMiddleware`.)*
2. **Route preprocessing (`route.PreMiddleware`):**
   - Runs before `a.Mux` global middleware (for example, `MarkStaticRouteMiddleware` on static and media routes).
3. **Global `a.Mux` middleware registered in `app.Initialize()` (in execution order):**
   - `middleware.AllowedHosts` (from `django.APPVAR_ALLOWED_HOSTS`)
   - `loggerMiddleware` (logs `<method> <duration> <ip> <path>`; controlled by `APPVAR_ROUTE_LOGGING_ENABLED`, `APPVAR_DEBUG`, and `APPVAR_STATIC_ROUTE_LOGGING_ENABLED`)
   - `django.ContextDataStoreMiddleware` (attaches a per-request datastore to `r.Context()`)
   - `middleware.Recoverer(a.ServerError)` (when `django.APPVAR_RECOVERER` is `true`, the default)
   - `middleware.BufferMiddleware` (buffers non-static responses; skips buffering when `django.IsStaticRouteRequest(r)` is `true`)
4. **App namespace and per-route middleware (`route.Middleware`):**
   - App routes registered via `AppConfig.BuildRouting` also attach the owning `AppConfig` to `r.Context()` (`django.AppFromContext(r.Context())`).

---

## Request Utilities and Signals

### Client IP (`django.GetIP`)

`django.GetIP(r *http.Request) string` returns the client IP using `mux.GetIP`, automatically passing `django.APPVAR_REQUESTS_PROXIED` from settings.

### Per-Request Context Data Store

`django.ContextDataStoreMiddleware` attaches a `map[any]interface{}` datastore to every request handled through `a.Mux`:

- `django.ContextDataStoreGet(requestOrContext any, key any, defaultValue ...interface{}) interface{}`
- `django.ContextDataStoreGetOK(requestOrContext any, key any) (interface{}, bool)`
- `django.ContextDataStoreSet(requestOrContext any, key any, value interface{})`
- `django.ContextDataStoreDelete(requestOrContext any, key string) bool`
- `django.ContextDataStore(requestOrContext any) map[any]interface{}`

> [!IMPORTANT]
> `ContextDataStore*` helpers type-assert the stored `*datastore` from `r.Context()` and **will panic** if called on a request or `context.Context` that did not pass through `django.ContextDataStoreMiddleware` (for example, in unit tests invoking a handler directly without `app.Mux`).

### Skipping Middleware for Static Routes

Static and media routes run `django.MarkStaticRouteMiddleware` during `route.Preprocess(...)`:

- `django.IsStaticRouteRequest(r *http.Request) bool`: Reports whether the request was marked as a static/media route.
- `django.RequestWithStaticMark(r *http.Request, isStatic bool) *http.Request`: Sets or updates the static route mark on the request context.
- `django.NonStaticMiddleware(middleware mux.Middleware) mux.Middleware`: Wraps a middleware so it is bypassed when `IsStaticRouteRequest(r)` is `true` (used by apps such as `session`).

### Request Signals (`core.SIGNAL_BEFORE_REQUEST` / `core.SIGNAL_AFTER_REQUEST`)

`django.RequestSignalMiddleware` sends `*core.HttpSignal{W, R, H}` to `core.SIGNAL_BEFORE_REQUEST` before invoking `a.Mux`, and sends `core.SIGNAL_AFTER_REQUEST` after the handler returns.

If a `SIGNAL_BEFORE_REQUEST` listener writes a custom response and returns `django.ErrServeCanceled`, `RequestSignalMiddleware` immediately stops and skips calling the downstream handler and `SIGNAL_AFTER_REQUEST`. See [Hooks and Signals](./hooks.md) for usage examples.
