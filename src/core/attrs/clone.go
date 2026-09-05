package attrs

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Nigel2392/versatile"
)

// MustClone clones a model object, it does so by creating a new object of the same type, and copying all Definer fields.
//
// Extra functions can be provided to clone values from the old to new object.
//
// This function panics on any error during the cloning process.
func MustClone[T Definer](ctx context.Context, old T, copy ...func(old T, new T)) T {
	newObj, err := Clone(ctx, old, copy...)
	if err != nil {
		panic(err)
	}
	return newObj
}

// Clone clones a model object, it does so by creating a new object of the same type, and copying all Definer fields.
//
// Extra functions can be provided to clone values from the old to new object.
func Clone[T any](ctx context.Context, old T, copy ...func(old T, new T)) (v T, err error) {
	if any(old) == any(v) {
		return v, nil
	}

	if any(old) == nil {
		return *new(T), nil
	}

	switch v := any(old).(type) {
	case Definer:
		return cloneDefiner(ctx, v, copy)
	}

	rv := reflect.ValueOf(old)
	return rClone[T](ctx, rv.Type(), rv, copy)
}

func rClone[T any](ctx context.Context, typ reflect.Type, val reflect.Value, copy []func(old T, new T)) (v T, err error) {
	switch {
	case val.Kind() == reflect.Struct:

	case val.Kind() == reflect.Pointer && typ.Elem().Kind() == reflect.Struct:

	}

	return
}

func cloneDefiner[T any](ctx context.Context, old Definer, copy []func(old T, new T)) (v T, err error) {
	var (
		newObj  = NewObject[Definer](ctx, old)
		newDefs = Define(ctx, newObj)
		newFlds = newDefs.Fields()

		oldDefs = Define(ctx, old)
		oldFlds = oldDefs.Fields()
	)

	if len(newFlds) != len(oldFlds) {
		return newObj.(T), fmt.Errorf(
			"cannot copy with fields length mismatch (old != new): %d != %d",
			len(oldFlds), len(newFlds),
		)
	}

	var cloner func() (any, error)
	for i, field := range oldDefs.Fields() {
		oldVal := field.GetValue()

		cloner, err = versatile.Method[func() (any, error)](oldVal, []string{"AttrsClone", "Clone", "Copy"},
			versatile.WithContext(ctx),
			versatile.WithFuncArgs(
				versatile.Arg(newObj),
				versatile.Arg(newFlds[i]),
			),
		)
		if err == nil {
			oldVal, err = cloner()
			if err != nil {
				return newObj.(T), err
			}
			goto setValue
		}

		oldVal, err = Clone(ctx, oldVal)
		if err != nil {
			return newObj.(T), err
		}

	setValue:
		if err := newFlds[i].SetValue(oldVal, true); err != nil {
			return newObj.(T), err
		}
	}

	for _, fn := range copy {
		fn(old.(T), newObj.(T))
	}

	return newObj.(T), nil
}
