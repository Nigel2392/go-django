package attrs

import (
	"reflect"

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
func RegisterFormFieldType(valueOfType any, getField func(opts ...func(fields.Field)) fields.Field) {
	var typ reflect.Type
	switch v := valueOfType.(type) {
	case reflect.Type:
		typ = v
	default:
		typ = reflect.TypeOf(valueOfType)
	}
	goldcrest.Register(HookFormFieldForType, 100,
		FormFieldGetter(func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
			if field_v.IsValid() && field_v.Type() == typ || new_field_t_indirected == typ {
				return getField(opts...), true
			}
			return nil, false
		}),
	)
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
