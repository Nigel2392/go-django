package attrs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"iter"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/shopspring/decimal"
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
	SourceField() string

	// The target field for the relation - this is the field in the target model, or in the through model.
	TargetField() string
}

// RelationTarget is an interface for defining a relation target.
//
// This is the target model for the relation, which can be used to define the relation in a more generic way.
type RelationTarget interface {
	// From represents the source model for the relationship.
	//
	// If this is nil then the current interface value is the source model.
	From() RelationTarget

	// The target model for the relationship.
	Model() Definer

	// Field retrieves the field in the target model for the relationship.
	//
	// This can be nil, in such cases the relationship should use the primary field of the target model.
	//
	// If a through model is used, the target field should still target the actual target model,
	// the through model should then use this field to link to the target model.
	Field() Field
}

// Relation is an interface for defining a relation between two models.
//
// This provides a very abstract way of defining relations between models,
// which can be used to define relations in a more generic way.
type Relation interface {
	RelationTarget

	Type() RelationType

	// A through model for the relationship.
	//
	// This can be nil, but does not have to be.
	// It can support a one to one relationship with or without a through model,
	// or a many to many relationship with a through model.
	Through() Through
}

// ModelMeta represents the meta information for a model.
//
// This is used to store information about the model, such as relational information,
// and other information that is not part of the model itself.
//
// Models which implement the `Definer` interface
type ModelMeta interface {
	// Model returns the model for this meta
	Model() Definer

	// Forward returns the forward relations for this model
	Forward(relField string) (Relation, bool)

	// ForwardMap returns the forward relations map for this model
	ForwardMap() *orderedmap.OrderedMap[string, Relation]

	// Reverse returns the reverse relations for this model
	Reverse(relField string) (Relation, bool)

	// ReverseMap returns the reverse relations map for this model
	ReverseMap() *orderedmap.OrderedMap[string, Relation]

	// IterForward iterates over the forward relations for this model
	IterForward() iter.Seq2[string, Relation]

	// IterReverse iterates over the reverse relations for this model
	IterReverse() iter.Seq2[string, Relation]

	// Storage returns a value stored on the model meta.
	//
	// This is used to store values that are not part of the model itself,
	// but are needed for the model or possible third party libraries to function.
	//
	// Values can be stored on the model meta using the `attrs.StoreOnMeta` helper function.
	Storage(key string) (any, bool)
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

type CanRelatedName interface {
	Field
	RelatedName() string
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
