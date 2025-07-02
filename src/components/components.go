package components

import (
	"context"
	"io"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

type ComponentFunc = django_reflect.Function

type Registry interface {
	Register(name string, componentFn ComponentFunc)
	Render(name string, args ...interface{}) Component
}

type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

type ComponentRegistry struct {
	ns_components map[string]map[string]*django_reflect.Func
	components    map[string]*django_reflect.Func
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		ns_components: make(map[string]map[string]*django_reflect.Func),
		components:    make(map[string]*django_reflect.Func),
	}
}

func (r *ComponentRegistry) newComponent(fn ComponentFunc) *django_reflect.Func {
	return django_reflect.NewFunc(fn, reflect.TypeOf((*Component)(nil)).Elem())
}

func (r *ComponentRegistry) Namespace(name string) Registry {
	return &namespace{
		name:              name,
		ComponentRegistry: r,
	}
}

func (r *ComponentRegistry) Register(name string, componentFn ComponentFunc) {
	if strings.Contains(name, ".") && !(strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".")) {
		var parts = strings.SplitN(name, ".", 2)
		var namespace = r.Namespace(parts[0])
		namespace.Register(parts[1], componentFn)
		return
	}
	r.components[name] = r.newComponent(componentFn)
}

func (r *ComponentRegistry) Render(name string, args ...interface{}) Component {
	if c, ok := r.components[name]; ok {
		var ret = c.Call(args...)
		return ret[0].(Component)
	}

	var hasDot = strings.Contains(name, ".")
	if !hasDot {
		return nil
	}

	var parts = strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return nil
	}

	var ns = parts[0]
	var n = parts[1]

	if c, ok := r.ns_components[ns][n]; ok {
		var ret = c.Call(args...)
		return ret[0].(Component)
	}

	return nil
}

var (
	RegistryObject = NewComponentRegistry()
	Register       = RegistryObject.Register
	Namespace      = RegistryObject.Namespace
	Render         = RegistryObject.Render
)
