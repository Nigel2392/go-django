package attrs

import (
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

var (
	ErrEmptyString      = errors.New("empty string")
	ErrConvertingString = errors.New("error converting string to number")
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// DefinerList converts a slice of []T where the underlying type is of type Definer to []Definer.
func DefinerList[T Definer](list []T) []Definer {
	var n = len(list)
	if n == 0 {
		return nil
	}
	var l = make([]Definer, n)
	for i, v := range list {
		l[i] = v
	}
	return l
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
