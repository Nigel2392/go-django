package components

import (
	"reflect"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
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
			c.rTyp.NumIn() == len(args)-1 || c.rTyp.NumIn() == len(args) || c.rTyp.NumIn() > len(args),
			"component must have fewer or equal number of arguments as the number of arguments passed to Call",
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
				var cnvrted, ok = attrs.RConvert(
					&valueOf, typ,
				)
				if !ok {
					assert.Fail("could not convert %v to %v", valueOf, typ)
				}
				values = reflect.Append(values, *cnvrted)
			}
			if values.Len() > 0 {
				in = append(in, values)
			}
		} else {
			var arg = args[i]
			var valueOf = reflect.ValueOf(arg)
			var cnvrted, ok = attrs.RConvert(
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
