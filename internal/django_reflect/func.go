package django_reflect

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	dj_errors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/pkg/errors"
)

//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

type Function = interface{} // func(...interface{}) -> Component

const CodeFunctionError dj_errors.GoCode = "FunctionError"

var (
	ErrFunction       = dj_errors.New(CodeFunctionError, "function error")
	ErrTypeMismatch   = ErrFunction.WithCause(dj_errors.TypeMismatch)
	ErrNotFunc        = ErrFunction.WithCause(errors.New("fn must be a function"))
	ErrArgCount       = ErrFunction.WithCause(errors.New("argument count mismatch"))
	ErrReturnCount    = ErrFunction.WithCause(errors.New("return count mismatch"))
	ErrNilObject      = ErrFunction.WithCause(errors.New("object is nil"))
	ErrMethodNotFound = ErrFunction.WithCause(errors.New("method not found"))
)

type funcArg struct {
	currIdx int
	vals    []reflect.Value
}

type funcArgWrapper struct {
	argMap map[reflect.Type]funcArg
}

func cloneArgsMap(m map[reflect.Type]funcArg) map[reflect.Type]*funcArg {
	var out = make(map[reflect.Type]*funcArg, len(m))
	for k, v := range m {
		nv := v
		out[k] = &nv
	}
	return out
}

func argFromMap(t reflect.Type, direct map[reflect.Type]*funcArg, a, c map[reflect.Type]reflect.Type) (ret reflect.Type, arg *funcArg, conv bool, ok bool) {
	orig := t
	if arg, ok = direct[t]; ok {
		return t, arg, false, ok
	}

	t, ok = a[orig]
	if arg, ok = direct[t]; ok {
		return t, arg, false, ok
	}

	t, ok = c[orig]
	if arg, ok = direct[t]; ok {
		return t, arg, true, ok
	}

	return orig, nil, false, false
}

func (fw *funcArgWrapper) wrap(src reflect.Value, srcTyp reflect.Type, dst reflect.Type, injectCtx bool) reflect.Value {

	if srcTyp == dst {
		return src
	}

	var (
		foundArgsIn   = make(map[int]reflect.Value)
		newFuncInputs = make([]reflect.Type, 0, srcTyp.NumIn())
		out           = make([]reflect.Type, 0, srcTyp.NumOut())
		seenM         = make(map[reflect.Type]struct{})
		argMap        = cloneArgsMap(fw.argMap)
		srcTypNumIn   = srcTyp.NumIn()

		foundTypes   int
		assignables  = make(map[reflect.Type]reflect.Type)
		convertibles = make(map[reflect.Type]reflect.Type)
	)

outer:
	for i := 0; i < srcTypNumIn; i++ {
		rIn := srcTyp.In(i)

		if foundTypes == len(argMap) {
			break
		}

		if _, ok := seenM[rIn]; ok {
			continue
		}

		seenM[rIn] = struct{}{}

		if _, ok := argMap[rIn]; ok {
			foundTypes++
			continue
		}

		for l := range argMap {
			if l.AssignableTo(rIn) || l.Kind() == reflect.Interface && rIn.AssignableTo(l) {
				assignables[rIn] = l
				foundTypes++
				continue outer
			}

			if (isSafeConversion(rIn, l) && l.ConvertibleTo(rIn)) || (l.Kind() == reflect.Interface && rIn.ConvertibleTo(l)) {
				convertibles[rIn] = l
				foundTypes++
				continue outer
			}
		}
	}

	for i := range srcTypNumIn {
		rIn := srcTyp.In(i)

		if len(argMap) == 0 {
			newFuncInputs = append(newFuncInputs, rIn)
			continue
		}

		if t, a, conv, ok := argFromMap(rIn, argMap, assignables, convertibles); ok {
			v := a.vals[a.currIdx]
			if conv {
				v = v.Convert(rIn)
			}

			foundArgsIn[i] = v
			a.currIdx++
			if a.currIdx == len(a.vals) {
				delete(argMap, t)
			}
			continue
		}

		newFuncInputs = append(newFuncInputs, rIn)
	}

	for i := range srcTyp.NumOut() {
		out = append(out, srcTyp.Out(i))
	}

	var isNewVariadic = srcTyp.IsVariadic()
	if isNewVariadic {
		if _, ok := foundArgsIn[srcTypNumIn-1]; ok {
			isNewVariadic = false
		}
	}

	var inLen = srcTypNumIn
	var newFuncType = reflect.FuncOf(newFuncInputs, out, isNewVariadic)
	return reflect.MakeFunc(newFuncType, func(in []reflect.Value) []reflect.Value {
		var newIn = make([]reflect.Value, 0, inLen)
		var inIdx = 0
		for loopIdx := 0; loopIdx < inLen; loopIdx++ {
			if a, ok := foundArgsIn[loopIdx]; ok {
				newIn = append(newIn, a)
			} else {
				newIn = append(newIn, in[inIdx])
				inIdx++
			}
		}

		if srcTyp.IsVariadic() {
			return src.CallSlice(newIn)
		}

		return src.Call(newIn)
	})
}

