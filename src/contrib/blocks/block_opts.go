package blocks

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type OptFunc[T any] func(T)

func runOpts[T1 any, T2 func(T1) | OptFunc[T1]](opts []T2, t T1) {
	for _, opt := range opts {
		opt(t)
	}
}

func WithValidators[T any](validators ...func(context.Context, interface{}) error) OptFunc[T] {
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

func reflectSetter[T Block](t T, fieldName string, value interface{}) bool {
	var field = reflect.ValueOf(t).Elem().FieldByName(fieldName)
	if field.IsValid() {
		field.Set(reflect.ValueOf(value))
		return true
	}
	return false
}

func WithMin[T Block](min int) OptFunc[T] {
	return func(t T) {
		if !reflectSetter(t, "Min", min) {
			panic(fmt.Errorf("Min is not a valid field of %T", t)) //lint:ignore ST1005 ignore this lint
		}
	}
}

func WithMax[T Block](max int) OptFunc[T] {
	return func(t T) {
		if !reflectSetter(t, "Max", max) {
			panic(fmt.Errorf("Max is not a valid field of %T", t)) //lint:ignore ST1005 ignore this lint
		}
	}
}

func WithBlockField[T Block](name string, block Block) OptFunc[T] {
	return func(t T) {
		var method, ok = attrs.Method[func(string, Block)](t, "AddField")
		if !ok {
			panic(fmt.Errorf("AddField is not a valid method of %T", t))
		}
		method(name, block)
	}
}
