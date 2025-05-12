# `attrs` package implementation details

The `attrs` package provides a way to define and manage model attributes.

The way to define these models is by implementing the [`Definer`](./interfaces.md#definer) interface.

This interface provides a way to retrieve the field definitions for the model.

---

## Implementing the `Definer` interface

To implement the `Definer` interface, you need to implement the `FieldDefs` method.

This method should return a `Definitions` interface, which provides the field definitions for the model.

---

### Automatically defining fields with `attrs` struct tags

The following struct tags are recognized by the `attrs` package:

#### Nullable Fields

- `attrs:"null"`

whether the field allows null values, this takes no value

#### Blank Fields

- `attrs:"blank"`

whether the field allows blank values, this takes no value

#### Read-only Fields

- `attrs:"readonly"`

whether the field is read-only, this takes no value

#### Primary Key Fields

- `attrs:"primary"`

whether the field is a primary key, this takes no value

#### Labels

- `attrs:"label"`

the label for the field, this takes a string value

#### Help Texts

- `attrs:"helptext"`

the help text for the field, this takes a string value

#### The column name

- `attrs:"column"`

the name of the column in the database, this takes a string value

#### Minimum Length

- `attrs:"min_length"`

the minimum length of the field, this takes a number value

#### Maximum Length

- `attrs:"max_length"`

the maximum length of the field, this takes a number value

#### Minimum Value

- `attrs:"min_value"`

the minimum value of the field, this takes a number value

#### Maximum Value

- `attrs:"max_value"`

the maximum value of the field, this takes a number value

#### Foreign Key Fields

- `attrs:"fk"`

The foreign key for the field, this takes the following formats:

**Will automatically use the primary key of the foreign model**

`fk:<package>.<type>`

**Will use the specified field as the primary key**

`fk:<package>.<type>,<field>`

#### One-to-One Relationships

- `attrs:"o2o"`

The one-to-one relationship for the field, this takes the following formats:

**will automatically use the primary key of the foreign model**

`o2o:<package>.<type>`

**will use the specified field as the primary key**

`o2o:<package>.<type>,<field>`

**will use the specified field as the primary key, and will use the specified through model**

The through model's source field will be the ID of the source model, and the target field will be the ID of the target model.

`o2o:<package>.<type>,<field>,<through>,<through_source>,<through_target>`

#### Default Values

- `attrs:"default"`

Default values can be set via struct tags, the `attrs` package will automatically parse the struct tags and set the default values.

The following default values are supported:

- any integer type
- any unsigned integer type
- any float type
- any string type
- any bool type
- `sql.NullBool`
- `sql.NullInt16`
- `sql.NullInt32`
- `sql.NullInt64`
- `sql.NullFloat64`
- `sql.NullString`
- `sql.NullTime`

---

#### Example struct

These can be chained inside of the struct tag, separated by a semicolon.

Multiple values can be separated by a comma.

Example:

```go
type MyStruct struct {
    ID   int    `attrs:"primary;readonly;label=ID;helptext=The unique identifier of the model"`
    Name string `attrs:"label=Name;helptext=The name of the model"`
    Age  int    `attrs:"label=Age;helptext=The age of the model"`
    Related *Related  `attrs:"fk:github.com/your/package.Related,ID"`
}

func (m *MyStruct) FieldDefs() attrs.Definitions {
    return attrs.AutoDefinitions(m)
}
```

### Manually defining fields

You can also manually define fields by calling the `attrs.Define` function.

This function takes your struct type as an argument, along with the field definitions.

The field definitions can be created using the `attrs.NewField`, or any other method that returns an `attrs.Field` interface type.

#### FieldConfig

The `attrs.FieldConfig` struct can be used to configure the field.

- `Null                 bool` - Wether the field allows null values.

- `Blank                bool` - Wether the field allows blank values.

- `ReadOnly             bool` - Wether the field is read-only.

- `Primary              bool` - Wether the field is a primary key.

- `NameOverride         string`

An optional override for the field name.

This can be used to override the field name, so your `ID` field can be adressable as `MyID` instead of `ID`.

- `Label                string` - The label for the field.

- `HelpText             string` - The help text for the field.

- `Column               string` - The name of the column in the database.

- `MinLength            int64` - The minimum length of the field.

- `MaxLength            int64` - The maximum length of the field.

- `MinValue             float64` - The minimum value of the field.

- `MaxValue             float64` - The maximum value of the field.

- `Attributes           map[string]interface{}` - The attributes for the field.

- `RelForeignKey        Relation` - The related (many to one) object for the field.

- `RelManyToMany        Relation` - A many to many relationship for the field.

- `RelOneToOne          Relation` - A one to one relationship for the field.

- `RelForeignKeyReverse Relation` - A one to many relationship for the field.

- `Default              any` - The default value for the field.

- `Validators           []func(interface{}) error` - Validators for the field.

- `FormField            func(opts ...func(fields.Field)) fields.Field`

A function which returns a form field for the field.

The options are automatically generated by the field itself, and then passed to this function.

- `WidgetAttrs          map[string]string` - The attributes for the form widget of this field.

- `FormWidget           func(FieldConfig) widgets.Widget` - The actual form widget for the field.

- `Setter               func(Definer, interface{}) error` - A custom setter for the field.

- `Getter               func(Definer) (interface{}, bool)` - A custom getter for the field.

- `OnInit               func(Definer, *FieldDef, *FieldConfig) *FieldConfig`

A function that is called when the field is initialized.

This can be used to set additional configuration for the field, based on the field itself.

#### Example

Example:

```go
type MyStruct struct {
    ID   int
    Name string
    Age  int
    Related *Related
}

func (m *MyStruct) FieldDefs() attrs.Definitions {
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
        attrs.NewField(m, "Age", &attrs.FieldConfig{
            Label:    "Age",
            HelpText: "The age of the model",
            MinValue: 0,
            MaxValue: 100,
        }),
        attrs.NewField(m, "Related", &attrs.FieldConfig{
            RelForeignKey: attrs.Relate(&Related{}, "", nil),
            Column:        "related_id",
        }),
    )
}
```
