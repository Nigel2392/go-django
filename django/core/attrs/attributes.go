package attrs

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/django/core/assert"
)

func SetMany(d Definer, values map[string]interface{}) error {
	for name, value := range values {
		if err := assert.Err(set(d, name, value, false)); err != nil {
			return err
		}
	}
	return nil
}

func Set(d Definer, name string, value interface{}) error {
	return set(d, name, value, false)
}

func ForceSet(d Definer, name string, value interface{}) error {
	return set(d, name, value, true)
}

func Get[T any](d Definer, name string) T {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {

		var method, ok = Method[T](d, name)
		if ok {
			return method
		}

		assert.Fail(
			"get (%T): no field named %q",
			d, name,
		)
	}

	var v = f.GetValue()
	switch t := v.(type) {
	case T:
		return t
	case *T:
		return *t
	default:
		assert.Fail(
			"get (%T): field %q is not of type %T",
			d, name, v,
		)
	}
	return *(new(T))
}

func ToString(v any) string {
	if v == nil {
		return ""
	}

	//switch v := v.(type) {
	//case Stringer:
	//	return v.ToString()
	//case fmt.Stringer:
	//	return v.String()
	//}

	if stringer, ok := v.(Stringer); ok {
		return stringer.ToString()
	}

	return toString(v)
}

func toString(v any) string {
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

	var r = reflect.ValueOf(v)
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
	}

	return fmt.Sprintf("%v", v)
}

func Method[T any](obj interface{}, name string) (n T, ok bool) {
	if obj == nil {
		return n, false
	}

	var (
		v = reflect.ValueOf(obj)
		m = v.MethodByName(name)
	)
checkValid:
	if !m.IsValid() {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			goto checkValid
		}
		return n, false
	}

	var i = m.Interface()
	if i == nil {
		return n, false
	}

	n, ok = i.(T)
	return n, ok
}

func set(d Definer, name string, value interface{}, force bool) error {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		return assert.Fail(
			fmt.Sprintf("set (%T): no field named %q", d, name),
		)
	}

	return f.SetValue(value, force)
}
