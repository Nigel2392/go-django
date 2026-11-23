# HTTP Integration

`queries` provides two HTTP helper packages for integrating database transactions and model lookups with Go-Django handlers and `github.com/Nigel2392/mux`:

- `github.com/Nigel2392/go-django/queries/src/middleware`: per-request database transaction middleware (`TransactionMiddleware`)
- `github.com/Nigel2392/go-django/queries/src/views`: generic detail view helper (`ObjectView[T]`)

---

## Transaction Middleware

`TransactionMiddleware` wraps each incoming HTTP request in a database transaction via `queries.StartTransaction(req.Context())`, injects the transaction context into `req`, calls the next handler, commits when `ServeHTTP` returns, and defers a rollback.

```go
package middleware

type TransactionOptions struct {
	Deadline time.Duration
}

func TransactionMiddleware(opts TransactionOptions) func(next mux.Handler) mux.Handler
```

### Usage

Because the transaction is attached to `req.Context()`, any `QuerySet` bound via `.WithContext(r.Context())` or `queries.GetQuerySetWithContext(r.Context(), model)` participates in the request transaction automatically (see [QuerySet](./queryset/queryset.md)).

```go
package main

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	qmiddleware "github.com/Nigel2392/go-django/queries/src/middleware"
	"github.com/Nigel2392/mux"
)

func routes(r *mux.Mux) {
	r.Use(qmiddleware.TransactionMiddleware(qmiddleware.TransactionOptions{}))

	r.Post("/orders/create", func(w http.ResponseWriter, req *http.Request) {
		order := &Order{Status: "pending"}
		_, err := queries.GetQuerySetWithContext(req.Context(), order).Create(order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}
```

### Caveats

- **`TransactionOptions.Deadline` is currently ignored:** `TransactionMiddleware` constructs the middleware wrapper without copying `opts` onto its struct field, so `opts.Deadline` has no effect. If you need a per-request deadline, apply `context.WithTimeout` inside your own middleware or handler before `StartTransaction`.
- **Commit and rollback behavior:** `TransactionMiddleware` calls `tx.Commit(ctx)` unconditionally after `next.ServeHTTP` returns, and always defers `tx.Rollback(ctx)`. If a handler panics or invokes `except.Fail` (see [Errors](../../docs/errors.md)), the deferred rollback runs instead of committing; on normal returns where commit succeeds, the deferred rollback may log an already-finished transaction error depending on the SQL driver.
- **Error reporting uses `except.Fail`:** If `StartTransaction` or `tx.Commit` fails, the middleware logs the error and calls `except.Fail(500, "Internal Server Error")`.

---

## Generic `ObjectView[T]`

`github.com/Nigel2392/go-django/queries/src/views` defines `ObjectView[T]`, a generic detail view built on `django_views.BaseView` (`type BaseView = django_views.BaseView`) that resolves a model instance from a URL path parameter and populates template context. See [Views](../../docs/views.md) for how `BaseView`, `Checker`, and `ContextGetter` execute during a request.

```go
package views

type BaseView = django_views.BaseView

type ObjectView[T attrs.Definer] struct {
	BaseView
	ObjectVar   string
	ContextVar  string // optional, used to store the object in the context
	Object      T
	Permissions []string
}

func (v *ObjectView[T]) Check(w http.ResponseWriter, req *http.Request) error
func (v *ObjectView[T]) Fail(w http.ResponseWriter, req *http.Request, err error)
func (v *ObjectView[T]) GetQuerySet(r *http.Request) *queries.QuerySet[T]
func (v *ObjectView[T]) GetObject(r *http.Request, pk any) (*queries.Row[T], error)
func (v *ObjectView[T]) GetContext(req *http.Request) (ctx.Context, error)
```

### How `ObjectView[T]` Works

1. `Check` verifies `permissions.HasPermission(req, v.Permissions...)` and returns `errors.PermissionDenied` if unauthorized.
2. `GetContext` reads the URL variable `mux.Vars(req).Get(ObjectVar)` (defaulting `ObjectVar` to `"pk"`).
3. It instantiates a zero model with `attrs.NewObject[T]`, locates the model's `Primary()` field, scans the URL string into the primary key (`primary.Scan(pk)`), and calls `GetObject(req, primary.GetValue())`.
4. `GetObject` runs `v.GetQuerySet(r).Filter(primaryField.Name(), pk).Get()` (`queries.GetQuerySet(v.Object).WithContext(r.Context())`).
5. On success, `GetContext` sets two template context variables (using `ContextVar`, which defaults to `"object"`):
   - `<ContextVar>` (`"object"`): `row.Object` (`T`)
   - `<ContextVar>Row` (`"objectRow"`): `row` (`*queries.Row[T]`, giving templates access to annotations or through-model fields)

### Usage Example

```go
package articles

import (
	"github.com/Nigel2392/go-django/queries/src/views"
	django_views "github.com/Nigel2392/go-django/src/views"
)

func ArticleDetailView() django_views.View {
	return &views.ObjectView[*Article]{
		BaseView: django_views.BaseView{
			AllowedMethods: []string{"GET"},
			TemplateName:   "articles/detail.html",
		},
		ObjectVar:   "article_id",
		ContextVar:  "article", // sets "article" and "articleRow" in template context
		Object:      &Article{},
		Permissions: []string{"articles.view_article"},
	}
}
```

### Caveats

- **`Fail` returns HTTP 500 for `errors.NoRows`:** The default `ObjectView.Fail` checks `errors.Is(err, errors.PermissionDenied)` for HTTP `403`, and falls back to `except.Fail(500, err)` for all other errors — including `errors.NoRows` ([Errors](./errors.md)). Embed `*views.ObjectView[T]` and override `Fail` if you want missing records or invalid primary keys (`errors.FieldNull`) to return `404` or `400`.
- **Empty primary key requirement:** `ObjectView` requires the model to define a `Primary()` field in `FieldDefs` and assumes `mux` route variables provide `ObjectVar` (or `"pk"`).
