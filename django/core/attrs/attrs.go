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
	Set(name string, value interface{}) error
	Get(name string) interface{}
	Field(name string) (f Field, ok bool)
	ForceSet(name string, value interface{}) error
	Primary() Field
	Fields() []Field
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
