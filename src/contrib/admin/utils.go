package admin

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
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
