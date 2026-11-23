# Chooser

The `chooser` app (`github.com/Nigel2392/go-django/contrib/chooser`) provides modal selection widgets and JSON-backed paginated list endpoints for picking related model instances in forms without requiring the admin interface.

## Standalone `contrib/chooser` vs `contrib/admin/chooser`

Go-Django provides two separate chooser packages depending on where your forms run:

| Feature | `contrib/chooser` (Standalone) | `contrib/admin/chooser` (Admin-integrated) |
| --- | --- | --- |
| App installation | `chooser.NewAppConfig()` in `django.Apps(...)` | Hooks automatically into `contrib/admin` on `OnModelsReady` |
| Route mounting | `/chooser/<<model_key>>/<<chooser_key>>/list/` (`chooser:list`) | Mounted under admin model routes (`chooser/<<chooser_key>>/list/` & `create/`) |
| Search integration | Uses [`contrib/search`](./search.md) (`[]search.SearchField`) | Uses `contrib/admin/searches` (`[]searches.SearchField`) |
| Inline creation (`CreatePage`) | Not supported (list-only) | Supported via `CreatePage *ChooserFormPage[T]` |
| Permissions / Admin requirement | Public route; does not require `admin` | Requires `admin` app and model registration in admin |

If you are building admin panels for models already registered in the [Admin App](./admin/app.md), use `github.com/Nigel2392/go-django/contrib/admin/chooser`. For public-facing forms or standalone apps, use `github.com/Nigel2392/go-django/contrib/chooser` documented below.

---

## Table of Contents

