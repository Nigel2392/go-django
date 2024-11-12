package attrs

import (
	"encoding/json"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/shopspring/decimal"
)

func init() {
	RegisterFormFieldType(
		json.RawMessage([]byte{}),
		func(opts ...func(fields.Field)) fields.Field {
			return fields.JSONField[json.RawMessage](opts...)
		},
	)
	RegisterFormFieldType(
		decimal.Decimal{},
		func(opts ...func(fields.Field)) fields.Field {
			return fields.DecimalField(opts...)
		},
	)
	RegisterFormFieldType(
		mediafiles.SimpleStoredObject{},
		func(opts ...func(fields.Field)) fields.Field {
			return fields.FileField("", opts...)
		},
	)
}

// Definer is the interface that wraps the FieldDefs method.
//
// FieldDefs retrieves the field definitions for the model.
type Definer interface {
	// Retrieves the field definitions for the model.
	FieldDefs() Definitions
}

// Definitions is the interface that wraps the methods for a model's field definitions.
//
// This is some sort of management- interface which allows for simpler and more uniform management of model fields.
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

	// Retrieves the underlying model instance.
	Instance() Definer

	// Retrieves the related model instance, if any.
	Rel() Definer

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

	// Retrieves the form field for the field.
	//
	// This is used to generate forms for the field.
	FormField() fields.Field

	// Validates the field's value.
	Validate() error
}

type Namer interface {
	// Retrieves the name of the field.
	//
	// This is the name that is used to identify the field in the definitions.
	//
	// It is also the name that is used mainly in forms.
	Name() string
}

type Stringer interface {
	// ToString returns a string representation of the value.
	//
	// This should be the human-readable version of the value, for example for a list display.
	ToString() string
}

type Labeler interface {
	// Label returns the human-readable name of the field.
	//
	// This is the name that is displayed to the user in for example, forms and column headers.
	Label() string
}

type Helper interface {
	// HelpText returns a description of the field.
	//
	// This is displayed to the user in for example, forms.
	HelpText() string
}

type Scanner interface {
	// ScanAttribute scans the value of the attribute.
	//
	// This is used to set the value of the field from a raw value.
	ScanAttribute(src any) error
}
