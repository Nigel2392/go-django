package attrs

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type ToStringConverter interface {
	ToString() string
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
