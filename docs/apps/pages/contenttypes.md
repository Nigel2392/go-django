# Page Definitions and Registration

**Note**: This documentation page was (in part) generated by ChatGPT and has been fully reviewed by [Nigel2392](github.com/Nigel2392).

Custom page types are defined via a `PageDefinition`, which extends a content type definition and includes additional settings for panel configuration and allowed parent/child types.

```go
type PageDefinition struct {
    *contenttypes.ContentTypeDefinition

    // Serve the page with a custom view.
    ServePage func(page Page) PageView

    // Panels to display when creating a new page.
    AddPanels func(r *http.Request, page Page) []admin.Panel

    // Panels to display when editing a page.
    EditPanels func(r *http.Request, page Page) []admin.Panel

    // Retrieve a page instance for the given ID.
    GetForID func(ctx context.Context, ref pages_models.PageNode, id int64) (Page, error)

    // Callbacks for node updates and deletion.
    OnReferenceUpdate       func(ctx context.Context, ref pages_models.PageNode, id int64) error
    OnReferenceBeforeDelete func(ctx context.Context, ref pages_models.PageNode, id int64) error

    // Controls for creation and hierarchy.
    DissallowCreate bool
    DisallowRoot    bool
    ParentPageTypes []string
    ChildPageTypes  []string

    // Internal maps for quick lookup.
    _parentPageTypes map[string]struct{}
    _childPageTypes  map[string]struct{}
}
```

- **Register(definition *PageDefinition)**  
  Registers a custom page definition with the Pages app. This integrates the new page type into the admin interface and routing.

- **DefinitionForType(typeName string)** and **DefinitionForObject(page Page)**  
  Retrieve the page definition based on type name or page instance.

- **ListDefinitions() / ListRootDefinitions() / ListDefinitionsForType(typeName string)**  
  Return lists of registered page definitions.

Registering a new page definition is fairly simple, an example is shown [here](./pages.md#registering-the-blog-page-model).
