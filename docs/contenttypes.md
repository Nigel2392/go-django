# Content Types

Content types are metadata descriptors (`github.com/Nigel2392/go-django/src/core/contenttypes`) that represent model types across your Go-Django application. They provide consistent type identifiers, human-readable labels, SQL/JSON serialization, and lookup callbacks for generic apps such as `contrib/admin` and form choosers.

---

## Interfaces and `BaseContentType`

The package defines two interfaces for describing a model type:

```go
type GenericContentType interface {
	PkgPath() string
	AppLabel() string
	TypeName() string
	ShortTypeName() string
	Model() string
}

type ContentType interface {
	GenericContentType
	New() interface{}
}
```

- **`PkgPath()`**: Full Go import path of the model package (e.g. `github.com/Nigel2392/myapp`).
- **`AppLabel()`**: Final segment of `PkgPath()` (e.g. `myapp`).
- **`TypeName()`**: Full canonical type name (`PkgPath() + "." + Model()`, e.g. `github.com/Nigel2392/myapp.Todo`).
- **`ShortTypeName()`**: Short alias using the app label (`AppLabel() + "." + Model()`, e.g. `myapp.Todo`).
- **`Model()`**: Go struct name (e.g. `Todo`).
- **`New()`**: Instantiates a new pointer/value of the underlying model type as `interface{}`.

### `BaseContentType[T]`

`BaseContentType[T any]` is the standard implementation provided by the package. It implements `GenericContentType`, `sql.Scanner`, `driver.Valuer`, `json.Marshaler`, `json.Unmarshaler`, and `DBType() dbtype.Type` (returning `dbtype.Text`):

```go
type BaseContentType[T any] struct {
	// unexported fields
}

func NewContentType[T any](p T) *BaseContentType[T]
func NewRContentType[T any](p reflect.Type) *BaseContentType[T]
func ChangeBaseType[OldT, NewT any](c *BaseContentType[OldT]) *BaseContentType[NewT]

func (c *BaseContentType[T]) DBType() dbtype.Type
func (c *BaseContentType[T]) String() string
func (c *BaseContentType[T]) PkgPath() string
func (c *BaseContentType[T]) AppLabel() string
func (c *BaseContentType[T]) Model() string
func (c *BaseContentType[T]) New() T
func (c *BaseContentType[T]) TypeName() string
func (c *BaseContentType[T]) ShortTypeName() string
func (c *BaseContentType[T]) Scan(src interface{}) error
func (c BaseContentType[T]) Value() (driver.Value, error)
func (c BaseContentType[T]) MarshalJSON() ([]byte, error)
func (c *BaseContentType[T]) UnmarshalJSON(data []byte) error
```

- **`NewContentType`**: Builds a `*BaseContentType[T]` from a struct value or pointer.
- **`NewRContentType`**: Builds a `*BaseContentType[T]` from a `reflect.Type`.
- **`ChangeBaseType`**: Converts `*BaseContentType[OldT]` to `*BaseContentType[NewT]` while preserving the underlying type metadata.
- **`DBType()` / `String()`**: Returns `dbtype.Text` for database schema mapping, and `<pkgPath>.<modelName>` for formatting.
- **`Scan` / `UnmarshalJSON`**: Looks up the type string (or alias) via `DefinitionForType` and populates the receiver. Returns `ErrInvalidScanType` when scanning a non-string, non-`[]byte` value.

Do note that `*BaseContentType[T]` only satisfies the non-generic `ContentType` interface when `T` is `any` (because `New() T` must match `New() interface{}`). For concrete `T` types, `*BaseContentType[T]` satisfies `GenericContentType`; use `ChangeBaseType[T, any](ct)` when you need a `ContentType`.

---

## `ContentTypeDefinition`

`ContentTypeDefinition` binds a model type to human-readable metadata, instance factories, and query callbacks:

```go
type ContentTypeDefinition struct {
	ContentObject     any
	GetLabel          func(ctx context.Context) string
	GetPluralLabel    func(ctx context.Context) string
	GetDescription    func(ctx context.Context) string
	GetInstanceLabel  func(any) string
	GetObject         func() any
	GetInstance       func(context.Context, interface{}) (interface{}, error)
	GetInstances      func(ctx context.Context, amount, offset uint) ([]interface{}, error)
	GetInstancesByIDs func(context.Context, []interface{}) ([]interface{}, error)
	Aliases           []string
}
```

### Fields

- **`ContentObject`**: Sample instance of the model (struct or pointer to struct). Required when registering.
- **`GetLabel`**: Returns the singular, human-readable label for the model.
- **`GetPluralLabel`**: Returns the pluralized label for the model.
- **`GetDescription`**: Returns a description of what the model represents.
- **`GetInstanceLabel`**: Returns a display label for a specific model instance.
- **`GetObject`**: Factory returning a fresh instance of the model.
- **`GetInstance`**: Retrieves a single model instance by primary key or identifier.
- **`GetInstances`**: Retrieves a paginated slice of model instances.
- **`GetInstancesByIDs`**: Retrieves multiple model instances by a slice of identifiers.
- **`Aliases`**: Additional type names or legacy package paths that should resolve to this definition.

