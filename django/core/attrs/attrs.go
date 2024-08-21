package attrs

import (
	"encoding/json"

	"github.com/Nigel2392/django/forms/fields"
)

func init() {
	RegisterFormFieldType(
		json.RawMessage([]byte{}),
		func(opts ...func(fields.Field)) fields.Field {
			return fields.JSONField[json.RawMessage](opts...)
		},
	)
}

type Definer interface {
	FieldDefs() Definitions
}

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

type Field interface {
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

type Namer interface {
	Name() string
}

type Stringer interface {
	ToString() string
}

type Labeler interface {
	Label() string
}

type Helper interface {
	HelpText() string
}

type Scanner interface {
	ScanAttribute(src any) error
}
