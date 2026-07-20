package models

import (
	"reflect"

	"github.com/Nigel2392/go-django/internal/bitch"
	"github.com/Nigel2392/go-django/queries/src/models/state"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const (
	flagNone   bitch.Flag = 0
	flagFromDB bitch.Flag = 1 << (iota - 1)
	flagStateDisabled
)

type ModelOptions struct {
	// base model information, used to  extract the model / proxy chain
	Base         *BaseModelInfo
	ReflectValue *reflect.Value
	Defs         *attrs.ObjectDefinitions
	State        *state.ModelState
	Flags        bitch.Flag
}

func OptionsFromModel(modelObject any) (options *ModelOptions, err error) {
	var modelObj *Model
	if m, ok := modelObject.(*Model); ok {
		modelObj = m
	}

	if modelObj == nil {
		modelObj, err = ExtractModel(modelObject.(attrs.Definer))
		if err != nil {
			return nil, err
		}
	}

	if modelObj.internals == nil {
		return nil, ErrModelInitialized
	}

	return modelObj.internals, nil
}
