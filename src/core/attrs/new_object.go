package attrs

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

func createIfIface[T Definer](ctx context.Context, v any) (T, bool) {
	var obj T = v.(T)
	switch v := v.(type) {
	case CanCreateObject[T]:
		obj = v.CreateObject(ctx, v.(T))
	case CanCreateObject[Definer]:
		obj = v.CreateObject(ctx, v.(Definer)).(T)
	default:
		var zero T
		return zero, false
	}

	var rVal = reflect.ValueOf(v)
	var rNew = reflect.ValueOf(obj)
	if !rNew.IsValid() || rNew.IsNil() {
		return obj, false
	}
	if rVal.Type() != rNew.Type() {
		return obj, false
	}

	if rVal.IsValid() && !rVal.IsNil() && rVal.CanAddr() && rNew.CanAddr() {
		assert.False(
			rVal.UnsafeAddr() == rNew.UnsafeAddr(),
			"the new object must not be the same as the original object (%d == %d)",
			rVal.UnsafeAddr(), rNew.UnsafeAddr(),
		)
	}

	return obj, true
}

func setup[T Definer](ctx context.Context, obj any) T {
	if setupObj, ok := obj.(CanSetup); ok {
		setupObj.Setup(ctx)
	}
	return obj.(T)
}

type objectRegistry struct {
	types map[reflect.Type]func(context.Context, any) Definer
}

var objectFuncRegistry = &objectRegistry{
	types: make(map[reflect.Type]func(context.Context, any) Definer),
}

func RegisterNewObjectFunc[T any](fn func(ctx context.Context, original T) Definer) {
	var t = reflect.TypeFor[T]()
	objectFuncRegistry.types[t] = func(ctx context.Context, a any) Definer {
		return fn(ctx, a.(T))
	}
}

func RegisterNewObjectFuncReflect(typ reflect.Type, fn func(ctx context.Context, original any) Definer) {
	objectFuncRegistry.types[typ] = fn
}

// Creates a new object from the given Definer type.
//
// This function should always be used to create new objects
// from a Definer type, as it will ensure that the object
// is properly set up and initialized.
//
// This function takes the following types of input:
// - A reflect.Type of the Definer to create an object from.
// - A string which is assumed to be the content type of T
// - A contenttypes.ContentType from which a T can be derived.
// - Any other value which can be safely cast to T
func NewObject[T Definer](ctx context.Context, definer any) T {
	var (
		obj      any
		definerT reflect.Type
	)
	switch v := definer.(type) {
	case reflect.Type:
		if v.Kind() == reflect.Ptr {
			definerT = v
			obj = reflect.New(v.Elem()).Interface()
		} else {
			definerT = reflect.PointerTo(v)
			obj = reflect.New(v).Interface()
		}
	case contenttypes.ContentType:
		obj = v.New()
	case string:
		var cTypeDef = contenttypes.DefinitionForType(v)
		assert.True(
			cTypeDef != nil,
			"NewObject requires a valid content type, got %s",
		)
		obj = cTypeDef.Object()
	default:
		obj = v
	}

	var rT = reflect.TypeOf(obj)
	if fn, ok := objectFuncRegistry.types[rT]; ok {
		return setup[T](ctx, fn(ctx, obj))
	}

	var newObj, ok = createIfIface[T](ctx, obj)
	if ok {
		return setup[T](ctx, newObj)
	}

	if definerT == nil {
		definerT = reflect.TypeOf(definer)
	}
	assert.True(
		definerT.Kind() == reflect.Ptr,
		"NewObject requires a pointer to a Definer type, got %s",
	)

	var cTypeDef = contenttypes.DefinitionForObject(definerT)
	if cTypeDef != nil {
		return cTypeDef.Object().(T)
	}

	definerT = definerT.Elem()
	var newObjT = reflect.New(definerT)
	return setup[T](ctx, newObjT.Interface())
}
