# Content Types

Content types are helper structures that simplify managing models and their human-readable representations.

## The `ContentType` Interface

The `ContentType` interface is defined as follows:

```go
type ContentType interface {
    PkgPath() string
    AppLabel() string
    TypeName() string
    Model() string
    New() interface{}
}
```

- **PkgPath**: Returns the package path of the model (e.g., `github.com/Nigel2392/mypackage`).
- **AppLabel**: Returns the app label, or last segment of the package path (e.g., `mypackage`).
- **TypeName**: Returns the full type name, including the package path (e.g., `github.com/Nigel2392/mypackage.MyModel`).
- **Model**: Returns the model name (e.g., `MyModel`).
- **New**: Returns a new instance of the model.

### ShortcutContentType Interface

The `ShortcutContentType` interface extends `ContentType` to allow a shorter alias to reference the model in code.

```go
type ShortcutContentType[T any] interface {
    ContentType
    ShortTypeName() string
}
```

- **ShortTypeName**: Returns a short alias, using the app label and model name (e.g., `mypackage.MyModel`).

## BaseContentType

`BaseContentType` implements the `ContentType` interface and provides SQL and JSON support.

```go
type BaseContentType[T any] struct {
    // Fields like rType, rTypeElem, pkgPath, modelName
}

func (c *BaseContentType[T]) PkgPath() string
func (c *BaseContentType[T]) AppLabel() string
func (c *BaseContentType[T]) TypeName() string
func (c *BaseContentType[T]) Model() string
func (c *BaseContentType[T]) New() T
func (c *BaseContentType[T]) ShortTypeName() string
func (c *BaseContentType[T]) Scan(src interface{}) error
func (c BaseContentType[T]) Value() (driver.Value, error)
func (c BaseContentType[T]) MarshalJSON() ([]byte, error)
func (c *BaseContentType[T]) UnmarshalJSON(data []byte) error
```

## ContentTypeDefinition Struct

The `ContentTypeDefinition` struct registers each model's content type with the registry, including metadata like aliases, label, and description.

### Attributes

- **ContentObject**: The model itself, which must be a struct or a pointer to a struct.
- **GetLabel func() string**: Returns the human-readable name of the model.
- **GetPluralLabel func() string**: Returns the plural name of the model.
- **GetDescription func() string**: Returns a description of the model.
- **GetInstanceLabel func(any) string**: Returns a label for an instance of the model.
- **GetObject func() any**: Returns a new instance of the model.
- **GetInstance func(interface{}) (interface{}, error)**: Retrieves an instance by ID.
- **GetInstances func(amount, offset uint) ([]interface{}, error)**: Retrieves multiple instances with pagination.
- **Aliases**: List of alternate names for referencing the model.

### Methods

- **ContentType() ContentType**: Returns the `ContentType` instance for the model.
- **Label() string**: Returns the model's human-readable name.
- **PluralLabel() string**: Returns the pluralized name.
- **Description() string**: Returns a description of the model.
- **InstanceLabel(instance any) string**: Returns the name of a model instance.
- **Object() any**: Returns a new instance of the model.
- **Instance(id interface{}) (interface{}, error)**: Retrieves an instance by its ID.
- **Instances(amount, offset uint) ([]interface{}, error)**: Retrieves a list of model instances.

## Working with the Content Type Registry

Content types are registered and managed via the `contenttypes` package, which provides registry management functions:

- **Register(definition *ContentTypeDefinition)**: Registers a content type and aliases.
- **Aliases(typeName string) []string**: Returns aliases for a model's type name.
- **ReverseAlias(alias string) string**: Returns the type name for an alias.
- **RegisterAlias(alias string, typeName string)**: Registers an alias for a type name.
- **DefinitionForType(typeName string) *ContentTypeDefinition**: Retrieves the definition by type name.
- **DefinitionForObject(obj any) *ContentTypeDefinition**: Retrieves the definition for an object.
- **ListDefinitions() []*ContentTypeDefinition**: Lists all registered content type definitions.
- **DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition**: Retrieves the definition by package and type name.
- **GetInstance(typeName string, id interface{}) (interface{}, error)**: Returns an instance by ID.
- **GetInstances(typeName string, amount, offset uint) ([]interface{}, error)**: Returns a list of instances with pagination.
