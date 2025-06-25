package drivers

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type Type int

const (
	TypeText Type = iota
	TypeString
	TypeChar
	TypeInt
	TypeUint
	TypeFloat
	TypeBool
	TypeUUID
	TypeBytes
	TypeJSON
	TypeBLOB
	TypeTimestamp
	TypeLocalTime
	TypeDateTime

	DEFAULT_TYPE = TypeText // Default type used when no specific type is registered
)

func (t Type) String() string {
	return TYPES.String(t)
}

// CanDBType is an interface that defines a method to get the database type of a value.
//
// It can be implemented by both [attrs.Field], or the [reflect.Type] returned by [attrs.Field.Type]
type CanDBType interface {
	DBType() Type
}

var TYPES = typeRegistry{
	byType:  make(map[reflect.Type]Type),
	byKind:  make(map[reflect.Kind]Type),
	strings: make(map[Type]string),
}

type typeRegistry struct {
	byType  map[reflect.Type]Type
	byKind  map[reflect.Kind]Type
	strings map[Type]string
}

func (r *typeRegistry) registerType(typ reflect.Type, dbType Type) {
	if typ == nil {
		return
	}
	if _, exists := r.byType[typ]; exists {
		logger.Warnf(
			"Type %s already registered with type %s, overwriting with %d",
			typ.String(), r.byType[typ], dbType,
		)
	}
	r.byType[typ] = dbType
}

func (r *typeRegistry) registerKind(knd reflect.Kind, dbType Type) {
	if knd == reflect.Invalid {
		return
	}
	if _, exists := r.byKind[knd]; exists {
		logger.Warnf(
			"Kind %s already registered with type %d, overwriting with %d",
			knd.String(), r.byKind[knd], dbType,
		)
	}
	r.byKind[knd] = dbType
}

func (r *typeRegistry) String(typ Type) string {
	if str, exists := r.strings[typ]; exists {
		return str
	}
	return "UNKNOWN"
}

func (r *typeRegistry) Add(srcTyp any, dbType Type, stringRep string, forceKind ...bool) {
	var (
		typ reflect.Type
		knd reflect.Kind
	)
	switch v := srcTyp.(type) {
	case reflect.Type:
		typ = v
	case reflect.Value:
		typ = v.Type()
	case reflect.Kind:
		knd = v
	default:
		typ = reflect.TypeOf(srcTyp)
	}

	var useKind bool
	if len(forceKind) > 0 {
		useKind = forceKind[0]

		if typ != nil && useKind {
			knd = typ.Kind()
		}
	}

	if typ != nil {
		logger.Debugf("Registering type %s as %d", typ.String(), dbType)
		r.registerType(typ, dbType)
	}

	if useKind || knd != reflect.Invalid {
		logger.Debugf("Registering kind %s as %d", knd.String(), dbType)
		r.registerKind(knd, dbType)
	}

	if stringRep != "" {
		if _, exists := r.strings[dbType]; exists {
			logger.Warnf(
				"Type %d already registered with string %s, overwriting with %s",
				dbType, r.strings[dbType], stringRep,
			)
		}
		r.strings[dbType] = stringRep
	}
}

func (r *typeRegistry) For(typ reflect.Type) (dbType Type, exists bool) {
	if typ == nil {
		goto retFalse
	}

	if dbType, exists = r.byType[typ]; exists {
		return dbType, true
	}

	if dbType, exists = r.byKind[typ.Kind()]; exists {
		return dbType, true
	}

retFalse:
	return DEFAULT_TYPE, false
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
func DBType(field attrs.FieldDefinition) (dbType Type, ok bool) {
	var fieldType = FieldType(field)
	var fieldVal = reflect.New(fieldType).Elem()
	if dbTypeDefiner, ok := fieldVal.Interface().(CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	} else if dbTypeDefiner, ok := field.(CanDBType); ok {
		return dbTypeDefiner.DBType(), true
	} else {
		return TYPES.For(fieldType)
	}
}
