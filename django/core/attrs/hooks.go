package attrs

import (
	"reflect"

	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

const (
	HookFormFieldForType = "attrs.FormFieldForType"
	DefaultForType       = "attrs.DefaultForType"
)

type FormFieldGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)
type DefaultGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)

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
