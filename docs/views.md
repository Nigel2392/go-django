# Views

Views handle HTTP requests and produce responses in Go-Django. Rather than relying solely on standalone functions, the `views` package (`github.com/Nigel2392/go-django/src/views`) uses composable Go structs, optional interfaces, and mixins inspected by `views.Serve` and `views.Invoke`.

## The `View` Interface

At minimum, a view must implement `views.View`:

```go
type View interface {
    // ServeXXX is a placeholder method that is never invoked directly.
    // Implement ServeGET, ServePOST, etc., or template/control interfaces instead.
    ServeXXX(w http.ResponseWriter, req *http.Request)
}
```

Implementing `ServeGET`, `ServePOST`, `ServePUT`, etc., binds the handler to that HTTP verb:

```go
package myapp

import "net/http"

type IndexView struct{}

func (v *IndexView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *IndexView) ServeGET(w http.ResponseWriter, req *http.Request) {
    w.Write([]byte("Hello, World!"))
}
```

---

## `Serve<METHOD>` Signatures and Critical Behavior

`views.Invoke` supports two `Serve<METHOD>` signatures (plus the `HttpContextHandler` fallback):

1. `Serve<METHOD>(w http.ResponseWriter, req *http.Request)` (`http.HandlerFunc`)
2. `Serve<METHOD>(w http.ResponseWriter, req *http.Request, context ctx.Context) error` (`views.ContextHandlerFunc`)
3. `ServeHTTP(w http.ResponseWriter, req *http.Request, context ctx.Context)` (`views.HttpContextHandler`)

> [!IMPORTANT]
> **Critical behavior of `Serve<METHOD>`:**
>
> 1. **No automatic template rendering:** If a view defines `Serve<METHOD>(w, req)` or `Serve<METHOD>(w, req, context) error` (or `HttpContextHandler.ServeHTTP`), `views.Invoke` invokes that method and **returns `nil` immediately**—it does **not** fall through to `TryServeTemplateView` to render `GetTemplate` / `GetTemplates`.
> 2. **Returned errors from `Serve<METHOD>(w, req, context) error` are discarded:** `views.MethodServeContext` does not inspect or propagate the `error` returned by `ContextHandlerFunc`. Returning an error from `ServeGET(w, req, context)` will **not** invoke `ErrorHandler.HandleError` or trigger an HTTP error page. Handle errors explicitly inside your method (for example by calling `except.Fail` or `v.HandleError(...)`).

To let Go-Django automatically render templates via `TryServeTemplateView`, **omit** `Serve<METHOD>` and populate data in `GetContext` (as `views.BaseView` does), or invoke `views.TryServeTemplateView` / `tpl.FRender` directly from inside your `Serve<METHOD>` method:

```go
type MyTemplateView struct {
    views.BaseView
}

func (v *MyTemplateView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *MyTemplateView) GetTemplate(req *http.Request) string {
    return "mycustomapp/index.tmpl"
}

func (v *MyTemplateView) GetContext(req *http.Request) (ctx.Context, error) {
    c := ctx.RequestContext(req)
    c.Set("Title", "Home Page")
    return c, nil
}
```

---

## Core View Interfaces and Execution Pipeline

All view interfaces live in `github.com/Nigel2392/go-django/src/views`:

