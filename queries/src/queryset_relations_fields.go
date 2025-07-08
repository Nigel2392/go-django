package queries

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// newSettableRelation creates a new instance of a SettableRelation or SettableMultiThroughRelation.
//
// It checks if the type is a slice or a pointer, and returns a new instance of the appropriate type.
func newSettableRelation[T any](typ reflect.Type) T {
	var setterTyp = typ
	if setterTyp.Kind() == reflect.Ptr {
		setterTyp = setterTyp.Elem()
	}

	if setterTyp.Kind() == reflect.Slice {
		var n = reflect.MakeSlice(setterTyp, 0, 0)
		var sliceVal = n.Interface()
		if n.Type().Implements(reflect.TypeOf((*T)(nil)).Elem()) {
			return sliceVal.(T)
		}
		return n.Addr().Interface().(T)
	}

	var newVal = reflect.New(setterTyp)
	if newVal.Type().Implements(reflect.TypeOf((*T)(nil)).Elem()) {
		return newVal.Interface().(T)
	}

	return newVal.Addr().Interface().(T)
}

var (
	_TYP_DEFINER                = reflect.TypeOf((*attrs.Definer)(nil)).Elem()
	_TYP_REL                    = reflect.TypeOf((*Relation)(nil)).Elem()
	_TYP_RELVALUE               = reflect.TypeOf((*RelationValue)(nil)).Elem()
	_TYP_THROUGH_RELVALUE       = reflect.TypeOf((*ThroughRelationValue)(nil)).Elem()
	_TYP_MULTI_RELVALUE         = reflect.TypeOf((*MultiRelationValue)(nil)).Elem()
	_TYP_MULTI_THROUGH_RELVALUE = reflect.TypeOf((*MultiThroughRelationValue)(nil)).Elem()
)

