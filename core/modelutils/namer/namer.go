package namer

import (
	"reflect"

	"github.com/Nigel2392/go-django/core"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/modelutils"
)

// GetAppName returns the name of the app the model belongs to.
//
// If the model implements the core.AppNamer interface, then the AppName method is called.
func GetAppName(m any) string {
	switch m := m.(type) {
	case core.AppNamer:
		return m.AppName()
	default:
		var v = reflect.ValueOf(m)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			return GetAppName(v.Interface())
		}
		return httputils.GetPkgPath(m)
	}
}

// GetModelName returns the name of the model.
//
// If the model implements the core.Namer interface, then the Name method is called.
func GetModelName(m any) string {
	var newM = modelutils.DePtr(m)
	m = newM.Interface()
	if name, isNamer := getNameOf(m); isNamer {
		return name
	}

	var newPtr = modelutils.NewPtr(m)
	m = newPtr.Interface()
	if name, isNamer := getNameOf(m); isNamer {
		return name
	}

	// Finally if both the interface with and without the pointer are not Namer,
	// then just return the name of the type.
	var modelType = reflect.TypeOf(m)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	return modelType.Name()
}

func getNameOf(m any) (s string, isNamer bool) {
	switch m := m.(type) {
	case core.Namer:
		return m.NameOf(), true
	}
	return "", false
}
