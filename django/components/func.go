package components

import (
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
		var val = reflect.ValueOf(arg)
		var argTyp = val.Type()

		assert.True(
			argTyp.AssignableTo(c.rTyp.In(i)) || argTyp.ConvertibleTo(c.rTyp.In(i)),
			"argument %d must be of type %s, got %s",
			i,
			c.rTyp.In(i),
		)

		if argTyp.ConvertibleTo(c.rTyp.In(i)) {
			in[i] = reflect.ValueOf(arg).Convert(c.rTyp.In(i))
		} else {
			in[i] = reflect.ValueOf(arg)
		}
	}

	var out = c.rVal.Call(in)
	return out[0].Interface()

}
