package attrs

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
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

// CanCreateObject is an interface for models that can create new objects of the same type.
//
// If the type is not the same as the model (for example when embedding a model),
// the newly created object will not be used.
type CanCreateObject[T Definer] interface {
	// CreateObject creates a new object being the same
	// type as the model, and returns it.
	CreateObject() T
}

// A model can implement the CanSetup interface
// to perform any setup that is needed for the model.
//
// This is called when the model is created with the [NewObject] function.
type CanSetup interface {
	Setup()
}

// Keys of attributes defined with the `Attrs()` method on fields.
//
// These are used to store extra information about the field.
//
// We provide some default keys which might be useful for implementing an ORM, but any keys can be used.
const (
	// AttrNameKey (string) is the name of the field.
	AttrNameKey = "field.name"

	// AttrMaxLengthKey (int64) is the maximum length of the field.
	AttrMaxLengthKey = "field.max_length"

	// AttrMinLengthKey (int64) is the minimum length of the field.
	AttrMinLengthKey = "field.min_length"

	// AttrMinValueKey (float64) is the minimum value of the field.
	AttrMinValueKey = "field.min_value"

	// AttrMaxValueKey (float64) is the maximum value of the field.
	AttrMaxValueKey = "field.max_value"

	// AttrAllowNullKey (bool) is whether the field allows null values.
	AttrAllowNullKey = "field.allow_null"

	// AttrAllowBlankKey (bool) is whether the field allows blank values.
	AttrAllowBlankKey = "field.allow_blank"

	// AttrAllowEditKey (bool) is whether the field is read-only.
	AttrAllowEditKey = "field.read_only"

	// AttrIsPrimaryKey (bool) is whether the field is a primary key.
	AttrIsPrimaryKey = "field.primary"

	// AttrAutoIncrementKey (bool) is whether the field is an auto-incrementing field.
	AttrAutoIncrementKey = "field.auto_increment"

	// AttrUniqueKey (bool) is whether the field is a unique field.
	AttrUniqueKey = "field.unique"

	// AttrReverseAliasKey (string) is the reverse alias of the field.
	AttrReverseAliasKey = "field.reverse_alias"
)

// Definer is the interface that wraps the FieldDefs method.
//
// FieldDefs retrieves the field definitions for the model.
type Definer interface {
	// Retrieves the field definitions for the model.
	FieldDefs() Definitions
}

// A binder is a value which can be bound to a model.
//
// Any fields should call the `Bind` method to bind the value to the model,
// this has to be done when:
// - the field value is set 	    (SetValue method is called)
// - the field value is retrieved   (GetValue method is called)
// - the default value is retrieved (GetDefault method is called)
// - the field value is scanned     (Scan method is called)
// - the driver.Value is retrieved  (Value method is called)
type Binder interface {
	// Bind binds the value to the model.
	BindToModel(model Definer, field Field) error
}

// BindValueToModel binds the given model and field to the value.
func BindValueToModel(model Definer, field Field, value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case reflect.Value:
		if !v.IsValid() {
			return nil
		}
		value = v.Interface()
	case *reflect.Value:
		if v == nil || !v.IsValid() {
			return nil
		}
		value = v.Interface()
	}

	if binder, ok := value.(Binder); ok {
		return binder.BindToModel(model, field)
	}
	return nil
}

// A field in a struct can implement the Embedded interface
// to bind itself to the [Definer] which should be the top-most model.
//
// It allows for multiple values in a chain of embedded models
// to be bound to the top-most model.
//
// This is only called when [NewField] is called.
type Embedded interface {
	BindToEmbedder(embedder Definer) error
}