```go
type View interface {
    ServeXXX(w http.ResponseWriter, req *http.Request)
}

type ControlledView interface {
    View
    TakeControl(w http.ResponseWriter, req *http.Request, v View)
}

type MethodsView interface {
    View
    Methods() []string
}

type BindableView interface {
    View
    Bind(w http.ResponseWriter, req *http.Request) (View, error)
}

type ContextGetter interface {
    View
    GetContext(req *http.Request) (ctx.Context, error)
}

type TemplateGetter interface {
    GetTemplate(req *http.Request) string
}

type TemplatesGetter interface {
    GetTemplates(req *http.Request) []string
}

type TemplateKeyer interface {
    GetBaseKey() string
}

type SetupView interface {
    View
    Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request)
}

type ErrorHandler interface {
    HandleError(w http.ResponseWriter, req *http.Request, err error, code int)
}

type Renderer interface {
    View
    ContextGetter
    Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error
}

type TemplateRenderer interface {
    Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error
}

type TemplatesRenderer interface {
    Render(w http.ResponseWriter, req *http.Request, templateName []string, context ctx.Context) error
}

type Checker interface {
    View
    Check(w http.ResponseWriter, req *http.Request) error
    Fail(w http.ResponseWriter, req *http.Request, err error)
}

type ViewMixins interface {
    View
    mixins.Definer[View]
}
```

### How `views.Serve` and `views.Invoke` Work

Wrap any `View` as a standard `http.Handler` with `views.Serve(view)`, or call `views.Invoke(view, w, req, allowedMethods...)` directly:

1. **Error handler selection:** Uses `ErrorHandler.HandleError` if `view` implements `ErrorHandler`, otherwise defaults to `except.Fail(code, err)`.
2. **Method check:** If `allowedMethods` is non-empty and `req.Method` is not in it, logs and invokes the error handler with `http.StatusMethodNotAllowed`.
3. **Mixin and binding loop:** Walks `mixins.Mixins` on the view (`Checker.Check` / `Fail`, then `SetupView.Setup`). If `Setup` returns a `nil` `ResponseWriter` or `*http.Request`, `Invoke` stops immediately. If the view implements `BindableView`, `Bind` creates a per-request bound view and the loop repeats on the bound view.
4. **`ControlledView`:** If the original view implements `ControlledView`, `TakeControl(w, req, boundView)` runs and `Invoke` returns `nil`.
5. **`Serve<METHOD>(w, req)`:** Runs `MethodServe`; if matched, returns `nil`.
6. **Context preparation:** Calls `GetViewContext(req, boundView)` (using `ContextGetter` or falling back to `ctx.RequestContext(req)`), setting `"Request"` and `"View"` on the context.
7. **`Serve<METHOD>(w, req, ctx)` / `HttpContextHandler`:** Runs `MethodServeContext`; if matched, returns `nil`.
8. **Automatic template rendering:** Calls `TryServeTemplateView(w, req, []View{boundView, original}, context)`, checking `Renderer`, `TemplatesGetter`/`TemplateGetter`, `TemplateKeyer`, and `TemplatesRenderer`/`TemplateRenderer` before calling `tpl.FRender`.

> [!WARNING]
> **Caveat on `MethodsView` with `views.Serve`:** When `views.Serve(view)` inspects a `MethodsView`, it first appends `view.Methods()` into `allowedFnMethods`, and then attempts to build `allowedMethods` by keeping only entries from `view.Methods()` that are **not** already in `allowedFnMethods`. Because all entries were just added, `allowedMethods` ends up empty, so `views.Serve` does not enforce `Methods()` automatically. If you need strict HTTP method enforcement, route by HTTP verb (`m.Get(...)`, `m.Post(...)`) or call `views.Invoke(view, w, req, view.Methods()...)`. Also note that `views.Serve` checks `Serve<METHOD>` using `http.HandlerFunc`, not `ContextHandlerFunc`.

---

## Serving a View

```go
import (
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/mux/v2"
)

myCustomApp.Routing = func(m mux.Multiplexer) {
    m.Handle(mux.GET, "/", views.Serve(&IndexView{}), "index")
}
```

---

## Built-in Generic Views

### `views.BaseView`

`views.BaseView` implements `View`, `MethodsView`, `ContextGetter`, `TemplatesGetter`, and `TemplatesRenderer`:

```go
var AboutView = &views.BaseView{
    AllowedMethods:  []string{http.MethodGet},
    BaseTemplateKey: "mycustomapp",
    TemplateName:    "mycustomapp/about.tmpl", // string or []string
    GetContextFn: func(req *http.Request) (ctx.Context, error) {
        c := ctx.RequestContext(req)
        c.Set("Title", "About")
        return c, nil
    },
}
```

### `views.DetailView[T]`

`DetailView[T]` (`src/views/detail.go`) implements `ControlledView` and `BindableView` to fetch and render a single object looked up from a URL route variable:

- `ContextName`: Template context key for the fetched object (defaults to `"object"` in `Setup`).
- `URLArgName`: URL route variable name passed to `mux.Vars(req).Get(...)` (defaults to `"primary"` in `Setup`; returns `views.ErrMissingArg` if empty).
- `GetObjectFn`: `func(req *http.Request, urlArg string) (T, error)`
- `ChangeContextFn`: Optional hook to modify or wrap the context after setting `ContextName`.
- `PostMethod`: Optional handler hook run on `POST` before template rendering.
- `OnErrorFn`: Custom error callback (falls back to `except.Fail(500, ...)`).

### `views.FormView[T forms.Form]`

`FormView[T]` (`src/views/form_view.go`) implements `TemplatesView` and `Checker` for handling form display, initialization, validation, and saving:

- `GetFormFn`: `func(req *http.Request) T`
- `GetInitialFn`: `func(req *http.Request) map[string]interface{}`
- `ValidFn`: `func(req *http.Request, form T) error` (runs on valid `POST` after `SaveableForm.Save()`, if implemented)
- `InvalidFn`: `func(req *http.Request, form T) error` (runs on validation failure or `ValidFn` error)
- `SuccessFn`: `func(w http.ResponseWriter, req *http.Request, form T)` (if set, short-circuits template rendering on success, typically to redirect)
- `CheckPermissions` / `FailsPermissions`: Hook into `Checker.Check` / `Checker.Fail`.

---

## List Views (`src/views/list`)

The `list` subpackage (`github.com/Nigel2392/go-django/src/views/list`) provides a generic `list.View[T attrs.Definer]` for querying, sorting, paginating, rendering, inline-editing, and exporting model lists.

### Creating a `list.View[T]`

Column headers accept `func(context.Context) string` (use `trans.S("...")` from `github.com/Nigel2392/go-django/src/core/trans`):

```go
import (
    "net/http"

    queries "github.com/Nigel2392/go-django/queries/src"
    "github.com/Nigel2392/go-django/src/core/ctx"
    "github.com/Nigel2392/go-django/src/core/trans"
    "github.com/Nigel2392/go-django/src/views/list"
)

var ViewEmails = &list.View[*UserMailProfileProxy]{
    Model:            &UserMailProfileProxy{},
    AllowedMethods:   []string{http.MethodGet},
    BaseTemplateKey:  "main",
    TemplateName:     "mailmgmt/emails/emails.tmpl",
    PageParam:        "page",
    AmountParam:      "limit",
    OrderParam:       "order",
    OrderableColumns: []string{"Email"},
    MaxAmount:        50,
    DefaultAmount:    10,

    QuerySet: func(r *http.Request) *queries.QuerySet[*UserMailProfileProxy] {
        return queries.
            GetQuerySetWithContext(r.Context(), &UserMailProfileProxy{}).
            OrderBy("User.Email")
    },

    GetContextFn: func(r *http.Request, qs *queries.QuerySet[*UserMailProfileProxy]) (ctx.Context, error) {
        c := ctx.RequestContext(r)
        c.Set("view.query", r.URL.Query().Get("search"))
        return c, nil
    },

    ListColumns: []list.ListColumn[*UserMailProfileProxy]{
        list.Column[*UserMailProfileProxy](trans.S("Email"), "Email"),
        list.BooleanFieldColumn[*UserMailProfileProxy](trans.S("Active"), "IsActive"),
    },
}
```

### Column Constructors