var _contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

type Argument interface {
	Type() reflect.Type
	Arg() any
}

type argument[T any] struct {
	v T
}

func Arg[T any](v T) Argument {
	return argument[T]{v: v}
}

func (f argument[T]) Type() reflect.Type {
	var rt = reflect.TypeOf(new(T)).Elem()
	return rt
}

func (f argument[T]) Arg() any {
	return f.v
}

func WithFuncArgs(args ...any) func(*FuncConfig) {
	var fw = &funcArgWrapper{
		argMap: make(map[reflect.Type]funcArg),
	}

	var context reflect.Value
	for _, arg := range args {
		var (
			rv reflect.Value
			rt reflect.Type
		)

		switch a := arg.(type) {
		case reflect.Value:
			rv = a
			rt = a.Type()
		case Argument:
			rv = reflect.ValueOf(a.Arg())
			rt = a.Type()
		default:
			rv = reflect.ValueOf(arg)
			rt = rv.Type()
		}

		if context.Kind() == reflect.Invalid && (rt == _contextType || rt.Implements(_contextType)) {
			if rv.Kind() == reflect.Invalid {
				rv = reflect.New(rt).Elem()
			}
			context = rv
			continue
		}

		l, ok := fw.argMap[rt]
		if ok {
			l.vals = append(l.vals, rv)
		} else {
			l.vals = []reflect.Value{rv}
		}

		fw.argMap[rt] = l
	}

	return func(fc *FuncConfig) {
		if context.Kind() != reflect.Invalid {
			fc.InjectContext = context
		}
		if len(fw.argMap) > 0 {
			fc.Wrappers = append(fc.Wrappers, fw.wrap)
		}
	}
}

type FuncConfig struct {
	InjectContext reflect.Value
	Wrappers      []func(src reflect.Value, srcTyp reflect.Type, dst reflect.Type, injectCtx bool) (newSrc reflect.Value)
	Decorators    []func(fn func([]reflect.Value) []reflect.Value) func([]reflect.Value) []reflect.Value
}

func (f FuncConfig) wrap(fnType reflect.Type, fnVal reflect.Value, out reflect.Type, injectCtx bool) (reflect.Type, reflect.Value, error) {
	// Apply generic wrappers before validation
	var lastOut = out
	for _, wrap := range f.Wrappers {
		if err := validateIsFunc(fnType); err != nil {
			return fnType, fnVal, err
		}
		fnVal = wrap(fnVal, fnType, lastOut, injectCtx)
		fnType = fnVal.Type()
		lastOut = fnType
	}

	return fnType, fnVal, nil
}

