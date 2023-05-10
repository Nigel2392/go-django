package modelutils

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/core"
)

// Get a default display for the given model.
//
// If the model implements the DisplayableModel interface, the String() method will be called.
func GetModelDisplay(mdl any, list bool) string {
	return getModelDisplay(mdl, list, true)
}

// Get a default display for the given model.
func getModelDisplay(mdl any, list, firstIteration bool) string {
	if list {
		switch mdlType := mdl.(type) {
		case core.ListDisplayer:
			return mdlType.ListDisplay()
		}
	}

	switch mdlType := mdl.(type) {
	case core.DisplayableField:
		return mdlType.Display()
	case fmt.Stringer:
		return mdlType.String()
	}

	if firstIteration {
		if reflect.TypeOf(mdl).Kind() == reflect.Ptr {
			mdl = reflect.ValueOf(mdl).Elem().Interface()
			return getModelDisplay(mdl, list, false)
		} else if reflect.TypeOf(mdl).Kind() == reflect.Struct {
			// If the model is a struct, and we still havent found it
			// We need to turn the non-pointer struct into a pointer struct.
			var ptr = reflect.New(reflect.TypeOf(mdl))
			// Set the value of the pointer struct to the value of the struct.
			ptr.Elem().Set(reflect.ValueOf(mdl))
			// Get the interface of the pointer struct.
			mdl = ptr.Interface()
			return getModelDisplay(mdl, list, false)
		}
	}

	var namedTyp = reflect.TypeOf(mdl)
	if namedTyp.Kind() == reflect.Ptr {
		namedTyp = namedTyp.Elem()
	}
	return namedTyp.Name()

}

// Get a new model from the given model.
//
// If ptr is true, a pointer to the model will be returned.
func NewOf(m any, ptr bool) any {
	var typeOf = reflect.TypeOf(m)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	if ptr {
		return reflect.New(typeOf).Interface()
	}
	return reflect.New(typeOf).Elem().Interface()
}

// Get a new slice of the given model.
func GetNewModelSlice(m any) any {
	var typeOf = reflect.TypeOf(m)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	return reflect.MakeSlice(reflect.SliceOf(typeOf), 0, 0).Interface()
}
