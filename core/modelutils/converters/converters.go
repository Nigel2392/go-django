package converters

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Nigel2392/go-django/core/views/interfaces"
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
		sl = append(sl, v)
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

func Convert[T any](s string) (T, error) {
	var t = any(*(new(T)))
	var v T
	var err error
	// var wentDefault bool
	switch t.(type) {
	case int:
		var i int64
		i, err = strconv.ParseInt(s, 10, 64)
		v = *(*T)(unsafe.Pointer(&i))
	case int8:
		var i int64
		i, err = strconv.ParseInt(s, 10, 8)
		var i8 int8 = int8(i)
		v = *(*T)(unsafe.Pointer(&i8))
	case int16:
		var i int64
		i, err = strconv.ParseInt(s, 10, 16)
		var i16 int16 = int16(i)
		v = *(*T)(unsafe.Pointer(&i16))
	case int32:
		var i int64
		i, err = strconv.ParseInt(s, 10, 32)
		var i32 int32 = int32(i)
		v = *(*T)(unsafe.Pointer(&i32))
	case int64:
		var i int64
		i, err = strconv.ParseInt(s, 10, 64)
		v = *(*T)(unsafe.Pointer(&i))
	case uint:
		var i uint64
		i, err = strconv.ParseUint(s, 10, 64)
		v = *(*T)(unsafe.Pointer(&i))
	case uint8:
		var i uint64
		i, err = strconv.ParseUint(s, 10, 8)
		var i8 uint8 = uint8(i)
		v = *(*T)(unsafe.Pointer(&i8))
	case uint16:
		var i uint64
		i, err = strconv.ParseUint(s, 10, 16)
		var i16 uint16 = uint16(i)
		v = *(*T)(unsafe.Pointer(&i16))
	case uint32:
		var i uint64
		i, err = strconv.ParseUint(s, 10, 32)
		var i32 uint32 = uint32(i)
		v = *(*T)(unsafe.Pointer(&i32))
	case uint64:
		var i uint64
		i, err = strconv.ParseUint(s, 10, 64)
		v = *(*T)(unsafe.Pointer(&i))
	case float64:
		var f float64
		f, err = strconv.ParseFloat(s, 64)
		v = *(*T)(unsafe.Pointer(&f))
	case float32:
		var f float64
		f, err = strconv.ParseFloat(s, 32)
		var f32 float32 = float32(f)
		v = *(*T)(unsafe.Pointer(&f32))
	case bool:
		var b bool
		b, err = strconv.ParseBool(s)
		v = *(*T)(unsafe.Pointer(&b))
	case string:
		v = any(s).(T)
	case []byte:
		v = any([]byte(s)).(T)
	default:
		switch t := t.(type) {
		case interfaces.Field:
			var typeOfField = reflect.TypeOf(t)
			var newOf reflect.Value
			if typeOfField.Kind() == reflect.Ptr {
				typeOfField = typeOfField.Elem()
				newOf = reflect.New(typeOfField)
			} else {
				newOf = reflect.New(typeOfField).Elem()
			}
			t = newOf.Interface().(interfaces.Field)
			err = t.FormValues([]string{s})
			v = any(t).(T)
		default:
			err = fmt.Errorf("cannot convert %T automatically. this can be done implementing interfaces.Field on type %t", t, t)
		}
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
