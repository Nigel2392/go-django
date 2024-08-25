# Fields

Fields are the basic building blocks of forms.  
They are used to create form elements such as text inputs, checkboxes, radio buttons, etc.

A field is a struct that implements the `Field` interface.

As with most of the forms- interfaces, the `Field` interface is quite extensive.

Fields are defined in their own `fields` package, which is a sub-package of the `forms` package.

## The `Field` interface

The `Field` interface is defined as follows:

```go
type Field interface {
 Attrs() map[string]string
 SetAttrs(attrs map[string]string)
 Hide(hidden bool)

 SetName(name string)
 SetLabel(label func() string)
 SetHelpText(helpText func() string)
 SetValidators(validators ...func(interface{}) error)
 SetWidget(widget widgets.Widget)

 Name() string
 Label() string
 HelpText() string
 Validate(value interface{}) []error
 Widget() widgets.Widget
 HasChanged(initial, data interface{}) bool

 Clean(value interface{}) (interface{}, error)
 ValueToForm(value interface{}) interface{}
 ValueToGo(value interface{}) (interface{}, error)
 Required() bool
 SetRequired(b bool)
 ReadOnly() bool
 SetReadOnly(b bool)
 IsEmpty(value interface{}) bool
}
```

* `Attrs` returns the attributes of the field.  
  This is a map of attributes that will be rendered in the HTML form.
* `SetAttrs` sets the attributes of the field.
* `Hide` sets the field to be hidden.
* `SetName` sets the name of the field.
* `SetLabel` sets the label of the field.
* `SetHelpText` sets the help text of the field.
* `SetValidators` sets the validators of the field.
* `SetWidget` sets the widget of the field.
* `Name` returns the name of the field.
* `Label` returns the label of the field.
* `HelpText` returns the help text of the field.
* `Validate` validates the value of the field.
* `Widget` returns the widget of the field.
* `HasChanged` checks if the value of the field has changed.
* `Clean` cleans the value of the field.
* `ValueToForm` converts the value to a form value.
* `ValueToGo` converts the value to a Go value.
* `Required` returns if the field is required.
* `SetRequired` sets if the field is required.
* `ReadOnly` returns if the field is read-only.
* `SetReadOnly` sets if the field is read-only.
* `IsEmpty` checks if the value passed is a zero value.

As seen, some setters take a function which returns a string.

This is to allow for dynamic values to be set, in practise this might help with translations.

## Field types

There are many different types of fields available in the `fields` package.

Most fields use the base- fields implementation, but some fields have expanded on it.

Fields are often adressd by their functions instead of instantiating the raw structs, this keeps the code uniform and allows for a more flexible API.

Most field- initializers take a set of options, these are functions that set the field's properties.

The following fields- functions are available by default:

### `NewField(type_ func() string, opts ...func(fields.Field)) *BaseField`

This is the base field, it is used to create new fields.

The `type_` function should return the type of the field, this is used to identify the field in the form.

The `opts` functions are used to set the field's properties.

### `CharField(opts ...func(fields.Field))`

A field for text input.

### `EmailField(opts ...func(fields.Field))`

A field for email input.

Automatically validates the input to be a valid email address using `mail.ParseAddress`.

### `BooleanField(opts ...func(fields.Field))`

A field for boolean input.

This field is rendered as a checkbox.

Setting this to required is not recommended, as the checkbox cannot be unchecked.

### `DateField(typ widgets.DateWidgetType, opts ...func(fields.Field))`

A field for date input.

The `typ` parameter is the type of the date widget to use.

It can be one of the following:

* `DateWidgetTypeDate`
* `DateWidgetTypeDateTime`

#### `DateWidgetTypeDate`

The default html date input.

#### `DateWidgetTypeDateTime`

The default html datetime-local input.

### `NumberField[T widgets.NumberType](opts ...func(fields.Field))`

A field for number input.

The `T` parameter is the type of the number widget to use.

It can be any GO float, int or uint type and will automatically be validated.

### `MarshallerField[T any](encoder func(io.Writer) fields.Encoder, decoder func(io.Reader) fields.Decoder, opts ...func(fields.Field)) *MarshallerFormField[T]`

A field for marshalling data.

The `encoder` function is used to encode the data to a writer.

The `decoder` function is used to decode the data from a reader.

### `JSONField[T any](opts ...func(fields.Field)) *JSONFormField[T]`

A field for JSON input.

This field will automatically marshal and unmarshal the data to and from JSON.

If any errors occur during marshalling or unmarshalling, a validation error will be returned.

### `Protect(w fields.Field, errFn func(err error) error) *ProtectedFormField`

A field that protects another field by catching any errors which occur during validation.

It will call the `errFn` function with the error that occurred.

This way you can handle the error in a custom way, I.E. for password fields.

## Example

Example usage of initializing a formfield with some options set.

```go
var myField = fields.CharField(
    // Sets the label of the field
    Label(label any)

    // Sets the help text of the field
    HelpText(helpText any)

    // Sets the name of the field (required for forms)
    Name(name string)

    // Sets the field to be required or not
    Required(b bool)

    // Sets the field to be read-only or not
    ReadOnly(b bool)

    // A regex pattern to validate the processed value of the field
    Regex(regex string)

    // A minimum length for the processed value of the field
    // 
    // If the value is a string, this will be the length of the string
    // 
    // If the value is a number, this will be the number minimum value
    MinLength(min int)

    // A maximum length for the processed value of the field
    //
    // If the value is a string, this will be the length of the string
    // 
    // If the value is a number, this will be the number maximum value
    MaxLength(max int)

    // Sets the widget of the field
    // 
    // This is the widget that will be used to render the field in the form
    // 
    // This allows for the most off-the-shelf customization of fields
    Widget(w widgets.Widget)

    // Sets the field to be hidden
    // 
    // This will hide the field in the form
    Hide(b bool)

    // Sets more validators for the field
    // 
    // These validators will be executed with the processed value of the field
    Validators(validators ...func(interface{}) error)
)
```