func WithContext(ctx context.Context) func(*FuncConfig) {
	return func(c *FuncConfig) {
		if ctx == nil {
			c.InjectContext = reflect.ValueOf(&ctx).Elem()
		} else {
			c.InjectContext = reflect.ValueOf(ctx)
		}
	}
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
		return n, errors.Wrapf(err, "method %s on %T is not compatible with %v", name, obj, fnT)
	}

	var i = converted.Interface()
	if i == nil {
		return n, errors.Wrapf(ErrTypeMismatch, "method %s on %T is nil, cannot cast to %v", name, obj, fnT)
	}

	n, ok := i.(T)
	if !ok {
		return n, errors.Wrapf(ErrTypeMismatch, "method %s on %T is not of type %v, got %T", name, obj, fnT, i)
	}

	return n, nil
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

	if err := validateIsFunc(out); err != nil {
		return reflect.Value{}, err
	}
	if err := validateIsFunc(fnType); err != nil {
		return reflect.Value{}, err
	}

	if fnVal.Kind() == reflect.Func && fnVal.IsNil() {
		return reflect.Value{}, ErrNotFunc
	}

	var config = &FuncConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if err := validateIsFunc(fnType); err != nil {
		return reflect.Value{}, err
	}

	// get context to inject (if any)
	var injectCtx bool
	var ctxVal reflect.Value
	if config.InjectContext.Kind() != reflect.Invalid {
		if fnType.NumIn() > 0 && fnType.In(0) == _contextType {
			if out.NumIn() == 0 || out.In(0) != _contextType {
				injectCtx = true
				ctxVal = config.InjectContext
			}
		}
	}

	// apply wrappers, these might change fnType
	fnType, fnVal, err := config.wrap(fnType, fnVal, out, injectCtx)
	if err := validateIsFunc(fnType); err != nil {
		return fnVal, err
	}

	// fast-path for matching or easily convertible func types
	var canQuickReturn = !injectCtx && len(config.Wrappers) == 0 && len(config.Decorators) == 0
	if canQuickReturn {
		if fnType == out {
			return fnVal, nil
		}
		if fnType.ConvertibleTo(out) {
			return fnVal.Convert(out), nil
		}
	}

	// Build Argument converters
	argBuilder, expectedCallInLength, err := compileArgBuilders(fnType, out, injectCtx, ctxVal)
	if err != nil {
		return reflect.Value{}, err
	}

	// Build Return converters
	retConverter, err := compileReturnConverter(fnType, out)
	if err != nil {
		return reflect.Value{}, err
	}

	var function = func(in []reflect.Value) []reflect.Value {
		callIn := make([]reflect.Value, 0, expectedCallInLength)
		// safe because callIn doesn't live past this function
		callInPtr := (*[]reflect.Value)(noescape(unsafe.Pointer(&callIn)))
		argBuilder(in, callInPtr)
		return retConverter(fnVal.Call(callIn))
	}

	for _, dec := range config.Decorators {
		function = dec(function)
	}

	return reflect.MakeFunc(out, function), nil
}

type argStep func(inVal reflect.Value, callIn *[]reflect.Value)

