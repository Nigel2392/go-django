package admin

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
)

var (
	ErrModelNotFound = errors.New("model not found")
	ErrAppNotFound   = errors.New("app not found")
)

func FindDefinition(model attrs.Definer) *ModelDefinition {
	var modelType = reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	for head := AdminSite.Apps.Front(); head != nil; head = head.Next() {
		var app = head.Value
		for front := app.Models.Front(); front != nil; front = front.Next() {
			var modelDef = front.Value
			var typ = modelDef.rModel()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}

			if typ == modelType {
				return modelDef
			}
		}
	}

	return nil
}

func GetModelInstance(appName, modelName string, id interface{}) (attrs.Definer, error) {
	app, ok := AdminSite.Apps.Get(appName)
	if !ok {
		return nil, ErrAppNotFound
	}

	model, ok := app.Models.Get(modelName)
	if !ok {
		return nil, ErrModelNotFound
	}

	return model.GetInstance(id)
}

func GetModelInstanceList(appName, modelName string, amount, offset uint) ([]attrs.Definer, error) {
	app, ok := AdminSite.Apps.Get(appName)
	if !ok {
		return nil, ErrAppNotFound
	}

	model, ok := app.Models.Get(modelName)
	if !ok {
		return nil, ErrModelNotFound
	}

	if err := except.Assert(
		model.GetList, http.StatusInternalServerError,
		"GetList not implemented for model %s", model.GetName(),
	); err != nil {
		return nil, err
	}

	instance := model.NewInstance()
	fieldNames := attrs.FieldNames(instance, nil)

	return model.GetList(
		amount, offset, fieldNames,
	)
}
