package drivers

import (
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// CanDBType is an interface that defines a method to get the database type of a value.
//
// It can be implemented by both [attrs.Field], or the [reflect.Type] returned by [attrs.Field.Type]
type CanDBType interface {
	DBType() dbtype.Type
}

// CanDBTypeString is a type that can be used to define a field that can return a database type as a string.
type CanDBTypeString interface {
	DBType() string
}

// FieldType returns the reflect.Type of the field definition.
//
// It does so by calling [attrs.FieldDefinition.Type] on the field.
//
// If the field is a relation, it will return the primary field's reflect.Type of the related model.
func FieldType(field attrs.FieldDefinition) reflect.Type {
	if field == nil {
		return nil
	}

	var rel = field.Rel()
	if rel != nil {
		var relField = rel.Field()
		if relField == nil {
			relField = rel.Model().FieldDefs().Primary()
		}
		if relField != nil {
			field = relField
		}
	}

	var fTyp = field.Type()
	if field.Type().Implements(reflect.TypeOf((*attrs.Definer)(nil)).Elem()) {
		// if the field is a definer, we return the type of the underlying object
		var definerType = fTyp.Elem()
		var newObj = reflect.New(definerType)
		var definer = newObj.Interface().(attrs.Definer)
		var fieldDefs = definer.FieldDefs()
		return FieldType(fieldDefs.Primary())
	}

	return field.Type()
}

// DBType returns the database type of the field definition.
//
// It checks if the field implements [CanDBType] and calls its [CanDBType.DBType] method.
// If it does not implement [CanDBType], it will check the [TYPES] registry to find the database type
// based on the field's reflect.Type.
func DBType(field attrs.FieldDefinition) (dbType dbtype.Type, ok bool) {
	var fieldType = FieldType(field)
	var fieldVal = reflect.New(fieldType).Elem()

	if dbTypeDefiner, ok := fieldVal.Interface().(CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	}

	if dbTypeDefiner, ok := fieldVal.Interface().(CanDBTypeString); ok {
		var dbTypeStr = dbTypeDefiner.DBType()
		var dbType, ok = dbtype.NewFromString(dbTypeStr)
		if ok {
			return dbType, true
		}
	}

	if dbTypeDefiner, ok := field.(CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	}

	if dbTypeDefiner, ok := field.(CanDBTypeString); ok {
		var dbTypeStr = dbTypeDefiner.DBType()
		var dbType, ok = dbtype.NewFromString(dbTypeStr)
		if ok {
			return dbType, true
		}
	}

	return dbtype.TYPES.For(fieldType)
}
