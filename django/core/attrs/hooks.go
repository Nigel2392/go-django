package attrs

import (
	"reflect"

	"github.com/Nigel2392/django/forms/fields"
)

const (
	HookFormFieldForType = "attrs.FormFieldForType"
	DefaultForType       = "attrs.DefaultForType"
)

type FormFieldGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)
type DefaultGetter func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)
