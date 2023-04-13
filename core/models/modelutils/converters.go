package modelutils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/core"
)

func ConvertInt(s string) (any, error) {
	return strconv.Atoi(s)
}

func ConvertUint(s string) (any, error) {
	var i, err = strconv.Atoi(s)
	return uint(i), err
}

func ConvertFloat(s string) (any, error) {
	return strconv.ParseFloat(s, 64)
}

func ConvertBool(s string) (any, error) {
	return strconv.ParseBool(s)
}

func ConvertString(s string) (any, error) {
	return s, nil
}

func ConvertSlice[T any](s string) ([]T, error) {
	var sl = make([]T, 0)
	var parts = strings.Split(s, ";")
	for _, part := range parts {
		var v, err = Convert[T](part)
		if err != nil {
			return sl, err
		}
		sl = append(sl, v.(T))
	}
	return sl, nil
}

func ConvertMap[T1 comparable, T2 any](s1, s2 string) (any, any, error) {
	var (
		key any
		val any
		err error
	)

	key, err = Convert[T1](s1)
	if err != nil {
		return *(new(T1)), *(new(T2)), err
	}

	val, err = Convert[T2](s2)
	if err != nil {
		return *(new(T1)), *(new(T2)), err
	}

	return key.(T1), val.(T2), nil
}

func Convert[T any](s string) (any, error) {
	var v any
	var err error
	switch any(*(new(T))).(type) {
	case int, int8, int16, int32, int64:
		v, err = ConvertInt(s)
	case uint, uint8, uint16, uint32, uint64:
		v, err = ConvertUint(s)
	case float64, float32:
		v, err = ConvertFloat(s)
	case bool:
		v, err = ConvertBool(s)
	case string:
		v, err = ConvertString(s)
	case []byte:
		v = []byte(s)
	default:
		var typ = reflect.TypeOf(new(T)).Elem()
		if typ.Kind() == reflect.Slice {
			v, err = ConvertSlice[T](s)
		} else {
			var t = new(T)
			switch t := any(t).(type) {
			case core.FromStringer:
				if err := t.FromString(s); err != nil {
					return nil, err
				}
			default:
				err = fmt.Errorf("cannot convert %v automatically. please provide a Convert(string) (any, error) function", typ)
			}
		}
		//	else if typ.Kind() == reflect.Map {
		//		v, err = ConvertMap(s)
		//	}
	}
	return v, err
}

func ParseValue(v string, t reflect.Type) (any, error) {
	var val any
	var err error
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err = ConvertInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err = ConvertUint(v)
	case reflect.Float32, reflect.Float64:
		val, err = ConvertFloat(v)
	case reflect.Bool:
		val, err = ConvertBool(v)
	case reflect.String:
		val, err = ConvertString(v)
	}
	return val, err
}
