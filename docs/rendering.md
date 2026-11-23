# Rendering Templates

The `tpl` package (`github.com/Nigel2392/go-django/src/core/filesystem/tpl`) wraps Go's `html/template` with filesystem layering, per-app base templates, context processors, request-scoped template functions, and fragment overrides.

## Initial Setup

Store your templates under a namespaced directory—typically `templates/<appname>/`—and register them during `AppConfig.Init()` (or via `AppConfig.TemplateConfig`).

### Creating Templates

```bash
mkdir -p templates/mycustomapp
```

Create a base template (`templates/mycustomapp/base.tmpl`):

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Get "Title" }}</title>
</head>
<body>
    {{ template "content" . }}
</body>
</html>
```

Create child templates (`templates/mycustomapp/index.tmpl` and `templates/mycustomapp/about.tmpl`):

```html
{{ define "content" }}
    <h1>Welcome to the Index Page</h1>
{{ end }}
```

```html
{{ define "content" }}
    <h1>Welcome to the About Page</h1>
{{ end }}
```

### Registering Templates (`tpl.Add`)

Register template filesystems with `tpl.Add(tpl.Config{...})` (or set `AppConfig.TemplateConfig` so Go-Django calls `tpl.Add` automatically during initialization):

```go
//go:embed templates/**
var tplFSFull embed.FS
var tplFS = filesystem.Sub(tplFSFull, "templates")

