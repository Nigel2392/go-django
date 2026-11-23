# Static Files

Static files are assets—such as CSS, JavaScript, fonts, and images—served directly to clients by the `staticfiles` package (`github.com/Nigel2392/go-django/src/core/filesystem/staticfiles`).

By default, Go-Django serves static files under the `/static/` URL prefix (`django.APPVAR_STATIC_URL`). Internally, `staticfiles` stores registered filesystems in a `filesystem.MultiFS`, so **the last registered matching filesystem wins** when multiple filesystems contain the same path.

## Configuration

For details on `MatchFS` and matcher functions, see [Filesystem](./filesystem.md#matchfs).

`staticfiles` exposes package-level helpers backed by `staticfiles.EntryHandler` (`*staticfiles.FileHandler`), or you can instantiate an isolated handler with `staticfiles.NewFileHandler()`.

### `AddFS(filesys fs.FS, matches func(path string) bool)`

Registers an `fs.FS` with `staticfiles.EntryHandler`. If `matches` is non-nil, `filesys` is wrapped in a `filesystem.MatchFS` to skip non-matching paths before touching the filesystem.

### `Open(name string) (fs.File, error)`

Opens `name` across the registered filesystems in reverse registration order (last registered wins).

### `Collect(fn func(path string, f fs.File) error) error`

Walks every filesystem registered in `FileHandler` and invokes `fn(path, f)` for each non-directory file. Use `Collect` to pre-hash, minify, copy, or inspect static assets at startup.

> [!NOTE]
> **Caveat on `Collect`:** `FileHandler.Collect` unwraps top-level `*filesystem.MatchFS` (`fs.FS()`) and `*filesystem.MultiFS` (`fs.FS()...`) wrappers before walking them with `fs.WalkDir`, which means matcher predicates on a top-level `MatchFS` are **not** enforced during `Collect`.

---

## Settings and Middleware Behavior

Static file serving is configured in `app.Initialize()` using these settings (see [Configuring](./configuring.md#core-application-settings)):

- `django.APPVAR_STATIC_URL` (`string`, default `"/static/"`): URL prefix where `staticfiles.EntryHandler` is mounted. Setting this to `""` falls back to `"/static/"`.
- `django.APPVAR_STATIC_ROUTE_CACHING_MAXAGE` (`int`, default `3600`): Wraps `staticfiles.EntryHandler` with `middleware.Cache(maxAge)` when `maxAge > 0`. Set to `0` (or negative) to disable cache headers.
- `django.APPVAR_STATIC_ROUTE_LOGGING_ENABLED` (`bool`, default `false`): Static requests only log (at `logger.DBG`) when **both** `django.APPVAR_DEBUG` and `django.APPVAR_STATIC_ROUTE_LOGGING_ENABLED` are `true`.

The static route is registered with `route.Preprocess(django.MarkStaticRouteMiddleware)`. As a result:

- Requests to static URLs return `true` from `django.IsStaticRouteRequest(r)`.
- `middleware.BufferMiddleware` skips response buffering for static routes.
- Any middleware wrapped in `django.NonStaticMiddleware(...)` (such as session middleware) is bypassed on static requests.

---

## Resolving Static URLs

Use `django.Static(path)` (or `app.Static(path)`) in Go code and `{{ static "..." }}` in templates:

- Relative paths (such as `"myapp/css/style.css"`) are prefixed with `APPVAR_STATIC_URL` (producing `"/static/myapp/css/style.css"`).
- Paths already beginning with `"/"` or full URLs containing a scheme/host (`"https://..."`) are returned unchanged.

---

## Example

```go
package main

import (
    "embed"
    "fmt"
    "io/fs"

    django "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/core/filesystem"
    "github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
)

//go:embed assets/**
var staticFS embed.FS

func main() {
    app := django.App(
        django.Configure(map[string]interface{}{
            django.APPVAR_ALLOWED_HOSTS: []string{"*"},
            django.APPVAR_STATIC_URL:    "/static/",
        }),
    )

    subFS := filesystem.Sub(staticFS, "assets/static")
    staticfiles.AddFS(subFS, filesystem.MatchPrefix("myapp/"))

    if err := app.Initialize(); err != nil {
        panic(err)
    }

    err := staticfiles.Collect(func(path string, f fs.File) error {
        stat, err := f.Stat()
        if err != nil {
            return err
        }
        fmt.Println("Collected", path, stat.Size())
        return nil
    })
    if err != nil {
        panic(err)
    }
}
```
