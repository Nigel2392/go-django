package attrs

import (
	"reflect"

	"github.com/Nigel2392/django/forms/fields"
)

const (
	HookFormFieldForType = "attrs.FormFieldForType"
)

type FormFieldGetter func(f Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)
