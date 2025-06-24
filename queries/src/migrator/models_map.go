package migrator

import (
	"reflect"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-signals"
)

var modelMap = make(map[reflect.Type]django.AppConfig)

func getModelApp(model any) django.AppConfig {
	var typ = reflect.TypeOf(model)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if app, ok := modelMap[typ]; ok {
		return app
	}
	return nil
}

var _, _ = core.OnDjangoReady.Listen(func(s signals.Signal[any], a any) error {
	var app = a.(*django.Application)
	var apps = app.Apps
	for head := apps.Front(); head != nil; head = head.Next() {
		var (
			app = head.Value
		)

		var models = app.Models()
		for _, model := range models {
			var typ = reflect.TypeOf(model)
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			if _, ok := modelMap[typ]; !ok {
				modelMap[typ] = app
			}
		}
	}
	return nil
})
