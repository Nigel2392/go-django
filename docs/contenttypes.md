# Content Types

Content types are helper structures which allow for simpler management of models and their human-readable representations.

## The `ContentType` interface

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

* `PkgPath` returns the package path of the model.  
  This is the import path of the package containing the model.  
  I.e. `github.com/Nigel2392/mypackage`.
* `AppLabel` returns the app label of the model.  
  This is the name of the app the model is in.  
  I.e. `myapp`.
* `TypeName` returns the full type name of the model.  
  This is the full package path and the model name.
  It is generally used in the database for generic relations.  
  I.e. `github.com/Nigel2392/mypackage.MyModel`.
* `Model` returns the name of the model.  
  This is the name of the model.  
  I.e. `MyModel`.
* `New` returns a new instance of the model.  
  This is a new instance of the model, created with reflection.  
  It works mostly like `new(MyModel)`, but with reflection and one key difference:  
  It returns the an interface with the underlying type of the model.

## Content type definitions

The ContentTypeDefinition struct is used to register each model's content type to the registry.

It allows for the definition of the model, as well as some additional metadata, such as aliases, a label and a description.

### Attributes

It has the following attributes:

#### `ContentObject any`

The model itself.

This must be either a struct, or a pointer to a struct.

#### `GetLabel func() string`

A function that returns the human-readable name of the model.

This can be used to provide a custom name for the model.

#### `GetDescription func() string`

A function that returns a description of the model.

This should return an accurate description of the model and what it represents.

#### `GetObject func() any`

A function that returns a new instance of the model.

This should return a new instance of the model that can be safely typecast to the
correct model type.

#### `Aliases []string`

A list of aliases for the model.

This can be used to provide additional names for the model and make it easier to
reference the model in code from the registry.

For example, after a big refactor or renaming of a model, you can add the old name
as an alias to make it easier to reference the model in code.

This should be the full type name of the model, including the package path.

## Working with the content type registry

Content types are registered to their own package.

It is possible to create a custom registry for content types, but this is not recommended.

### Functions

The default registry is available in the `contenttypes` package.

It exposes a few package-level functions to better work with content types.

#### `Register(definition *ContentTypeDefinition)`

Register a new content type definition.

This will add the content type to the registry, as well as any aliases.

#### `Aliases(typeName string) []string`

Return a list of aliases for a given type name.

This must be the full type name of the model, including the package path.

Example:

```go
aliases := contenttypes.Aliases("github.com/Nigel2392/mypackage.MyModel")
```

#### `ReverseAlias(alias string) string`

Return the type name for a given alias.

This will return the full type name of the model, including the package path.

Aliases are case sensitive, and must be registered with the global registry before they can be used.

This can be done by calling `RegisterAlias` on the global registry, or by calling `Register` with a definition that has the alias-list set.

#### `RegisterAlias(alias string, typeName string)`

Register an alias for a given type name.

This will add the alias to the global registry, and make it available for use in the `ReverseAlias` function.

Example:

```go
contenttypes.RegisterAlias("custompackage.MyModel", "github.com/Nigel2392/mypackage.MyModel")
```

It can later be retrieved with:

```go
typeName := contenttypes.ReverseAlias("custompackage.MyModel")
```

Or to retrieve the actual type definition:

```go
definition := contenttypes.DefinitionForType("custompackage.MyModel")
```

#### `DefinitionForType(typeName string) *ContentTypeDefinition`

Return the content type definition for a given type name.

This will return the definition for the model, if it is registered.

It will first try to check if there are any aliases registered with that name, and return the definition for the first alias found.

If no aliases are found, it will return the definition for the type name provided, if any.

#### `DefinitionForObject(obj any) *ContentTypeDefinition`

Return the content type definition for a given object.

This will return the definition for the model, if it is registered.

#### `ListDefinitions() []*ContentTypeDefinition`

Return a list of all registered content type definitions.

#### `DefinitionForPackage(toplevelPkgName string, typeName string) *ContentTypeDefinition`

Return the content type definition for a given package and type name.

This will return the definition for the model, if it is registered.

It is fully capable of handling aliases and will return the correct definition for the model.

Example:

```go
contenttypes.RegisterAlias(
  "custompackage.MyModel",
  "github.com/Nigel2392/mypackage.MyModel",
)

definition := contenttypes.DefinitionForPackage(
  "github.com/Nigel2392/mypackage", "MyModel",
)

definition := contenttypes.DefinitionForPackage(
  "custompackage", "MyModel",
)
```
