# Proxy Models

A **proxy model** is a model whose table is **automatically `INNER JOIN`ed** in every query for an embedding model. This pattern works well for shared base tables — for example, a `Page` table shared by `BlogPage`, `NewsPage`, and other subtypes.

Unlike a regular `ForeignKey` that requires `Select("*", "Relation.*")` or `SelectRelated("Relation")`, proxy model fields and joins are included unconditionally whenever the embedding model is queried.

---

## How Proxy Models Work

When a struct embeds `models.Model` and contains a proxy pointer field (`*ProxyModel` embedded anonymously, or a named `*ProxyModel` field tagged with `` `proxy:"true"` ``), `buildModelChain` records the proxy relationship and `m.Model.Define(ctx, m, ...)` automatically registers a `ProxyField` in the model's field definitions.

Every `QuerySet` for the outer model automatically adds the proxy's `JOIN` and selects its fields (`addProxyChain`).

---

## Rules and Struct Tags

1. Both the proxy model and the embedding model must embed `models.Model` (directly or via a value-embedded struct) and implement `attrs.Definer`.
2. In the outer struct's field declaration order, `models.Model` (or the value-embedded struct containing `models.Model`) **must appear before** any proxy pointer fields.
3. A struct field is treated as a proxy if it is a pointer to a struct (`*T`) and either:
   - it is an anonymous embedded pointer (`*ProxyModel`) whose target embeds `models.Model`, **or**
   - it is a named (or anonymous) pointer field tagged with `` `proxy:"true"` ``.
   You can opt out of automatic proxy detection on any field with `` `proxy:"false"` `` or `` `proxy:"-"` ``.
4. Proxy relations **cannot be nullable**.
5. Supported proxy struct tags (`queries/src/models/chain.go` and `model.go`):
   - `` `proxy:"true"` `` / `` `proxy:"false"` `` / `` `proxy:"-"` ``: Explicitly enables or disables proxy detection on a `*Struct` field.
   - `` `auto:"true"` ``: Automatically allocates a new instance of a nil proxy field via `attrs.NewObject` during `Model.Setup`.
   - `` `ctype:"ContentTypeField"` `` and `` `target:"TargetPKField"` ``: Field names on a polymorphic proxy model storing the embedder's content type and primary key (used when the proxy does not implement `CanTargetDefiner`).

---

## Basic Pattern (Anonymous or Named Proxy Field)

```go
package models

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type ProxyModel struct {
	models.Model
	ID          int64
	Title       string
	Description string
}

func (b *ProxyModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		attrs.Unbound("Title"),
		attrs.Unbound("Description"),
	)
}

type ProxiedModel struct {
	models.Model // Must appear BEFORE proxy fields
	*ProxyModel  // Embedded pointer = automatic proxy relation (or `Proxy *ProxyModel \`proxy:"true"\``)
	ID           int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (p *ProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		attrs.Unbound("CreatedAt"),
		attrs.Unbound("UpdatedAt"),
	)
}
```

Creating and querying `ProxiedModel` saves and loads `ProxyModel` automatically:

```go
obj := models.Setup(ctx, &ProxiedModel{
	ProxyModel: &ProxyModel{
		Title:       "Hello",
		Description: "World",
	},
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
})

if err := obj.Save(ctx); err != nil {
	return err
}

row, err := queries.GetQuerySet(&ProxiedModel{}).
	WithContext(ctx).
	Filter("ProxyModel.Title", "Hello").
	First()
if err != nil {
	return err
}

fmt.Println(row.Object.ID, row.Object.ProxyModel.Title)
```

---

## Alternative: Explicit Proxy Field via `fields.OneToOne`

You can also declare a proxy field manually inside `FieldDefs` using `fields.OneToOne` with `IsProxy: true` (for instance, to customize `ColumnName` or `AllowEdit`, without relying on struct tags):

```go
type ProxiedModel struct {
	models.Model
	ProxyModel *ProxyModel `proxy:"-"` // suppress automatic tag-based registration
	ID         int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (p *ProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		fields.OneToOne[*ProxyModel]("ProxyModel", &fields.FieldConfig{
			IsProxy:    true,
			Nullable:   false,
			AllowEdit:  true,
			ColumnName: "proxy_id",
		}),
		attrs.Unbound("CreatedAt"),
		attrs.Unbound("UpdatedAt"),
	)
}
```

---

## Polymorphic Proxy Models (`CanTargetDefiner` and `CanControlSaving`)

When a shared proxy table stores a content-type string and target primary key pointing back to multiple concrete embedder types, configure `ctype` / `target` tags or implement `models.CanTargetDefiner`:

```go
type CanTargetDefiner interface {
	attrs.Definer
	TargetContentTypeField() attrs.FieldDefinition
	TargetPrimaryField() attrs.FieldDefinition
}
```

Example implementation on `Page`:

```go
func (p *Page) TargetContentTypeField() attrs.FieldDefinition {
	defs := p.FieldDefs(context.Background())
	f, _ := defs.Field("ContentType")
	return f
}

func (p *Page) TargetPrimaryField() attrs.FieldDefinition {
	defs := p.FieldDefs(context.Background())
	f, _ := defs.Field("PageID")
	return f
}
```

`proxyField` (`queries/src/models/proxy_field.go`) uses these two fields to construct a two-condition `INNER JOIN`:

`lhs.<PrimaryKey> = rhs.<TargetPrimaryField> AND rhs.<TargetContentTypeField> = <ContentType(lhs)>`

And on `SaveObject`, `proxyField` automatically populates `<TargetPrimaryField>` and `<TargetContentTypeField>` before calling `Save(ctx)` on the proxy object — unless the proxy model implements `models.CanControlSaving`:

```go
type CanControlSaving interface {
	attrs.Definer
	ControlsEmbedderSaving() bool
}
```

When a polymorphic proxy returns `true` from `ControlsEmbedderSaving()`, `Model.Define` leaves `AllowEdit: false` on the generated `proxyField`. `proxyField.Save` then returns `errors.NotImplemented` (which `Model.SaveObject` logs and skips), allowing the embedder or proxy to coordinate saving manually (e.g., inside `SaveObject` or `BeforeSave`).

---

## Chained Proxy Relations

Proxy joins chain across foreign keys and nested proxies. Querying a model with `Select("*", "ProxiedModel.*")` automatically loads `ProxiedModel` and any proxies attached to `ProxiedModel`:

```go
type LinkedToProxiedModel struct {
	models.Model
	ID           int64
	ProxiedModel *ProxiedModel
}

func (l *LinkedToProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return l.Model.Define(ctx, l,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		fields.ForeignKey[*ProxiedModel]("ProxiedModel", "proxied_model_id"),
	)
}
```

```go
row, err := queries.GetQuerySet(&LinkedToProxiedModel{}).
	WithContext(ctx).
	Select("*", "ProxiedModel.*").
	Filter("ID", id).
	First()
if err != nil {
	return err
}

fmt.Println(row.Object.ProxiedModel.ProxyModel.Title)
```
