package djangoapps

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2"
	"github.com/elliotchance/orderedmap/v2"
)

// AppConfig is the interface that must be implemented by all Django applications.
//
// The AppConfig interface is used to define the structure of a Django application.
//
// It can be used to define routes, middleware, templates, and other options / handlers.
//
// The implementation of this interface can be found in [./src/apps/apps.go].
type BaseAppConfig interface {
	// The application name.
	//
	// This is used to identify the application.
	//
	// An application cannot be registered twice - the name MUST be unique.
	Name() string

	// Dependencies for the application.
	//
	// This can be used to define dependencies for the application.
	//
	// All of theses dependencies must be registered before the application is initialized.
	Dependencies() []string

	// Commands for the application.
	//
	// These commands can be used to run tasks from the command line.
	//
	// The commands are registered in the Django command registry.
	Commands() []command.Command

	// All apps have been initialized before OnReady() is called.
	OnReady() error

	// BuildRouting is used to define the routes for the application.
	// It can also be used to define middleware for the application.
	//
	// A Mux object is passed to the function which can be used to define routes.
	BuildRouting(mux mux.Multiplexer)

	Processors() []func(ctx.ContextWithRequest)

	Templates() *tpl.Config
}

var apps = struct {
	objects *orderedmap.OrderedMap[reflect.Type, BaseAppConfig]
	names   *orderedmap.OrderedMap[string, BaseAppConfig]
}{
	objects: orderedmap.NewOrderedMap[reflect.Type, BaseAppConfig](),
	names:   orderedmap.NewOrderedMap[string, BaseAppConfig](),
}

func Apps(yield func(BaseAppConfig) bool) {
	for head := apps.names.Front(); head != nil; head = head.Next() {
		if !yield(head.Value) {
			break
		}
	}
}

func RegisterModel(obj any, app BaseAppConfig) {
	apps.objects.Set(reflect.TypeOf(obj), app)

	appName := app.Name()
	if !apps.names.Has(appName) {
		apps.names.Set(appName, app)
	}
}
func GetAppForName(name string) (BaseAppConfig, bool) {
	return apps.names.Get(name)
}

func GetAppForModel(obj any) (BaseAppConfig, bool) {
	var rt reflect.Type
	switch o := obj.(type) {
	case reflect.Value:
		rt = o.Type()
	case reflect.Type:
		rt = o
	default:
		rt = reflect.TypeOf(obj)
	}
	return apps.objects.Get(rt)
}
