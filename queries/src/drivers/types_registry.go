package drivers

import (
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// FieldType returns the reflect.Type of the field definition.
//
// It does so by calling [attrs.FieldDefinition.Type] on the field.
//
// If the field is a relation, it will return the related models' primary field reflect.Type
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

	if dbTypeDefiner, ok := field.(dbtype.CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	}

	if dbTypeDefiner, ok := field.(dbtype.CanDBTypeString); ok {
		var dbTypeStr = dbTypeDefiner.DBType()
		var dbType, ok = dbtype.NewFromString(dbTypeStr)
		if ok {
			return dbType, true
		}
	}

	var fieldType = FieldType(field)
	var fieldVal = reflect.New(fieldType).Elem()
	if dbTypeDefiner, ok := fieldVal.Interface().(dbtype.CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	}

	if dbTypeDefiner, ok := fieldVal.Interface().(dbtype.CanDBTypeString); ok {
		var dbTypeStr = dbTypeDefiner.DBType()
		var dbType, ok = dbtype.NewFromString(dbTypeStr)
		if ok {
			return dbType, true
		}
	}

	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	return dbtype.TYPES.For(fieldType)
}
