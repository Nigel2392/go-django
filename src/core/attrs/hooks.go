package attrs

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/goldcrest"
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
