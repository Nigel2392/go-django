# Proxy Models

A **proxy model** is a model that is **always automatically JOINed** to another model in every query. It is ideal for shared base tables — for example, a generic `Page` model that every content type (BlogPage, NewsPage, etc.) always references.

Unlike a regular FK relation that you must explicitly `Select("*", "Relation.*")` to load, proxy models are loaded **unconditionally** in every query. You cannot opt out of the join.

---

## How Proxy Models Work

When a model embeds a pointer to another model (`*ProxyModel`), and `models.Model.Define()` sees that embedded pointer field in the struct, it automatically registers it as a proxy relation in the model's metadata. From that moment on, every QuerySet for the embedding model will INNER JOIN the proxy model's table.

The JOIN condition is based on the primary key of the proxy model matching the FK column stored on the main model.

---

## Rules

1. Both the proxy model **and** the embedding model must embed `models.Model`.
2. Both models must implement `attrs.Definer`.
3. The embedding model must have a `*ProxyModel` **pointer field** (not a value, not a named field you define yourself — the embedded pointer is discovered automatically).
4. Proxy models **cannot be nullable**. The JOIN is always required.

---

## Basic Pattern

```go
type ProxyModel struct {
    models.Model
    ID          int64
    Title       string
    Description string
}

func (b *ProxyModel) FieldDefs() attrs.Definitions {
    return b.Model.Define(b,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("Title"),
        attrs.Unbound("Description"),
    )
}

// ProxiedModel automatically JOINs ProxyModel in every query
type ProxiedModel struct {
    models.Model
    *ProxyModel       // <-- embedded pointer = proxy relation
    ID        int64
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (p *ProxiedModel) FieldDefs() attrs.Definitions {
    return p.Model.Define(p,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        attrs.Unbound("CreatedAt"),
        attrs.Unbound("UpdatedAt"),
    )
    // Note: the proxy field itself is not listed here — it is discovered
    // automatically from the embedded *ProxyModel pointer.
}
```

Creating a `ProxiedModel` will also create the `ProxyModel` in the database:

```go
var obj = models.Setup(&ProxiedModel{
    ProxyModel: &ProxyModel{
        Title:       "Hello",
        Description: "World",
    },
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
})

if err := obj.Save(context.Background()); err != nil {
    log.Fatal(err)
}
```

Querying loads both models automatically:

```go
var row, err = queries.GetQuerySet(&ProxiedModel{}).
    WithContext(ctx).
    Filter("ID", obj.ID).
    First()

// Both ProxiedModel and ProxyModel fields are available
fmt.Println(row.Object.ID)
fmt.Println(row.Object.ProxyModel.Title) // "Hello"
```

---

## Filtering Across the Proxy Join

Because the JOIN is always present, you can filter on proxy model fields using the dot notation:

```go
var row, err = queries.GetQuerySet(&ProxiedModel{}).
    WithContext(ctx).
    Filter("ProxyModel.Title", "Hello").
    First()
```

---

## Alternative: Explicit Proxy Field via `fields.OneToOne`

Instead of an embedded pointer, you can also declare the proxy field explicitly using `fields.OneToOne` with `IsProxy: true`. This gives you more control over the column name:

```go
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
        fields.OneToOne[*ProxyModel]("ProxyModel", &fields.FieldConfig{
            IsProxy:  true,
            Nullable: false, // Must be false for proxy models!
        }),
        attrs.Unbound("CreatedAt"),
        attrs.Unbound("UpdatedAt"),
    )
}
```

---

## Polymorphic Proxy Models (`CanTargetDefiner`)

When the proxy model needs a **polymorphic JOIN** — i.e., the proxy table stores a content type string and a generic primary key that can point to different target models — implement `models.CanTargetDefiner`:

```go
type CanTargetDefiner interface {
    TargetContentTypeField() attrs.FieldDefinition
    TargetPrimaryField() attrs.FieldDefinition
}
```

The content type field must store the type name of the target model, and the primary field must store the target's PK. The ORM uses these to build the correct polymorphic JOIN condition.

```go
func (p *Page) TargetContentTypeField() attrs.FieldDefinition {
    defs := p.FieldDefs()
    f, _ := defs.Field("ContentType")
    return f
}

func (p *Page) TargetPrimaryField() attrs.FieldDefinition {
    defs := p.FieldDefs()
    f, _ := defs.Field("PageID")
    return f
}
```

---

## Chained Proxy Relations

Proxy models can chain further — a model that JOINs `ProxiedModel` will also transitively get the JOIN to `ProxyModel`, since the ORM walks the full proxy chain automatically:

```go
type LinkedToProxiedModel struct {
    models.Model
    ID           int64
    ProxiedModel *ProxiedModel
}

func (l *LinkedToProxiedModel) FieldDefs() attrs.Definitions {
    return l.Model.Define(l,
        attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
        fields.ForeignKey[*ProxiedModel]("ProxiedModel", "proxied_model_id"),
    )
}
```

When you query `LinkedToProxiedModel` with `Select("*", "ProxiedModel.*")`, the result will include `ProxiedModel` fields AND `ProxiedModel.ProxyModel` fields:

```go
var row, _ = queries.GetQuerySet(&LinkedToProxiedModel{}).
    WithContext(ctx).
    Select("*", "ProxiedModel.*").
    Filter("ID", id).
    First()

fmt.Println(row.Object.ProxiedModel.ProxyModel.Title) // fully traversed
```
