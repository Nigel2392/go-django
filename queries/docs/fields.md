# Fields Package (`queries/src/fields`)

The `fields` package provides field types, typed relation constructors, virtual expression fields, and model embedding helpers for `FieldDefs(ctx context.Context)`.

All typed relation constructors (`ForeignKey`, `OneToOne`, `ManyToMany`) return an `attrs.UnboundFieldConstructor`. When passed to `m.Model.Define(ctx, m, ...)` or `attrs.Make(ctx, m, ...)`, the ORM binds each field to the model instance automatically.

---

## Relation Shortcuts and `FieldConfig`

### `fields.ForeignKey[T]`

Creates a **many-to-one** foreign key field (`*ForeignKeyField[T]`). The struct field should be `*TargetModel` (or `*queries.RelFK[*TargetModel]`).

```go
func ForeignKey[T any](name, columnName string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

```go
type Todo struct {
	models.Model
	ID   int64
	User *User
}

func (m *Todo) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		fields.ForeignKey[*User]("User", "user_id"),
	)
}
```

### `fields.OneToOne[T]`

Creates a **one-to-one** relation field (`*OneToOneField[T]`). The struct field is `*TargetModel` (direct FK column) or `*queries.RelO2O[*TargetModel, *ThroughModel]` (through table).

```go
func OneToOne[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

```go
type User struct {
	models.Model
	ID      uint64
	Profile *Profile
}

func (m *User) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true}),
		fields.OneToOne[*Profile]("Profile", &fields.FieldConfig{
			ColumnName: "profile_id",
			AllowEdit:  true,
		}),
	)
}
```

> **Caveat (`OneToOneField.AllowEdit`):** Unlike `ForeignKeyField`, `OneToOneField` does **not** override `AllowEdit()` or `AllowDBEdit()`, which default to `false` in `RelationField`. If the `OneToOne` column should be persisted on `INSERT`/`UPDATE` or edited in forms, set `AllowEdit: true` in `FieldConfig`.

### `fields.ManyToMany[T]`

Creates a **many-to-many** relation field (`*ManyToManyField[T]`). The struct field type must be `*queries.RelM2M[*TargetModel, *ThroughModel]`.

```go
func ManyToMany[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor
```

`ManyToMany` requires a through relation configured via `FieldConfig.Rel` (with an `*attrs.ThroughModel` passed to `attrs.Relate`) — it panics if `conf` is omitted or lacks a through model:

```go
fields.ManyToMany[*queries.RelM2M[*Tag, *PostTag]]("Tags", &fields.FieldConfig{
	Rel: attrs.Relate(&Tag{}, "", &attrs.ThroughModel{
		This:   &PostTag{},
		Source: "PostRef",
		Target: "TagRef",
	}),
})
```

### `fields.FieldConfig` and `fields.DataModelFieldConfig`

`FieldConfig` embeds `DataModelFieldConfig`:

```go
type DataModelFieldConfig struct {
	Label      any
	HelpText   any
	ResultType reflect.Type // Explicit result type override
	Ref        attrs.Field  // Outer field reference for change signals / form hooks
	Formfield  fields.Field // Explicit override form field
}

type FieldConfig struct {
	DataModelFieldConfig
	ScanTo            any
	Nullable          bool
	AllowEdit         bool
	IsProxy           bool
	IsReverse         any // bool, func() bool, or func(attrs.Field) bool
	ReverseName       string
	NoReverseRelation bool
	ColumnName        string
	TargetField       string
	Through           attrs.Through
	Rel               attrs.Relation
}
```

| Field | Purpose |
|---|---|
| `ScanTo` | Destination pointer when scanning/binding; defaults to `&model.<Name>` |
| `Nullable` | Marks the relation column as nullable (`AllowNull()`) |
| `AllowEdit` | Controls `AllowDBEdit()` (and `AllowEdit()` when `IsProxy` is true) on `RelationField` |
| `IsProxy` | Marks as a proxy relation (always auto-joined; cannot be nullable; suppresses reverse relation) |
| `IsReverse` | Explicitly marks the field as a reverse relation (`bool`, `func() bool`, or `func(attrs.Field) bool`) |
| `ReverseName` | Custom name for the auto-generated reverse relation on the target model |
| `NoReverseRelation` | Suppresses reverse relation generation on the target model |
| `ColumnName` | Database column name for direct FK / O2O relations |
| `TargetField` | Target field name on the related model (defaults to its primary key) |
| `Through` | `attrs.Through` definition used when building `Rel` automatically |
| `Rel` | Explicit `attrs.Relation` (for example `attrs.Relate(&Tag{}, "", &attrs.ThroughModel{...})`) |
| `Label`, `HelpText` | UI label and help text (string or translation getter) |
| `ResultType` | Overrides `reflect.Type` inferred from type parameter `T` |
| `Ref`, `Formfield` | Custom outer `attrs.Field` reference and `forms/fields.Field` override |

---

## Core Field Implementations

### `DataModelField[T]`

`NewDataModelField[T](forModel attrs.Definer, dst any, name string, cnf ...DataModelFieldConfig) *DataModelField[T]` provides reflection-backed and `DataModel`/`ModelDataStore`-backed attribute access (`attrs.Field`). It implements `ColumnName() == ""`, `AllowEdit() == false`, and `AllowDBEdit() == false`, and serves as the base type for `ExpressionField` and `RelationField`.

### `ExpressionField[T]`

`NewVirtualField[T](forModel attrs.Definer, dst any, name string, expr expr.Expression) *ExpressionField[T]` embeds `*DataModelField[T]` and implements `queries.VirtualField` and `queries.AliasField`. See [Virtual Fields](./virtual_fields.md).

### `RelationField[T]`, `ForeignKeyField[T]`, `OneToOneField[T]`, `ManyToManyField[T]`

`NewRelatedField[T](forModel attrs.Definer, name string, cnf *FieldConfig) *RelationField[T]` implements:

- `queries.TargetClauseField` and `queries.TargetClauseThroughField` (`INNER JOIN` when `!Nullable && !hasMany()`, `LEFT JOIN` otherwise)
- `queries.ProxyField` and `queries.ProxyThroughField`
- `queries.ForUseInQueriesField` (`ForSelectAll()` returns `false` on base `RelationField`, overridden to `true` on `ForeignKeyField`)
- `queries.SaveableField` (cascades `Save(ctx)` to related `Save(ctx)` / `Save(ctx, parent)` targets on FK/O2O)
- `attrs.CanRelatedName` (default reverse names: `<Model>Reverse` for O2O / O2M, `<Model>Set` for M2O / M2M)

Specialized constructors:

- `NewForeignKeyField[T]` (`ForeignKeyField[T]`): Overrides `AllowEdit() == true`, `AllowDBEdit() == true`, and `ForSelectAll() == true`. (`ForeignKeyReverseField[T]` via `NewForeignKeyReverseField[T]` represents reverse FK sets.)
- `NewOneToOneField[T]` (`OneToOneField[T]`): Wraps `RelationField[T]` with `RelOneToOne`. Does **not** override `AllowEdit()` / `AllowDBEdit()`, so `FieldConfig.AllowEdit` must be `true` for DB edits. (`OneToOneReverseField[T]` represents reverse O2O.)
- `NewManyToManyField[T]` (`ManyToManyField[T]`): Wraps `RelationField[T]` with `RelManyToMany`; `CanMigrate()` returns `false` and `Value()` returns `(nil, nil)` because the junction table stores rows separately.

---

## Embed Helper (`fields.Embed`)

`fields.Embed` unpacks fields from a struct field or `attrs.Definer` directly into another model's `FieldDefs(ctx context.Context)`:

```go
type EmbedOptions struct {
	AutoInit    bool  // Initialize the embedded pointer if nil
	EmbedFields []any // Subset of field names/definitions to embed; nil embeds all
}

func Embed(ctx context.Context, nameOrScan any, options ...EmbedOptions) func(d attrs.Definer) []attrs.Field
```

Example:

```go
func (m *BlogPage) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		fields.Embed(ctx, "Page", fields.EmbedOptions{AutoInit: true}),
		attrs.NewField(m, "Author", nil),
	)
}
```

---

## Form Fields and Widgets (`fields/formfields`)

Package `github.com/Nigel2392/go-django/queries/src/fields/formfields` is automatically blank-imported when you import `queries/src/models`. Its `init()` hooks into Go-Django forms:

- Registers `attrs.HookFormFieldForType` (priority `100`) to map relation fields onto `ForeignKeyFormField`, `OneToOneFormField`, or `ManyToManyFormField`. Reverse `RelOneToMany` fields and reverse `RelOneToOne` fields whose target field is non-primary return `(nil, true)` (no form field generated).
- Registers `forms.FormTemplateStaticHookName` and `forms.FormTemplateFSHookName` with embedded static assets (`forms/css/model-multiple-select.css`) and templates (`forms/widgets/model-select.html`, `forms/widgets/model-multiple-select.html`).

Key types:

- `BaseRelationField`: Holds `*django_formfields.BaseField`, `attrs.FieldDefinition`, and `attrs.Relation`.
- `ForeignKeyFormField`: Uses `ModelSelectWidget` (`*ModelSelect`) to render a single `<select>` of related model instances.
- `OneToOneFormField`: Embeds `ForeignKeyFormField` and implements `modelforms.AfterModelFieldSaver` (`SaveField` delegates to `queries.OneToOneQuerySet(backRef)` to call `Clear(ctx)` or `SetTarget(obj)`).
- `ManyToManyFormField`: Uses `*MultiSelectWidget[attrs.Definer]` and implements `modelforms.AfterModelFieldSaver` (`SaveField` delegates to `queries.ManyToManyQuerySet(backRef)` to call `ClearTargets()` or `SetTargets(objects)`).
- `ModelSelect` / `ModelSelectWidget(allowBlank, blankLabel, opts, atts)`: `chooser.BaseChooser`-backed `<select>` widget translating primary keys to and from `attrs.Definer` objects.
- `MultiSelectWidget[T]`: Multi-select widget querying selected and unselected rows via `RelM2M` / `QuerySet`, validating submitted PKs, and supplying CSS/JS `Media()`.

See [Form Fields](../../docs/forms/fields.md) and [Widgets](../../docs/forms/widgets.md) for general form usage.
