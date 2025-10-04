package django_reflect

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/pkg/errors"
)

type Function = interface{} // func(...interface{}) -> Component

var (
	ErrTypeMismatch   = errors.New("type mismatch")
	ErrNotFunc        = errors.New("fn must be a function")
	ErrArgCount       = errors.New("argument count mismatch")
	ErrReturnCount    = errors.New("return count mismatch")
	ErrNilObject      = errors.New("object is nil")
	ErrMethodNotFound = errors.New("method not found")
)

// Method retrieves a method from an object.
//
// The generic type parameter must be the type of the method.
func Method[T Function](obj interface{}, name string, opts ...func(*FuncConfig)) (n T, err error) {
	if obj == nil {
		return n, ErrNilObject
	}

	var (
		v = reflect.ValueOf(obj)
		m = v.MethodByName(name)
	)
checkValid:
	if !m.IsValid() {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			goto checkValid
		}
		return n, ErrMethodNotFound
	}

	var fnT = reflect.TypeOf(n)
	converted, err := RCastFunc(fnT, m, opts...)
	if err != nil {
		return n, errors.Wrapf(
			err, "method %s on %T is not compatible with %v",
			name, obj, fnT,
		)
	}

	var i = converted.Interface()
	if i == nil {
		return n, errors.Wrapf(
			ErrTypeMismatch,
			"method %s on %T is nil, cannot cast to %v",
			name, obj, fnT,
		)
	}

	n, ok := i.(T)
	if !ok {
		return n, errors.Wrapf(
			ErrTypeMismatch,
			"method %s on %T is not of type %v, got %T",
			name, obj, fnT, i,
		)
	}

	return n, nil
}

func CastFunc[OUT Function](fn any, opts ...func(*FuncConfig)) (OUT, error) {
	var nT = new(OUT)
	var outTyp = reflect.TypeOf(nT).Elem()
	var rVal, err = RCastFunc(outTyp, fn, opts...)
	if err != nil {
		return *nT, err
	}
	return rVal.Interface().(OUT), nil
}

type FuncConfig struct {
	// wrappers are applied before any validation occurs
	// this /could/ be used to set default values for parameters
	// such as context.Context, http.Request, etc.
	Wrappers []func(src reflect.Value, srcTyp reflect.Type, dst reflect.Type) (newSrc reflect.Value)

	// decorators are applied after validation occurs
	// you can use decorators to execute code before and after
	// the function is called. it is also possible
	// to modify the input and output parameters.
	Decorators []func(fn reflectFunc) reflectFunc
}

type reflectFunc = func([]reflect.Value) []reflect.Value

func wrapWithContext(ctx context.Context) func(src reflect.Value, srcTyp reflect.Type, dst reflect.Type) reflect.Value {
	return func(src reflect.Value, srcTyp reflect.Type, dst reflect.Type) reflect.Value {
		if srcTyp.NumIn() == 0 {
			return src
		}
		if srcTyp.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return src
		}
		if dst.NumIn() != 0 && dst.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() {
			return src
		}

		var newFuncInputs = make([]reflect.Type, 0, srcTyp.NumIn()-1)
		for i := 1; i < srcTyp.NumIn(); i++ {
			newFuncInputs = append(newFuncInputs, srcTyp.In(i))
		}

		var out = make([]reflect.Type, 0, srcTyp.NumOut())
		for i := 0; i < srcTyp.NumOut(); i++ {
			out = append(out, srcTyp.Out(i))
		}

		// New function that adds context.Context as the first parameter
		// The new source function returned has one less input parameter
		// than the original.
		return reflect.MakeFunc(reflect.FuncOf(newFuncInputs, out, srcTyp.IsVariadic()), func(in []reflect.Value) []reflect.Value {
			var newIn = make([]reflect.Value, 0, len(in)+1)
			newIn = append(newIn, reflect.ValueOf(ctx))
			newIn = append(newIn, in...)
			return src.Call(newIn)
		})
	}
}

func WrapWithContext(ctx context.Context) func(*FuncConfig) {
	return func(c *FuncConfig) {
		c.Wrappers = append(c.Wrappers, wrapWithContext(ctx))
	}
}

