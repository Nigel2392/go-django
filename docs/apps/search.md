# Search

The `search` package (`github.com/Nigel2392/go-django/contrib/search`) is a lightweight search abstraction and backend registry (it has no `AppConfig` or database tables of its own). Together with the built-in `searchqueries` backend (`github.com/Nigel2392/go-django/contrib/search/backends/searchqueries`), it provides weighted multi-field querying over ORM `QuerySet`s as well as an interface for external search indexes.

## Table of Contents

- [Enabling the Default `searchqueries` Backend](#enabling-the-default-searchqueries-backend)
- [Core Interfaces and Types](#core-interfaces-and-types)
- [Defining Searchable Models and Fields](#defining-searchable-models-and-fields)
- [Running Searches and Managing Indexes](#running-searches-and-managing-indexes)
- [How `QueriesSearchBackend` Builds Queries](#how-queriessearchbackend-builds-queries)
- [Registering Custom Search Backends](#registering-custom-search-backends)
- [Notes and Caveats](#notes-and-caveats)

---

## Enabling the Default `searchqueries` Backend

Because `contrib/search` is a library and registry rather than an `AppConfig`, you enable the default ORM backend with a blank import—typically in `main.go`:

```go
import (
	_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"
)
```

Importing `contrib/search/backends/searchqueries` runs its `init()` function, registering `&QueriesSearchBackend{}` under the name `"queries"` as the default backend.

To override the active backend globally via configuration, set `search.APPVAR_SEARCH_BACKEND`:

```go
const APPVAR_SEARCH_BACKEND = "search.backend" // string
```

```go
django.Configure(map[string]any{
	search.APPVAR_SEARCH_BACKEND: "queries",
})
```

See [Configuring](../configuring.md) for application settings.

---

## Core Interfaces and Types

```go
type SearchField interface {
	Weight() int8   // higher is more important
	Field() string  // <fieldname> or <path.to.related.fieldname>
	Lookup() string // icontains, istartswith, exact, etc. (see expr.LookupFilter)
}

type SearchableModel interface {
	attrs.Definer
	SearchableFields() []SearchField
}

type Searchable interface {
	BuildSearchQuery(fields []BuiltSearchField, query string) (Searchable, error)
}

type SearchBackend interface {
	AddToSearchIndex(ctx context.Context, models []any) (int64, error)
	RemoveFromSearchIndex(ctx context.Context, models []any) (int64, error)
	Search(fields []BuiltSearchField, query string, searchable any) (any, error)
}

type BackendDefiner interface {
	SearchBackend() SearchBackend
}

type BuiltSearchField struct {
	ModelField  attrs.FieldDefinition
	ModelMeta   attrs.ModelMeta
	SearchField SearchField
}
```

- `NewSearchField(weight int8, field string, lookup string) SearchField` creates a `SearchField`.
- `SearchableModel` lets a model declare default search fields when no explicit fields are passed to `search.Search`.
- `Searchable` allows a query object returned by `SearchBackend.Search` to perform an additional post-processing step (`BuildSearchQuery`).
- `BackendDefiner` lets an individual model supply its own `SearchBackend` instead of using the configured global backend.
- `BuildSearchFields[T attrs.Definer](model T, fields []SearchField) ([]BuiltSearchField, error)` resolves each field path (including dot-separated relation paths such as `"Author.Username"`) using `attrs.WalkMetaFields`.

---

## Defining Searchable Models and Fields

Implement `SearchableFields() []search.SearchField` on your model, or construct `[]search.SearchField` slices ad hoc at query time:

```go
package articles

import (
	"github.com/Nigel2392/go-django/contrib/search"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
)

type Article struct {
	models.Model `table:"articles"`
	ID           int64
	Title        string
	Summary      string
	Body         string
}

func (a *Article) SearchableFields() []search.SearchField {
	return []search.SearchField{
		search.NewSearchField(10, "Title", expr.LOOKUP_ICONTANS),
		search.NewSearchField(5, "Summary", expr.LOOKUP_ICONTANS),
		search.NewSearchField(1, "Body", expr.LOOKUP_ICONTANS),
	}
}
```

Lookups come from `github.com/Nigel2392/go-django/queries/src/expr` (for example `expr.LOOKUP_ICONTANS`, `expr.LOOKUP_IEXACT`, `expr.LOOKUP_ISTARTSWITH`); see [Lookups](../../queries/docs/expressions/lookups.md) for all supported lookup filters.

---

## Running Searches and Managing Indexes

```go
func Search[MODEL attrs.Definer, SEARCHABLE any](
	model MODEL,
	query string,
	searchable SEARCHABLE,
	search ...SearchField,
) (SEARCHABLE, error)

func IndexModels[T SearchableModel](ctx context.Context, models []T) (int64, error)
func RemoveModelsFromIndex[T SearchableModel](ctx context.Context, models []T) (int64, error)
```

- If `search...` arguments are omitted, `Search` uses `model.SearchableFields()` (if `model` implements `SearchableModel`). If no fields are resolved, `Search` returns `searchable, nil` unchanged.
- `Search` passes the resolved `BuiltSearchField` slice and `searchable` (such as a `*queries.QuerySet[*Article]`) to the model's `SearchBackend`.
- `IndexModels` and `RemoveModelsFromIndex` forward `models` to the resolved backend's `AddToSearchIndex` / `RemoveFromSearchIndex`.

Example querying articles with `Search`:

```go
baseQS := queries.GetQuerySet(&Article{}).WithContext(ctx).Filter("Published", true)

// Uses Article.SearchableFields() automatically:
resultsQS, err := search.Search(&Article{}, userQuery, baseQS)
if err != nil {
	return nil, err
}

rows, err := resultsQS.All()
```

Or pass explicit `SearchField` arguments to override the model defaults (this is also how [`contrib/chooser`](./chooser.md) filters modal lists):

```go
resultsQS, err := search.Search(
	&Article{},
	userQuery,
	baseQS,
	search.NewSearchField(3, "Title", expr.LOOKUP_ICONTANS),
	search.NewSearchField(1, "Author.Username", expr.LOOKUP_ISTARTSWITH),
)
```

---

## How `QueriesSearchBackend` Builds Queries

`QueriesSearchBackend` (`github.com/Nigel2392/go-django/contrib/search/backends/searchqueries`) translates `[]search.BuiltSearchField` directly into ORM expressions on the `searchable` `QuerySet`:

1. `AddToSearchIndex` and `RemoveFromSearchIndex` are no-ops returning `(0, nil)` because queries run directly against SQL tables.
2. For each `BuiltSearchField`:
   - If `Lookup()` is empty, it defaults to `expr.LOOKUP_IEXACT`.
   - It creates an `expr.When("<field>__<lookup>", query).Then(sf.SearchField.Weight())`.
   - It creates a filter predicate `expr.Expr(fieldName, lookup, query)`.
3. Using `attrutils.NewBuilder(searchable)`, it chains:
   - `.Annotate("_relevance", expr.Case(cases))`
   - `.Filter(expr.Or(orexprs...))`
   - `.OrderBy("-_relevance")`

Rows matching at least one field are returned ordered by descending `_relevance`.

---

## Registering Custom Search Backends

To plug in a custom search backend (such as full-text search or an external search service), implement `search.SearchBackend` and register it with `RegisterSearchBackend`:

```go
func GetDefaultSearchBackend() (string, bool)
func RegisterSearchBackend(name string, backend SearchBackend, setDefault ...bool)
func GetSearchBackend(name string) (SearchBackend, bool)
func GetSearchBackendForModel[T any](model T) (SearchBackend, error)
```

Backend resolution order in `GetSearchBackendForModel`:

1. If `model` implements `BackendDefiner`, call `model.SearchBackend()`.
2. Otherwise, check `django.Global.Settings` for `search.APPVAR_SEARCH_BACKEND`.
3. If unset, fall back to the backend registered with `setDefault = true` (such as `"queries"`).
4. If neither exists, return an `errors.NoDatabase` error.

---

## Notes and Caveats

- **Weight ordering in `Search` vs SQL `CASE WHEN`**: `search.Search` sorts `BuiltSearchField` in **ascending** order of `Weight()` (`int(wa - wb)`) before calling `backend.Search`. Because `QueriesSearchBackend` builds `expr.Case(cases)` in slice order, SQL `CASE WHEN` evaluates lower-weight `WHEN` clauses before higher-weight ones if a row matches multiple fields.
- **Nil receiver on `BackendDefiner` during indexing**: `IndexModels[T]` and `RemoveModelsFromIndex[T]` call `GetSearchBackendForModel(*new(T))`. If `T` is a pointer type (such as `*Article`) implementing `BackendDefiner`, `SearchBackend()` receives a `nil` pointer and must not access instance fields.
- **`expr.LOOKUP_ICONTANS` spelling**: Note the constant name in `queries/src/expr` is `expr.LOOKUP_ICONTANS` (without the second `I`).
- **Missing backend error**: Calling `search.Search` or `chooser` search without importing `contrib/search/backends/searchqueries` or registering another default backend fails with `No default search backend configured and model ... does not define one`.
