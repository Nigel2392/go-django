# Filters

The `filters` package (`github.com/Nigel2392/go-django/contrib/filters`) is a pure Go generic library for building form-backed `queries.QuerySet` filters. It has no `AppConfig`, database tables, or routes of its own—you can use it standalone in any HTTP view or attach filter specs directly to `admin.ListViewOptions.Filters`.

## Table of Contents

- [Core Types and API](#core-types-and-api)
- [Implementing `FilterSpec` with `BaseFilterSpec`](#implementing-filterspec-with-basefilterspec)
- [Standalone View Usage](#standalone-view-usage)
- [Admin `ListViewOptions.Filters` Integration](#admin-listviewoptionsfilters-integration)
- [Notes and Caveats](#notes-and-caveats)

---

## Core Types and API

`contrib/filters` validates query parameters through a `*forms.BaseForm`, then runs each matching `FilterSpec[T]` sequentially against a `*queries.QuerySet[T]`:

```go
type FilterSpec[T attrs.Definer] interface {
	Name() string
	Field(r *http.Request) fields.Field
	Filter(r *http.Request, value interface{}, object *queries.QuerySet[T]) (*queries.QuerySet[T], error)
}

type BaseFilterSpec[T any] struct {
	SpecName  string
	FormField fields.Field
	Apply     func(req *http.Request, value interface{}, object T) (T, error)
}

type Filters[T attrs.Definer] struct {
	// contains filtered or unexported fields
}

var ErrForm = errors.New("form is not valid")

func NewFilters[T attrs.Definer](ctx context.Context, formPrefix string, specs ...FilterSpec[T]) *Filters[T]
func (f *Filters[T]) Add(spec FilterSpec[T])
func (f *Filters[T]) Specs() *orderedmap.OrderedMap[string, FilterSpec[T]]
func (f *Filters[T]) Form() *forms.BaseForm
func (f *Filters[T]) EditContext(key string, c ctx.Context)
func (f *Filters[T]) Filter(r *http.Request, data map[string][]string, object *queries.QuerySet[T]) (*queries.QuerySet[T], error)
```

- `NewFilters(ctx, formPrefix, specs...)` creates a `*Filters[T]` backed by a `*forms.BaseForm` with an optional prefix (for example `"filters"`, which prefixes HTML input names as `"filters-<SpecName>"`).
- `Add(spec)` appends or replaces a `FilterSpec[T]` by `spec.Name()`.
- `Form()` returns the underlying `*forms.BaseForm` for rendering inputs, labels, and errors in templates.
- `EditContext(key, c)` implements `ctx.Editor`, setting `c.Data()[key]` to `*Filters[T]` and `c.Set("<key>_form", f.Form())`.
- `Filter(r, data, qs)` resets the form, adds every spec's `Field(r)`, binds `data` (typically `r.URL.Query()`), runs `forms.IsValid`, and applies each spec's `Filter(r, cleanedVal, qs)`. If form validation fails, it returns the original `qs` alongside an error wrapping `ErrForm` and all field/form validation errors.

---

## Implementing `FilterSpec` with `BaseFilterSpec`

Because `FilterSpec[T]` expects `*queries.QuerySet[T]` as the filtered object, instantiate `BaseFilterSpec` with `T = *queries.QuerySet[YourModel]` (or `*queries.QuerySet[attrs.Definer]` for admin list views):

```go
searchSpec := &filters.BaseFilterSpec[*queries.QuerySet[attrs.Definer]]{
	SpecName:  "search",
	FormField: fields.CharField(fields.HelpText(trans.S("Search by title or path"))),
	Apply: func(req *http.Request, value interface{}, qs *queries.QuerySet[attrs.Definer]) (*queries.QuerySet[attrs.Definer], error) {
		if fields.IsZero(value) {
			return qs, nil
		}

		return qs.Filter(expr.Or(
			expr.Q("Title__icontains", value),
			expr.Q("Path__icontains", value),
		)), nil
	},
}
```

Always check `fields.IsZero(value)` inside `Apply` so empty query parameters leave the `QuerySet` unfiltered.

See [Form Fields](../forms/fields.md) and [Expressions](../../queries/docs/expressions/expressions.md) for available form fields and `QuerySet` filter expressions.

---

## Standalone View Usage

You can use `Filters[T]` directly inside any HTTP handler and pass its form to a template:

```go
package articles

import (
	"errors"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/filters"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

func ListArticles(w http.ResponseWriter, r *http.Request) {
	f := filters.NewFilters[*Article](
		r.Context(),
		"filter",
		&filters.BaseFilterSpec[*queries.QuerySet[*Article]]{
			SpecName:  "title",
			FormField: fields.CharField(fields.Label("Title")),
			Apply: func(req *http.Request, value interface{}, qs *queries.QuerySet[*Article]) (*queries.QuerySet[*Article], error) {
				if fields.IsZero(value) {
					return qs, nil
				}
				return qs.Filter("Title__icontains", value), nil
			},
		},
		&filters.BaseFilterSpec[*queries.QuerySet[*Article]]{
			SpecName:  "published",
			FormField: fields.BooleanField(fields.Label("Published only")),
			Apply: func(req *http.Request, value interface{}, qs *queries.QuerySet[*Article]) (*queries.QuerySet[*Article], error) {
				if published, ok := value.(bool); ok && published {
					return qs.Filter("Published", true), nil
				}
				return qs, nil
			},
		},
	)

	qs := queries.GetQuerySet(&Article{}).WithContext(r.Context())
	filteredQS, err := f.Filter(r, r.URL.Query(), qs)
	if err != nil && !errors.Is(err, filters.ErrForm) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := filteredQS.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tplCtx := ctx.RequestContext(r)
	f.EditContext("filters", tplCtx) // exposes "filters" and "filters_form"
	tplCtx.Set("articles", rows)

	if err := tpl.FRender(w, tplCtx, "articles", "articles/list.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

In your template, render `filters_form` inside a `method="GET"` form:

```html
<form method="get">
    {{ $form := .Get "filters_form" }}
    {{ range $field := $form.Fields }}
        <div class="form-field">
            {{ $field.Label }}
            {{ $field.Field }}
            {{ range $err := $field.Errors }}
                <p class="error">{{ $err }}</p>
            {{ end }}
        </div>
    {{ end }}
    <button type="submit">Apply</button>
</form>
```

---

## Admin `ListViewOptions.Filters` Integration

In `contrib/admin`, `admin.ListViewOptions.Filters` accepts `[]filters.FilterSpec[attrs.Definer]`:

```go
ListView: admin.ListViewOptions{
	Filters: []filters.FilterSpec[attrs.Definer]{
		&filters.BaseFilterSpec[*queries.QuerySet[attrs.Definer]]{
			SpecName:  "search",
			FormField: fields.CharField(fields.HelpText(trans.S("Search by title or URL path"))),
			Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[attrs.Definer]) (*queries.QuerySet[attrs.Definer], error) {
				if fields.IsZero(value) {
					return object, nil
				}
				return object.Filter(expr.Or(
					expr.Q("Title__icontains", value),
					expr.Q("Path__icontains", value),
				)), nil
			},
		},
	},
}
```

When `Filters` is non-empty, `contrib/admin` automatically builds `filters.NewFilters[attrs.Definer](r.Context(), "filters")`, binds `r.URL.Query()`, filters the list `QuerySet`, and renders a filter side panel (`SidePanelFilters`).

See [Admin Models](./admin/models.md) and [Views](../views.md) for more list view options.

---

## Notes and Caveats

- **`Apply` must be non-nil**: `BaseFilterSpec.Filter` calls `logger.Fatalf(1, ...)` if `Apply == nil`, terminating the process. Always provide an `Apply` function.
- **Form prefixing**: Passing a non-empty `formPrefix` to `NewFilters` scopes query parameter names (e.g., `"filters"` expects `?filters-search=foo`), avoiding collisions with pagination parameters such as `page` or `amount`.
- **Re-adding dynamic fields safely**: `Filters.Filter` resets the internal form and rebuilds `FormFields` from each spec's `Field(r)` on every call, so a single `*Filters[T]` instance can be reused across requests if `Field(r)` returns a fresh field instance per call; however, avoid sharing mutable field state across concurrent requests.
- **Validation failure behavior in `contrib/admin`**: In the built-in admin list view, any error returned by `filterForm.Filter(...)` (including `filters.ErrForm`) fails with HTTP `500` (`except.AssertNil`). Keep filter fields optional (`fields.Required(false)`) unless strict rejection is desired.