func compileArgBuilders(srcFnTyp, dstFnTyp reflect.Type, injectCtx bool, ctxVal reflect.Value) (func([]reflect.Value, *[]reflect.Value), int, error) {
	var (
		numInSrc = srcFnTyp.NumIn()
		numInDst = dstFnTyp.NumIn()
	)

	if !injectCtx {
		if numInSrc > numInDst && !dstFnTyp.IsVariadic() {
			return nil, 0, errors.Wrapf(ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v", numInDst, numInSrc)
		}
		if numInSrc < numInDst && !srcFnTyp.IsVariadic() {
			return nil, 0, errors.Wrapf(ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v", numInDst, numInSrc)
		}
	}

	srcTypes := make([]reflect.Type, numInSrc)
	for i := 0; i < numInSrc; i++ {
		srcTypes[i] = srcFnTyp.In(i)
	}

	dstTypes := make([]reflect.Type, numInDst)
	for i := 0; i < numInDst; i++ {
		dstTypes[i] = dstFnTyp.In(i)
	}

	isSrcVariadic := srcFnTyp.IsVariadic()
	isDstVariadic := dstFnTyp.IsVariadic()

	expectedLength := numInSrc
	if isSrcVariadic {
		expectedLength = 3 // buffer for variadic growth
	}

	var steps []argStep
	var srcIdx = 0
	if injectCtx {
		srcIdx = 1
	}

	for i := 0; i < numInDst; i++ {
		// Handle Source Variadic where destination is NOT variadic (or we are not at the last dest element)
		if isSrcVariadic && srcIdx == numInSrc-1 && !(isDstVariadic && i == numInDst-1) {
			var variadicType = srcTypes[srcIdx].Elem()
			conv := compileTypeConverter(dstTypes[i], variadicType)
			idx := i // capture
			steps = append(steps, func(inVal reflect.Value, callIn *[]reflect.Value) {
				val, ok := conv(inVal)
				if !ok {
					assert.Fail(errors.Wrapf(ErrTypeMismatch, "could not convert %T [%d]: (%v) to %v", inVal.Interface(), idx, inVal, variadicType))
				}
				*callIn = append(*callIn, val)
			})
			continue
		}

		// Handle Dest Variadic (this is always the last element i == numInDst-1)
		if isDstVariadic && i == numInDst-1 {
			var argTyp = dstTypes[i] // this is a slice type

			// If src is also variadic and we are at the variadic arg
			if isSrcVariadic && srcIdx == numInSrc-1 {
				var castType = srcTypes[srcIdx]
				conv := compileTypeConverter(argTyp, castType)
				steps = append(steps, func(inVal reflect.Value, callIn *[]reflect.Value) {
					val, ok := conv(inVal)
					if !ok {
						assert.Fail(errors.Wrapf(ErrTypeMismatch, "could not convert %T (%v) to %v", inVal.Interface(), inVal, castType))
					}
					*callIn = append(*callIn, val)
				})
			} else {
				// Dest is variadic, but source expects discrete arguments (or discrete then variadic)
				// We must unroll the slice into remaining srcIdxs.
				var elemType = argTyp.Elem()

				// Precompile converters for the remaining discrete source args
				var discConvLen int
				for s := srcIdx; s < numInSrc; s++ {
					if isSrcVariadic && s == numInSrc-1 {
						break
					}
					discConvLen++
				}

				var discreteConverters []converterFunc
				if discConvLen > 0 {
					discreteConverters = make([]converterFunc, 0, discConvLen)
					for s := srcIdx; s < numInSrc; s++ {
						if isSrcVariadic && s == numInSrc-1 {
							break
						}

						discreteConverters = append(
							discreteConverters,
							compileTypeConverter(elemType, srcTypes[s]),
						)
					}
				}

				var variadicConv converterFunc
				var variadicDstType reflect.Type
				if isSrcVariadic {
					variadicDstType = srcTypes[numInSrc-1].Elem()
					variadicConv = compileTypeConverter(elemType, variadicDstType)
				}

				steps = append(steps, func(inVal reflect.Value, callIn *[]reflect.Value) {
					inLen := inVal.Len()

					// Validate length
					if isSrcVariadic {
						if inLen < len(discreteConverters) {
							assert.Fail(errors.Wrapf(ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v", numInDst, inLen))
						}
					} else {
						if inLen != len(discreteConverters) {
							assert.Fail(errors.Wrapf(ErrArgCount, "function must have the same number of arguments as the output function (%v), got %v", numInDst, inLen))
						}
					}

					// Process discrete
					for j, conv := range discreteConverters {
						elem := inVal.Index(j)
						val, ok := conv(elem)
						if !ok {
							assert.Fail(errors.Wrapf(ErrTypeMismatch, "could not convert %T [%d]: (%v) to %v", elem.Interface(), j, elem, srcTypes[srcIdx+j]))
						}
						*callIn = append(*callIn, val)
					}

					// Process variadic
					if isSrcVariadic {
						for j := len(discreteConverters); j < inLen; j++ {
							elem := inVal.Index(j)
							val, ok := variadicConv(elem)
							if !ok {
								assert.Fail(errors.Wrapf(ErrTypeMismatch, "could not convert %T [%d]: (%v) to %v", elem.Interface(), j, elem, variadicDstType))
							}
							*callIn = append(*callIn, val)
						}
					}
				})
			}
			srcIdx++
			continue
		}

		// Standard Match
		if srcIdx >= numInSrc {
			return nil, 0, errors.Wrapf(ErrArgCount, "function must have the same number of arguments as the output function (%v), got too many", numInDst)
		}
		var typ = srcTypes[srcIdx]
		conv := compileTypeConverter(dstTypes[i], typ)
		steps = append(steps, func(inVal reflect.Value, callIn *[]reflect.Value) {
			val, ok := conv(inVal)
			if !ok {
				assert.Fail(errors.Wrapf(ErrTypeMismatch, "could not convert %T (%v) to %v", inVal.Interface(), inVal, typ))
			}
			*callIn = append(*callIn, val)
		})
		srcIdx++
	}

	builder := func(in []reflect.Value, callIn *[]reflect.Value) {
		if injectCtx {
			*callIn = append(*callIn, ctxVal)
		}
		for i, step := range steps {
			step(in[i], callIn)
		}
	}

	return builder, expectedLength, nil
}

func compileReturnConverter(srcFnTyp, dstFnTyp reflect.Type) (func([]reflect.Value) []reflect.Value, error) {
	var (
		numOutSrc = srcFnTyp.NumOut()
		numOutDst = dstFnTyp.NumOut()
		outZero   reflect.Type
	)

	if numOutDst > 0 {
		outZero = dstFnTyp.Out(0)
	}

	switch {
	case numOutDst == 0:
		return func(res []reflect.Value) []reflect.Value { return []reflect.Value{} }, nil
	case numOutSrc == numOutDst && canConvertSimple(srcFnTyp.Out(0), dstFnTyp.Out(0)):
		// keep exact matcher inside closure
	case numOutDst == 1 && isErrType(outZero) && numOutSrc > 0 && isErrType(srcFnTyp.Out(numOutSrc-1)):
		// keep exact matcher inside closure
	case numOutDst == 2 && numOutSrc > 1 && isErrType(dstFnTyp.Out(1)) && isErrType(srcFnTyp.Out(numOutSrc-1)) && (isLiteralAny(outZero) || isAnySlice(outZero)):
		// keep exact matcher inside closure
	case numOutDst == 1 && (isLiteralAny(outZero) || isAnySlice(outZero)) && numOutSrc >= 1:
		// keep exact matcher inside closure
	default:
		return nil, errors.Wrapf(ErrReturnCount,
			"function must return the same number of values as the output function, src != dest (%v), got %v: %s != %s",
			numOutDst, numOutSrc, srcFnTyp, dstFnTyp,
		)
	}

	dstTypes := make([]reflect.Type, numOutDst)
	retConverters := make([]converterFunc, numOutDst)
	for i := 0; i < numOutDst; i++ {
		dstTypes[i] = dstFnTyp.Out(i)

		var srcType = srcFnTyp.Out(i)
		if numOutDst == 1 && isErrType(outZero) && numOutSrc > 0 && isErrType(srcFnTyp.Out(numOutSrc-1)) {
			srcType = srcFnTyp.Out(numOutSrc - 1)
		} else if numOutDst == 2 && numOutSrc > 1 && isErrType(dstFnTyp.Out(1)) && isErrType(srcFnTyp.Out(numOutSrc-1)) && (isLiteralAny(outZero) || isAnySlice(outZero)) {
			if i == 1 {
				srcType = srcFnTyp.Out(numOutSrc - 1)
			} else {
				srcType = _literalAny
			}
		} else if numOutDst == 1 && (isLiteralAny(outZero) || isAnySlice(outZero)) && numOutSrc >= 1 {
			srcType = _literalAny
		}

		if isAnySlice(dstTypes[i]) {
			// For slice returns generated dynamically
			retConverters[i] = func(v reflect.Value) (reflect.Value, bool) { return v, true }
		} else {
			retConverters[i] = compileTypeConverter(srcType, dstTypes[i])
		}
	}

	return func(res []reflect.Value) []reflect.Value {
		if len(res) == 0 {
			return []reflect.Value{}
		}

		switch {
		case numOutDst == 0:
			return []reflect.Value{}
		case numOutSrc == numOutDst && canConvertSimple(srcFnTyp.Out(0), dstFnTyp.Out(0)):
			// standard processing below
		case numOutDst == 1 && isErrType(outZero) && numOutSrc > 0 && isErrType(srcFnTyp.Out(numOutSrc-1)):
			res = res[numOutSrc-1:]
		case numOutDst == 2 && numOutSrc > 1 && isErrType(dstFnTyp.Out(1)) && isErrType(srcFnTyp.Out(numOutSrc-1)) && (isLiteralAny(outZero) || isAnySlice(outZero)):
			if len(res) > 2 || isAnySlice(outZero) {
				var slice = reflect.MakeSlice(reflect.SliceOf(_literalAny), 0, len(res)-1)
				for i := 0; i < len(res)-1; i++ {
					slice = reflect.Append(slice, res[i])
				}
				return []reflect.Value{slice, res[len(res)-1]}
			}
		case numOutDst == 1 && (isLiteralAny(outZero) || isAnySlice(outZero)) && numOutSrc >= 1:
			if len(res) > 1 || isAnySlice(outZero) {
				var slice = reflect.MakeSlice(reflect.SliceOf(_literalAny), 0, len(res))
				for i := 0; i < len(res); i++ {
					slice = reflect.Append(slice, res[i])
				}
				return []reflect.Value{slice}
			}
		default:
			assert.Fail(errors.Wrapf(ErrReturnCount, "function must return the same number of values as the output function (%v), got %v", numOutDst, numOutSrc))
		}

		for i, curr := range res {
			var cnvrted, ok = retConverters[i](curr)
			if !ok {
				assert.Fail(errors.Wrapf(ErrReturnCount, "function return value %v is not convertible to %v", curr.Type(), dstTypes[i]))
			}
			res[i] = cnvrted
		}
		return res
	}, nil
}

type converterFunc func(reflect.Value) (reflect.Value, bool)

func ctc_t_invalid(_ reflect.Value) (reflect.Value, bool) { return reflect.Value{}, false }

func ctc_t_eq(v reflect.Value) (reflect.Value, bool) { return v, true }

func ctc_t_stringer_conv(fromV reflect.Value) (reflect.Value, bool) {
	if s, ok := fromV.Interface().(fmt.Stringer); ok {
		return reflect.ValueOf(s.String()), true
	}
	if b, ok := fromV.Interface().(error); ok {
		return reflect.ValueOf(b.Error()), true
	}
	return reflect.Value{}, false
}

func compileTypeConverter(fromT, toT reflect.Type) converterFunc {
	if fromT == toT {
		return ctc_t_eq
	}

	if fromT.Kind() == reflect.Interface {
		if fromT.AssignableTo(toT) {
			return ctc_t_eq
		}

		return func(fromV reflect.Value) (reflect.Value, bool) {
			if !fromV.IsNil() {
				underlying := fromV.Elem()
				uType := underlying.Type()
				if uType == toT {
					return underlying, true
				}
				if uType.AssignableTo(toT) {
					return underlying, true
				}
				if uType.ConvertibleTo(toT) {
					return underlying.Convert(toT), true
				}
				if toT.Kind() == reflect.Interface && uType.Implements(toT) {
					return underlying, true
				}
			}
			if fromT.AssignableTo(toT) {
				return fromV, true
			}
			if fromT.ConvertibleTo(toT) {
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
			if toT.Kind() == reflect.Interface && fromT.Implements(toT) {
				return fromV, true
			}
			return reflect.Value{}, false
		}
	}

	if fromT.AssignableTo(toT) {
		return ctc_t_eq
	}

	if fromT.ConvertibleTo(toT) {
		if toT.Kind() == reflect.String && !toT.ConvertibleTo(fromT) {
			return ctc_t_stringer_conv
		}
		return func(v reflect.Value) (reflect.Value, bool) { return v.Convert(toT), true }
	}

	if toT.Kind() == reflect.Interface && fromT.Implements(toT) {
		return ctc_t_eq
	}

	return ctc_t_invalid
}

func validateIsFunc(fnType reflect.Type) error {
	if fnType != nil && fnType.Kind() == reflect.Func {
		return nil
	}
	return ErrNotFunc.Wrapf(
		"expected a function, got %T", fnType,
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
