package models

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

func ExtractModel(def attrs.Definer) (*Model, error) {
	var info = getModelChain(def)
	if info == nil {
		return nil, fmt.Errorf("the definer %T does not embed a model", def)
	}

	return extractFromInfo(info, def)
}

func extractFromInfo(info *BaseModelInfo, def attrs.Definer) (*Model, error) {
	var rVal = reflect.ValueOf(def)
	if rVal.Kind() != reflect.Ptr || rVal.IsNil() {
		return nil, fmt.Errorf(
			"the definer %T must be a valid pointer to a struct: %w",
			def, ErrObjectInvalid,
		)
	}

	if info == nil {
		return nil, fmt.Errorf(
			"the definer %T does not have an embedded Model field: %w",
			def, ErrModelEmbedded,
		)
	}

	// retrieve the model field by its index chain
	var modelValue = rVal.Elem().FieldByIndex(info.base.Index)
	if modelValue.Kind() != reflect.Struct {
		return nil, ErrModelEmbedded
	}

	// check if the model is addressable
	if !modelValue.CanAddr() {
		return nil, ErrModelAdressable
	}

	// return the model POINTER.
	var modelPtr = modelValue.Addr().Interface()
	return modelPtr.(*Model), nil
}
