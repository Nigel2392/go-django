package components

import (
	"context"
	"io"
	"reflect"

	"github.com/Nigel2392/django/core/assert"
)

type ComponentFunc = interface{} // func(...interface{}) -> Component

type reflectFunc struct {
	fn   ComponentFunc
	rTyp reflect.Type
	rVal reflect.Value
}

func (c *reflectFunc) Call(args ...interface{}) interface{} {

	assert.True(c.rTyp.Kind() == reflect.Func, "component must be a function")
	assert.True(c.rTyp.NumIn() == len(args), "component must have the same number of arguments as the number of arguments passed to Call")
	assert.True(c.rTyp.NumOut() == 1, "component must return a single value of type interface { Render(ctx context.Context, w io.Writer) error }")

	in := make([]reflect.Value, len(args))
	for i, arg := range args {

		assert.True(
			in[i].Type().AssignableTo(c.rTyp.In(i)) || in[i].Type().ConvertibleTo(c.rTyp.In(i)),
			"argument %d must be of type %s, got %s",
			i,
			c.rTyp.In(i),
		)

		if in[i].Type().ConvertibleTo(c.rTyp.In(i)) {
			in[i] = reflect.ValueOf(arg).Convert(c.rTyp.In(i))
		} else {
			in[i] = reflect.ValueOf(arg)
		}
	}

	var out = c.rVal.Call(in)
	return out[0].Interface()

}

type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

type ComponentRegistry struct {
	components map[string]*reflectFunc
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]*reflectFunc),
	}
}

func (r *ComponentRegistry) newComponent(fn ComponentFunc) *reflectFunc {
	rTyp := reflect.TypeOf(fn)
	rVal := reflect.ValueOf(fn)

	return &reflectFunc{
		fn:   fn,
		rTyp: rTyp,
		rVal: rVal,
	}
}

func (r *ComponentRegistry) Register(name string, componentFn ComponentFunc) {
	r.components[name] = r.newComponent(componentFn)
}

func (r *ComponentRegistry) Render(name string, args ...interface{}) Component {
	component, ok := r.components[name]
	assert.True(ok, "component %s not found", name)

	return component.Call(args).(Component)
}
