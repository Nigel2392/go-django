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

// InterfaceList converts a slice of []T where the underlying type is of type Definer to []any.
func InterfaceList[T Definer](list []T) []any {
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
