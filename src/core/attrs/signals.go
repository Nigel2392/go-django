package attrs

import (
	"reflect"
	"sync"

	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
)

type SignalModelMeta struct {
	Definer     Definer
	Definitions StaticDefinitions
	Meta        ModelMeta
}

type SignalThroughModelMeta struct {
	Source      Definer
	Target      Definer
	ThroughInfo Through
	Meta        ModelMeta
}

// The following signals are available for hooking into the `attrs` package's model registration process.
//
// It can be hooked into to add custom logic before a model is registered.
//
// Example usage:
//
//	func init() {
//		attrs.OnBeforeModelRegister.Listen(func(s signals.Signal[attrs.Definer], obj attrs.Definer) error {
//			// Do something before the model is registered
//			return nil
//		})
//	}
var (
	modelSignalPool = signals.NewPool[SignalModelMeta]()

	throughSignalPool = signals.NewPool[SignalThroughModelMeta]()

	// A signal that is called before a model is registered.
	//
	// This can be used to add custom logic before a model is registered.
	OnBeforeModelRegister = modelSignalPool.Get("attrs.OnBeforeModelRegister")

	// A signal that is called after a model is registered.
	//
	// This can be used to add custom logic after a model is registered.
	OnModelRegister = modelSignalPool.Get("attrs.OnModelRegister")

	// A signal that is called when a through model is registered.
	//
	// This is only sent from the forward relation side,
	// not when the through model is actually registered.
	OnThroughModelRegister = throughSignalPool.Get("attrs.OnThroughModelRegister")

	// ResetDefinitions is a signal that can be sent to reset the static definitions of all models.
	//
	// This should be done after all models have been registered, so that the static definitions have enough information to be built correctly.
	ResetDefinitions = signals.New[any]("attrs.ResetDefinitions")

	//_, _ = OnModelRegister.Listen(func(s signals.Signal[SignalModelMeta], smm SignalModelMeta) error {
	//	return ResetDefinitions.Send(nil)
	//})

	// This makes sure to reset the static definitions after all models have been registered.
	//
	// This is so reverse fields are visible in the static definitions - these
	// can only be built after all models have been registered.
	_, _ = ResetDefinitions.Listen(func(s signals.Signal[any], a any) error {
		for _, meta := range modelReg {
			meta.definitions = newStaticDefinitions(NewObject[Definer](meta.model))
		}
		return nil
	})
)

const (
	HookFormFieldForType = "attrs.FormFieldForType"
	DefaultForType       = "attrs.DefaultForType"
)

type FormFieldGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)
type DefaultGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)

// RegisterFormFieldType registers a field type for a given valueOfType.
//
// getField is a function that returns a fields.Field for the given valueOfType.
//
// The valueOfType can be a reflect.Type or any value, in which case the reflect.TypeOf(valueOfType) will be used.
//
// This is a shortcut function for the `HookFormFieldForType` hook.
//
// Example usage:
//
//		RegisterFormFieldType(
//			json.RawMessage{},
//			func(opts ...func(fields.Field)) fields.Field {
//				return fields.JSONField[json.RawMessage](opts...)
//			},
//	 	)

var (
	registeredFormFieldTypes = make(map[reflect.Type]FormFieldGetter)
	_registerFormFieldHook   = &sync.Once{}
)

func registerFormFieldHook() {
	_registerFormFieldHook.Do(func() {
		goldcrest.Register(HookFormFieldForType, 100,
			FormFieldGetter(func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {

				var valTyp = field_v.Type()
				if field_v.IsValid() && valTyp != nil {
					if getter, exists := registeredFormFieldTypes[valTyp]; exists {
						return getter(f, new_field_t_indirected, field_v, opts...)
					}
				}

				if getter, exists := registeredFormFieldTypes[new_field_t_indirected]; exists {
					return getter(f, new_field_t_indirected, field_v, opts...)
				}

				return nil, false
			}),
		)
	})
}

func RegisterFormFieldType(valueOfType any, getField func(opts ...func(fields.Field)) fields.Field) {
	var typ reflect.Type
	switch v := valueOfType.(type) {
	case reflect.Type:
		typ = v
	case reflect.Value:
		typ = v.Type()
	default:
		typ = reflect.TypeOf(valueOfType)
	}

	// Register the getter function for the type
	registeredFormFieldTypes[typ] = func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		return getField(opts...), true
	}

	registerFormFieldHook()
}

func RegisterFormFieldGetter(valueOfType any, getField FormFieldGetter) {
	var typ reflect.Type
	switch v := valueOfType.(type) {
	case reflect.Type:
		typ = v
	case reflect.Value:
		typ = v.Type()
	default:
		typ = reflect.TypeOf(valueOfType)
	}

	// Register the getter function for the type
	registeredFormFieldTypes[typ] = getField

	registerFormFieldHook()
}

// RegisterDefaultType registers a default value to be used for that specific type.
//
// This is useful when implementing custom types.
//
// Example usage:
//
//	RegisterDefaultType(
//		json.RawMessage{},
//		func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool) {
//			if field_v.IsValid() && field_v.Type() == reflect.TypeOf(json.RawMessage{}) {
//				return json.RawMessage{}, true
//			}
//			return nil, false
//		},
//	)
func RegisterDefaultType(valueOfType any, getDefault func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)) {
	var typ reflect.Type
	switch v := valueOfType.(type) {
	case reflect.Type:
		typ = v
	default:
		typ = reflect.TypeOf(valueOfType)
	}
	goldcrest.Register(DefaultForType, 100,
		DefaultGetter(func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool) {
			if field_v.IsValid() && field_v.Type() == typ || new_field_t_indirected == typ {
				return getDefault(f, new_field_t_indirected, field_v)
			}
			return nil, false
		}),
	)
}

// var valueInitMap = make(map[reflect.Type]func(instType reflect.Type, instField reflect.StructField, instValue reflect.Value) (reflect.Value, error))
