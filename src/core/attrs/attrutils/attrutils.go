package attrutils

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var (
	ErrEmptyString      = errors.New("empty string")
	ErrConvertingString = errors.New("error converting string to number")
)

func AttrFromMap[T any](attrMap map[string]any, attrName string) (T, bool, error) {
	var n T
	if v, ok := attrMap[attrName]; ok {
		if t, ok := v.(T); ok {
			return t, true, nil
		}

		var (
			rT = reflect.TypeOf((*T)(nil)).Elem()
			vT = reflect.TypeOf(v)
			vV = reflect.ValueOf(v)
		)

		if vT.AssignableTo(rT) {
			return vV.Interface().(T), true, nil
		}

		if vT.ConvertibleTo(rT) {
			return vV.Convert(rT).Interface().(T), true, nil
		}

		// If we reach here, the type is not convertible or assignable to T
		return n, false, fmt.Errorf(
			"expected type %T for attribute %q, but got %T",
			n, attrName, v,
		)
	}

	return n, false, nil
}

// CastToNumber converts a value of any type (int, float, string) to a number of type T.
// It returns the converted value and an error if the conversion fails.
func CastToNumber[T any](v any) (T, error) {
	var zero T
	var rv = reflect.ValueOf(v)
	var rt = reflect.TypeOf(zero)
	// if they passed a pointer to a number, grab the value
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.String {
		var str = rv.String()
		if str == "" {
			return zero, ErrEmptyString
		}
		rv, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return zero, errors.Join(
				ErrConvertingString, err,
			)
		}
		return CastToNumber[T](rv)
	}

	if !rv.Type().ConvertibleTo(rt) {
		panic(fmt.Sprintf("cannot convert %T to %s", v, rt))
	}

	cv := rv.Convert(rt)
	return cv.Interface().(T), nil
}

// InterfaceList converts a slice of []T to []any.
func InterfaceList[T any](list []T) []any {
	var n = len(list)
	if n == 0 {
		return nil
	}
	var l = make([]any, n)
	for i, v := range list {
		l[i] = v
	}
	return l
}
