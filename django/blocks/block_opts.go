package blocks

import "reflect"

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

func WithLabel[T any](label string) OptFunc[T] {
	return func(t T) {
		reflectSetter(t, "LabelFunc", func() string { return label })
	}
}

func WithHelpText[T any](text string) OptFunc[T] {
	return func(t T) {
		reflectSetter(t, "HelpFunc", func() string { return text })
	}
}
