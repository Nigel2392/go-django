# Proxy models documentation

Proxy models are a powerful feature in Go-Django Queries that allow you to embed other models into your own model.
These models provide a way to create a model that is always joined with the source model by default, this cannot be disabled.
This might be useful for embedding models that are always used together, such as a `Page` model that is always used with a `BlogPage` model.

---

## Rules

There are however a few rules for proxy models:

* A proxy model must embed the `models.Model` struct (much like a regular model).
* A proxy model must implement the `attrs.Definer` interface.
* The top-level (embedder) model must embed the `models.Model` struct.
* The top-level model must implement the `attrs.Definer` interface.
* The top-level model must embed a pointer to the proxy model, not a value.

---

## `CanTargetDefiner` Interface

This interface can be implemented by a proxy model to define which fields
are used to create a proper join between the target model and the proxy model.

The content type and primary field are used to then generate that join.

From the database perspective, the content type field holds the content type of the target model,
and the primary field holds the primary key of the target model.

```go
type CanTargetDefiner interface {
    TargetContentTypeField() attrs.FieldDefinition
    TargetPrimaryField() attrs.FieldDefinition
}
```

## Example of defining a proxied model

```go
type Page struct {
    // Embedding Model struct is required for proxy models
    models.Model
    
    // This is the primary key for the target model
    PageID       int

    // Content type of the target model
    ContentType  *contenttypes.BaseContentType[attrs.Definer]

    // Content fields of the Page model
    ID           int
    Title        string
    Content      string
}


func (p *Page) TargetContentTypeField() attrs.FieldDefinition {
    var defs = p.FieldDefs()
    var f, _ = defs.Field("PageContentType")
    return f
}

func (p *Page) TargetPrimaryField() attrs.FieldDefinition {
    var defs = p.FieldDefs()
    var f, _ = defs.Field("PageID")
    return f
}

func (m *Page) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{
            Primary: true,
        }),
        attrs.NewField(m, "Title"),
        attrs.NewField(m, "Content"),
        attrs.NewField(m, "PageID"),
        attrs.NewField(m, "ContentType"),
    )
}

type BlogPage struct {
    // Embedding the Model struct is required for models with a proxy model
    models.Model

    // Embedding the Page model
    *Page
    
    // Extra fields for the BlogPage model
    Author string
}

func (m *BlogPage) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        // Embed the fields of the Page model.
        // These fields will be available if Page is not nil.
        fields.Embed("Page"),
        attrs.NewField(m, "Author"),
    )
}

// Optionally the proxy model can be implemented like so:

type ProxyModel struct {
    models.Model
    ID          int64
}

func (b *ProxyModel) FieldDefs() attrs.Definitions {
    return b.Model.Define(b,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
    )
}

type ProxiedModel struct {
    models.Model
    ProxyModel *ProxyModel
    ID         int64
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

func (p *ProxiedModel) FieldDefs() attrs.Definitions {
    return p.Model.Define(p,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        // Embed a O2O field with the IsProxy flag set to true.
        // This will ensure that the field is treated as a proxy model, and automatically
        // joined with the source model.
        fields.OneToOne[*ProxyModel]("ProxyModel", &fields.FieldConfig{
            IsProxy:  true,
            Nullable: false, // Proxy models cannot be nullable!
        }),
        attrs.Unbound("CreatedAt"),
        attrs.Unbound("UpdatedAt"),
    )
}
```