func RCastFunc(out reflect.Type, fn any, opts ...func(*FuncConfig)) (reflect.Value, error) {
	var (
		fnType reflect.Type
		fnVal  reflect.Value
	)

	switch f := fn.(type) {
	case reflect.Value:
		fnType = f.Type()
		fnVal = f
	default:
		fnType = reflect.TypeOf(f)
		fnVal = reflect.ValueOf(f)
	}

	var config = &FuncConfig{}
	for _, opt := range opts {
		opt(config)
	}

	for _, wrap := range config.Wrappers {
		// validating before wrapping is OK because
		// we also validate after the loop continues
		if err := validateIsFunc(fnType); err != nil {
			return reflect.Value{}, err
		}

		fnVal = wrap(fnVal, fnType, out)
		fnType = fnVal.Type()
	}

	if err := validateIsFunc(fnType); err != nil {
		return reflect.Value{}, err
	}
	if err := validateIsFunc(out); err != nil {
		return reflect.Value{}, err
	}

	if fnType == out || fnType.ConvertibleTo(out) {
		return fnVal.Convert(out), nil
	}

	var (
		numInSrc  = fnType.NumIn()
		numInDst  = out.NumIn()
		numOutSrc = fnType.NumOut()
		numOutDst = out.NumOut()
	)

	switch {
	case numInSrc > numInDst && !out.IsVariadic():
		return reflect.Value{}, errors.Wrapf(
			ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v",
			numInDst, numInSrc,
		)
	case numInSrc < numInDst && !fnType.IsVariadic():
		return reflect.Value{}, errors.Wrapf(
			ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v",
			numInDst, numInSrc,
		)
	}

	switch {
	case numOutDst == 0: // func(...)
	// ignore all return values
	case numOutSrc == numOutDst && canConvertSimple(fnType.Out(0), out.Out(0)): // func(...) ...
	// exact match
	case numOutDst == 1 && isErrType(out.Out(0)) && numOutSrc > 0 && isErrType(fnType.Out(numOutSrc-1)): // func(...) error
	// last return value is error and only error, ignore other return values
	case numOutDst == 2 && (isLiteralAny(out.Out(0)) || isAnySlice(out.Out(0))) && isErrType(out.Out(1)) && numOutSrc > 1 && isErrType(fnType.Out(numOutSrc-1)): // func(...) (interface{}, error) or func(...) ([]interface{}, error)
	// if len of res is greater than 2, we can create a slice and return the first value as []interface{} + error
	case numOutDst == 1 && (isLiteralAny(out.Out(0)) || isAnySlice(out.Out(0))) && numOutSrc >= 1: // func(...) interface{} or func(...) []interface{}
	// if len of res is greater than 1, we can create a slice and return the first value as []interface{}
	default:
		return reflect.Value{}, errors.Wrapf(
			ErrReturnCount, "function must return the same number of values as the output function (%v), got %v",
			numOutDst, numOutSrc,
		)
	}

	var function = func(in []reflect.Value) []reflect.Value {
		var callIn = make([]reflect.Value, 0, len(in))
	argLoop:
		for i := 0; i < len(in); i++ {
			switch {
			case i >= numInSrc && !fnType.IsVariadic() || i >= numInSrc && !out.IsVariadic():
				assert.Fail(errors.Wrapf(
					ErrArgCount,
					"function must have the same number of arguments as the output function (%v), got %v",
					numInDst, len(in),
				))

			case fnType.IsVariadic() && i == numInSrc-1:
				// handle variadic parameters
				var variadicType = fnType.In(i).Elem()
				for j := i; j < len(in); j++ {
					var argTyp = in[j].Type()
					var argVal, ok = convertType(argTyp, variadicType, in[j])
					if !ok {
						assert.Fail(errors.Wrapf(
							ErrTypeMismatch,
							"could not convert %T [%d]: (%v) to %v",
							in[j].Interface(), j, in[j], variadicType,
						))
					}
					callIn = append(callIn, argVal)
				}
				break argLoop

			case out.IsVariadic() && i >= numInDst-1:

				if i >= numInSrc {
					assert.Fail(errors.Wrapf(
						ErrArgCount,
						"function must have the same number of arguments as the output function (%v), got %v",
						numInDst, len(callIn),
					))
				}

				// handle variadic parameters
				var argTyp = in[i].Type()
				var castType = fnType.In(i)
				switch castType.Kind() {
				case reflect.Slice:
					if argTyp.Kind() == reflect.Slice {
						var conv, ok = convertType(argTyp, castType, in[i])
						if !ok {
							assert.Fail(errors.Wrapf(
								ErrTypeMismatch,
								"could not convert %T (%v) to %v",
								in[i].Interface(), in[i], castType,
							))
						}
						callIn = append(callIn, conv)
						break argLoop
					}

					var elemType = castType.Elem()
					var sliceVal = reflect.MakeSlice(castType, 0, 0)
					for j := i; j < len(in); j++ {
						var argTyp = in[j].Type()
						var argVal, ok = convertType(argTyp, elemType, in[j])
						if !ok {
							assert.Fail(errors.Wrapf(
								ErrTypeMismatch,
								"could not convert %T [%d]: (%v) to %v",
								in[j].Interface(), j, in[j], elemType,
							))
						}
						sliceVal = reflect.Append(sliceVal, argVal)
					}

					callIn = append(callIn, sliceVal)
					break argLoop

				default:
					if argTyp.Kind() == reflect.Slice {
						for j := 0; j < in[i].Len(); j++ {
							var elem = in[i].Index(j)
							var elemTyp = elem.Type()
							var argVal, ok = convertType(elemTyp, castType, elem)
							if !ok {
								continue argLoop
							}
							callIn = append(callIn, argVal)
						}
						break argLoop
					}

					var argVal, ok = convertType(argTyp, castType, in[i])
					if !ok {
						assert.Fail(errors.Wrapf(
							ErrTypeMismatch,
							"could not convert %T (%v) to %v",
							in[i].Interface(), in[i], castType,
						))
					}

					callIn = append(callIn, argVal)
				}
			}

			var typ = fnType.In(i)
			var argTyp = in[i].Type()
			var argVal, ok = convertType(argTyp, typ, in[i])
			if !ok {
				assert.Fail(errors.Wrapf(
					ErrTypeMismatch,
					"could not convert %T (%v) to %v",
					in[i].Interface(), in[i], typ,
				))
			}

			callIn = append(callIn, argVal)
		}

		if len(callIn) < fnType.NumIn() && !fnType.IsVariadic() || len(callIn) > fnType.NumIn() && !fnType.IsVariadic() {
			assert.Fail(errors.Wrapf(
				ErrArgCount,
				"function must have the same number of arguments as the output function (%v), got %v",
				numInDst, len(callIn),
			))
		}

		return callConvertedFunc(out, fnVal, callIn)
	}

	for _, dec := range config.Decorators {
		function = dec(function)
	}

	return reflect.MakeFunc(out, function), nil
}

