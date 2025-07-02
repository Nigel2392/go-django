package django_reflect

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
)

type Function = interface{} // func(...interface{}) -> Component

type Func struct {
	Fn          Function
	Type        reflect.Type
	Value       reflect.Value
	ReturnTypes []reflect.Type
	BeforeExec  func(in []reflect.Value) error

	// RequiresIn is a list of types that the function requires as input.
	requiresIn map[int]reflect.Type
}

func NewFunc(fn Function, returns ...reflect.Type) *Func {
	var rTyp = reflect.TypeOf(fn)
	var rVal = reflect.ValueOf(fn)

	assert.True(
		rTyp.Kind() == reflect.Func,
		"fn must be a function",
	)

	var funcVal = &Func{
		Fn:    fn,
		Type:  rTyp,
		Value: rVal,
	}

	if len(returns) > 0 {
		funcVal = funcVal.Returns(returns...)
	}

	return funcVal
}

func (c *Func) AdheresTo(fn any) bool {
	var fnType = reflect.TypeOf(fn)
	assert.True(
		fnType.Kind() == reflect.Func,
		"fn must be a function, got %s", fnType.Kind(),
	)

	if c.Type == fnType || c.Type.ConvertibleTo(fnType) {
		return true
	}

	if c.Type.NumIn() != fnType.NumIn() ||
		c.Type.NumOut() != fnType.NumOut() {
		return false
	}

	var variadicIndex = c.Type.NumIn() - 1
	for i := 0; i < c.Type.NumIn(); i++ {
		var typ = c.Type.In(i)
		var fnTyp = fnType.In(i)

		// check if the types match for variadic parameters
		if i == variadicIndex && c.Type.IsVariadic() {
			if typ.Kind() != reflect.Slice {
				return false
			}

			if fnTyp.Kind() != reflect.Slice {
				return false
			}

			if typ.Elem() != fnTyp.Elem() && !typ.Elem().ConvertibleTo(fnTyp.Elem()) {
				return false
			}

			continue
		}

		// check if the types match for non-variadic parameters
		switch {
		case typ == fnTyp:
			// Types match, do nothing
		case typ.ConvertibleTo(fnTyp):
			// Type is convertible to function type, do nothing
		case fnTyp.Kind() == reflect.Interface && typ.Implements(fnTyp) ||
			typ.Kind() == reflect.Interface && fnTyp.Implements(typ):
			// Type implements the interface, do nothing
		default:
			return false
		}
	}
	return true
}

func (c *Func) Requires(index int, typ reflect.Type) *Func {
	assert.True(
		index < c.Type.NumIn(),
		"index %v is out of bounds for function with %v input parameters",
		index, c.Type.NumIn(),
	)

	assert.True(
		c.Type.In(index) == typ || c.Type.In(index).ConvertibleTo(typ) || typ.Kind() == reflect.Interface && c.Type.In(index).Implements(typ),
		"function input parameter %v is not convertible to %v for required parameter at index %v",
		c.Type.In(index), typ, index,
	)

	if c.requiresIn == nil {
		c.requiresIn = make(map[int]reflect.Type)
	}
	c.requiresIn[index] = typ
	return c
}

func (c *Func) Returns(returns ...reflect.Type) *Func {
	assert.True(
		c.Type.NumOut() == len(returns),
		"function must return the same number of values as the number of types passed to Returns",
	)

	for i, typ := range returns {
		assert.True(
			c.Type.Out(i) == typ || c.Type.Out(i).ConvertibleTo(typ) || typ.Kind() == reflect.Interface && c.Type.Out(i).Implements(typ),
			"function return value %v is not convertible to %v",
			c.Type.Out(i), typ,
		)
	}

	c.ReturnTypes = returns
	return c
}

func (c *Func) CallFunc(in []reflect.Value) []interface{} {

	if c.BeforeExec != nil {
		var err = c.BeforeExec(in)
		assert.True(
			err == nil, "BeforeExec function returned an error: %v", err,
		)
	}

	for i, typ := range c.requiresIn {
		assert.True(
			in[i].Type() == typ || in[i].Type().ConvertibleTo(typ) || typ.Kind() == reflect.Interface && in[i].Type().Implements(typ),
			"function input parameter %v is not convertible to %v",
			in[i].Type(), typ,
		)
	}

	var out = c.Value.Call(in)
	if len(out) == 0 {
		return []interface{}{}
	}

	var results = make([]interface{}, len(out))
	for i, curr := range out {
		if i >= len(c.ReturnTypes) && len(c.ReturnTypes) == 0 {
			results[i] = curr.Interface()
			continue
		}

		assert.False(
			i >= len(c.ReturnTypes),
			"function must return %v values, got %v",
			len(c.ReturnTypes), len(out),
		)

		var typ = c.ReturnTypes[i]
		var currType = curr.Type()
		assert.True(
			currType == typ || currType.ConvertibleTo(typ) || typ.Kind() == reflect.Interface && currType.Implements(typ),
			"function return value %v is not convertible to %v",
			currType, typ,
		)

		if curr.Type() != typ {
			if typ.Kind() == reflect.Interface {
				var newVal = reflect.New(typ)
				newVal.Elem().Set(curr)
				curr = newVal.Elem()
			} else {
				curr = curr.Convert(typ)
			}
		}

		results[i] = curr.Interface()
	}

	return results
}

func argsStr(args []interface{}) string {
	var sb = make([]string, 0, len(args))
	for _, arg := range args {
		sb = append(sb, fmt.Sprintf("%T", arg))
	}
	return fmt.Sprintf("[%s]", strings.Join(sb, ", "))
}

func (c *Func) Call(args ...interface{}) []interface{} {

	if c.Type.IsVariadic() {
		assert.True(
			len(args) >= c.Type.NumIn()-1,
			"function must have at least %v arguments, got %v",
			c.Type.NumIn()-1, len(args),
		)
	} else {
		assert.True(
			c.Type.NumIn() == len(args) || c.Type.IsVariadic() && len(args) >= c.Type.NumIn()-1,
			"function %T must have the same number of arguments as the number of arguments passed to Call (%v), got %v",
			c.Fn, c.Type.NumIn(), len(args), argsStr(args),
		)
	}

	var variadicIndex = c.Type.NumIn() - 1
	var in = make([]reflect.Value, 0, c.Type.NumIn())

	for i := 0; i < c.Type.NumIn(); i++ {
		var typ = c.Type.In(i)
		if c.Type.IsVariadic() && i == variadicIndex {
			var values = reflect.MakeSlice(typ, 0, 0)
			for j := variadicIndex; j < len(args); j++ {
				var valueOf = reflect.ValueOf(args[j])
				var cnvrted, ok = RConvert(
					&valueOf, typ.Elem(),
				)
				if !ok {
					assert.Fail("could not convert %T (%v) to %v", valueOf.Interface(), valueOf, typ)
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
			var cnvrted, ok = RConvert(
				&valueOf, typ,
			)
			if !ok {
				assert.Fail("could not convert %T (%v) to %v", valueOf.Interface(), valueOf, typ)
			}
			in = append(in, *cnvrted)
		}

		//if argTyp.ConvertibleTo(c.Type.In(i)) {
		//	in[i] = reflect.ValueOf(arg).Convert(c.Type.In(i))
		//} else {
		//	in[i] = reflect.ValueOf(arg)
		//}
	}

	return c.CallFunc(in)
}
