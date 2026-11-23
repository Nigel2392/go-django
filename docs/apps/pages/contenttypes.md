# Page Definitions and Registration

Custom page types are registered with the `pages` app using a `*pages.PageDefinition`, which extends Go-Django's [`*contenttypes.ContentTypeDefinition`](../../contenttypes.md) with admin panels, hierarchy constraints, prefetching, lifecycle callbacks, and custom view serving.

See the [Pages Application](./readme.md) overview and the [Example Blog App](../../examples/blog.md) for end-to-end usage.

## The `Page` Interface

Every custom page model must implement `pages.Page`:

```go
type Page interface {
	attrs.Definer
	ID() int64
	Reference() *PageNode
	Save(c context.Context) error
}
```

> [!IMPORTANT]
> `PageAppConfig.Check` verifies at startup that `ID()` and `Reference()` are defined **directly** on your page struct rather than promoted from an embedded `*pages.PageNode`. Always implement both methods explicitly on your custom type.

## `PageDefinition`

```go
type PageDefinition struct {
	*contenttypes.ContentTypeDefinition

	// Serve the page with this view instead of http.Handler on the model.
	ServePage func(page Page) PageView

	// Panels for the model when creating or editing a page in admin.
	AddPanels  func(r *http.Request, page Page) []admin.Panel
	EditPanels func(r *http.Request, page Page) []admin.Panel

	// Query for an instance of this model by its ID.
	GetForID func(ctx context.Context, ref *PageNode, id int64) (Page, error)

	// Lifecycle callbacks hooked to SignalNodeUpdated and SignalNodeBeforeDelete.
	OnReferenceUpdate       func(ctx context.Context, ref *PageNode, id int64) error
	OnReferenceBeforeDelete func(ctx context.Context, ref *PageNode, id int64) error

	// Prefetches applied when loading the specific instance (when GetForID is nil).
	Prefetch admin.Prefetch

	// Creation and tree placement rules.
	DisallowCreate  bool
	DisallowRoot    bool
	IsValidChild    func(parent Page) (bool, error)
	IsValidParent   func(child Page) (bool, error)
	ParentPageTypes []string
	ChildPageTypes  []string
}
```

### `PageDefinition` Methods

- `IsValidParentType(typeName string) bool`: Reports whether `typeName` is allowed by `ParentPageTypes` (returns `true` if `ParentPageTypes` is empty).
- `IsValidChildType(typeName string) bool`: Reports whether `typeName` is allowed by `ChildPageTypes` (returns `true` if `ChildPageTypes` is empty).
- `Label(ctx context.Context) string`: Calls `GetLabel(ctx)` if set, otherwise returns `""`.
- `Description(ctx context.Context) string`: Calls `GetDescription(ctx)` if set, otherwise returns `""`.
- `ContentType() contenttypes.ContentType`: Returns the underlying `ContentTypeDefinition.ContentType()` (or `nil` if `ContentTypeDefinition == nil`).
- `AppLabel() string`: Returns `ContentType().AppLabel()`.
- `Model() string`: Returns `ContentType().Model()`.
- `PageView(page Page) PageView`: Calls `ServePage(page)` if set, otherwise returns `nil`.

## Registration and Lookup Functions

- `Register(definition *PageDefinition)`: Registers a `PageDefinition` in the pages registry and registers its `ContentTypeDefinition` with `contenttypes`. Also connects `OnReferenceUpdate` to `SignalNodeUpdated` and `OnReferenceBeforeDelete` to `SignalNodeBeforeDelete`. Panics if `definition == nil`, `ContentObject == nil`, or the content type is already registered.
- `Specific(ctx context.Context, node *PageNode, withRelated bool) (Page, error)`: Resolves the concrete `Page` for `node`. Returns `node, nil` if `node.PageID == 0` or no definition is registered. Uses `definition.GetForID` if provided; otherwise queries `definition.ContentObject` by primary key, applying `definition.Prefetch.SelectRelated` and `definition.Prefetch.PrefetchRelated` when `withRelated` is `true`, and copies `*node` into `row.Object.Reference()`.
- `DefinitionForType(typeName string) *PageDefinition`: Returns the definition for `typeName` (resolving content-type aliases via `contenttypes.ReverseAlias`).
- `DefinitionForObject(page Page) *PageDefinition`: Returns the definition for the concrete type of `page`.
- `ListDefinitions() []*PageDefinition`: Returns all registered definitions sorted by model name and description.
- `ListRootDefinitions() []*PageDefinition`: Returns all definitions where `DisallowRoot == false`.
- `ListDefinitionsForType(typeName string) []*PageDefinition`: Returns allowed child definitions for `typeName` matching `ChildPageTypes` and each child's `ParentPageTypes` (or all definitions if `ChildPageTypes` is empty).
- `FilterCreatableDefinitions(definitions []*PageDefinition, otherChecks ...func(*PageDefinition) bool) []*PageDefinition`: Filters out definitions with `DisallowCreate == true` and any definition failing `otherChecks`.

## Example

```go
package blog

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/pages"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type BlogPage struct {
	models.Model `table:"blog_pages"`
	Page         *pages.PageNode `proxy:"true"`
	Body         string
}

func (b *BlogPage) ID() int64 {
	if b.Page == nil {
		return 0
	}
	return b.Page.PageID
}

func (b *BlogPage) Reference() *pages.PageNode {
	if b.Page == nil {
		b.Page = &pages.PageNode{}
	}
	return b.Page
}

func (b *BlogPage) Save(ctx context.Context) error {
	qs := queries.GetQuerySetWithContext(ctx, &BlogPage{}).ExplicitSave()
	if b.ID() == 0 {
		_, err := qs.Create(b)
		return err
	}
	_, err := qs.Update(b)
	return err
}

func (b *BlogPage) FieldDefs(ctx context.Context) attrs.Definitions {
	ref := b.Reference()
	return b.Model.Define(ctx, b,
		attrs.NewField(ref, "PageID", &attrs.Config{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.NewField(ref, "Title", &attrs.Config{IsEmbedded: true}),
		attrs.NewField(ref, "Slug", &attrs.Config{IsEmbedded: true, Blank: true}),
		attrs.NewField(ref, "UrlPath", &attrs.Config{IsEmbedded: true, ReadOnly: true}),
		attrs.NewField(b, "Body"),
	)
}

func init() {
	pages.Register(&pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject:  &BlogPage{},
			GetLabel:       trans.S("Blog Page"),
			GetDescription: trans.S("A blog article page."),
		},
		AddPanels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.TitlePanel(admin.FieldPanel("Title")),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Body"),
			}
		},
		EditPanels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.TitlePanel(admin.FieldPanel("Title")),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Body"),
			}
		},
	})
}
```

## Caveats

- **`IsValidChild` and `IsValidParent` are not currently called**: Hierarchy constraints in `AddChildren`, `MoveNode`, `ListDefinitionsForType`, and admin page creation enforce `ParentPageTypes`, `ChildPageTypes`, and `DisallowRoot` (via `IsValidParentType` and `IsValidChildType`). The function fields `IsValidChild` and `IsValidParent` on `PageDefinition` are never invoked by `contrib/pages`.
