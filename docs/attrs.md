# Attributes

Attributes are used to define the properties of a model.

This is useful for providing form functionality and more.

Embedded struct support is easily implemented by your own code.

The `Attributes` interface allows for customizability of the model and it's formfields when auto-generating forms for it's fields, setting & retrieving values and more.

## SQLC Plugin

A SQLC plugin has been implemented to help generate [the definer interface](#definer-interface) for your models.

This plugin is not included in the main package, but can be found in the `cmd` subpackage.

It can be installed by running the following command:

```bash
go install github.com/Nigel2392/go-django/cmd/go-django-definitions@latest
```

This will install the plugin in your `$GOPATH/bin` directory and can later be referenced in your sqlc configuration file.

### SQLC Plugin Configuration

To use the plugin, you need to add the `go-django-definitions` plugin to your sqlc configuration file.

To learn more about SQLC, please refer to the [SQLC documentation](https://docs.sqlc.dev/).

An example configuration file is provided below:
  
```yaml
version: "2"
plugins:
  - name: go-django-definitions-plugin
    process:
      # go install github.com/Nigel2392/go-django/cmd/go-django-definitions@latest
      cmd: "go-django-definitions"
sql:
  - schema: "./schema.sql"
    queries: "./queries.sql"
    engine: "mysql"
    codegen:
    - out: ./gen
      # The name of the plugin as defined above in the plugins dictionary.
      plugin: go-django-definitions-plugin
      options:
        # required, name of the package to generate
        package: "mypackage" 

        # optional, default is "<package>_definitions.go"
        out: "mydefinitions.go" 

        # optional, default is false, generates extra functions to easily set up the admin panel for the included models
        generate_admin_setup: true

        # optional, default is false, generates extra methods to adhere to models.Saver, models.Updater, models.Deleter and models.Reloader interfaces
        generate_models_methods: true

        # optional, see https://docs.sqlc.dev/en/stable/reference/config.html
        initialisms: ["id", "api", "url"] 

        # optional, see https://docs.sqlc.dev/en/stable/howto/rename.html
        rename: 

    gen:
      go:
        package: "mypackage"
        out: "./gen"
        emit_json_tags: true
        emit_prepared_queries: true
        emit_result_struct_pointers: true
        emit_interface: true
        query_parameter_limit: 8
```

## Interfaces

To keep things uniform when working with attributes we have implemented a few interfaces.

These interfaces are already implemented in the framework, but for the sake of customizability you can implement them yourself.

### `Definer` interface

The `Definer` interface is used to define the properties of a model.

This interface has a single method, and should be defined on the model itself.

The interface definition:

```go
type Definer interface {
    FieldDefs() attrs.Definitions
}
```

### `Definitions` interface

The definitions interface should be considered some kind of management struct for your fields.

This is the primary way we recommend to interact with your fields, setting values and more.

The interface is defined as follows:

```go
type Definitions interface {
    // Set sets the value of the field with the given name (or panics if not found).
    Set(name string, value interface{}) error

    // Retrieves the value of the field with the given name (or panics if not found).
    Get(name string) interface{}

    // Retrieves the field with the given name.
    //
    // If the field is not found, the second return value will be false.
    Field(name string) (f Field, ok bool)

    // Set sets the value of the field with the given name (or panics if not found).
    //
    // This method will allow setting the value of a field that is marked as not editable.
    ForceSet(name string, value interface{}) error

    // Retrieves the primary field.
    Primary() Field

    // Retrieves a slice of all fields.
    //
    // The order of the fields is the same as they were defined.
    Fields() []Field

    // Retrieves the number of fields.
    Len() int
}
```

### `Scanner` interface

The `Scanner` interface is used to scan a value into a field.

This should be a method on the value of the field.

I.E. to scan a value into a `StringField`, the method should be defined on the `StringField` struct.

Example:

```go
type StringField struct {
    Value string
}

func (s *StringField) ScanAttribute(src any) error {
    if v, ok := src.(string); ok {
        s.Value = v
        return nil
    }
    return errors.New("Invalid type")
}

type MyStruct struct {
    MyField StringField
}
```

This allows for more complex logic when setting values.

The interface is defined as follows:

```go
type Scanner interface {
    ScanAttribute(src any) error
}
```

### `Field` Interface

The field interface is used to define the properties of a field.

The interface itself also adheres to the following interfaces:

- `Labeler`
- `Helper`
- `Stringer`
- `Namer`

This allows each field to easily define labels, help texts, string representations and names for forms.

The full interface is defined as follows:

```go
type Field interface { // size=16 (0x10)
    Labeler
    Helper
    Stringer
    Namer
    Instance() Definer
    IsPrimary() bool
    AllowNull() bool
    AllowBlank() bool
    AllowEdit() bool
    GetValue() interface{}
    GetDefault() interface{}
    SetValue(v interface{}, force bool) error
    FormField() fields.Field
    Validate() error
}
```

As seen it is a quite extensive interface.

This is to allow for maximum customizability of the fields, while not sacrificing functionality in forms.

The methods are explained as follows:

- `Instance() Definer`  
  Returns the struct instance of the field.
- `IsPrimary() bool`  
  Returns whether the field is a primary identifier of this object.  
  For example, this can be used to easily identify the primary key field in a database.
- `AllowNull() bool`  
  Returns whether the field allows nil values.  
  If this is not true; a panic will be raised if the return object of `reflect.ValueOf` returns false when calling `IsValid` on that object.
- `AllowBlank() bool`  
  Returns whether the field allows blank (zero) values in forms.  
  If this is false, formfields generated by this (by default) will be required.  
  The framework does not always panic if a zero value is passed to a field that does not allow it, we only panic if the field's value is zero (reflect.IsZero) and not of types:
  - `bool`
  - `int/int8/int16/int32/int64`
  - `uint/uint8/uint16/uint32/uint64/uintptr`
  - `float32/float64`
  - `complex64/complex128`
- `AllowEdit() bool`  
  Mark this field as non-editable (editable by default).  
  This means a panic will be raised if the value of this field gets set.  
  Values still can be read at all times. Setting the value can also still occur, if the `force` parameter is set to true.
- `GetValue() interface{}`  
  Returns the value of the field.
- `GetDefault() interface{}`  
  Returns a default value for the field.  
  This can be overridden on the `Definer` struct by creating a method called `GetDefault<FieldName>` or optionally by populating the `Default` key in the `FieldConfig` struct.  
  When populating in the `FieldConfig` struct, a function that returns the default value for this is also allowed.  
  *warning*: Do not use pointer values as a default - this may cause unexpected behavior.
- `SetValue(v interface{}, force bool) error`  
  Sets the value of the field.  
  If `force` is set to true, the value will be set regardless of the `AllowEdit` setting.  
  If the value is not of the correct type, a panic will be raised.
- `FormField() fields.Field`
  Returns a form field for this field.  
  This is used to auto-generate forms for the model.
- `Validate() error`
  Validates the field.  
  This can be used to check if the field is valid.  
  If the field is not valid, an error should be returned.  
  If the field is valid, nil should be returned.  
  The value of the field will still be set, even if the field is not valid - errors should be handled by the caller.

#### `Namer` interface

The `Namer` interface is used to define the name of a field.

This is useful for form generation and possibly even database columns, etc.

The interface is defined as follows:

```go
type Namer interface {
    Name() string
}
```

For a form, this is the name attribute of the field; any other implementation may vary.

#### `Stringer` interface

The `Stringer` interface is used to define the string representation of a field.

This interface varies from the `fmt.Stringer` interface; the method is called `ToString` instead of `String`.

It is mainly used in list representations; and should provide a human-readable string representation of the field.

The interface is defined as follows:

```go
type Stringer interface {
    ToString() string
}
```

#### `Labeler` interface

The `Labeler` interface is used to define the label of a field.

This should be the human-readable representation of the name of the field.

I.E. if your field's name is `firstName`, an appropriate label might be `First Name`.

Internally it is used in forms to generate labels for the fields or in lists to generate column headers.

The interface is defined as follows:

```go
type Labeler interface {
    Label() string
}
```

#### `Helper` interface

The `Helper` interface is used to define a help text for a field.

This is useful for providing additional information about the field, such as in forms.

The interface is defined as follows:

```go
type Helper interface {
    HelpText() string
}
```

## Structs

### `FieldDef` struct

The `FieldDef` struct is used to define the properties of a field.

It adheres to the `Field` interface, and should provide enough capabilities to customize fields to your liking.

The struct definition is not important; and has only private properties.

Creating a new field is done by calling the `NewField` function.

Example:

```go
var myStruct = myStruct{
  MyField: "value",
}

var myField = attrs.NewField(myStruct, "MyField", attrs.FieldConfig{
    // ... config data
})
```

### `FieldConfig` struct

The `FieldConfig` struct is used to define the configuration of a field.

The struct is defined as follows:

```go
type FieldConfig struct {
    Null       bool
    Blank      bool
    ReadOnly   bool
    Primary    bool
    Label      string
    HelpText   string
    Default    any
    Validators []func(interface{}) error
    FormField  func(opts ...func(fields.Field)) fields.Field
    FormWidget func(FieldConfig) widgets.Widget
    Setter     func(Definer, interface{}) error
    Getter     func(Definer) (interface{}, bool)
}
```

The struct is used to define the properties of a field, as well as setters, getters, form fields, etc.

The setters/getters also take an argument of type `Definer`.

This is the parent struct which the field belongs to, and thus can be safely cast to the parent struct's type.

#### Properties

- `Null bool`  
  Whether the field allows nil values.
- `Blank bool`  
  Whether the field allows blank (zero) values.
- `ReadOnly bool`  
  Whether the field is read-only.
- `Primary bool`  
  Whether the field is a primary key.
- `Label string`  
  The label of the field.
- `HelpText string`  
  The help text of the field.
- `Default any`  
  The default value of the field.
- `Validators []func(interface{}) error`  
  A slice of validators for the field.
- `FormField func(opts ...func(fields.Field)) fields.Field`  
  A function that returns a self-defined form field for the field.
- `FormWidget func(FieldConfig) widgets.Widget`  
  A function that returns a custom form widget for the field.
- `Setter func(Definer, interface{}) error`  
  A function that sets the value of the field.
- `Getter func(Definer) (interface{}, bool)`  
  A function that retrieves the value of the field.

## Defining Model Attributes

To define the attributes of a model, you need to implement the `Definer` interface on the model.

The `FieldDefs` method should return a `Definitions` struct.

Let's create a simple struct on which we will be defining the attributes:

```go
type myInt int

type MyModel struct {
    ID   int
    Name string
    Bio  string
    Age  myInt
}
```

Now we need to configure the fields of this struct.

Formfields will be generated automatically, but we want to customize the Bio field to use a `Textarea` widget.

We also want to add a validator to the `Age` field to ensure it is greater than 0.

The `FieldDefs` method should be implemented as follows:

```go
func (m *MyModel) FieldDefs() attrs.Definitions {
  return attrs.Define(m,
    attrs.NewField(m, "ID", &attrs.FieldConfig{
      Primary:  true,
      ReadOnly: true,
      Label:    "ID",
      HelpText: "The unique identifier of the model",
    }),
    attrs.NewField(m, "Name", &attrs.FieldConfig{
      Label:    "Name",
      HelpText: "The name of the model",
    }),
    attrs.NewField(m, "Bio", &attrs.FieldConfig{
      Label:    "Biography",
      HelpText: "The biography of the model",
      FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
        return widgets.NewTextarea(nil)
      },
    }),
    attrs.NewField(m, "Age", &attrs.FieldConfig{
      Label:    "Age",
      HelpText: "The age of the model",
      Validators: []func(interface{}) error{
        func(v interface{}) error {
          if v.(myInt) <= myInt(0) {
            return errors.New("Age must be greater than 0")
          }
          return nil
        },
      },
    }),
  )
}
```

This will define the fields of the model, and customize the Bio field to use a `Textarea` widget, and add a validator to the Age field.

To then set or retrieve the values; you could use the attrs package- level functions, or directly interact with the `Definitions` struct.

### Embedding Structs

Embedding a struct is relatively easy; there are a few options which each depend on the already existing implementation of the underlying struct.

In short; if the underlying struct implements the `Definer` interface, you can embed it directly using it's `FieldDefs().Fields()` method.

If the underlying struct does not implement the `Definer` interface, you can still embed it, but you will have to manually define the fields.

#### Option 1: The embedded struct implements the `Definer` interface

When the embedded struct implements the `Definer` interface, you can embed it's fields directly.

Let's extend the previous example with an embedded struct:

```go
type MyTopLevelModel struct {
    MyModel
    Address string
}
```

The `MyModel` struct implements the `Definer` interface, so we can embed it directly.

The `FieldDefs` method should be implemented as follows:

```go
func (m *MyTopLevelModel) FieldDefs() attrs.Definitions {
    var fields = m.MyModel.FieldDefs().Fields()
    fields = append(fields, attrs.NewField(m, "Address", &attrs.FieldConfig{
        Label:    "Address",
        HelpText: "The address of the model",
    }))
    return attrs.Define(m, fields...)
}
```

#### Option 2: Embedding structs which do not implement the `Definer` interface

When the embedded struct does not implement the `Definer` interface, you will have to manually define the fields.

Let's pretend the `MyModel` struct does not implement the `Definer` interface:

```go
type MyModel struct {
    ID   int
    Name string
    Bio  string
    Age  myInt
}

type MyTopLevelModel struct {
    MyModel
    Address string
}
```

The `FieldDefs` method should be implemented in a way to still include the fields of the embedded struct:

```go
func (m *MyTopLevelModel) FieldDefs() attrs.Definitions {
    return attrs.Define(m,
        attrs.NewField(m.MyModel, "ID", &attrs.FieldConfig{
            Primary:  true,
            ReadOnly: true,
            Label:    "ID",
            HelpText: "The unique identifier of the model",
        }),
        attrs.NewField(m.MyModel, "Name", &attrs.FieldConfig{
            Label:    "Name",
            HelpText: "The name of the model",
        }),
        attrs.NewField(m.MyModel, "Bio", &attrs.FieldConfig{
            Label:    "Biography",
            HelpText: "The biography of the model",
            FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
                return widgets.NewTextarea(nil)
            },
        }),
        attrs.NewField(m.MyModel, "Age", &attrs.FieldConfig{
            Label:    "Age",
            HelpText: "The age of the model",
            Validators: []func(interface{}) error{
                func(v interface{}) error {
                    if v.(myInt) <= myInt(0) {
                        return errors.New("Age must be greater than 0")
                    }
                    return nil
                },
            },
        }),
        attrs.NewField(m, "Address", &attrs.FieldConfig{
            Label:    "Address",
            HelpText: "The address of the model",
        }),
    )
}
```

As seen in the example, we have to manually define the fields of the embedded struct.

We must still bind the fields to the embedded struct, even if the `FieldDefs` method is called on the parent struct.

This allows for a more flexible way of defining fields, and allows for more complex models and overrides.

### Automatic definitions of fields

The `attrs` package provides a way to automatically define fields for a struct.

This can be done using struct-tags.

- null  
    Whether the field allows nil values.
- blank  
    Whether the field allows blank (zero) values.
- readonly  
    Whether the field is read-only.
- primary  
    Whether the field is a primary key.
- label  
    The label of the field.
- helptext  
    The help text of the field.
- default  
    The default value of the field.
    Allowed types for the struct tags are:
  - `string` (no parentheses)
  - `int/int8/int16/int32/int64`
  - `uint/uint8/uint16/uint32/uint64/uintptr`
  - `float32/float64`
  - `bool`

Let's create a struct to define with struct tags:

```go
type MyModel struct {
    ID   int    `attrs:"primary;readonly;label=ID;helptext=The unique identifier of the model"`
    Name string `attrs:"label=Name;helptext=The name of the model"`
    Bio  string `attrs:"label=Biography;helptext=The biography of the model"`
    Age  myInt  `attrs:"label=Age;helptext=The age of the model"`
}
```

The `FieldDefs` method should be implemented as follows:

```go
func (m *MyModel) FieldDefs() attrs.Definitions {
    return attrs.AutoDefinitions(m)
}
```

Optionally, you could pass a variadic list of fields/strings to the `AutoDefinitions` function to include only those fields.

This allows for more flexibility when defining fields.

Example:

```go
func (m *MyModel) FieldDefs() attrs.Definitions {
    return attrs.AutoDefinitions(m, "ID", "Name")
}
```

This will only include the `ID` and `Name` fields in the definitions, and exclude the `Bio` and `Age` fields.

## Package- level functions

### `FieldNames(d any, exclude []string) []string`

A shortcut for getting the names of all fields in a Definer.

The exclude parameter can be used to exclude certain fields from the result.

This function is useful when you need to get the names of all fields in a  
model, but you want to exclude certain fields (e.g. fields that are not editable).

### `PrimaryKey(d Definer) interface{}`

PrimaryKey retrieves the primary key of a Definer.

This function will panic if the primary key is not found.

### `DefinerList[T Definer](list []T) []Definer

DefinerList converts a slice of []T where the underlying type is of type Definer to []Definer.

### `SetMany(d Definer, values map[string]interface{}) error`

SetMany sets multiple fields on a Definer.

The values parameter is a map where the keys are the names of the fields to set.

The values must be of the correct type for the fields.

### `Set(d Definer, name string, value interface{}) error`

Set sets the value of a field on a Definer.

If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.

If the field is marked as non editable, this function will panic.

### `ForceSet(d Definer, name string, value interface{}) error`

ForceSet sets the value of a field on a Definer.

If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.

This function will allow setting the value of a field that is marked as not editable.

### `Get[T any](d Definer, name string) T`

Get retrieves the value of a field on a Definer.

If the field is not found, this function will panic.

Type assertions are used to ensure that the value is of the correct type,
as well as providing less work for the caller.

### `ToString(v any) string`

ToString converts a value to a string.

This should be the human-readable representation of the value.

### `Method[T any](obj interface{}, name string) (n T, ok bool)`

Method retrieves a method from an object.

The generic type parameter must be the type of the method.

### `AutoDefinitions[T Definer](instance T, include ...any) Definitions`

AutoDefinitions automatically generates definitions for a struct.

It does this by iterating over the fields of the struct and checking for the
`attrs` tag. If the tag is present, it will parse the tag and generate the
definition.

If the `include` parameter is provided, it will only generate definitions for
the fields that are included.

### `Define(d Definer, fieldDefinitions ...Field) *ObjectDefinitions`

Define creates a new object definitions.

This can then be returned by the FieldDefs method of a model
to make it comply with the Definer interface.

### `NewField[T any](instance *T, name string, conf *FieldConfig) *FieldDef`

NewField creates a new field definition for the given instance.

This can then be used for managing the field in a more abstract way.

### `RegisterFormFieldType(valueOfType any, getField func(opts ...func(fields.Field)) fields.Field)`

RegisterFormFieldType registers a field type for a given valueOfType.

getField is a function that returns a fields.Field for the given valueOfType.

The valueOfType can be a reflect.Type or any value, in which case the reflect.TypeOf(valueOfType) will be used.

This is a shortcut function for the `HookFormFieldForType` hook.

### `RConvert(v *reflect.Value, t reflect.Type) (*reflect.Value, bool)`

RConvert converts a reflect.Value to a different type.

If the value is not convertible to the type, the original value is returned.

If the pointer of `v` is invalid, a new value of type `t` is created, and the pointer is set to it, then the pointer is returned.

### `RSet(src, dst *reflect.Value, convert bool) (canset bool)`

RSet sets a value from one reflect.Value to another.

If the destination value is not settable, this function will return false.

If the source value is not immediately assignable to the destination value, and the convert parameter is true,

the source value will be converted to the destination value's type.

If the source value is not immediately assignable to the destination value, and the convert parameter is false,
this function will return false.