func validateIsFunc(fnType reflect.Type) error {
	if fnType != nil && fnType.Kind() == reflect.Func {
		return nil
	}
	return errors.Wrapf(
		ErrNotFunc, "expected a function, got %T", fnType,
	)
}

var _errType = reflect.TypeOf((*error)(nil)).Elem()
var _literalAny = reflect.TypeOf((*interface{})(nil)).Elem()

func isErrType(t reflect.Type) bool {
	return t.AssignableTo(_errType) || t == _errType || (t.Kind() == reflect.Interface && t.Implements(_errType))
}

func isLiteralAny(t reflect.Type) bool {
	return t == _literalAny
}

func isAnySlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem() == _literalAny
}

func callConvertedFunc(dstFnTyp reflect.Type, srcFnVal reflect.Value, convertedArgs []reflect.Value) []reflect.Value {
	var res = srcFnVal.Call(convertedArgs)
	if len(res) == 0 {
		return []reflect.Value{}
	}

	var (
		srcFnTyp  = srcFnVal.Type()
		numOutSrc = srcFnTyp.NumOut()
		numOutDst = dstFnTyp.NumOut()
		outZero   reflect.Type
	)

	if numOutDst > 0 {
		outZero = dstFnTyp.Out(0)
	}

	switch {
	case numOutDst == 0: // func(...)
		// ignore all return values
		return []reflect.Value{}
	case numOutSrc == numOutDst && canConvertSimple(srcFnTyp.Out(0), dstFnTyp.Out(0)): // func(...) ...
		// exact match
	case numOutDst == 1 && isErrType(outZero) && numOutSrc > 0 && isErrType(srcFnVal.Type().Out(numOutSrc-1)): // func(...) error
		// last return value is error and only error, ignore other return values
		res = res[numOutSrc-1:]
	case numOutDst == 2 && // func(...) (interface{}, error)
		numOutSrc > 1 &&
		isErrType(dstFnTyp.Out(1)) &&
		isErrType(srcFnVal.Type().Out(numOutSrc-1)) &&
		(isLiteralAny(outZero) || isAnySlice(outZero)):

		// if len of res is greater than 2, we can create a slice and return the first value as []interface{}
		// no further conversions are required in this case.
		if len(res) > 2 || isAnySlice(outZero) {
			var slice = reflect.MakeSlice(reflect.SliceOf(_literalAny), 0, len(res)-1)
			for i := 0; i < len(res)-1; i++ {
				slice = reflect.Append(slice, res[i])
			}
			return []reflect.Value{slice, res[len(res)-1]} // return []interface{}, error
		}

		// continue to convert the results as normal
	case numOutDst == 1 && (isLiteralAny(outZero) || isAnySlice(outZero)) && numOutSrc >= 1: // func(...) interface{}
		// if len of res is greater than 1, we can create a slice and return the first value as []interface{}
		if len(res) > 1 || isAnySlice(outZero) {
			var slice = reflect.MakeSlice(reflect.SliceOf(_literalAny), 0, len(res))
			for i := 0; i < len(res); i++ {
				slice = reflect.Append(slice, res[i])
			}
			return []reflect.Value{slice} // return []interface{}
		}

		// continue to convert the results as normal
	default:
		assert.Fail(errors.Wrapf(
			ErrReturnCount, "function must return the same number of values as the output function (%v), got %v",
			numOutDst, numOutSrc,
		))
	}

	var results = make([]reflect.Value, len(res))
	for i, curr := range res {
		var typ = dstFnTyp.Out(i)
		var currType = curr.Type()

		var cnvrted, ok = convertType(currType, typ, curr)
		if !ok {
			assert.Fail(errors.Wrapf(
				ErrReturnCount,
				"function return value %v is not convertible to %v",
				currType, typ,
			))
		}

		results[i] = cnvrted
	}
	return results
}

