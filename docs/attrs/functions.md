# `attrs` Package Functions

The `attrs` package (`github.com/Nigel2392/go-django/src/core/attrs`) provides package-level helpers for building model definitions, reading and writing field values, creating model instances, and traversing model metadata.

---

## Defining and Constructing Models

### `Define`

Retrieves the [`Definitions`](./interfaces.md#definitions) for `model`, ensuring `ctx` carries a valid attribute context and reusing cached definitions when `ctx` was initialized with `AttributeContext`.

```go
func Define(ctx context.Context, model Definer) Definitions
```

Always prefer `attrs.Define(ctx, model)` over calling `model.FieldDefs(ctx)` directly: if `ctx` has no attribute context attached, `Define` attaches `CtxFlagNone` so `Make` will not panic.

### `Make`

Creates an `*ObjectDefinitions` from a model instance and variadic field arguments.

```go
func Make[T1 Definer, T2 any](ctx context.Context, d T1, fieldDefinitions ...T2) *ObjectDefinitions
```

`Make` requires `ctx` to be an attribute context (panicking otherwise). It unpacks `fieldDefinitions` using `UnpackFieldsFromArgsIter` and binds each field to the returned `*ObjectDefinitions`. You can optionally chain `.WithTableName("custom_table")` on the result.

### `AutoDefinitions`

Generates `Definitions` automatically for a struct by inspecting exported fields and parsing `attrs` struct tags.

```go
func AutoDefinitions[T Definer](ctx context.Context, instance T, include ...any) Definitions
```

If `include` is provided, only the specified field names (or `"*"` for all struct fields, `Field` values, or any argument supported by `UnpackFieldsFromArgs`) are included. Note that calling `AutoDefinitions` inside `FieldDefs(ctx)` still requires `ctx` to be an attribute context because `AutoDefinitions` calls `Make`.

### `AutoFieldList`

Builds the `[]Field` slice used by `AutoDefinitions`.

```go
func AutoFieldList[T1 Definer](ctx context.Context, instance T1, include ...any) []Field
```

Anonymous (embedded) struct fields, unexported fields, and fields tagged `attrs:"-"` are skipped automatically.

### `NewField`

Creates a `*FieldDef` bound to a struct field on `instance` with optional `*Config`.

```go
func NewField(instance any, name string, conf ...*Config) *FieldDef
```

If no `conf` is passed and the struct field has tags, `NewField` parses the `attrs` tag automatically. Any callbacks registered via `RegisterConfigChange` are also applied during initialization, followed by `Config.OnInit` if set.

### `Unbound`

Creates an [`UnboundFieldConstructor`](./interfaces.md#unboundfieldconstructor-and-unboundfield) that defers binding `NewField` until `Make` (or `FieldsMap.Set`) supplies the concrete `Definer` instance.

```go
func Unbound(name string, cnf ...*Config) UnboundFieldConstructor
```

### `UnpackFieldsFromArgs` and `UnpackFieldsFromArgsIter`

Normalize heterogeneous field arguments into `[]Field` or an `iter.Seq2[Field, error]` sequence.

```go
func UnpackFieldsFromArgs[T1 Definer, T2 any](ctx context.Context, definer T1, args ...T2) ([]Field, error)
func UnpackFieldsFromArgsIter[T1 Definer, T2 any](ctx context.Context, definer T1, args ...T2) iter.Seq2[Field, error]
```

Both support `Field`, `[]Field`, `UnboundFieldConstructor`, `[]UnboundFieldConstructor`, `[]UnboundField`, `[]any`, `string` field names, and generator functions taking `()`, `(context.Context)`, `(T1)`, `(Definer)`, `(context.Context, T1)`, or `(context.Context, Definer)` and returning `Field`, `[]Field`, `[]any`, or `(..., error)`. `UnpackFieldsFromArgsIter` additionally accepts `iter.Seq2[Field, error]` iterators directly or returned from functions.

### `RegisterConfigChange`

Registers a configuration mutator callback that runs whenever `NewField` constructs the named field on `model`.

```go
func RegisterConfigChange(model Definer, field string, change func(*Config))
```

### `NewObject` and `Setup`

`NewObject` instantiates and initializes a new model instance from a `reflect.Type`, `contenttypes.ContentType`, `*contenttypes.ContentTypeDefinition`, content type `string`, or existing value.

```go
func NewObject[T any](ctx context.Context, definer any) T
func Setup[T any](ctx context.Context, obj any) T
```

`NewObject` checks any custom factory registered with `RegisterNewObjectFunc` or `RegisterNewObjectFuncReflect`, then [`CanCreateObject`](./interfaces.md#cancreateobject-and-cansetup), then `contenttypes`, falling back to `reflect.New`. `Setup` binds embedded structs via `embedder.Init.Bind` and invokes `CanSetup.Setup(ctx)` if implemented.

### `RegisterNewObjectFunc` and `RegisterNewObjectFuncReflect`

Register custom instantiation functions consulted by `NewObject` for a specific pointer type.

```go
func RegisterNewObjectFunc[T any](fn func(ctx context.Context, original T) Definer)
func RegisterNewObjectFuncReflect(typ reflect.Type, fn func(ctx context.Context, original any) Definer)
```

---

## Reading and Writing Field Values

### `Get`

Retrieves a typed field value from a `Definer` or `Definitions`, supporting dot-separated relation paths (for example `"Author.Name"`).

```go
func Get[T any](ctx context.Context, d any, name string) T
```

If an intermediate relation field is `nil`, `Get` allocates a zero value of that field's type to continue walking if it implements `Definer`. It fails (panics via `assert.Fail`) if the field path does not exist or cannot be assigned/converted to `T`.

### `Set` and `ForceSet`

Set the value of a single field on a `Definer`. `Set` enforces `AllowEdit()`, `AllowBlank()`, and `AllowNull()` checks and runs validation; `ForceSet` passes `force = true` to `Field.SetValue`, bypassing editability and blank checks and suppressing change signals on `FieldDef`.

```go
func Set(ctx context.Context, d Definer, name string, value interface{}) error
func ForceSet(ctx context.Context, d Definer, name string, value interface{}) error
```

### `SetMany`

Sets multiple fields on `d` from a map using `Set` semantics (`force = false`).

```go
func SetMany(ctx context.Context, d Definer, values map[string]interface{}) error
```

### `PrimaryKey` and `SetPrimaryKey`

Read or update the primary key of a model, field, or `Definitions`.

```go
func PrimaryKey(ctx context.Context, obj any) interface{}
func SetPrimaryKey(ctx context.Context, d Definer, value interface{}) error
```

`PrimaryKey` accepts `Definer`, `Field`, `Definitions`, `reflect.Value`, or raw scalar values (integers, strings, `[]byte`, `driver.Valuer`, `fmt.Stringer`), returning `uint64` for signed/unsigned integers, `string` for strings/`[]byte`/`fmt.Stringer`, or the underlying value. Both `PrimaryKey` and `SetPrimaryKey` attempt the `FastGet`/`FastSet` fast path first when models are registered and fall back to `Define(ctx, d)`.

### `FastGet` and `FastSet`

Read or write a struct field directly by `reflect.StructField` index cached during model registration, avoiding `FieldDefs(ctx)` allocation.

```go
func FastGet(modelInstance reflect.Value, fieldName string) (val reflect.Value, typ reflect.Type, hasField bool, err error)
func FastSet(modelInstance reflect.Value, fieldName string, src any) (wasSet, hasField bool, err error)
```

`modelInstance` must be a pointer to the registered model struct. Passing `""` for `fieldName` selects the model's primary key field. Fields without a `StructField` or whose type implements [`Binder`](./interfaces.md#binder-embedded-and-embedder) return an error and require falling back to `Define(ctx, d)`.

### `Method`

Looks up a method by name on `obj` and returns a typed function wrapper if found.

```go
type Function interface{}

func Method[T Function](obj interface{}, name string) (n T, ok bool)
```

---

## Attribute Context Functions

Defined in `src/core/attrs/define.go`, attribute contexts carry flags, per-context `Definitions` caches, and deferred validation/signal callbacks across nested `Define` and `Make` calls.

```go
type ContextFlag bitcheck.Flag

const (
	CtxFlagNone            ContextFlag = 0
	CtxFlagRegistering     ContextFlag = 1 << (iota - 1)
	CtxFlagDeferSignals
	CtxFlagDeferValidation
	CtxFlagEmbedded
)

type ContextSignalFlag bitcheck.Flag

const (
	SigFlagNone    ContextSignalFlag = 0
	SigFlagSignals ContextSignalFlag = 1 << (iota - 1)
	SigFlagValidate
)
```

- `AttributeContext(ctx context.Context) (context.Context, *Context)`: Returns an updated `context.Context` and `*Context` pointer. If `ctx` already holds a `*Context`, it clones it (preserving the shared deferred signal callback slice while keeping parent flags immutable).
- `IsAttributeContext(ctx context.Context) bool`: Reports whether `ctx` contains either a `ContextFlag` or a `*Context`.
- `ContextWithFlags(ctx context.Context, flag ContextFlag, set bool) context.Context`: Sets or clears `flag` on the attribute context (cloning any existing `*Context`).
- `ContextHasFlag(ctx context.Context, flag ContextFlag) bool`: Checks whether `flag` is active on `ctx`.
- `Context.Reset()`, `Context.Clone()`, `Context.Listen(sig, fn)`, `Context.Send(sig)`: Manage per-context caches and deferred callbacks.

---

## Field Attributes and Utility Functions

### `FieldAttribute` and `GetFromAttributes`

`FieldAttribute` fetches a typed entry from a registered model's cached field attributes (`MetaStorageKeyAttrs`) and caches the attribute map inside `ctx` for faster repeated lookups.

```go
func FieldAttribute[T any](ctx context.Context, model Definer, fieldName string, attrName string) (context.Context, T, bool)
func GetFromAttributes[T any](attrMap map[string]any, key string) (T, bool)
```

### `IsZero`

Checks whether a value should be treated as unset/zero for field defaults and model form initialization.

```go
func IsZero(value interface{}) bool
```

**Caveats**:

- `IsZero` explicitly excludes `bool` and `string` kinds (`rv.Kind() != reflect.Bool && rv.Kind() != reflect.String`), so `IsZero(false)` and `IsZero("")` always return `false`.
- The internal slice helper `versatileIsZeroSlice` returns `(true, true)` whenever the reflected kind is `reflect.Slice` or `reflect.Array`, so `IsZero` treats any slice or array as zero regardless of whether it is empty or populated.

### `CanAutoIncrement`

Reports whether `field` is a non-nullable primary key of signed or unsigned integer kind.

```go
func CanAutoIncrement(field FieldDefinition) bool
```

### `ColumnName` and `NiceName`

Convert Go `CamelCase` field identifiers to `snake_case` database column names or human-readable title labels (preserving uppercase `ID`, `PK`, `UUID`, and `URL` tokens in `NiceName`).

```go
func ColumnName(str string) string
func NiceName(str string) string
```

### `ToString`

Converts any value into a human-readable string. It checks `string`, struct `contenttypes.DefinitionForObject` `InstanceLabel`, `ToStringConverter`, `*mail.Address`, `time.Time` (`RFC3339`), `fmt.Stringer`, `error`, `driver.Valuer`, primitives, `[]byte`/`[]rune`, and element-wise joined slices.

```go
func ToString(v any) string
```

### `FieldNames` and `DefinerList`

```go
func FieldNames(d any, exclude []string) []string
func DefinerList[T Definer](list []T) []Definer
```

`FieldNames` extracts field names (excluding `exclude`) from a `Definer`, `Definitions`, `StaticDefinitions`, `[]Field`, `[]FieldDefinition`, `[]string`, or arbitrary struct via reflection. `DefinerList` converts `[]T` to `[]Definer`.

### `BindValueToModel` and `IsEmbeddedField`

```go
func BindValueToModel(model Definer, field Field, value any) error
func IsEmbeddedField[T FieldDefinition](f T) bool
```

`BindValueToModel` invokes `Binder.BindToModel(model, field)` if `value` (or addressable `reflect.Value`) implements [`Binder`](./interfaces.md#binder-embedded-and-embedder). `IsEmbeddedField` returns `true` if `f` implements [`Embedder`](./interfaces.md#binder-embedded-and-embedder) and `Embedded()` returns `true`.

### `IsOptionsType` and `TryOptions`

Inspect and evaluate `Config.Options` values (see [Implementation](./implementation.md#options-and-automatic-select-widgets)).

```go
func IsOptionsType(optVal any) (acceptsModel, isOptions bool)
func TryOptions(ctx context.Context, optVal any, modelType Definer) (opts []Option, ok bool, err error)
```

### `ModelMixins`

Iterates over exported anonymous mixin fields on `obj` that implement the internal `IsModelMixin()` marker, yielding `(mixin, depth)` while skipping depth `0` (the model itself).

```go
func ModelMixins(obj Definer, topdown bool) iter.Seq2[any, int]
```

---

## `attrutils` Subpackage

Import path: `github.com/Nigel2392/go-django/src/core/attrs/attrutils`.

### Attribute, Number, and Slice Helpers

```go
var (
	ErrEmptyString      = errors.New("empty string")
	ErrConvertingString = errors.New("error converting string to number")
)

func AttrFromMap[T any](attrMap map[string]any, attrName string) (T, bool, error)
func CastToNumber[T any](v any) (T, error)
func InterfaceList[T any](list []T) []any
```

- `AttrFromMap`: Looks up `attrName` in `attrMap` and performs direct type assertion, assignability, or convertibility checks to `T`, returning an error on type mismatch.
- `CastToNumber`: Converts numeric values, pointers to numbers, or numeric strings to `T`. **Caveat**: string inputs are parsed via `strconv.ParseUint(str, 10, 64)`, so negative or floating-point string representations return `ErrConvertingString`. Unconvertible types panic.
- `InterfaceList`: Converts a typed slice `[]T` into `[]any`.

### Struct Field and Method Cache

Thread-safe caches storing `*reflect.StructField` and `reflect.Method` lookups keyed by struct `reflect.Type`:

```go
func GetStructField(typIndirected reflect.Type, name string) (*reflect.StructField, bool)
func GetStructMethod(typIndirected reflect.Type, name string) (reflect.Method, bool)
func AddStructField(typIndirected reflect.Type, name string, field *reflect.StructField)
func AddStructMethod(typIndirected reflect.Type, name string, method reflect.Method)
```

### `Builder` Pattern Helpers

Wrap fluent or immutable builder objects (such as QuerySets) so methods can be invoked dynamically by name with automatic type conversion via `versatile.ConvertToType`:

```go
type ObjectBuilder[T any] interface {
	CallBuilderMethod(methodName string, args ...any) (T, []any, error)
}

func BuilderMethod[BUILDER any](bld BUILDER, methodName string, args ...any) (BUILDER, []any, error)

type Builder[T any] struct { /* ... */ }

func NewBuilder[T any](t T) *Builder[T]
func (b *Builder[T]) Orig() T
func (b *Builder[T]) Ref() T
func (b *Builder[T]) Exec(fn func(bld T) (T, error)) (T, error)
func (b *Builder[T]) Call(methodName string, args ...any) (T, []any, error)
func (b *Builder[T]) HasMethod(methodName string) bool
func (b *Builder[T]) Chain(methodName string, args ...any) *Builder[T]
func (b *Builder[T]) Error() error
```

---

## `changes` Subpackage

Import path: `github.com/Nigel2392/go-django/src/core/attrs/changes`.

The `changes` subpackage compares two instances of the same `attrs.Definer` model field-by-field using `contrib/admin/compare` and returns rendered HTML diffs for changed fields:

```go
type Change struct {
	Name string        `json:"name"`
	HTML template.HTML `json:"html"`
}

func Changes(ctx context.Context, oldV, newV attrs.Definer) ([]Change, error)
```

If `reflect.TypeOf(oldV) != reflect.TypeOf(newV)`, `Changes` returns an `errors.TypeMismatch` error. Fields without a registered comparator (`errors.NotImplemented`) or returning `compare.ErrSkip` are skipped.
