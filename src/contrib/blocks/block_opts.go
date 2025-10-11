package blocks

import (
	"reflect"
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

func WithLabel[T Block](label any) OptFunc[T] {
	return func(t T) {
		t.SetLabel(label)
	}
}

func WithHelpText[T Block](text any) OptFunc[T] {
	return func(t T) {
		t.SetHelpText(text)
	}
}

func WithDefault[T Block](def interface{}) OptFunc[T] {
	return func(t T) {
		t.SetDefault(def)
	}
}