- [Installation and Routing](#installation-and-routing)
- [Registering a Chooser Definition](#registering-a-chooser-definition)
- [Configuring the List View (`ChooserListPage`)](#configuring-the-list-view-chooserlistpage)
  - [Search Integration](#search-integration)
  - [Custom List Rendering and `WrappedModel`](#custom-list-rendering-and-wrappedmodel)
- [Using `ChooserWidget` in Models and Forms](#using-chooserwidget-in-models-and-forms)
- [Notes and Caveats](#notes-and-caveats)

---

## Installation and Routing

Add `chooser.NewAppConfig` to `django.Apps(...)`. If your choosers use `SearchFields`, also blank-import a search backend such as `contrib/search/backends/searchqueries`:

```go
package main

import (
	"github.com/Nigel2392/go-django/contrib/chooser"
	_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"
	django "github.com/Nigel2392/go-django/src"
)

func main() {
	app := django.App(
		django.Apps(
			chooser.NewAppConfig,
			// ... your apps
		),
	)
	_ = app
}
```

`chooser.NewAppConfig()` initializes an app named `"chooser"` that:

- Registers embedded modal and widget templates (`AppName: "chooser"`).
- Serves static assets under `chooser/` (`chooser/css/index.css` and `chooser/js/index.js`).
- Mounts the `chooser:list` route at `/chooser/<<model_key>>/<<chooser_key>>/list/`, where `model_key` is `contenttypes.NewContentType(model).ShortTypeName()` (for example `blog.Article`).
- Listens on `core.OnDjangoReady` to validate and initialize every registered chooser via `Setup(chooserKey)`.

---

## Registering a Chooser Definition

Register choosers using `chooser.Register`:

```go
const DEFAULT_KEY = "default"

func Register(chsr chooser, key ...string)
```

If `key` is omitted, `DEFAULT_KEY` (`"default"`) is used. You can register multiple choosers for the same model under different keys (for example `"default"` and `"active_only"`). Registering a duplicate key for the same model panics.

`ChooserDefinition[T]` configures the modal title, labels, preview text, extra JSON metadata, and list page behavior:

```go
type ChooserDefinition[T attrs.Definer] struct {
	ChooserKey    string
	Model         T
	Title         any // string or func(ctx context.Context) string
	Labels        map[string]func(ctx context.Context) string
	PreviewString func(ctx context.Context, instance T) string
	ExtraData     func(ctx context.Context, instance T) map[string]any

	ListPage    *ChooserListPage[T]
	DjangoApp   djangoapps.BaseAppConfig
	ContentType contenttypes.ContentType
	MediaFn     func() media.Media
}
```

- `Model` must be a non-nil pointer to a model registered with an installed Django app (checked via `djangoapps.GetAppForModel` during `Setup`).
- `Title` is required (`string` or `func(context.Context) string`, such as `trans.S(...)`).
- `PreviewString` formats the label shown inside the widget after a record is picked; if omitted, it falls back to `attrs.ToString(instance)` or `"<Type>(<pk>)"`.
- `ExtraData` attaches arbitrary JSON-serializable key/value pairs to each selectable item (`data-chooser-data`).
- `MediaFn` lets you include additional CSS/JS assets whenever `ChooserWidget.Media()` is rendered.

---

## Configuring the List View (`ChooserListPage`)

If `ListPage` is omitted, `Setup` creates a default `ChooserListPage[T]` with `AllowedMethods: []string{"GET"}`, `PerPage: 20`, and all model fields/labels from `attrs.GetModelMeta(c.Model)`.

```go
type ChooserListPage[T attrs.Definer] struct {
	Template       string
	SearchQueryVar string
	AllowedMethods []string
	Fields         []string
	Labels         map[string]func(ctx context.Context) string
	PerPage        uint64
	Columns        map[string]list.ListColumn[T]
	Format         map[string]func(v any) any
	QuerySet       func(r *http.Request, model T) (*queries.QuerySet[T], error)
	Filter         func(qs *queries.QuerySet[T], req *http.Request, searchValue string) *queries.QuerySet[T]
	SearchFields   []search.SearchField
	NewList        func(req *http.Request, results []T, def *ChooserDefinition[T]) any
	BoundView      func(w http.ResponseWriter, req *http.Request, v *ChooserListPage[T], d *ChooserDefinition[T]) (views.View, error)
}
```

### Search Integration

`ChooserListPage` integrates directly with [`contrib/search`](./search.md). When `SearchFields` is non-empty, `chooser/views/list.tmpl` renders a search input (using query parameter `SearchQueryVar`, defaulting to `"search"`), applies `Filter` if provided, and calls `search.Search(qs.Meta().Model(), searchValue, qs, v.SearchFields...)`:

```go
package myapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/chooser"
	"github.com/Nigel2392/go-django/contrib/search"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func registerDomainChooser() {
	chooser.Register(&chooser.ChooserDefinition[*MailDomain]{
		Model: &MailDomain{},
		Title: trans.S("Choose Mail Domain"),
		PreviewString: func(ctx context.Context, d *MailDomain) string {
			return fmt.Sprintf("%s", d.DomainName)
		},
		ListPage: &chooser.ChooserListPage[*MailDomain]{
			PerPage:        15,
			SearchQueryVar: "q",
			Fields:         []string{"ID", "DomainName", "Description"},
			Labels: map[string]func(ctx context.Context) string{
				"DomainName": trans.S("Domain"),
				"q":          trans.S("Search"),
				"q...":       trans.S("Filter domains..."),
			},
			SearchFields: []search.SearchField{
				search.NewSearchField(2, "DomainName", expr.LOOKUP_ICONTANS),
				search.NewSearchField(1, "Description", expr.LOOKUP_ICONTANS),
			},
			QuerySet: func(r *http.Request, model *MailDomain) (*queries.QuerySet[*MailDomain], error) {
				return queries.GetQuerySet(model).Filter("IsActive", true).OrderBy("DomainName"), nil
			},
		},
	})
}
```

Custom search button and input placeholder labels can be overridden in `ChooserListPage.Labels` using the keys `<SearchVar>` (defaults to `"search"`) and `<SearchVar>...` (defaults to `"search..."`).

### Custom List Rendering and `WrappedModel`

If you override `ChooserListPage.NewList`, use `WrappedModel[T]`, `WrapModel`, and `WrapModels` to attach preview text and extra metadata to each item:

```go
type WrappedModel[T attrs.Definer] struct {
	Model      T
	Definition *ChooserDefinition[T]
	Context    context.Context
}

func WrapModel[T attrs.Definer](ctx context.Context, def *ChooserDefinition[T], model T) *WrappedModel[T]
func WrapModels[T attrs.Definer](ctx context.Context, def *ChooserDefinition[T], models []T) []*WrappedModel[T]
func (w *WrappedModel[T]) PreviewHTML() string
func (w *WrappedModel[T]) ExtraData() map[string]any
```

Your rendered HTML elements should include `class="godjango-chooser-list-group"` along with `data-chooser-value`, `data-chooser-preview`, and `data-chooser-data` attributes so the frontend Stimulus controller detects item clicks.

The `chooser:list` endpoint returns a JSON `ChooserResponse`:

```go
type ChooserResponse struct {
	HTML      string         `json:"html"`
	Preview   string         `json:"preview,omitempty"`
	ExtraData map[string]any `json:"data,omitempty"`
	PK        any            `json:"pk,omitempty"`
}
```

---

## Using `ChooserWidget` in Models and Forms

Construct a chooser widget with `NewChooserWidget`:

```go
func NewChooserWidget(
	model attrs.Definer,
	widgetAttrs, chooserAttrs map[string]string,
	chooserKey ...string,
) *ChooserWidget
```

- `model`: A non-nil pointer to the target model whose chooser is registered.
- `widgetAttrs`: HTML attributes applied to the hidden `<input>`.
- `chooserAttrs`: HTML attributes applied to the wrapper `<div class="chooser-widget ...">` (which also receives `data-controller="chooser"`, `data-chooser-listurl-value`, and `data-chooser-title-value`).
- `chooserKey`: Optional key matching `chooser.Register(..., key)` (defaults to `chooser.DEFAULT_KEY`).

Use `NewChooserWidget` inside a model field's `FormWidget` callback or directly on a form field:

```go
attrs.NewField(m, "Domain", &attrs.Config{
	RelForeignKey: attrs.Relate(&MailDomain{}, "", nil),
	Column:        "domain_id",
	Label:         trans.S("Mail Domain"),
	FormWidget: func(fc attrs.Config) widgets.Widget {
		return chooser.NewChooserWidget(
			fc.RelForeignKey.Model(),
			fc.WidgetAttrs,
			nil,
			chooser.DEFAULT_KEY,
		)
	},
})
```

On form submission:

- `ChooserWidget.ValueToGo` accepts an existing `attrs.Definer` or scans the submitted primary key into a new instance of `model`.
- `ChooserWidget.Validate` queries the database to confirm a matching row exists for non-zero primary key values, returning `errors.NoRows` if none is found.
- Make sure your template renders `form.Media()` (or includes `chooser/css/index.css` and `chooser/js/index.js`) so the modal JavaScript and styles load; see [Forms](../forms/readme.md) and [Widgets](../forms/widgets.md).

---

## Notes and Caveats

- **Register choosers before constructing widgets**: `NewChooserWidget` immediately looks up the chooser map by `contenttypes.NewContentType(model).ShortTypeName()` and panics if no chooser has been registered for that model yet.
- **Model must belong to a Django app**: `ChooserDefinition.Setup` runs on `core.OnDjangoReady` and returns an error if `Model` is not listed in an installed app's `ModelObjects` (or otherwise discoverable via `djangoapps.GetAppForModel`).
- **`Filter` hook requires `SearchFields`**: In `ChooserListPage.FilterQuerySet`, if `len(v.SearchFields) == 0`, the method returns the unmodified `QuerySet` early before invoking `v.Filter`. Use `QuerySet` when filtering without `SearchFields`, or configure at least one `SearchField` when using `Filter`.
- **No search backend configured**: Searching in `ChooserListPage` calls `search.Search(...)`, which requires a registered backend such as `_ "github.com/Nigel2392/go-django/contrib/search/backends/searchqueries"`; if an error occurs during search, `FilterQuerySet` falls back to returning the original `QuerySet`.
