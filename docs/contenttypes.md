# Content Types

Content types are helper structures that simplify managing models and their human-readable representations.

## The `ContentType` Interface

The ContentType interface itself stores the string representations of the model's content type.

It can be used to retrieve the definition from the registry.

The Base content type implementation provided by Go-Django implements the sql.Scanner and driver.Valuer interfaces, allowing it to be used as attributes on your models.

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
- **GetInstancesByIDs([]interface{}) ([]interface{}, error)**: Retrieves a list of instances by their IDs.
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
- **InstancesByIDs(ids []interface{}) ([]interface{}, error)**: Retrieves a list of instances by their IDs.
  **warning**: This method is optional and falls back to calling `Instance` for each ID in a gourotine if not implemented.

## Working with the Content Type Registry

Content types are registered and managed via the `contenttypes` package, which provides registry management functions:

- **Register(definition *ContentTypeDefinition)**: Registers a content type and aliases.
- **Aliases(typeName string) []string**: Returns aliases for a model's type name.
- **ReverseAlias(alias string) string**: Returns the type name for an alias.
- **RegisterAlias(alias string, typeName string)**: Registers an alias for a type name.
- **EditDefinition(def *ContentTypeDefinition)**: Edits a content type definition, allowing for easily changing certain properties.
- **ListDefinitions() []*ContentTypeDefinition**: Lists all registered content type definitions.
- **DefinitionForType(typeName string) *ContentTypeDefinition**: Retrieves the definition by type name.
- **DefinitionForObject(obj any) *ContentTypeDefinition**: Retrieves the definition for an object.
- **ListDefinitions() []*ContentTypeDefinition**: Lists all registered content type definitions.
- **DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition**: Retrieves the definition by package and type name.
- **GetInstance(typeName string, id interface{}) (interface{}, error)**: Returns an instance by ID.
- **GetInstances(typeName string, amount, offset uint) ([]interface{}, error)**: Returns a list of instances with pagination.
- **GetInstancesByIDs(typeName string, ids []interface{}) ([]interface{}, error)**: Returns a list of instances queried by their IDs.

## Examples

### Registering a Content Type

Registering a content type is straightforward, and it has to be done once for each model to easily work with Go-Django.

This allows for using the object in the admin area, or form widgets with minimal setup.

The model we will work with can be copied from our [todo's example](./examples/todos.md).

```go
package main

import (
    "github.com/Nigel2392/go-django/src/core/contenttypes"
)

contenttypes.Register(&contenttypes.ContentTypeDefinition{
    // The model itself.
    //
    // This must be either a struct, or a pointer to a struct.
    ContentObject: &Todo{},

    // A function that returns the human-readable name of the model.
    //
    // This can be used to provide a custom name for the model.
    GetLabel: func() string {
        return "Todo Model"
    },

    // A function to return a pluralized version of the model's name.
    //
    // This can be used to provide a custom plural name for the model.
    GetPluralLabel: func() string {
        return "Todo Models"
    },

    // A function that returns a description of the model.
    //
    // This should return an accurate description of the model and what it represents.
    GetDescription: func() string {
        return "A model to represent todos."
    },

    // A function which returns the label for an instance of the content type.
    //
    // This is used to get the human-readable name of an instance of the model.
    GetInstanceLabel: func(instance any) string {
        return instance.(*Todo).Title
    },

    // A function that returns a new instance of the model.
    // 
    // This should return a new instance of the model that can be safely typecast to the
    // correct model type.
    GetObject: func() any {
        return &Todo{}
    },

    // A function to retrieve an instance of the model by its ID.
    GetInstance: func(identifier any) (interface{}, error) {
        return GetTodoByID(
            context.Background(),
            identifier,
        )
    },

    // A function to get a list of instances of the model.
    GetInstances: func(amount, offset uint) ([]interface{}, error) {
        var todos, err = ListAllTodos(
            context.Background(), amount, offset,
        )
        return todos, nil
    },

    // A function to get a list of instances of the model by a list of IDs.
    //
    // Falls back to calling Instance for each ID if GetInstancesByID is not implemented.
    GetInstancesByIDs: func([]interface{}) ([]interface{}, error) {
        return GetTodosByIDs(context.Background(), ids)
    }

    // A list of aliases for the model.
    //
    // This can be used to provide additional names for the model and make it easier to
    // reference the model in code from the registry.
    //
    // For example, after a big refactor or renaming of a model, you can add the old name
    // as an alias to make it easier to reference the model in code.
    //
    // This should be the full type name of the model, including the package path.
    Aliases: []string{
        // Full type name of the model.
        "github.com/Nigel2392/todos.Todo",
    },
})
```

#### Extra aliases explanation

The `Aliases` field is used to provide additional names for the model, making it easier to reference the model in code from the registry.

Say you have the following project structure:

```sh
github.com/Nigel2392/todos
├── models
│   └── todo.go
└── views
    └── todo.go
```

The model is currently in the models package, able to be referred as `github.com/Nigel2392/todos/models.Todo`.

But as the project grows, we might want to move the model to another package:

```sh
github.com/Nigel2392/todos
├── views
│   └── todo.go
└── todo.go
```

The model can now be referred to as `github.com/Nigel2392/todos.Todo`.

Except, the database does not know this. It still references the old model name, `github.com/Nigel2392/todos/models.Todo`, thus breaking the application.

This means we will either:

 1. Have to update the database to reflect the new model name.
 2. Add the old model name as an alias to the new model name.

The second option is the easiest and most efficient way to handle this situation.

```go
Aliases: []string{
    // Full type name of the model.
    "github.com/Nigel2392/todos/models.Todo",
},
```

### Using the Content Type Registry

Once the content type is registered, you can use the registry to access the model's content type and other metadata.

This can be done by referring to the model with a string, or by using the model's type directly.

You can then preform actions with the content type definition.

```go

// Get the content type definition for the model.
var (
    // Reverse lookup by alias (from the previous ContentTypeDefinition example).
    todoContentType = contenttypes.DefinitionForType("github.com/Nigel2392/todos/models.Todo")
    
    // Forward lookup by type name.
    todoContentType = contenttypes.DefinitionForType("github.com/Nigel2392/todos.Todo")

    // Directly using the model type.
    todoContentType = contenttypes.DefinitionForObject(&Todo{})

    // Using the toplevel package name and type name.
    todoContentType = contenttypes.DefinitionForPackage("todos", "Todo")
)
```
