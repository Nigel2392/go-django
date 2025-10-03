package blocks

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/trans"
)

type OptFunc[T any] func(T)

func runOpts[T1 any, T2 func(T1) | OptFunc[T1]](opts []T2, t T1) {
	for _, opt := range opts {
		opt(t)
	}
}

func WithValidators[T any](validators ...func(interface{}) error) OptFunc[T] {
	return func(t T) {
		var validatorField = reflect.ValueOf(t).Elem().FieldByName("Validators")
		if validatorField.IsValid() {
			for _, validator := range validators {
				validatorField.Set(reflect.Append(validatorField, reflect.ValueOf(validator)))
			}
		}
	}
}

func reflectSetter[T any](t T, fieldName string, value interface{}) {
	var field = reflect.ValueOf(t).Elem().FieldByName(fieldName)
	if field.IsValid() {
		field.Set(reflect.ValueOf(value))
	}
}

func WithLabel[T any](label any) OptFunc[T] {
	return func(t T) {
		reflectSetter(t, "LabelFunc", func(ctx context.Context) string {
			var label, ok = trans.GetText(ctx, label)
			if !ok {
				return ""
			}
			return label
		})
	}
}

func WithHelpText[T any](text any) OptFunc[T] {
	return func(t T) {
		reflectSetter(t, "HelpFunc", func(ctx context.Context) string {
			var label, ok = trans.GetText(ctx, text)
			if !ok {
				return ""
			}
			return label
		})
	}
}