func canConvertSimple(from, to reflect.Type) bool {
	if from == to {
		return true
	}
	if to.Kind() == reflect.Interface && from.Implements(to) {
		return true
	}
	if from.AssignableTo(to) {
		return true
	}
	if from.ConvertibleTo(to) && to.ConvertibleTo(from) {
		return true
	}
	return false
}

func convertType(fromT, toT reflect.Type, fromV reflect.Value) (reflect.Value, bool) {
	// Exact type match
	if fromT == toT {
		return fromV, true
	}

	// If source is an interface, try unwrapping its dynamic value.
	if fromT.Kind() == reflect.Interface && !fromV.IsNil() {
		underlying := fromV.Elem() // dynamic value
		uType := underlying.Type()

		// Direct match after unwrapping
		if uType == toT {
			return underlying, true
		}
		// Assignable
		if uType.AssignableTo(toT) {
			return underlying, true
		}
		// Convertible
		if uType.ConvertibleTo(toT) {
			return underlying.Convert(toT), true
		}
		// If destination is a broader interface implemented by underlying
		if toT.Kind() == reflect.Interface && uType.Implements(toT) {
			return underlying, true
		}
	}

	// Assignable (covers pointer/interface assignment cases)
	if fromT.AssignableTo(toT) {
		return fromV, true
	}

	// Direct convertible (numeric, etc.)
	if fromT.ConvertibleTo(toT) {
		// If converting to string, check for Stringer or error interface
		// we check the reverse of ConvertibleTo because integers should not
		// be converted to strings implicitly.
		if toT.Kind() == reflect.String && !toT.ConvertibleTo(fromT) {
			if s, ok := fromV.Interface().(fmt.Stringer); ok {
				return reflect.ValueOf(s.String()), true
			}

			if b, ok := fromV.Interface().(error); ok {
				return reflect.ValueOf(b.Error()), true
			}

			return reflect.Value{}, false
		}

		return fromV.Convert(toT), true
	}

	// Widen to interface
	if toT.Kind() == reflect.Interface && fromT.Implements(toT) {
		return fromV, true
	}

	return reflect.Value{}, false
}

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