// setRelatedObjects sets the related objects for the given relation name and type.
//
// it provides a uniform way to set related objects on a model instance,
// allowing to handle different relation types and through models.
//
// used in [rows.compile] to set the related objects on the parent object.
//
// through models will not be set using [ThroughModelSetter.SetThroughModel] here
// to avoid cluttered, complex and unreadable code in the switch statement.
func setRelatedObjects(relName string, relTyp attrs.RelationType, obj attrs.Definer, relatedObjects []Relation) {

	var fieldDefs = obj.FieldDefs()
	var field, ok = fieldDefs.Field(relName)
	if !ok {
		panic(fmt.Sprintf("relation %s not found in field defs of %T", relName, obj))
	}

	var (
		fieldType  = field.Type()
		fieldValue = field.GetValue()
	)

	switch relTyp {
	case attrs.RelManyToOne:
		// handle foreign keys
		//
		// no through model is expected
		if len(relatedObjects) > 1 {
			panic(fmt.Sprintf("expected at most one related object for %s, got %d", relName, len(relatedObjects)))
		}

		var relatedObject attrs.Definer
		if len(relatedObjects) > 0 {
			relatedObject = relatedObjects[0].Model()
		}

		switch {
		case fieldType.Implements(_TYP_RELVALUE):
			// If the field is a RelationValue, we can set the related object directly
			var rel = fieldValue.(RelationValue)
			rel.SetValue(relatedObject)
			field.SetValue(rel, true)

		case fieldType.Implements(_TYP_DEFINER):
			// If the field is a Definer, we can set the related object directly
			// This is useful for fields that are not RelationValue but still need to be set
			field.SetValue(relatedObject, true)
		}

	case attrs.RelOneToMany:
		// handle reverse foreign keys
		//
		// a through model is not expected
		var related = make([]attrs.Definer, len(relatedObjects))
		for i, relatedObj := range relatedObjects {
			related[i] = relatedObj.Model()
		}

		if dm, ok := obj.(DataModel); ok {
			dm.DataStore().SetValue(relName, related)
		}

		switch {
		case fieldType == reflect.TypeOf(related):
			// If the field is a slice of Definer, we can set the related objects directly
			field.SetValue(related, true)

		case fieldType.Implements(_TYP_MULTI_RELVALUE):
			// If the field is a RelationValue, we can set the related objects directly
			var rel = fieldValue.(MultiRelationValue)
			rel.SetValues(related)
			field.SetValue(rel, true)

		case fieldType.Kind() == reflect.Slice:
			// If the field is a slice, we can set the related objects directly after
			// converting them to the appropriate type.
			var slice = reflect.MakeSlice(fieldType, len(relatedObjects), len(relatedObjects))
			for i, relatedObj := range relatedObjects {
				slice.Index(i).Set(reflect.ValueOf(relatedObj.Model()))
			}
			field.SetValue(slice.Interface(), true)

		default:
			panic(fmt.Sprintf("expected field %s to be a RelationValue, MultiRelationValue, or slice of Definer, got %s", relName, fieldType))
		}

	case attrs.RelOneToOne:
		// handle one-to-one relations
		//
		// a through model COULD BE expected, but it is not required
		if len(relatedObjects) > 1 {
			panic(fmt.Sprintf("expected at most one related object for %s, got %d", relName, len(relatedObjects)))
		}

		var relatedObject Relation
		if len(relatedObjects) > 0 {
			relatedObject = relatedObjects[0]
		}

		switch {
		case fieldType.Implements(_TYP_RELVALUE):
			var rel = fieldValue.(RelationValue)
			rel.SetValue(relatedObject.Model())
			field.SetValue(rel, true)

		case fieldType.Implements(_TYP_THROUGH_RELVALUE):
			// If the field is a SettableThroughRelation, we can set the related object directly
			var rel = fieldValue.(ThroughRelationValue)
			rel.SetValue(relatedObject.Model(), relatedObject.Through())
			field.SetValue(rel, true)

		case fieldType.Implements(_TYP_REL):
			// If the field is a Relation, we can set the related object directly
			field.SetValue(relatedObject, true)

		case fieldType.Implements(_TYP_DEFINER):

			// If the field is a Definer, we can set the related object directly
			field.SetValue(relatedObject.Model(), true)

		default:
			panic(fmt.Sprintf("expected field %s to be a RelationValue, ThroughRelationValue, Relation, or Definer, got %s", relName, fieldType))

		}

	case attrs.RelManyToMany:
		// handle many-to-many relations
		//
		// a through model is expected
		if dm, ok := obj.(DataModel); ok {
			dm.DataStore().SetValue(relName, relatedObjects)
		}

		switch {
		case fieldType.Implements(_TYP_MULTI_THROUGH_RELVALUE):
			// If the field is a SettableMultiRelation, we can set the related objects directly
			var rel = fieldValue.(MultiThroughRelationValue)
			rel.SetValues(relatedObjects)
			field.SetValue(rel, true)

		case fieldType.Implements(_TYP_MULTI_RELVALUE):
			// If the field is a SettableMultiRelation, we can set the related objects directly
			var rel = fieldValue.(MultiRelationValue)
			var definerList = make([]attrs.Definer, len(relatedObjects))
			for i, relatedObj := range relatedObjects {
				definerList[i] = relatedObj.Model()
			}
			rel.SetValues(definerList)
			field.SetValue(rel, true)

		case fieldType.Kind() == reflect.Slice && fieldType.Elem().Implements(_TYP_REL):
			// If the field is a slice, we can set the related objects directly
			var slice = reflect.MakeSlice(fieldType, len(relatedObjects), len(relatedObjects))
			for i, relatedObj := range relatedObjects {
				slice.Index(i).Set(reflect.ValueOf(relatedObj))
			}
			// fieldDefs.Set(relName, slice.Interface())
			field.SetValue(slice.Interface(), true)

		case fieldType.Kind() == reflect.Slice && fieldType.Elem().Implements(_TYP_DEFINER):
			// If the field is a slice of Definer, we can set the related objects directly
			var slice = reflect.MakeSlice(fieldType, len(relatedObjects), len(relatedObjects))
			for i, relatedObj := range relatedObjects {
				var relatedDefiner = relatedObj.Model()
				slice.Index(i).Set(reflect.ValueOf(relatedDefiner))
			}
			// fieldDefs.Set(relName, slice.Interface())
			field.SetValue(slice.Interface(), true)

		default:
			panic(fmt.Sprintf("expected field %s to be a SettableMultiRelation, SettableMultiThroughRelation, or a slice of Relation/Definer, got %s", relName, fieldType))
		}
	default:
		panic(fmt.Sprintf("unknown relation type %s for field %s in %T", relTyp, relName, obj))
	}

	// field.SetValue(fieldValue, true)
}