// Through is an interface for defining a relation between two models.
//
// This provides a very abstract way of defining relations between models,
// which can be used to define one to one relations or many to many relations.
type Through interface {
	// The through model itself.
	Model() Definer

	// The source field for the relation - this is a field in the through model linking to the source model.
	SourceField() string

	// The target field for the relation - this is a field in the through model linking to the target model.
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
	Field() FieldDefinition
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

// CanMeta is an interface for defining a model that can have meta information.
//
// This meta information is then stored on the ModelMeta interface.
type CanModelInfo interface {
	// ModelMetaInfo returns the meta information for the model.
	//
	// This is used to store information about the model, such as relational information,
	ModelMetaInfo(object Definer) map[string]any
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
	// which belong to the field with the given name.
	Forward(relField string) (Relation, bool)

	// Reverse returns the reverse relations for this model
	// which belong to the field with the given name.
	Reverse(relField string) (Relation, bool)

	// ForwardMap returns a copy the forward relations map for this model
	ForwardMap() *orderedmap.OrderedMap[string, Relation]

	// ReverseMap returns a copy of the reverse relations map for this model
	ReverseMap() *orderedmap.OrderedMap[string, Relation]

	// ContentType returns the content type for the model.
	ContentType() contenttypes.ContentType

	// Storage returns a value stored on the model meta.
	//
	// This is used to store values that are not part of the model itself,
	// but are needed for the model or possible third party libraries to function.
	//
	// Values can be stored on the model meta using the `attrs.StoreOnMeta` helper function.
	//
	// A model can also implement the `CanModelInfo` interface to store values on the model meta.
	Storage(key string) (any, bool)

	// Definitions returns the field definitions for the model.
	//
	// This is used to retrieve meta information about fields, such as their type,
	// and other information that is not part of the model itself.
	Definitions() StaticDefinitions
}

type staticDefinitions[T FieldDefinition] interface {
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
	Field(name string) (f T, ok bool)

	// Retrieves the primary field.
	Primary() T

	// Instance returns the underlying model instance.
	Instance() Definer

	// Retrieves a slice of all fields.
	//
	// The order of the fields is the same as they were defined.
	Fields() []T

	// Retrieves the number of fields.
	Len() int
}

type StaticDefinitions = staticDefinitions[FieldDefinition]

type FieldDefinition interface {
	Name() string
	Labeler
	Helper

	// Tag retrieves the tag value for the field with the given name.
	Tag(name string) string

	// Retrieves the underlying model instance.
	//
	// For a field definition, this is likely not an actual instance of the model,
	// for the Field interface, this is the actual model instance.
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
	// Retrieves the form field for the field.
	//
	// This is used to generate forms for the field.
	FormField() fields.Field

	// Validates the field's value.
	Validate() error
}

// Definitions is the interface that wraps the methods for a model's field definitions.
//
// This is some sort of management- interface which allows for simpler and more uniform management of model fields.
type Definitions interface {
	staticDefinitions[Field]

	// Set sets the value of the field with the given name (or panics if not found).
	Set(name string, value interface{}) error

	// Retrieves the value of the field with the given name (or panics if not found).
	Get(name string) interface{}
}

type Field interface {
	FieldDefinition

	// Scan the value of the field into your model.
	//
	// This allows for reading the value easily from the database.
	sql.Scanner

	// Return the value of the field as a driver.Value.
	//
	// This value should be used for storing the field in a database.
	//
	// If the field is nil or the zero value, the default value should be returned.
	driver.Valuer

	// ToString returns a string representation of the value.
	//
	// This should be the human-readable version of the value, for example for a list display.
	ToString() string

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
}

// CanRelatedName is an interface for fields that have a related name.
//
// This is used to define the name of the field in the related model.
type CanRelatedName interface {
	Field
	RelatedName() string
}

// CanOnModelRegister defines a method that is called when the model is registered.
//
// This method is called once, and only once.
//
// See [OnModelRegister] and [RegisterModel] for the implementation details.
type CanOnModelRegister interface {
	Field
	OnModelRegister(model Definer) error
}

// An unbound field is a field that is not yet bound to a model.
//
// It returns a field in case of a wrapper implementation,
// or an error in case the field cannot be bound to the model.
type UnboundField interface {
	Field
	// BindField binds the field to the model.
	BindField(model Definer) (Field, error)
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
