# Model Meta Information

The `attrs` package maintains a global metadata registry for registered models so callers, forms, and ORM backends can inspect fields, primary keys, content types, and forward/reverse relationships without rebuilding runtime field definitions on every access.

---

## Registration and Lookup Functions

### `RegisterModel`

Registers a model in the global `attrs` registry.

```go
func RegisterModel(model Definer)
```

Models returned from `django.AppConfig.Models()` are registered automatically during `django.Initialize`, followed by [`attrs.ResetDefinitions`](./hooks_signals.md#available-signals) to finalize reverse relations and static definitions. If the model type is already in the registry, `RegisterModel` returns immediately.

### `GetModelMeta`

Retrieves the [`ModelMeta`](#modelmeta) for a registered model.

```go
func GetModelMeta(model any) ModelMeta
```

`model` may be a `Definer`, a `reflect.Type` (value or pointer), or a `reflect.Value`. `GetModelMeta` panics if the model has not been registered with `RegisterModel`.

### `IsModelRegistered`

Reports whether a model has completed registration.

```go
func IsModelRegistered(model any) bool
```

`model` accepts a `Definer`, `reflect.Type`, `reflect.Value`, or `*reflect.Value`.

**Caveat**: `IsModelRegistered` checks `ok && meta.setup`. `meta.setup` is set to `true` only during the final pass triggered by [`attrs.ResetDefinitions`](./hooks_signals.md#available-signals), **not** by `attrs.RegisterModel` alone. If you register models manually outside `django.Initialize`, send `attrs.ResetDefinitions.Send(ctx, nil)` before checking `IsModelRegistered`.

### `IterModelMeta`

Returns an iterator over all registered `(Definer, ModelMeta)` pairs ordered topologically by forward dependencies and through models.

```go
func IterModelMeta() (iter.Seq2[Definer, ModelMeta], error)
```

Dependencies (target models and through models) appear before the models that reference them. Cyclic dependencies are tolerated rather than returning an error.

### `GetRelationMeta`

Looks up a relationship field on a registered model by field name (or reverse alias).

```go
func GetRelationMeta(m Definer, name string) (rel Relation, fwd bool, present bool)
```

- `rel`: The matching [`Relation`](./interfaces.md#relationtarget-relation-and-lazyrelation), or `nil` if not found.
- `fwd`: `true` if `name` is a forward relation, `false` if it is a reverse relation (or absent).
- `present`: `true` if the model is registered and has a relation under `name`.

This function never panics if the model or relation is missing.

### `StoreOnMeta` and `FieldAttribute`

Store arbitrary metadata on a registered model's `ModelMeta`, or retrieve cached field attributes (`MetaStorageKeyAttrs`).

```go
const MetaStorageKeyAttrs = "fields.attributes"

func StoreOnMeta(m Definer, key string, value any)
func FieldAttribute[T any](ctx context.Context, model Definer, fieldName string, attrName string) (context.Context, T, bool)
```

`StoreOnMeta` panics if `m` has not been registered. `FieldAttribute` reads the model's `MetaStorageKeyAttrs` map and stores it inside the returned `context.Context` so repeated lookups avoid hitting the global registry.

---

## `ModelMeta` and `CanModelInfo`

### `ModelMeta`

```go
type ModelMeta interface {
	Model() Definer
	Forward(relField string) (Relation, bool)
	Reverse(relField string) (Relation, bool)
	ForwardMap() *orderedmap.OrderedMap[string, Relation]
	ReverseMap() *orderedmap.OrderedMap[string, Relation]
	ContentType() contenttypes.ContentType
	Storage(key string) (any, bool)
	Definitions() StaticDefinitions
	Primary() FieldDefinition
}
```

- `Model`: Returns the registered prototype `Definer` instance.
- `Forward` / `Reverse`: Look up forward or reverse relations by field/alias name.
- `ForwardMap` / `ReverseMap`: Return the ordered maps of forward and reverse relations.
- `ContentType`: Returns the `contenttypes.ContentType` for the model.
- `Storage`: Retrieves values saved via `StoreOnMeta`, `CanModelInfo`, or internal registration keys such as `MetaStorageKeyAttrs` and `"through.model"`.
- `Definitions`: Returns the model's cached [`StaticDefinitions`](./interfaces.md#staticdefinitions).
- `Primary`: Returns the model's primary [`FieldDefinition`](./interfaces.md#fielddefinition).

### `CanModelInfo`

Models can implement [`CanModelInfo`](./interfaces.md#canmodelinfo) to populate `ModelMeta.Storage(...)` automatically when registered:

```go
func (m *MyModel) ModelMetaInfo(object attrs.Definer) map[string]any {
	return map[string]any{
		"ordering": []string{"-created_at"},
	}
}
```

---

## Relations and Through Models

### `RelationType`

```go
type RelationType int

const (
	RelNone RelationType = iota
	RelManyToOne
	RelOneToOne
	RelManyToMany
	RelOneToMany
)
```

`RelationType` implements `String() string`, `json.Marshaler` (`MarshalJSON`), and `json.Unmarshaler` (`UnmarshalJSON`), mapping between the constants and `"None"`, `"ManyToOne"`, `"OneToOne"`, `"ManyToMany"`, and `"OneToMany"`.

### `Relate` and `ReverseRelation`

```go
func Relate(target Definer, targetField string, through Through) Relation
func ReverseRelation(fromModel Definer, fromField Field, forward Relation) Relation
```

`Relate` constructs a forward relation target for [`attrs.Config`](./implementation.md#attrsconfig-reference) (`RelForeignKey`, `RelOneToOne`, `RelManyToMany`, or `RelForeignKeyReverse`). Pass `""` for `targetField` to target the related model's primary key. `ReverseRelation` constructs the inverse `Relation` used during reverse relation registration.

### `ThroughModel`

A concrete implementation of [`Through`](./interfaces.md#through-and-lazythrough):

```go
type ThroughModel struct {
	This   Definer
	Source string
	Target string
}
```

- `This`: The intermediary model instance.
- `Source`: Field name on `This` pointing to the source model.
- `Target`: Field name on `This` pointing to the target model.

### `ThroughDeferred` and `RelatedDeferred`

Create lazy relations and through models resolved on demand by `contenttypes.LoadModel(modelKey)` (using `<appName>.<ModelName>` or a `contenttypes.LazyRegistry` key):

```go
func ThroughDeferred(modelKey string, sourceField string, targetField string) Through
func RelatedDeferred(typ RelationType, modelKey string, targetField string, through Through) LazyRelation
```

### `ThroughMeta` and `ThroughModelMeta`

When a model is referenced as a `Through` model on a forward relation, `registerModel` stores a `ThroughMeta` under `meta.Storage("through.model")`:

```go
type ThroughMeta struct {
	IsThroughModel bool
	Source         Definer
	Target         Definer
	SourceField    string
	TargetField    string
}

func (t ThroughMeta) GetSourceField(targetModel Definer, throughDefs Definitions) Field
func (t ThroughMeta) GetTargetField(targetModel Definer, throughDefs Definitions) Field

func ThroughModelMeta(m Definer) ThroughMeta
```

`ThroughModelMeta(m)` returns the `ThroughMeta` for `m`, or a zero-value `ThroughMeta{}` if `m` is not a through model.

---

## Path and Relation Chain Walking

The `attrs` package provides utilities in `src/core/attrs/meta_walk.go` to traverse dot-separated field paths (passed as `[]string`) across registered model relations.

### `WalkMetaFieldsFunc`

Walks each segment in `path` starting from `m`, looking up fields and forward/reverse relations (forward relations take priority when both exist under the same name):

```go
type WalkFieldsFunc func(
	source Relation,
	meta ModelMeta,
	object Definer,
	field FieldDefinition,
	fieldRel Relation,
	part string,
	parts []string,
	idx int,
) (stop bool, err error)

func WalkMetaFieldsFunc(m Definer, path []string, fn WalkFieldsFunc) error
```

Returns `errors.FieldNotFound` if a segment does not exist on the current model, or `errors.AmbiguousColumn` if an intermediate segment is not a relation.

### `WalkMetaFields` and `PathMetaChain`

Builds a slice of `*pathMeta` entries representing every hop along `path`:

```go
type PathMetaChain []*pathMeta

func (c PathMetaChain) First() *pathMeta
func (c PathMetaChain) Last() *pathMeta

type pathMeta struct {
	Object      Definer
	Definitions StaticDefinitions
	Field       FieldDefinition
	Relation    Relation
	// ...
}

func (m *pathMeta) String() string
func (m *pathMeta) Parent() *pathMeta
func (m *pathMeta) Child() *pathMeta
func (m *pathMeta) CutAt() []*pathMeta

func WalkMetaFields(m Definer, path []string) (PathMetaChain, error)
```

### `WalkRelationChain` and `RelationChain`

Builds a doubly-linked chain of `RelationChainPart` nodes (suitable for SQL join planning), optionally appending the target model's primary/target field node when `includeFinalRel` is `true` and the final field is a relation:

```go
type RelationChainPart struct {
	Next      *RelationChainPart
	Prev      *RelationChainPart
	ChainPart string
	FieldRel  Relation
	Model     Definer
	Through   Through
	Field     FieldDefinition
	Depth     int
	// ...
}

func (p *RelationChainPart) Chain() []string

type RelationChain struct {
	Root   *RelationChainPart
	Final  *RelationChainPart
	Fields []FieldDefinition
	Chain  []string
}

func WalkRelationChain(m Definer, includeFinalRel bool, path []string) (*RelationChain, error)
```