### Methods

- **`ContentType() ContentType`**: Returns (and lazily caches) the `ContentType` for `ContentObject`.
- **`Name() string`**: Returns the underlying Go struct name of `ContentObject`.
- **`Label(ctx context.Context) string`**: Calls `GetLabel(ctx)` if set, otherwise checks whether `ContentObject` implements `Label(context.Context) string`, and falls back to `trans.T(ctx, p.Name())`.
- **`PluralLabel(ctx context.Context) string`**: Calls `GetPluralLabel(ctx)` if set, otherwise pluralizes `p.Label(ctx)`.
- **`Description(ctx context.Context) string`**: Calls `GetDescription(ctx)` if set, otherwise returns `""`.
- **`InstanceLabel(instance any) string`**: Calls `GetInstanceLabel(instance)` if set, otherwise uses `fmt.Stringer` if non-empty, or falls back to `<Object "<Model>">`.
- **`Object() any`**: Calls `GetObject()` if set, otherwise returns `p.ContentType().New()`.
- **`Instance(ctx context.Context, id interface{}) (interface{}, error)`**: Calls `GetInstance(ctx, id)`.
- **`Instances(ctx context.Context, amount, offset uint) ([]interface{}, error)`**: Calls `GetInstances(ctx, amount, offset)`.
- **`InstancesByIDs(ctx context.Context, ids []interface{}) ([]interface{}, error)`**: Calls `GetInstancesByIDs(ctx, ids)` if set; otherwise spawns one goroutine per ID calling `Instance(ctx, id)` and joins any returned errors.

---

## Content Type Registry

The package exposes a global `ContentTypeRegistry` (and `NewContentTypeRegistry()` if you need an isolated registry instance) with the following functions:

```go
func NewContentTypeRegistry() *ContentTypeRegistry
func Register(definition *ContentTypeDefinition) *ContentTypeDefinition
func EditDefinition(def *ContentTypeDefinition)
func Aliases(typeName string) []string
func ReverseAlias(alias string) string
func RegisterAlias(alias string, typeName string)
func ListDefinitions() []*ContentTypeDefinition
func DefinitionForType(typeName string) *ContentTypeDefinition
func DefinitionForObject(obj any) *ContentTypeDefinition
func DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition
func GetInstance(ctx context.Context, typeName string, id interface{}) (interface{}, error)
func GetInstances(ctx context.Context, typeName string, amount, offset uint) ([]interface{}, error)
func GetInstancesByIDs(ctx context.Context, typeName string, ids []interface{}) ([]interface{}, error)
func OnRegister(fn func(def *ContentTypeDefinition)) (signals.Receiver[*ContentTypeDefinition], error)
```

- **`Register`**: Registers `definition` and returns it. If the type is already registered, `Register` delegates to `EditDefinition` instead.
- **`EditDefinition`**: Updates non-nil callbacks and appends any new `Aliases` on an existing definition. Panics if the type is not yet registered.
- **`Aliases` / `RegisterAlias` / `ReverseAlias`**: Manages aliases mapping alternate names back to canonical `TypeName()` strings.
- **`ListDefinitions`**: Returns all registered definitions sorted stably by model name and description.
- **`DefinitionForType` / `DefinitionForObject` / `DefinitionForPackage`**: Looks up a `*ContentTypeDefinition` by full type name/alias, sample object, or `<pkg>.<Model>`.
- **`GetInstance` / `GetInstances` / `GetInstancesByIDs`**: Convenience wrappers that look up the definition for `typeName` and invoke its corresponding method with `ctx`.
- **`OnRegister`**: Subscribes a callback to the `"contenttypes.OnRegister"` signal whenever a new content type is registered.

---

## Auto-Registration via `AppConfig.ModelObjects`

When you list your models in [`AppConfig.ModelObjects`](./apps.md), `django.Application.Initialize()` automatically registers each model with both `djangoapps.RegisterModel` and `contenttypes.Register(&contenttypes.ContentTypeDefinition{ContentObject: model})`.

During registration, `contenttypes` automatically registers short aliases for every model:

- `<pkgAppLabel>.<Model>` using `contentType.AppLabel()` (the last segment of the Go package path).
- `<appName>.<Model>` using the owning Django `AppConfig.Name()`, when it differs from `AppLabel()`.

Additionally, if you import `github.com/Nigel2392/go-django/queries/src`, its `attrs.OnBeforeModelRegister` listener automatically fills in `GetInstance`, `GetInstances`, and `GetInstancesByIDs` on your `ContentTypeDefinition` whenever they are not already set.

---

## Registering and Using Content Types

