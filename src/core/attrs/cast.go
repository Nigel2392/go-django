package attrs

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type (
	ToIntConverter interface {
		ToInt() int64
	}
	ToStringConverter interface {
		ToString() string
	}
	ToFloatConverter interface {
		ToFloat() float64
	}
	ToBoolConverter interface {
		ToBool() bool
	}
	ToTimeConverter interface {
		ToTime() time.Time
	}
)

// ToString converts a value to a string.
//
// This should be the human-readable representation of the value.
//
// If the value is a struct with a content type, it will use the content type's InstanceLabel method to convert it to a string.
//
// time.Time, mail.Address, and error types are handled specially.
//
// If the value is a slice or array, it will convert each element to a string and join them with ", ".
//
// If all else fails, it will use fmt.Sprintf to convert the value to a string.
func ToString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	var r = reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	if r.Kind() == reflect.Struct {
		var cType = contenttypes.DefinitionForObject(
			v,
		)
		if cType != nil {
			return cType.InstanceLabel(v)
		}
	}

	return toString(r, v)
}

func toString(r reflect.Value, v any) string {
	switch v := v.(type) {
	case ToStringConverter:
		return v.ToString()
	case *mail.Address:
		return v.Address
	case time.Time:
		return v.Format(time.RFC3339)
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
	}

	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	switch r.Kind() {
	case reflect.String:
		return r.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(r.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(r.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(r.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(r.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(r.Bool())
	case reflect.Slice, reflect.Array:
		var b = make([]string, r.Len())
		for i := 0; i < r.Len(); i++ {
			b[i] = ToString(r.Index(i).Interface())
		}
		return strings.Join(b, ", ")
	case reflect.Struct:
		var cType = contenttypes.DefinitionForObject(
			v,
		)
		if cType != nil {
			return cType.InstanceLabel(v)
		}
	}

	return fmt.Sprintf("%v", v)
}

func ToInt(v any) (int64, error) {
	if v == nil {
		return 0, nil
	}

	if toInter, ok := v.(ToIntConverter); ok {
		return toInter.ToInt(), nil
	}

	var r = reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(r.Uint()), nil
	case reflect.Float32:
		return int64(r.Float()), nil
	case reflect.Float64:
		return int64(r.Float()), nil
	case reflect.Bool:
		if r.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseInt(r.String(), 10, 64)
	}

	if def, ok := v.(Definer); ok {
		var fieldDefs = def.FieldDefs()
		var prim = fieldDefs.Primary()
		var val, err = prim.Value()
		if err != nil {
			return 0, err
		}
		i, err := ToInt(val)
		return i, err
	}

	return 0, errors.TypeMismatch.Wrapf(
		"cannot convert %T to int64", v,
	)
}

func ToFloat(v any) (float64, error) {
	if v == nil {
		return 0, nil
	}

	if toFloat, ok := v.(ToFloatConverter); ok {
		return toFloat.ToFloat(), nil
	}

	var r = reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Float32:
		return float64(r.Float()), nil
	case reflect.Float64:
		return r.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(r.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(r.Uint()), nil
	case reflect.Bool:
		if r.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseFloat(r.String(), 64)
	}

	if def, ok := v.(Definer); ok {
		var fieldDefs = def.FieldDefs()
		var prim = fieldDefs.Primary()
		var val, err = prim.Value()
		if err != nil {
			return 0, err
		}
		f, err := ToFloat(val)
		return f, err
	}

	return 0, errors.TypeMismatch.Wrapf(
		"cannot convert %T to float64", v,
	)
}

func ToBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}

	if toBool, ok := v.(ToBoolConverter); ok {
		return toBool.ToBool(), nil
	}

	var r = reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Bool:
		return r.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Int() != 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Uint() != 0, nil
	case reflect.String:
		return strconv.ParseBool(r.String())
	}

	if def, ok := v.(Definer); ok {
		var fieldDefs = def.FieldDefs()
		var prim = fieldDefs.Primary()
		var val, err = prim.Value()
		if err != nil {
			return false, err
		}
		b, err := ToBool(val)
		return b, err
	}

	return IsZero(v), nil
}

func ToTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}

	if toTime, ok := v.(ToTimeConverter); ok {
		return toTime.ToTime(), nil
	}

	if toTime, ok := v.(interface{ Time() time.Time }); ok {
		return toTime.Time(), nil
	}

	var r = reflect.ValueOf(v)
	var timeTyp = reflect.TypeOf(time.Time{})
	if r.Type() == timeTyp {
		return r.Interface().(time.Time), nil
	}

	if r.Type().ConvertibleTo(timeTyp) {
		r = r.Convert(timeTyp)
		return r.Interface().(time.Time), nil
	}

	switch r.Kind() {
	case reflect.String:
		return time.Parse(time.RFC3339, r.String())
	case reflect.Int, reflect.Int64:
		return time.Unix(r.Int(), 0), nil
	}

	return time.Time{}, errors.TypeMismatch.Wrapf(
		"cannot convert %T to time.Time", v,
	)
}
