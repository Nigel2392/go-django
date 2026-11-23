# Context Package

The `ctx` package (`github.com/Nigel2392/go-django/src/core/ctx`) provides a flexible key-value context abstraction for passing data into templates and views. It supports plain map-backed contexts, HTTP request-aware contexts, custom context editors, and an optional build-tagged struct wrapper.

## Example Usage

```go
package main

import (
    "net/http"

    "github.com/Nigel2392/go-django/src/core/ctx"
    "github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
    // Create a request-backed context
    context := ctx.RequestContext(r)
    context.Set("Title", "Welcome to My Custom App")

    // Render the index template directly to w
    if err := tpl.FRender(w, context, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
    // Create a generic map-backed context
    context := ctx.NewContext(nil)
    context.Set("Title", "About My Custom App")

    // Render the template to HTML and write bytes to the response
    html, err := tpl.Render(context, "mycustomapp", "mycustomapp/about.tmpl")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write([]byte(html))
}
```

---

## Core Interfaces

### `Context`, `Setter`, and `Getter`

```go
package ctx

type Setter interface {
    Set(key string, value any)
}

type Getter interface {
    Get(key string) any
}

type Context interface {
    Setter
    Getter
    Data() map[string]any
    Clone(values ...any) (Context, error)
}
```

- `Set`: Stores a value under `key` (or delegates to `Editor.EditContext` if `value` implements `Editor`).
- `Get`: Retrieves the value stored at `key`.
- `Data`: Returns the underlying `map[string]any`.
- `Clone`: Clones the context and merges additional alternating `key, value` pairs parsed via `TemplateDictFunc`.

### `ContextWithRequest`

```go
type ContextWithRequest interface {
    Context
    Request() *http.Request
}
```

- `Request`: Returns the `*http.Request` associated with the context. Template request processors (`tpl.RequestProcessors`) and request functions (`tpl.RequestFuncs`) only run when the template context implements `ContextWithRequest` with a non-nil `Request()`.

---

## `NewContext` and `TemplateDictFunc`

`NewContext(m map[string]any) Context` creates a standard map-backed `Context` (allocating an empty map if `m == nil`).

`TemplateDictFunc(values ...interface{}) (map[string]interface{}, error)` converts alternating `key, value` pairs into a map (requiring an even number of arguments and `string` keys). Both `Clone(...)` and template dictionary helpers rely on this format.

---

## `HTTPRequestContext` and `RequestContext`

`RequestContext(r *http.Request) *HTTPRequestContext` creates an HTTP request context backed by `ctx.NewContext(nil)` (not `StructContext`), capturing `r` and `nosurf.Token(r)`:

```go
type HTTPRequestContext struct {
    Context
    HttpRequest *http.Request
    CsrfToken   string
}

func RequestContext(r *http.Request) *HTTPRequestContext
func (c *HTTPRequestContext) Request() *http.Request
func (c *HTTPRequestContext) CSRFToken() string
func (c *HTTPRequestContext) Set(key string, value any)
func (c *HTTPRequestContext) Get(key string) any
func (c *HTTPRequestContext) Clone(values ...interface{}) (Context, error)
```

`HTTPRequestContext.Get(key)` intercepts the following special keys before falling back to `c.Context.Get(key)`:

- `"csrf_token"`, `"CsrfToken"`, `"CSRFToken"`: returns `c.CsrfToken`.
- `"request"`, `"Request"`: returns `c.HttpRequest`.

---

## `Editor` (`EditContext`)

If a value passed to `Set(key, value)` implements `ctx.Editor`, `Set` calls `value.EditContext(key, c)` and returns without storing `value` automatically:

```go
type Editor interface {
    EditContext(key string, context Context)
}
```

> [!WARNING]
> Inside `EditContext`, **never** call `context.Set(key, u)` with the same `Editor` value—doing so invokes `u.EditContext` again in infinite recursion and causes a stack overflow. Store the value directly via `context.Data()[key] = u`, and only use `context.Set(...)` for derived values that do not trigger the same `Editor`.

```go
type User struct {
    Name string
}

func (u *User) EditContext(key string, context ctx.Context) {
    context.Data()[key] = u
    context.Set("UserName", u.Name)
}
```

---

## `StructContext` (Build-Tagged)

`StructContext` (`src/core/ctx/struct_context.go`) wraps a pointer to a Go struct, exposing exported struct fields via `Get`/`Set` and storing unknown keys in a fallback map. It is compiled **only** when the `struct_context` build tag is enabled (`//go:build struct_context`):

```go
type StructContext struct {
    DeniesContext bool
    // contains filtered or unexported fields
}

func NewStructContext(obj interface{}) Context
func (c *StructContext) Object() interface{}
```

- `NewStructContext` panics if `obj` is not a pointer.
- If `!c.DeniesContext` and `obj` implements `Setter` or `Getter`, `Set`/`Get` delegate to `obj` directly.
- Otherwise, `Set` and `Get` match struct field names via reflection before falling back to an internal map.
- Enable it at build/test time with `go build -tags struct_context` or `go test -tags struct_context ./...`.

---

## Using Context in Templates

Inside Go HTML templates, retrieve keys from a `ctx.Context` with `.Get`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Get "Title" }}</title>
</head>
<body>
    {{ template "content" . }}
</body>
</html>
```
