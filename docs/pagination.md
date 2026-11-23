# Pagination

The `pagination` package (`github.com/Nigel2392/go-django/src/core/pagination`) provides generic pagination over arbitrary slices (`Paginator[S, E]`) as well as database-backed `queries.QuerySet[T]` pagination (`QueryPaginator[T]`), complete with HTML/Templ navigation rendering.

## Core Interfaces

```go
package pagination

type PageObject[T any] interface {
    Count() int
    Results() []T
    PageNum() int
    HasNext() bool
    HasPrev() bool
    Next() int
    Prev() int
    Paginator() Pagination[T]
}

type Pagination[T any] interface {
    BaseURL() string
    Count() (int, error)
    Page(n int) (PageObject[T], error)
    NumPages() (int, error)
    PerPage() int
}
```

- `PageObject[T].Count`: Number of items on this specific page (may be smaller than `PerPage()`).
- `PageObject[T].Next` / `Prev`: Next/previous page number, or `-1` if none.
- `Pagination[T].BaseURL`: Base URL prepended to page links when rendering HTML navigation.
- `NewPageObject[T](ctx context.Context, paginator Pagination[T], num int, results []T) PageObject[T]`: Constructs a concrete page object implementing `PageObject[T]` as well as `FirstPage()`, `LastPage()`, `WithAttrs(attrs map[string]any)`, `HTML(...)`, and `Component(...)`.

---

## Slice-Based `Paginator[S, E]`

`Paginator` requires two type parameters (`S ~[]E, E any`)—the slice type and its element type:

```go
package myapp

import (
    "errors"
    "net/http"

    qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
    "github.com/Nigel2392/go-django/src/core/ctx"
    "github.com/Nigel2392/go-django/src/core/pagination"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
    c := ctx.RequestContext(r)

    paginator := pagination.Paginator[[]MyObject, MyObject]{
        Context: r.Context(),
        URL:     r.URL.Path,
        Amount:  15,
        GetObject: func(obj MyObject) MyObject {
            return obj
        },
        GetObjects: func(amount, offset int) ([]MyObject, error) {
            return getMyObjects(amount, offset)
        },
        GetCount: func() (int, error) {
            return getMyObjectTotalCount()
        },
    }

    pageNum := max(1, pagination.GetPageNum(r.URL.Query().Get("page")))

    page, err := paginator.Page(pageNum)
    if err != nil && !errors.Is(err, qerrors.NoRows) {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    c.Set("paginator", page)
}
```

- `Context`: Context passed into the resulting `PageObject` (used for translation and Templ rendering).
- `URL`: Base URL returned by `BaseURL()`.
- `Amount`: Number of items per page (if `0`, `Page` returns `errors.ValueError`).
- `GetObject`: Optional transform applied to each item returned by `GetObjects`.
- `GetObjects`: Callback receiving `(amount, offset)` to fetch the slice for the requested page.
- `GetCount`: Callback returning the total item count (cached after first lookup; panics if `nil`).

---

## Queryset-Based `QueryPaginator[T]`

Use `QueryPaginator[T attrs.Definer]` to paginate a `*queries.QuerySet[T]` directly with `Offset`, `Limit`, and `IterAll()`:

```go
paginator := &pagination.QueryPaginator[*Article]{
    Context: r.Context(),
    URL:     r.URL.Path,
    Amount:  20,
    BaseQuerySet: func() *queries.QuerySet[*Article] {
        return queries.GetQuerySet(&Article{}).OrderBy("-CreatedAt")
    },
}

pageNum := max(1, pagination.GetPageNum(r.URL.Query().Get("page")))
page, err := paginator.Page(pageNum)
```

- `BaseQuerySet`: Factory returning the base `*queries.QuerySet[T]`.
- `GetCount`: Optional custom count function `func(*queries.QuerySet[T]) (int64, error)`; defaults to `qs.Count()`.
- `GetObject`: Optional per-row wrapper/modifier `func(T) T`.

> [!WARNING]
> **Caveat on `QueryPaginator.BaseQuerySet`:** Always set `BaseQuerySet` to a non-nil function. When `BaseQuerySet` is `nil`, `GetQuerySet()` calls `qs.Context()` while `qs` is still `nil`, causing a panic.

---

## Page Numbers and Empty Pages

### `GetPageNum`

`pagination.GetPageNum(n any) int` parses integers, unsigned integers, pointers, or numeric strings into an `int`.

> [!WARNING]
> **Caveat on `GetPageNum` returning `0`:** For empty strings (`""`), unparseable strings, or unsupported types, `GetPageNum` returns `0`. Passing `0` to `paginator.Page(0)` calculates `offset = (0 - 1) * amount = -amount`. Always clamp with `max(1, pagination.GetPageNum(...))` before calling `Page`.

### Empty Pages (`drivers/errors.NoRows`)

When a page query yields zero results (`len(results) == 0`), both `Paginator.Page` and `QueryPaginator.Page` return a non-nil empty `PageObject[T]` (`nullPageObject`) paired with `github.com/Nigel2392/go-django/queries/src/drivers/errors.NoRows`. Check `errors.Is(err, errors.NoRows)` so empty result sets can still render cleanly without treating them as server errors.

---

## HTML and Templ Rendering

The concrete `pageObject[T]` returned by `Page` and `NewPageObject` provides helpers beyond the `PageObject[T]` interface:

- `FirstPage() int`: Always returns `1`.
- `LastPage() int`: Returns total page count from `NumPages()` (or `1` on error).
- `WithAttrs(attrs map[string]any)`: Attaches HTML attributes (such as `class` or HTMX `hx-*` attributes) to rendered `<a>` links.
- `Component(queryParam string, numPageNumbers int, queryParams url.Values) templ.Component`: Renders a Templ pagination bar (returns `nil` if total pages `<= 1`), preserving existing `queryParams` while updating `queryParam`.
- `HTML(queryParam string, numPageNumbers int, queryParams url.Values) template.HTML`: Renders `Component(...)` into `template.HTML` for Go `html/template` pages.

```html
{{ $page := (.Get "paginator") }}

{{ range $item := $page.Results }}
    <div class="list-item">
        <div class="list-item__header">
            {{ $item.Label }}
        </div>
    </div>
{{ else }}
    <div class="list-item">
        <div class="list-item__body">
            <p class="list-empty-text">{{ T "No items found" }}</p>
        </div>
    </div>
{{ end }}

{{ $page.HTML "page" 5 .Request.URL.Query }}
```
