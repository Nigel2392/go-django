# `attrs` package functions

Global functions in the `attrs` which can be used to manage your models.

---

## AutoDefinitions

AutoDefinitions automatically generates definitions for a struct.

It does this by iterating over the fields of the struct and checking for the
`attrs` tag. If the tag is present, it will parse the tag and generate the
definition.

If the `include` parameter is provided, it will only generate definitions for
the fields that are included.

### `AutoDefinitions[T Definer](instance T, include ...any) Definitions`

---

## Define

Define creates a new object definitions.
This can then be returned by the FieldDefs method of a model
to make it comply with the Definer interface.

### `Define(d Definer, fieldDefinitions ...Field) *ObjectDefinitions`

---

## FieldNames

A shortcut for getting the names of all fields in a Definer.

The exclude parameter can be used to exclude certain fields from the result.

This function is useful when you need to get the names of all fields in a
model, but you want to exclude certain fields (e.g. fields that are not editable).

---

### `FieldNames(d any, exclude []string) []string`

## SetPrimaryKey

SetPrimaryKey sets the primary key field of a Definer.

If the primary key field is not found, this function will panic.

### `SetPrimaryKey(d Definer, value interface{}) error`

---

## PrimaryKey

PrimaryKey returns the primary key field of a Definer.

If the primary key field is not found, this function will panic.

### `PrimaryKey(d Definer) interface{}`

---

## SetMany

SetMany sets multiple fields on a Definer.

The values parameter is a map where the keys are the names of the fields to set.

The values must be of the correct type for the fields.

### `SetMany(d Definer, values map[string]interface{}) error`

---

## Set

Set sets the value of a field on a Definer.

If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.

If the field is marked as non editable, this function will panic.

### `Set(d Definer, name string, value interface{}) error`

---

## ForceSet

ForceSet sets the value of a field on a Definer.

If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.

This function will allow setting the value of a field that is marked as not editable.

### `ForceSet(d Definer, name string, value interface{}) error`

---

## Get

Get retrieves the value of a field on a Definer.

If the field is not found, this function will panic.

Type assertions are used to ensure that the value is of the correct type,
as well as providing less work for the caller.

### `Get[T any](d Definer, name string) T`

---

## Method

Method retrieves a method from an object.

The generic type parameter must be the type of the method.

### `Method[T any](obj interface{}, name string) (n T, ok bool)`

---

## CastToNumber

CastToNumber converts a value of any type (int, float, string) to a number of type T.
It returns the converted value and an error if the conversion fails.

### `CastToNumber[T any](v any) (T, error)`

---

## ToString

ToString converts a value to a string.

This should be the human-readable representation of the value.

If the value is a struct with a content type, it will use the content type's InstanceLabel method to convert it to a string.

time.Time, mail.Address, and error types are handled specially.

If the value is a slice or array, it will convert each element to a string and join them with ", ".

If all else fails, it will use fmt.Sprintf to convert the value to a string.

### `ToString(v any) string`

---

## DefinerList

DefinerList converts a slice of []T where the underlying type is of type Definer to []Definer.

### `DefinerList[T Definer](list []T) []Definer`

---

## InterfaceList

InterfaceList converts a slice of []T to []any.

### `InterfaceList[T any](list []T) []any`
