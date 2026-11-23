# `attrs` Package Interfaces

The `attrs` package (`github.com/Nigel2392/go-django/src/core/attrs`) defines a set of interfaces for inspecting, validating, binding, and relating model fields in a uniform way.

---

## Core Model and Definition Interfaces

### `Definer`

Any model managed by `attrs` implements the [`Definer`](./implementation.md#implementing-the-definer-interface) interface to expose its field definitions for a given context.

```go
type Definer interface {
	// Retrieves the field definitions for the model.
	FieldDefs(context.Context) Definitions
}
```

### `StaticDefinitions`

`StaticDefinitions` is a type alias for `staticDefinitions[FieldDefinition]` and represents read-only structural metadata for a model's fields, used primarily inside [`ModelMeta`](./model-meta.md#modelmeta).

```go
type staticDefinitions[T FieldDefinition] interface {
	TableName() string
	Field(name string) (f T, ok bool)
	Primary() T
	Instance() Definer
	Fields() []T
	Len() int
}

type StaticDefinitions = staticDefinitions[FieldDefinition]
```

- `TableName` returns the database table name for the model (defaulting to `<app>_<snake_case_model>` when an app is registered).
- `Field` looks up a field by name, returning `false` if not found.
- `Primary` returns the primary key field definition, or `nil` if none is marked primary.
- `Instance` returns the underlying model instance (for `StaticDefinitions`, a freshly initialized object from `NewObject`).
- `Fields` returns all fields in their definition order.
- `Len` returns the number of defined fields.

### `Definitions`

`Definitions` wraps runtime field management for a model instance, embedding `staticDefinitions[Field]` and `CanSignalChanged`.

```go
type Definitions interface {
	staticDefinitions[Field]

	CanSignalChanged

	Context() context.Context

	// Set sets the value of the field with the given name (or panics if not found).
	Set(name string, value interface{}) error

	// Retrieves the value of the field with the given name (or panics if not found).
	Get(name string) interface{}
}
```

- `Context` returns the context passed when the definitions were created.
- `Set` updates a field value by name (enforcing editability and validation constraints).
- `Get` retrieves a field value by name (failing if the field does not exist).

### `CanSignalChanged`

`CanSignalChanged` lets models track field mutations and resets when values are scanned, updated, or saved.

```go
type CanSignalChanged interface {
	// SignalChange is called when a field is changed.
	SignalChange(f Field, value interface{})

	// SignalReset is called when a field is saved and its changed status should be reset.
	SignalReset(f Field)
}
```

---

## Field Interfaces

### `FieldDefinition`

`FieldDefinition` describes the static metadata and form configuration of a field without exposing value mutation or database scanning.

```go
type FieldDefinition interface {
	Name() string
	Labeler
	Helper

	Instance() Definer
	ColumnName() string
	Type() reflect.Type
	Attrs() map[string]any
	Rel() Relation
	IsPrimary() bool
	AllowNull() bool
	AllowBlank() bool
	AllowEdit() bool
	FormField() fields.Field
}
```

- `Name` returns the field's name (or its `NameOverride`).
- `Instance` returns the model instance associated with this field definition.
- `ColumnName` returns the database column name.
- `Type` returns the `reflect.Type` of the underlying field.
- `Attrs` returns the field's attribute map (including `Attr*Key` metadata).
- `Rel` returns the [`Relation`](#relationtarget-relation-and-lazyrelation) metadata if the field defines a relationship, or `nil`.
- `IsPrimary`, `AllowNull`, `AllowBlank`, and `AllowEdit` report the field's constraints.
- `FormField` builds or returns the `fields.Field` used for HTML form rendering and validation.

### `StructFieldDefinition`

`StructFieldDefinition` extends `FieldDefinition` for struct-backed fields so [`FastGet`](./functions.md#fastget-and-fastset) and [`FastSet`](./functions.md#fastget-and-fastset) can access struct field offsets directly.

```go
type StructFieldDefinition interface {
	FieldDefinition

	// Used in [FastGet]. Beware that the struct field may not have the right index list
	// or belong directly to the Definer it was found on.
	StructField() *reflect.StructField
}
```

### `Field`

`Field` extends `FieldDefinition`, `sql.Scanner`, and `driver.Valuer` with runtime value access, binding, default lookup, and validation.

```go
type Field interface {
	FieldDefinition

	sql.Scanner
	driver.Valuer

	FieldDefinitions() Definitions
	BindToDefinitions(definitions Definitions)

	ToString() string
	GetValue() interface{}
	GetDefault() interface{}
	SetValue(v interface{}, force bool) error
	Validate() error
}
```

- `FieldDefinitions` and `BindToDefinitions` link a field back to its owning `Definitions` so mutations can trigger `SignalChange`.
- `ToString` formats the field's current (or default) value as a human-readable string.
- `GetValue` retrieves the field's current Go value.
- `GetDefault` retrieves the field's configured default value.
- `SetValue` validates and sets the field value. When `force` is `false`, `SetValue` enforces `AllowEdit`, `AllowBlank`, and `AllowNull` checks and triggers change signaling.
- `Validate` runs length, numeric range, and custom `Validators` checks against the field's value.

### `Labeler` and `Helper`

Field definitions and custom field value types can implement `Labeler` and `Helper` to provide context-aware (and translatable) labels and help text.

```go
type Labeler interface {
	Label(ctx context.Context) string
}

type Helper interface {
	HelpText(ctx context.Context) string
}
```

### `Scanner` and `ToStringConverter`

Custom field value types can implement `Scanner` to customize raw value scanning in `FieldDef.SetValue`, or `ToStringConverter` to customize string formatting in [`attrs.ToString`](./functions.md#tostring).

```go
type Scanner interface {
	ScanAttribute(src any) error
}

type ToStringConverter interface {
	ToString() string
}
```

### `OptionsField`

Fields that expose selectable choices implement `OptionsField`:

```go
type OptionsField interface {
	Options(ctx context.Context) ([]Option, error)
}
```

---

## Object Lifecycle and Embedding Interfaces

### `CanCreateObject` and `CanSetup`

Used by [`attrs.NewObject`](./functions.md#newobject-and-setup) and [`attrs.Setup`](./functions.md#newobject-and-setup) to customize object instantiation and post-creation initialization.

```go
type CanCreateObject[T any] interface {
	CreateObject(ctx context.Context, source T) T
}

type CanSetup interface {
	Setup(ctx context.Context)
}
```

Note that `CanCreateObject[T]` expects the returned object to have the same concrete type as the source model; if the types differ, `NewObject` ignores the returned value.

### `Binder`, `Embedded`, and `Embedder`

- `Binder` allows a field's value to receive a reference to its owning model and `Field` whenever the value is read, set, scanned, or serialized via [`BindValueToModel`](./functions.md#bindvaluetomodel-and-isembeddedfield).
- `Embedded` lets an embedded field bind to the top-most embedding `Definer` during `NewField`.
- `Embedder` indicates whether a `FieldDefinition` belongs to an embedded or proxy model (checked by [`IsEmbeddedField`](./functions.md#bindvaluetomodel-and-isembeddedfield)).

```go
type Binder interface {
	BindToModel(model Definer, field Field) error
}

type Embedded interface {
	BindToEmbedder(embedder Definer) error
}

type Embedder interface {
	Embedded() bool
}
```

---

## Registration and Unbound Field Interfaces

### `CanOnModelRegister`

Implemented by fields that need a one-time callback during final model registration (invoked once per model lifetime during [`ResetDefinitions`](./hooks_signals.md#available-signals)).

```go
type CanOnModelRegister interface {
	Field
	OnModelRegister(model Definer, top FieldDefinition) error
}
```

### `CanRelatedName`, `CanReverseRelate`, and `CanIsReverse`

These interfaces control how relationships expose reverse accessors and indicate relationship direction:

```go
type CanRelatedName interface {
	Field
	RelatedName() string
}

type CanReverseRelate interface {
	Field
	AllowReverseRelation() bool
}

type CanIsReverse interface {
	Field
	IsReverse() bool
}
```

- `CanRelatedName` overrides the field's reverse alias on the target model.
- `CanReverseRelate` suppresses automatic reverse relation registration when `AllowReverseRelation()` returns `false`.
- `CanIsReverse` reports whether the field represents the reverse side of a relationship.

### `UnboundFieldConstructor` and `UnboundField`

Used with [`attrs.Unbound`](./functions.md#unbound) and [`attrs.Make`](./functions.md#make) to declare fields that are bound to a model instance at `Make` time instead of requiring the model instance upfront.

```go
type UnboundFieldConstructor interface {
	Name() string
	BindField(ctx context.Context, model Definer) (Field, error)
}

type UnboundField interface {
	Field
	UnboundFieldConstructor
}
```

### `CanModelInfo`

Models can implement `CanModelInfo` to store custom key-value metadata on [`ModelMeta`](./model-meta.md#modelmeta) during registration.

```go
type CanModelInfo interface {
	ModelMetaInfo(object Definer) map[string]any
}
```

---

## Relationship Metadata Interfaces

See [Model Meta](./model-meta.md) for concrete implementations and helper functions such as `Relate`, `ReverseRelation`, `ThroughDeferred`, and `RelatedDeferred`.

### `Through` and `LazyThrough`

Describe an intermediary ("through") model linking a source and target model.

```go
type Through interface {
	Model() Definer
	SourceField() string
	TargetField() string
}

type LazyThrough interface {
	Through
	ModelKey() string
}
```

### `RelationTarget`, `Relation`, and `LazyRelation`

Describe the target model, target field, direction, type, and optional through model of a relation.

```go
type RelationTarget interface {
	From() RelationTarget
	Model() Definer
	Field() FieldDefinition
}

type Relation interface {
	RelationTarget
	Type() RelationType
	Through() Through
}

type LazyRelation interface {
	Relation
	ModelKey() string
}
```

---

## ORM Relation Value Interfaces

The `attrs` package defines value-holder interfaces used by [`queries`](../../queries/docs/relations/relations.md) to read and write populated relation instances on model fields.

```go
type ValueRelation interface {
	// The target model of the relation.
	Model() Definer

	// The through model of the relation.
	Through() Definer
}

type RelationValue interface {
	GetValue() (obj Definer)
	SetValue(instance Definer)
}

type MultiRelationValue interface {
	GetValues() []Definer
	SetValues(instances []Definer)
}

type ThroughRelationValue interface {
	GetValue() (obj Definer, through Definer)
	SetValue(instance Definer, through Definer)
}

type MultiThroughRelationValue interface {
	SetValues(instances []ValueRelation)
	GetValues() []ValueRelation
}
```

- `RelationValue`: Single-object relation values without a through table (such as `OneToOne`).
- `MultiRelationValue`: Multi-object relation values without a through table (such as reverse `OneToMany` sets).
- `ThroughRelationValue`: Single-object `OneToOne` relation values carrying a through model instance.
- `MultiThroughRelationValue`: Multi-object `ManyToMany` relation values backed by `ValueRelation` pairs.
- In `queries`, the extended interfaces (`queries.RelationValue`, `queries.MultiRelationValue`, `queries.ThroughRelationValue`, `queries.MultiThroughRelationValue`) embed the corresponding `attrs` interface alongside `attrs.Binder` and `ParentInfo() *ParentInfo`. See [Queries Relations Documentation](../../queries/docs/relations/relations.md).
