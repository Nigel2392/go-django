# `attrs` Package Documentation

The `attrs` package enables defining and managing model attributes with support for auto-generated forms, field validation, and custom form widgets.

## Core Concept

Attributes describe the properties of model fields—useful for form generation, value setting, validation, and more.

Embedded struct support is possible, including manual and automatic field registration.

## Interfaces

These interfaces provide a uniform way to define and manage attributes:

---

### `Definer`

A model implementing this interface provides its own field definitions.

```go
import (
  "github.com/Nigel2392/go-django/src/core/attrs"
)

type Definer interface {
  FieldDefs() attrs.Definitions
}
```

---

### `Definitions`

Manages field definitions for a model.

```go
type Definitions interface {
  Set(name string, value interface{}) error
  ForceSet(name string, value interface{}) error
  Get(name string) interface{}
  Field(name string) (attrs.Field, bool)
  Primary() attrs.Field
  Fields() []attrs.Field
  TableName() string
  Instance() attrs.Definer
  Len() int
}
```

---

### `Field`

Represents a model field and its behavior.

```go
type Field interface {
  sql.Scanner
  driver.Valuer
  attrs.Labeler
  attrs.Helper
  attrs.Stringer
  attrs.Namer

  Instance() attrs.Definer
  Tag(name string) string
  ColumnName() string
  Type() reflect.Type
  Attrs() map[string]any

  Rel() attrs.Definer
  ForeignKey() attrs.Definer
  ManyToMany() attrs.Relation
  OneToOne() attrs.Relation

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

---

### `Scanner`

Allows scanning raw values into custom field types.

```go
type Scanner interface {
  ScanAttribute(src any) error
}
```

---

### Additional Small Interfaces

```go
type Namer interface       { Name() string }
type Stringer interface    { ToString() string }
type Labeler interface     { Label() string }
type Helper interface      { HelpText() string }
```

---

## Field Configuration

### `FieldConfig`

Used to configure individual fields:

```go
type FieldConfig struct {
  Null          bool
  Blank         bool
  ReadOnly      bool
  Primary       bool
  Label         string
  HelpText      string
  Column        string
  MinLength     int64
  MaxLength     int64
  MinValue      float64
  MaxValue      float64
  Attributes    map[string]interface{}
  WidgetAttrs   map[string]string
  Default       any
  RelForeignKey attrs.Definer
  RelManyToMany attrs.Relation
  RelOneToOne   attrs.Relation
  Validators    []func(interface{}) error
  FormField     func(opts ...func(fields.Field)) fields.Field
  FormWidget    func(FieldConfig) widgets.Widget
  Setter        func(attrs.Definer, interface{}) error
  Getter        func(attrs.Definer) (interface{}, bool)
}
```

---

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

---

### Embedding Structs

Embedding a struct is relatively easy; there are a few options which each depend on the already existing implementation of the underlying struct.

In short; if the underlying struct implements the `Definer` interface, you can embed it directly using it's `FieldDefs().Fields()` method.

If the underlying struct does not implement the `Definer` interface, you can still embed it, but you will have to manually define the fields.

---

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

---

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

---

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

---

## Common Utilities

```go
attrs.FieldNames(definer, []string{"Exclude"})
attrs.PrimaryKey(definer)
attrs.SetMany(definer, map[string]interface{}{"Name": "New"})
attrs.Set(definer, "Name", "New")
attrs.ForceSet(definer, "Name", "Forced")
attrs.Get[string](definer, "Name")
attrs.ToString(value)
```

---

## Form Field Hooking

Register form field widgets for custom types:

```go
RegisterFormFieldType(
  json.RawMessage{},
  func(opts ...func(fields.Field)) fields.Field {
    return fields.JSONField[json.RawMessage](opts...)
  },
)
```

---

## Notes

- Panics occur on invalid sets unless `force` is used.
- Default values can be set via struct tags, the FieldConfig type or methods like `GetDefault<FieldName>()`.