### Registering a Content Type Manually

Registering or customizing a `ContentTypeDefinition` lets you override labels, descriptions, aliases, and instance lookup behavior:

```go
package todos

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

func init() {
	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &Todo{},
		GetLabel: func(ctx context.Context) string {
			return "Todo Item"
		},
		GetPluralLabel: func(ctx context.Context) string {
			return "Todo Items"
		},
		GetDescription: func(ctx context.Context) string {
			return "A task item tracked by a user."
		},
		GetInstanceLabel: func(instance any) string {
			return instance.(*Todo).Title
		},
		GetObject: func() any {
			return &Todo{}
		},
		GetInstance: func(ctx context.Context, identifier any) (interface{}, error) {
			return GetTodoByID(ctx, identifier)
		},
		GetInstances: func(ctx context.Context, amount, offset uint) ([]interface{}, error) {
			return ListAllTodos(ctx, amount, offset)
		},
		GetInstancesByIDs: func(ctx context.Context, ids []interface{}) ([]interface{}, error) {
			return GetTodosByIDs(ctx, ids)
		},
		Aliases: []string{
			"github.com/example/todos/legacy.Todo",
		},
	})
}
```

### Handling Refactored Package Paths with Aliases

If a model moves from `github.com/example/todos/models.Todo` to `github.com/example/todos.Todo`, persisted references in the database may still contain the old `TypeName()`. Adding the previous full type name to `Aliases` preserves lookups without rewriting stored data:

```go
Aliases: []string{
	"github.com/example/todos/models.Todo",
},
```

### Querying the Registry

```go
// Lookup by canonical name, alias, short name, object, or package+model
defByAlias := contenttypes.DefinitionForType("github.com/example/todos/models.Todo")
defByName := contenttypes.DefinitionForType("github.com/example/todos.Todo")
defByShort := contenttypes.DefinitionForType("todos.Todo")
defByObj := contenttypes.DefinitionForObject(&Todo{})
defByPkg := contenttypes.DefinitionForPackage("todos", "Todo")

// Retrieve instances through the registry
item, err := contenttypes.GetInstance(ctx, "todos.Todo", 42)
items, err := contenttypes.GetInstances(ctx, "todos.Todo", 20, 0)
```

---

## `LazyRegistry`

`LazyRegistry` lets reusable packages depend on models (such as a custom user model) without hardcoding concrete types at compile time:

```go
func NewLazyRegistry(modelKey string, check func(*ContentTypeDefinition) bool) *LazyRegistry
func (r LazyRegistry) Load(typeNames ...string) *ContentTypeDefinition
func (r LazyRegistry) LoadString(typeNames ...string) string
func LoadModel(modelKey string, typeNames ...string) *ContentTypeDefinition
func LoadModelString(modelKey string, typeNames ...string) string
```

- **`NewLazyRegistry`**: Creates a registry under `modelKey` and attaches an `OnRegister` listener that indexes any newly registered `ContentTypeDefinition` where `check(def)` returns `true`.
- **`Load` / `LoadString`**: Returns the matching `*ContentTypeDefinition` or its `ShortTypeName()`. If no `typeNames` are passed, the registry's default model is used.
- **`LoadModel` / `LoadModelString`**: Global lookup helpers that resolve a `LazyRegistry` by `modelKey` and call `Load` or `LoadString`.

```go
package myapp

import (
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/mux/middleware/authentication"
)

var lazyUserModel = contenttypes.NewLazyRegistry("users.User", func(ctd *contenttypes.ContentTypeDefinition) bool {
	_, ok := ctd.ContentObject.(authentication.User)
	return ok
})

func ResolveUserContentType() *contenttypes.ContentTypeDefinition {
	// Loads the default matching model or a specific override if passed
	return lazyUserModel.Load()
}
```

---

## Caveats

- **Register `LazyRegistry` before models are registered**: `NewLazyRegistry` listens to `OnRegister`, which only fires when a type is registered for the first time. Models registered before `NewLazyRegistry` is called will not be added to that `LazyRegistry`, and re-registering an existing type via `Register` routes through `EditDefinition` without emitting `OnRegister`.
- **Alphabetical default selection in `LazyRegistry`**: When multiple models match `check(def)` and no `typeNames` are passed to `Load` or `LoadString`, the default is whichever matching model has the lexicographically smallest `ShortTypeName()`.
- **Panics on missing models or callbacks**:
  - `Load`, `LoadString`, `LoadModel`, and `LoadModelString` panic if `modelKey` is unknown or no matching model is registered (`LoadModel` falls back to `DefinitionForType(modelKey)` before panicking; `LoadModelString` does not).
  - `ContentTypeDefinition.Instance` and `ContentTypeDefinition.Instances` panic via `assert.Fail` if `GetInstance` or `GetInstances` is `nil`.
  - `EditDefinition` panics if called for an unregistered type.
