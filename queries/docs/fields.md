# Fields Package (`queries/src/fields`)

The `fields` package provides typed field constructors for defining relations and computed expressions on your models. These fields plug directly into the `queries` ORM and are used inside `FieldDefs()`.

> All typed constructors return an `attrs.UnboundFieldConstructor` (or a concrete field pointer). When used inside `m.Model.Define(m, ...)`, the ORM binds the field to the current model instance automatically.

---

## Relation Fields

Relation fields connect two models in the database. They are backed by the internal `RelationField[T]` type and implement the `queries.TargetClauseField` interface so the ORM knows how to build JOIN clauses.

### `fields.ForeignKey[T]`

Creates a **many-to-one** foreign key field. The struct field must be `*T`.

```go
// Signature:
func ForeignKey[T any](name, columnName string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

```go
// Example:
type Todo struct {
    models.Model
    ID   int
    User *User
}

func (m *Todo) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        fields.ForeignKey[*User]("User", "user_id"),
    )
}
```

### `fields.OneToOne[T]`

Creates a **one-to-one** relation field. The struct field must be `*T`.

```go
// Signature:
func OneToOne[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

```go
// Example:
type User struct {
    models.Model
    ID      uint64
    Profile *Profile
}

func (m *User) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
        fields.OneToOne[*Profile]("Profile", &fields.FieldConfig{
            ColumnName: "profile_id",
        }),
    )
}
```

### `fields.ManyToMany[T]`

Creates a **many-to-many** relation field. A `Through` table must be provided via `FieldConfig.Through` or `FieldConfig.Rel`. The type parameter `T` must be `*queries.RelM2M[*TargetModel, *ThroughModel]`.

```go
// Signature:
func ManyToMany[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

```go
// Example:
fields.ManyToMany[*queries.RelM2M[*Tag, *PostTag]]("Tags", &fields.FieldConfig{
    Through: &PostTag{},
})
```

> **Note:** `ManyToMany` will panic if no `Through` relation is defined in the `FieldConfig`.

### `fields.FieldConfig`

Used to configure relation fields. Common fields:

| Field | Type | Purpose |
|---|---|---|
| `ColumnName` | `string` | Database column name for FK reference |
| `Nullable` | `bool` | Whether the relation can be NULL |
| `IsProxy` | `bool` | Treat as a proxy model field (auto-joined always) |
| `IsReverse` | `any` | Mark as a reverse side of a relation (`bool`, `func() bool`, or `func(attrs.Field) bool`) |
| `ReverseName` | `string` | Custom name for the auto-generated reverse relation |
| `NoReverseRelation` | `bool` | Suppress auto-generation of the reverse side |
| `TargetField` | `string` | Name of the target field on the related model |
| `Through` | `attrs.Through` | Through table model for M2M |
| `Rel` | `attrs.Relation` | Custom relation object |

---

## Virtual / Expression Fields

### `fields.NewVirtualField[T]`

Creates a **read-only computed field** backed by a SQL expression. See the [Virtual Fields documentation](../virtual_fields.md) for full usage.

```go
// Signature:
func NewVirtualField[T any](forModel attrs.Definer, dst any, name string, expr expr.Expression) *ExpressionField[T]
```

---

## Embed Helper

### `fields.Embed`

Used inside `FieldDefs()` to embed the fields of a related struct (typically a proxy model). The named field must be a **pointer to a struct** implementing `attrs.Definer`.

```go
// Signature:
func Embed(nameOrScan any, options ...EmbedOptions) func(d attrs.Definer) []attrs.Field
```

```go
// Example: embed all fields of the Page struct
func (m *BlogPage) FieldDefs() attrs.Definitions {
    return m.Model.Define(m,
        fields.Embed("Page"),                      // embed all of Page's fields
        attrs.NewField(m, "Author", nil),
    )
}
```

`EmbedOptions` lets you control which fields are included:

```go
type EmbedOptions struct {
    AutoInit    bool     // auto-initialize the embedded pointer if nil
    EmbedFields []any    // embed only specific fields; nil = embed all
}
```

---

## Generic Foreign Key

### `fields.GenericForeignKey`

A polymorphic relation field that can point to any model type, identified by a content-type string and a primary key. Useful for comment systems, audit logs, etc.

```go
// Signature:
func GenericForeignKey[T attrs.Definer](definer T, name string, cnf *GenericForeignKeyConfig) interface{}
```

```go
type GenericForeignKeyConfig struct {
    Nullable         bool
    Target           attrs.Definer
    RelPrimary       string    // Field name on the target that stores the PK
    ContentTypeField string    // Field name on this model that stores the content type
    TargetField      string    // Field name on this model that stores the PK value
}
```
