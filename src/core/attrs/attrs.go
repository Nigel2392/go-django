package attrs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/shopspring/decimal"
)

type RelationType int

const (

	// ManyToOne is a many to one relationship, also known as a foreign key relationship.
	//
	// This means that the target model can have multiple instances of the source model,
	// but the source model can only have one instance of the target model.
	// This is the default type for a relation.
	RelManyToOne RelationType = iota

	// OneToOne is a one to one relationship.
	//
	// This means that the target model can only have one instance of the source model.
	// This is the default type for a relation.
	RelOneToOne

	// ManyToMany is a many to many relationship.
	//
	// This means that the target model can have multiple instances of the source model,
	// and the source model can have multiple instances of the target model.
	RelManyToMany

	// OneToMany is a one to many relationship, also known as a reverse foreign key relationship.
	//
	// This means that the target model can only have one instance of the source model,
	// but the source model can have multiple instances of the target model.
	RelOneToMany
)

func init() {
	RegisterFormFieldType(
		json.RawMessage{},
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

// Through is an interface for defining a relation between two models.
//
// This provides a very abstract way of defining relations between models,
// which can be used to define one to one relations or many to many relations.
type Through interface {
	// The through model itself.
	Model() Definer

	// The source field for the relation - this is the field in the source model.
	SourceField() Field

	// The target field for the relation - this is the field in the target model, or in the through model.
	TargetField() Field
}

// Relation is an interface for defining a relation between two models.
//
// This provides a very abstract way of defining relations between models,
// which can be used to define relations in a more generic way.
type Relation interface {
	// The target model for the relationship.
	Target() Definer

	// TargetField retrieves the field in the target model for the relationship.
	//
	// This can be nil, in such cases the relationship should use the primary field of the target model.
	//
	// If a through model is used, the target field should still target the actual target model,
	// the through model should then use this field to link to the target model.
	TargetField() Field

	// A through model for the relationship.
	//
	// This can be nil, but does not have to be.
	// It can support a one to one relationship with or without a through model,
	// or a many to many relationship with a through model.
	Through() Through
}

type TypedRelation interface {
	Relation
	Type() RelationType
}

// Definitions is the interface that wraps the methods for a model's field definitions.
//
// This is some sort of management- interface which allows for simpler and more uniform management of model fields.
type Definitions interface {
	// Set sets the value of the field with the given name (or panics if not found).
	Set(name string, value interface{}) error

	// Retrieves the value of the field with the given name (or panics if not found).
	Get(name string) interface{}

	// TableName retrieves the name of the table in the database.
	//
	// This can be used to generate the SQL for the model.
	//
	// This is the name of the table in the database.
	// It is not the name of the model in the code.
	TableName() string

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

	// Instance returns the underlying model instance.
	Instance() Definer

	// Retrieves a slice of all fields.
	//
	// The order of the fields is the same as they were defined.
	Fields() []Field

	// Retrieves the number of fields.
	Len() int
}

type Field interface {
	sql.Scanner

	// Return the value of the field as a driver.Value.
	//
	// This value should be used for storing the field in a database.
	//
	// If the field is nil or the zero value, the default value should be returned.
	driver.Valuer

	Labeler
	Helper
	Stringer
	Namer

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
	Rel() TypedRelation

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