tpl.Add(tpl.Config{
    AppName: "mycustomapp",
    FS:      tplFS,
    Bases: []string{
        "mycustomapp/base.tmpl",
    },
    Matches: filesystem.MatchAnd(
        filesystem.MatchPrefix("mycustomapp/"),
        filesystem.MatchExt(".tmpl"),
    ),
    Funcs: template.FuncMap{
        "upper": strings.ToUpper,
    },
})
```

Only files approved by `Matches` are exposed from `FS` (see [MatchFS](./filesystem.md#matchfs)). Because `tpl` uses `filesystem.MultiFS` internally, later `tpl.Add` calls with matching paths override earlier ones.

---

## Rendering Templates

Use `tpl.FRender` to write directly to an `io.Writer`, or `tpl.Render` to obtain a `template.HTML` string:

```go
func IndexHandler(w http.ResponseWriter, r *http.Request) {
    requestContext := ctx.RequestContext(r)
    requestContext.Set("Title", "My Custom App")

    if err := tpl.FRender(w, requestContext, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
    genericContext := ctx.NewContext(map[string]any{
        "Title": "About My Custom App",
    })

    html, err := tpl.Render(genericContext, "mycustomapp", "mycustomapp/about.tmpl")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write([]byte(html))
}
```

> [!IMPORTANT]
> **Base key behavior when rendering:** If you call `tpl.FRender(w, context, "mycustomapp/index.tmpl")` with a single path argument and omit `baseKey`, `normalizeTemplatePath` swaps `baseKey` to `""` and `path` to `["mycustomapp/index.tmpl"]`. When `baseKey == ""`, `getTemplateConfig` returns `nil`, meaning `Config.Bases` and per-config `Config.Funcs` are **not** applied—use this only for standalone templates that do not inherit from a base template. Always pass `baseKey` (for example `"mycustomapp"`) when your template defines blocks inside a `Bases` template.

---

## Context Processors, Overrides, and `ContextBinder`

### Global Processors and Overrides

- `tpl.Processors(funcs ...func(any))`: Runs on every non-nil template context before rendering. During `django.Initialize()`, Go-Django registers a processor that sets `"Application"` and `"Settings"` on any `ctx.Context`.
- `tpl.Overrides(funcs ...func(any) (any, error))`: Transforms or replaces the context object before template execution.
- `ContextBinder` interface (`BindContext(r *TemplateRenderer) any`): If the final context implements `ContextBinder`, `BindContext` is invoked and its return value becomes the executed template data.

### Context Processors (`RequestProcessors`)

Request processors run on every render where the context implements `ctx.ContextWithRequest` and `Request()` is non-nil:

```go
tpl.RequestProcessors(
    func(c ctx.ContextWithRequest) {
        c.Set("SiteName", "My Custom App")
    },
    func(c ctx.ContextWithRequest) {
        if c.Get("User") == nil {
            c.Set("User", &UnauthenticatedUser{})
        }
    },
)
```

---

## Template Functions

### Registering Custom Functions

- `tpl.Funcs(funcs template.FuncMap)`: Registers functions globally across all templates.
- `tpl.RequestFuncs(funcs func(*http.Request) template.FuncMap)`: Registers functions computed per request (evaluated when rendering with a `ctx.ContextWithRequest` whose `Request()` is non-nil).

> [!WARNING]
> **Caveat on `tpl.Config.RequestFuncs`:** Although `tpl.Config` exposes a `RequestFuncs` field and `MergeConfig` merges it, `TemplateRenderer` currently only evaluates global functions registered via `tpl.RequestFuncs(...)` when parsing templates for a request—per-`Config` `RequestFuncs` is not invoked. Register request-dependent template functions globally with `tpl.RequestFuncs(...)` instead.

### Built-in `tpl` Functions

`tpl.NewRenderer()` registers these functions on every renderer:

- `include(context any, baseKey string, path ...string) template.HTML`: Renders a sub-template via `tpl.Render` and returns its HTML (or the error message string if rendering fails).
- `add(x, y int) any`: Returns `x + y`.
- `sub(y, x int) any`: Returns `x - y`. Note the argument order for Go template pipelines: `{{ 10 | sub 3 }}` calls `sub(3, 10)` and evaluates to `7`.
- `safe(v any) template.HTML`: Marks a value as safe HTML (note that `django.Initialize()` overrides this with `func(s string) template.HTML`).
- `safeAttrs(v any) template.HTMLAttr`: Converts a `string`, `fmt.Stringer`, `map[string]string`, or `map[string]any` (supporting `string` and `[]string` values) into `template.HTMLAttr`.
- `safeURL(v any) template.URL`: Marks a value as a safe URL.
- `safeJS(v any) template.JS`: Marks a value as safe JavaScript.
- `safeCSS(v any) template.CSS`: Marks a value as safe CSS.

### Global Functions Added by `django.Initialize()`

During application initialization (`src/django.go`), Go-Django registers additional global `tpl.Funcs`:

- `safe(s string) template.HTML`: Wraps `s` as `template.HTML`.
- `json(v any) string`: Marshals `v` to JSON string.
- `static(path string) string`: Resolves a static asset URL via `app.Static(path)`.
- `media(path string) string`: Resolves a media URL via `app.Media(path)`.
- `repeat(str any, cnt any) template.HTML`: Repeats `str` `cnt` times.
- `make(args ...any) []any`: Builds a slice from arguments inside templates.
- `toString(v any) string`: Converts a value with `attrs.ToString(v)`.
- `pk(v attrs.Definer) any`: Returns the model's primary key (`attrs.PrimaryKey`).
- `cType(v attrs.Definer) any`: Returns `*contenttypes.ContentTypeDefinition` via `contenttypes.DefinitionForObject(v)`.
- `url(name string, args ...any) template.URL`: Reverses a named route via `a.Mux.Reverse` (panics if the route is not found or arguments are invalid).
- `field(obj attrs.Definer, name string) any`: Reads field `name` on `obj` via `attrs.Define`.
- `has_perm` / `has_object_perm`: Global functions delegating to `permissions.HasPermission` / `permissions.HasObjectPermission`.
- `component(name string, args ...interface{}) template.HTML`: Renders a registered component by name with `context.Background()`.
- `Translate(ctx any, s any, args ...any) string`: Translates string `s` using a `context.Context`, `*http.Request`, or direct message string.

### Request-Scoped Functions Added by `django.Initialize()`

When rendering with a non-nil `*http.Request`, `django.Initialize()` also provides these `tpl.RequestFuncs` (overriding non-request `has_perm`, `has_object_perm`, and `component` with request-aware variants):

- `csrf_token() template.HTML`: Returns `nosurf.Token(req)`.
- `csrf_field(cookieName ...string) template.HTML`: Renders `<input type="hidden" name="<name>" value="<token>">` (defaulting `name` to `nosurf.CookieName`).
- `request() *http.Request`: Returns the current `*http.Request`.
- `user() interface{}`: Returns `authentication.Retrieve(req)`.
- `has_perm(perm ...string) bool`: Calls `permissions.HasPermission(req, perm...)`.
- `has_object_perm(obj interface{}, perms ...string) bool`: Calls `permissions.HasObjectPermission(req, obj, perms...)`.
- `language() string`: Returns `trans.LocaleFromContext(req.Context()).String()`.
- `T(s string, args ...any) string`: Calls `trans.T(req.Context(), s, args...)`.
- `P(single, plural string, count int, args ...any) string`: Calls `trans.P(req.Context(), single, plural, count, args...)`.
- `component(name string, args ...interface{}) template.HTML`: Renders a registered component with `req.Context()`.

---

## Core `tpl` API Reference

### `Renderer`, `Config`, and `Template`

```go
type Template interface {
    Name() string
    Execute(w io.Writer, data any) error
}

type ContextBinder interface {
    BindContext(r *TemplateRenderer) any
}

type Renderer interface {
    Add(cfg Config)
    Processors(funcs ...func(any))
    Override(funcs ...func(any) (any, error))
    RequestProcessors(funcs ...func(ctx.ContextWithRequest))
    FirstRender() signals.Signal[*TemplateRenderer]
    GetTemplate(baseKey string, path ...string) Template
    GetTemplateFromString(name string, content string, funcs template.FuncMap) Template
    Render(buffer io.Writer, data any, appKey string, path ...string) error
    Funcs(funcs template.FuncMap)
    RequestFuncs(funcs func(*http.Request) template.FuncMap)
}

type Config struct {
    AppName                    string
    FS                         fs.FS
    Bases                      []string
    Matches                    func(path string) bool
    Funcs                      template.FuncMap
    RequestFuncs               func(*http.Request) template.FuncMap
    TemplateFragmentForRequest func(r *http.Request, baseTemplate string) (newBaseTemplate string, err error)
}
```

### Advanced Helpers

- `tpl.GetTemplate(baseKey string, path ...string) Template`: Resolves and returns a `Template` object without executing it yet.
- `tpl.GetTemplateFromString(name string, content string, funcs template.FuncMap) Template`: Compiles an inline template string with access to global functions and request functions.
- `tpl.MergeConfig(dst, src *Config) *Config`: Combines two `*tpl.Config` values (merging filesystems into a `MultiFS`, prepending `src.Bases`, `MatchOr`-ing matchers, chaining `TemplateFragmentForRequest`, and copying function maps).
- `Config.TemplateFragmentForRequest`: Dynamically switches the executed template block name for a given request (for example, returning a partial fragment name for HTMX requests instead of the full page base template).
- `tpl.FirstRender() signals.Signal[*TemplateRenderer]`: Fires once immediately before the very first template render executes (used internally by packages such as `forms` to lazily register widget templates and static assets).