type walkInfo struct {
	idx       int
	depth     int
	fieldDefs attrs.Definitions
	field     attrs.Field
	chain     []string
}

//  func (w walkInfo) path() string {
//  	var path = w.field.Name()
//  	if len(w.chain) > 1 {
//  		path = fmt.Sprintf("%s.%s", w.chain[:w.depth], path)
//  	}
//  	return path
//  }

func isNil(v reflect.Value) bool {
	if !v.IsValid() || v.Kind() == reflect.Pointer && v.IsNil() {
		return true
	}
	if v.Kind() == reflect.Ptr {
		return isNil(v.Elem())
	}
	return false
}

// walkFields traverses the fields of an object based on a chain of field names.
//
// It yields each field found at the last depth of the chain, allowing for
// custom processing of the field (e.g., collecting values).
func walkFieldValues(obj attrs.Definitions, chain []string, idx *int, depth int, yield func(walkInfo) bool) error {

	if depth > len(chain)-1 {
		return errors.FieldNotFound.WithCause(fmt.Errorf(
			"depth %d exceeds chain length %d", depth, len(chain),
		))
	}

	var fieldName = chain[depth]
	var field, ok = obj.Field(fieldName)
	if !ok {
		return errors.FieldNotFound.WithCause(fmt.Errorf(
			"field %s not found in object %T", fieldName, obj,
		))
	}

	if depth == len(chain)-1 {
		if !yield(walkInfo{
			idx:       *idx,
			depth:     depth,
			fieldDefs: obj,
			field:     field,
			chain:     chain,
		}) {
			return errStopIteration
		}
		*idx++     // Increment index for the next field found
		return nil // Found the field at the last depth
	}

	var fieldType = field.Type()
	var value = field.GetValue()
	var rVal = reflect.ValueOf(value)

	if isNil(rVal) && fieldType == nil {
		return errors.NilPointer.WithCause(fmt.Errorf(
			"field %s in object %T is nil", fieldName, obj,
		))
	}

	if isNil(rVal) && fieldType != nil && fieldType.Implements(reflect.TypeOf((*attrs.Binder)(nil)).Elem()) {
		value = newSettableRelation[attrs.Binder](fieldType)
		field.SetValue(value, true)
	}

	var rTyp = reflect.TypeOf(value)
	switch {
	case rTyp.Implements(reflect.TypeOf((*attrs.Definer)(nil)).Elem()):
		// If the field is a Definer, we can walk its fields
		var definer = value.(attrs.Definer).FieldDefs()
		if err := walkFieldValues(definer, chain, idx, depth+1, yield); err != nil {
			if errors.Is(err, errors.NilPointer) {
				return nil // Skip nil pointers
			}
			return fmt.Errorf("%s: %w", fieldName, err)
		}

	case rTyp.Implements(reflect.TypeOf((*RelationValue)(nil)).Elem()):
		// If the field is a RelationValue, we can walk its fields
		var relValue = value.(RelationValue)
		var model = relValue.GetValue()
		if model == nil {
			return nil
		}
		if err := walkFieldValues(model.FieldDefs(), chain, idx, depth+1, yield); err != nil {
			if errors.Is(err, errors.NilPointer) {
				return nil // Skip nil pointers
			}
			return fmt.Errorf("%s: %w", fieldName, err)
		}

	case rTyp.Implements(reflect.TypeOf((*MultiRelationValue)(nil)).Elem()):
		// If the field is a MultiRelationValue, we can walk its fields
		var multiRelValue = value.(MultiRelationValue)
		var relatedObjects = multiRelValue.GetValues()
		if len(relatedObjects) == 0 {
			return nil // Skip empty relations
		}

		for _, rel := range relatedObjects {
			var modelDefs = rel.FieldDefs()
			if err := walkFieldValues(modelDefs, chain, idx, depth+1, yield); err != nil {
				if errors.Is(err, errors.NilPointer) {
					continue // Skip nil relations
				}
				return fmt.Errorf("%s: %w", fieldName, err)
			}
		}

	case rTyp.Implements(reflect.TypeOf((*ThroughRelationValue)(nil)).Elem()):
		var value = value.(ThroughRelationValue)
		var model, _ = value.GetValue()
		if model == nil {
			return errors.NilPointer
		}
		if err := walkFieldValues(model.FieldDefs(), chain, idx, depth+1, yield); err != nil {
			if errors.Is(err, errors.NilPointer) {
				return nil // Skip nil pointers
			}
			return fmt.Errorf("%s: %w", fieldName, err)
		}

	case rTyp.Implements(reflect.TypeOf((*MultiThroughRelationValue)(nil)).Elem()):
		var value = value.(MultiThroughRelationValue)
		var relatedObjects = value.GetValues()
		if len(relatedObjects) == 0 {
			return nil // Skip empty relations
		}
		for _, rel := range relatedObjects {
			var modelDefs = rel.Model().FieldDefs()
			if err := walkFieldValues(modelDefs, chain, idx, depth+1, yield); err != nil {
				if errors.Is(err, errors.NilPointer) {
					continue // Skip nil relations
				}
				return fmt.Errorf("%s: %w", fieldName, err)
			}
		}

	case rTyp.Kind() == reflect.Slice && rTyp.Elem().Implements(reflect.TypeOf((*attrs.Definer)(nil)).Elem()):
		// If the field is a slice of Definer, we can walk its fields
		var slice = reflect.ValueOf(value)
		for i := 0; i < slice.Len(); i++ {
			var elem = slice.Index(i).Interface()
			if elem == nil {
				continue // Skip nil elements
			}
			if err := walkFieldValues(elem.(attrs.Definer).FieldDefs(), chain, idx, depth+1, yield); err != nil {
				if errors.Is(err, errors.NilPointer) {
					continue // Skip elements where the field is nil
				}
				return fmt.Errorf("%s[%d]: %w", fieldName, i, err)
			}
		}

	case rTyp.Kind() == reflect.Slice && rTyp.Elem().Implements(reflect.TypeOf((*Relation)(nil)).Elem()):
		// If the field is a slice of Relation, we can walk its fields
		var slice = reflect.ValueOf(value)
		for i := 0; i < slice.Len(); i++ {
			var elem = slice.Index(i).Interface()
			if elem == nil {
				continue // Skip nil elements
			}
			var rel = elem.(Relation)
			if rel.Model() == nil {
				continue // Skip relations with nil model
			}
			if err := walkFieldValues(rel.Model().FieldDefs(), chain, idx, depth+1, yield); err != nil {
				if errors.Is(err, errors.NilPointer) {
					continue // Skip elements where the field is nil
				}
				return fmt.Errorf("%s[%d]: %w", fieldName, i, err)
			}
		}

	default:
		return fmt.Errorf("expected field %s in object %T to be a Definer, slice of Definer, or slice of Relation, got %s", fieldName, obj.Instance(), rTyp)
	}
	return nil
}
