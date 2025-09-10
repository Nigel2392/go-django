package attrutils

import (
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

var __error_type = reflect.TypeOf((*error)(nil)).Elem()

type ObjectBuilder[T any] interface {
	CallBuilderMethod(methodName string, args ...any) (T, []any, error)
}

func BuilderMethod[BUILDER any](bld BUILDER, methodName string, args ...any) (BUILDER, []any, error) {
	if bldr, ok := any(bld).(ObjectBuilder[BUILDER]); ok {
		return bldr.CallBuilderMethod(methodName, args...)
	}

	return builderMethod(bld, methodName, args...)
}

func builderMethod[BUILDER any](bld BUILDER, methodName string, args ...any) (BUILDER, []any, error) {
	var rVal = reflect.ValueOf(bld)
	var method = rVal.MethodByName(methodName)
	if !method.IsValid() {
		return bld, nil, errors.FieldNotFound.Wrapf(
			"method %s not found on type %T", methodName, bld,
		)
	}

	var numIn = method.Type().NumIn()
	var in = make([]reflect.Value, 0, numIn)
	for i, arg := range args {
		var paramType = method.Type().In(i)
		var argVal = reflect.ValueOf(arg)

		if i == numIn-1 && method.Type().IsVariadic() {
			var remainingArgs = args[i:]
			for _, a := range remainingArgs {
				var converted, err = django_reflect.ConvertToType(reflect.ValueOf(a), paramType.Elem())
				if err != nil {
					return bld, nil, errors.Wrapf(err, "argument %d to method %s on %T", i, methodName, bld)
				}

				in = append(in, converted)
			}
			break
		}

		var converted, err = django_reflect.ConvertToType(argVal, paramType)
		if err != nil {
			return bld, nil, errors.Wrapf(err, "argument %d to method %s on %T", i, methodName, bld)
		}

		in = append(in, converted)
	}

	var out = method.Call(in)
	if len(out) == 0 {
		return bld, nil, nil
	}

	var (
		outIdxStart = 1
		outIdxEnd   = len(out) - 1
		retVal      = out[0]
		convertedQS BUILDER
	)

	converted, err := django_reflect.ConvertToType(retVal, rVal.Type())
	switch {
	case err == nil:
		convertedQS = converted.Interface().(BUILDER)
	case errors.Is(err, errors.TypeMismatch):
		outIdxStart = 0
		convertedQS = bld
	default:
		return bld, nil, errors.Wrapf(err, "return value of method %s on %T", methodName, bld)
	}

	var outErr error
	if len(out) > outIdxEnd && out[outIdxEnd].Type().Implements(__error_type) {
		outErr = out[outIdxEnd].Interface().(error)
		outIdxEnd--
	}

	var retArgs = make([]any, outIdxEnd-outIdxStart+1)
	for i := outIdxStart; i <= outIdxEnd; i++ {
		retArgs[i-outIdxStart] = out[i].Interface()
	}

	return convertedQS, retArgs, outErr
}

type Builder[T any] struct {
	orig T
	ref  T
	err  error
}

func NewBuilder[T any](t T) *Builder[T] {
	return &Builder[T]{
		orig: t,
		ref:  t,
	}
}

func (b *Builder[T]) Orig() T {
	return b.orig
}

func (b *Builder[T]) Ref() T {
	return b.ref
}

func (b *Builder[T]) Exec(fn func(bld T) (T, error)) (T, error) {
	if b.err != nil {
		return b.ref, b.err
	}

	var newRef, err = fn(b.ref)
	if err != nil {
		return b.ref, err
	}

	b.ref = newRef
	return b.ref, nil
}

func (b *Builder[T]) Call(methodName string, args ...any) (T, []any, error) {
	if b.err != nil {
		return b.ref, nil, b.err
	}

	var newRef, retArgs, err = builderMethod(b.ref, methodName, args...)
	var rVal = reflect.ValueOf(newRef)
	if err != nil || !rVal.IsValid() {
		return b.ref, retArgs, err
	}

	b.ref = newRef

	return b.ref, retArgs, err
}

func (b *Builder[T]) HasMethod(methodName string) bool {
	var rVal = reflect.ValueOf(b.ref)
	var method = rVal.MethodByName(methodName)
	return method.IsValid()
}

func (b *Builder[T]) Chain(methodName string, args ...any) *Builder[T] {
	if b.err != nil {
		return b
	}

	var newRef, out, err = builderMethod(b.ref, methodName, args...)
	if err != nil {
		b.err = err
		return b
	}

	if len(out) > 0 {
		b.err = errors.Errorf(
			"unexpected return values from method %s on %T: %v",
			methodName, b.ref, out,
		)
		return b
	}

	b.ref = newRef
	return b
}

func (b *Builder[T]) Error() error {
	return b.err
}