- `list.Column[T](header, getter)`: Generic column accepting a field name `string`, a `func(r *http.Request, defs attrs.Definitions, row T) interface{}`, or an existing `ListColumn[T]`.
- `list.FieldColumn[T](header, fieldName)` / `list.ProcessableFieldColumn[T, VAL](header, fieldName, process)`
- `list.FuncColumn[T](header, dataFn)`
- `list.BooleanColumnFunc[T](header, dataFn)` / `list.BooleanFieldColumn[T](header, fieldName)`
- `list.DateTimeColumnFunc[T](fmt, header, dataFn)` / `list.DateTimeFieldColumn[T](fmt, header, fieldName)`
- `list.HTMLColumn[T](header, getHTML)` / `list.HTMLFieldColumn[T](header, fieldName, getHTML...)`
- `list.LinkColumn[T](header, fieldName, getURL)`
- `list.TitleFieldColumn[T](wraps, getURL)`
- `list.OrderableColumn[T](wraps, orderParam)` (applied automatically for fields in `View.OrderableColumns`)
- `list.RenameColumn[T](col, fieldName)` / `list.ChangeColumnType[T1, T2](col, args...)`
- `list.RowSelectColumn[T](formName, isChecked, hasPermission, wraps, hdrAttrs, attrs)`
- `list.FieldCheckbox[T](header, fieldName, hasPermission)`
- `list.EditableColumn[T](header, list.EditableColumnConfig{...})`

### `BoundView`, `List`, `ListForm`, and Mixins

During request handling, `list.View[T]` binds into `*list.BoundView[T]` and attaches mixins:

- `list.ListObjectMixin[T]` (attached automatically): Builds `*list.List[T]`, sets `"view_list"` in the template context, and—when inline-editable columns and `list.SetAllowListEdit(ctx, true)` are active—builds `*list.ListForm[T]` (`"view_list_form"`) and saves validated `POST` submissions via `qs.BulkUpdate`.
- `list.ListExportMixin[T]`: Attach via `View.Mixins` to intercept `?_export=1` `GET` requests and stream downloads using `list.ExportJSON[T]`, `list.ExportCSV[T]`, or `list.ExportText[T]`, with `list.FieldExportColumn[T]` or custom `list.ExportColumn[T]`.

### List Context Helpers and Template Keys

Request `context.Context` helpers:

- `list.PaginatorFromContext[T](ctx)` and `list.PageFromContext[T](ctx)`
- `list.AllowListEdit(ctx)` / `list.SetAllowListEdit(ctx, allow)`
- `list.AllowListRowSelect(ctx)` / `list.SetAllowListRowSelect(ctx, allow)`

Template `ctx.Context` keys set by `list.View[T]`:

- `"view"`: The bound view (`*list.BoundView[T]`)
- `"view_paginator"`: `pagination.Pagination[T]`
- `"view_paginator_object"`: `pagination.PageObject[T]`
- `"view_max_amount"`, `"view_amount_param"`, `"view_page_param"`
- `"view_list"`: The rendered `list.StringRenderer` (`*list.List[T]`)
- `"view_list_form"`: `*list.ListForm[T]` (when editable columns are present)

---

## View Examples and Caching

For complete runnable examples covering common view patterns, see:

- [TemplateView](../examples/views/TemplateView.md)
- [DetailView](../examples/views/DetailView.md)
- [FormView](../examples/views/FormView.md)
- [ListView](../examples/views/ListView.md)
- [ListViewColumns](../examples/views/ListViewColumns.md)
- [CreateView](../examples/views/CreateView.md)
- [UpdateView](../examples/views/UpdateView.md)
- [DeleteView](../examples/views/DeleteView.md)
- [RedirectView](../examples/views/RedirectView.md)
- [JSONView](../examples/views/JSONView.md)
- [ErrorView](../examples/views/ErrorView.md)

To cache HTTP responses for views, see [Caching Views](./caching_views.md).
