# `attrs` package interfaces

The `attrs` package provides a set of interfaces that can be used to define and manage model attributes.

These interfaces provide a uniform way to define and manage attributes.

## `Definer`

Any model should implement the [`Definer`](./implementation.md#implementing-the-definer-interface) interface.

This interface provides a way to retrieve the field definitions for the model.

```go
type Definer interface {
    FieldDefs() Definitions
}
```

## `Definitions`

Definitions is the interface that wraps the methods for a model's field definitions.

This can be used to retrieve, set or overall manipulate the fields of a model.

```go
type Definitions interface {
    // TableName retrieves the name of the table in the database.
    //
    // This can be used to generate the SQL for the model.
    TableName() string

    // Retrieves the field with the given name.
    //
    // If the field is not found, the second return value will be false.
    Field(name string) (f attrs.Field, ok bool)

    // Retrieves the primary field.
    Primary() attrs.Field

    // Instance returns the underlying model instance.
    Instance() Definer

    // Retrieves a slice of all fields.
    //
    // The order of the fields is the same as they were defined.
    Fields() []attrs.Field

    // Retrieves the number of fields.
    Len() int

    // Set sets the value of the field with the given name (or panics if not found).
    Set(name string, value interface{}) error

    // Retrieves the value of the field with the given name (or panics if not found).
    Get(name string) interface{}
}
```

## `FieldDefinition`

The FieldDefinition interface is a sort- of static definition of a field.

It does not allow for setting, getting or retrieving the default value of the field.

These are used in the [`ModelMeta`](./model-meta.md) interface type.

```go
type FieldDefinition interface {
    Labeler
    Helper

    // The name of the field
    Name() string

    // Tag retrieves the tag value for the field with the given name.
    Tag(name string) string

    // Retrieves the underlying model instance.
    Instance() Definer

    // ColumnName retrieves the name of the column in the database.
    //
    // This can be used to generate the SQL for the field.
    ColumnName() string

    // Type returns the reflect.Type of the field.
    Type() reflect.Type

    // Attrs returns any extra attributes for the field, these can be used for multiple purposes.
    //
    // Additional info can be stored here, for example - if the field has a min / max length.
    Attrs() map[string]any

    // Rel etrieves the related model instance for a foreign key field.
    //
    // This could be used to generate the SQL for the field.
    Rel() Relation

    // Reports whether the field is the primary field.
    //
    // A model can technically have multiple primary fields, but this is not recommended.
    //
    // When for example, calling `Primary()` on the `Definitions` interface - only one will be returned.
    IsPrimary() bool

    // Reports whether the field is allowed to be null.
    //
    // If not, the field should panic when trying to set the value to nil / a reflect.Invalid value.
    AllowNull() bool

    // Reports whether the field is allowed to be blank.
    //
    // If not, the field should panic when trying to set the value to a blank value if the field is not of types:
    // bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, complex64, complex128.
    //
    // this means that for example, a string field should panic when trying to set the value to an empty string.
    AllowBlank() bool

    // Reports whether the field is allowed to be edited.
    //
    // If not, the field should panic when trying to set the value, unless the force parameter passed to the `SetValue` method is true.
    AllowEdit() bool
    // Retrieves the form field for the field.
    //
    // This is used to generate forms for the field.
    FormField() fields.Field

    // Validates the field's value.
    Validate() error
}
```

## `Field`

The `Field` interface is the interface that wraps the methods for a model's field.

This can be used to retrieve, set or overall manipulate the fields of a model.

Fields should also implement the before- mentioned `FieldDefinition` interface.

```go

type Field interface {
    FieldDefinition
    
    // Scan the value of the field into your model.
    //
    // This allows for reading the value easily from the database.
    sql.Scanner
    
    // Return the value of the field as a driver.Value.
    //
    // This value should be used for storing the field in a database.
    //
    // If the field is nil or the zero value, the default value should be returned.
    driver.Valuer
    
    // ToString returns a string representation of the value.
    //
    // This should be the human-readable version of the value, for example for a list display.
    ToString() string
    
    // Retrieves the value of the field.
    GetValue() interface{}
    
    // Retrieves the default value of the field.
    //
    // Fields should also check the main model instance for methods like `GetDefault<FieldName>` to retrieve the default value.
    GetDefault() interface{}
    
    // Sets the value of the field.
    //
    // If the field is not allowed to be edited, this method should panic.
    // If the field is not allowed to be null, this method should panic when trying to set the value to nil / a reflect.Invalid value.
    // If the field is not allowed to be blank, this method should panic when trying to set the value to a blank value if the field is not of types:
    SetValue(v interface{}, force bool) error
}
```

## `CanRelatedName`

A field can implement the `CanRelatedName` interface.

This interface is used to retrieve the remote- related name of the field.

This is to adress a reverse relationship.

```go
type CanRelatedName interface {
    Field
    RelatedName() string
}
```

## `Labeler`

A field can implement the `Labeler` interface.

This interface is used to retrieve the label of the field.

The value of the field can also implement the labeler interface - unless overridden this
should get returned by the `Label()` method instead.

```go
type Labeler interface {
    // Label returns the human-readable name of the field.
    //
    // This is the name that is displayed to the user in for example, forms and column headers.
    Label() string
}
```

## `Helper`

A field can implement the `Helper` interface.

This interface is used to retrieve the help text of the field.

The value of the field can also implement the helper interface - unless overridden this
should get returned by the `HelpText()` method instead.

```go
type Helper interface {
    // HelpText returns a description of the field.
    //
    // This is displayed to the user in for example, forms.
    HelpText() string
}
```

## `Scanner`

A field's value can implement the `Scanner` interface.

This interface is used to scan the value of the field into the value itself.

This can be useful for custom conversions or initialisation of the field.

```go
type Scanner interface {
    // ScanAttribute scans the value of the attribute.
    //
    // This is used to set the value of the field from a raw value.
    ScanAttribute(src any) error
}
```
