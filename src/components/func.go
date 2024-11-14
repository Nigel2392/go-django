package components

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

type ComponentFunc = interface{} // func(...interface{}) -> Component

type reflectFunc struct {
	fn   ComponentFunc
	rTyp reflect.Type
	rVal reflect.Value
}

func (c *reflectFunc) Call(args ...interface{}) interface{} {

	assert.True(c.rTyp.Kind() == reflect.Func, "component must be a function")
	if c.rTyp.IsVariadic() {
		assert.True(
			len(args) >= c.rTyp.NumIn()-1,
			"component must have at least %v arguments",
			c.rTyp.NumIn()-1,
		)
	} else {
		assert.True(c.rTyp.NumIn() == len(args), "component must have the same number of arguments as the number of arguments passed to Call")
	}
	assert.True(c.rTyp.NumOut() == 1, "component must return a single value of type interface { Render(ctx context.Context, w io.Writer) error }")

	variadicIndex := c.rTyp.NumIn() - 1
	in := make([]reflect.Value, 0, c.rTyp.NumIn())

	for i := 0; i < c.rTyp.NumIn(); i++ {
		var typ = c.rTyp.In(i)
		if c.rTyp.IsVariadic() && i == variadicIndex {
			var values = reflect.MakeSlice(typ, 0, 0)
			for j := variadicIndex; j < len(args); j++ {
				var valueOf = reflect.ValueOf(args[j])
				var cnvrted, ok = django_reflect.RConvert(
					&valueOf, typ.Elem(),
				)
				if !ok {
					assert.Fail("could not convert %v to %v", valueOf, typ)
				}
				values = reflect.Append(values, *cnvrted)
			}
			if values.Len() > 0 {
				for j := 0; j < values.Len(); j++ {
					in = append(in, values.Index(j))
				}
			}
		} else {
			var arg = args[i]
			var valueOf = reflect.ValueOf(arg)
			var cnvrted, ok = django_reflect.RConvert(
				&valueOf, typ,
			)
			if !ok {
				assert.Fail("could not convert %v to %v", valueOf, typ)
			}
			in = append(in, *cnvrted)
		}

		//if argTyp.ConvertibleTo(c.rTyp.In(i)) {
		//	in[i] = reflect.ValueOf(arg).Convert(c.rTyp.In(i))
		//} else {
		//	in[i] = reflect.ValueOf(arg)
		//}
	}

	var out = c.rVal.Call(in)
	return out[0].Interface()

}
