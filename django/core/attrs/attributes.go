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

func fieldNames(d Definer, exclude []string) []string {
	var excludeMap = make(map[string]struct{})
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var (
		fields = d.FieldDefs().Fields()
		n      = len(fields)
		names  = make([]string, 0, n)
	)

	for _, f := range fields {
		if _, ok := excludeMap[f.Name()]; ok {
			continue
		}
		names = append(names, f.Name())
	}

	return names
}

// A shortcut for getting the names of all fields in a Definer.
//
// The exclude parameter can be used to exclude certain fields from the result.
//
// This function is useful when you need to get the names of all fields in a
// model, but you want to exclude certain fields (e.g. fields that are not editable).
func FieldNames(d any, exclude []string) []string {
	if d == nil {
		return nil
	}
	if d, ok := d.(Definer); ok {
		return fieldNames(d, exclude)
	}
	var (
		rTyp       = reflect.TypeOf(d)
		excludeMap = make(map[string]struct{})
	)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var (
		n     = rTyp.NumField()
		names = make([]string, 0, n)
	)
	for i := 0; i < n; i++ {
		var f = rTyp.Field(i)
		if _, ok := excludeMap[f.Name]; ok {
			continue
		}
		names = append(names, f.Name)
	}

	return names
}

// SetMany sets multiple fields on a Definer.
//
// The values parameter is a map where the keys are the names of the fields to set.
//
// The values must be of the correct type for the fields.
func SetMany(d Definer, values map[string]interface{}) error {
	for name, value := range values {
		if err := assert.Err(set(d, name, value, false)); err != nil {
			return err
		}
	}
	return nil
}

// Set sets the value of a field on a Definer.
//
// If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.
//
// If the field is marked as non editable, this function will panic.
func Set(d Definer, name string, value interface{}) error {
	return set(d, name, value, false)
}

// ForceSet sets the value of a field on a Definer.
//
// If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.
//
// This function will allow setting the value of a field that is marked as not editable.
func ForceSet(d Definer, name string, value interface{}) error {
	return set(d, name, value, true)
}

// Get retrieves the value of a field on a Definer.
//
// If the field is not found, this function will panic.
//
// Type assertions are used to ensure that the value is of the correct type,
// as well as providing less work for the caller.
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

// ToString converts a value to a string.
//
// This should be the human-readable representation of the value.
func ToString(v any) string {
	if v == nil {
		return ""
	}

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

// Method retrieves a method from an object.
//
// The generic type parameter must be the type of the method.
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

// RConvert converts a reflect.Value to a different type.
//
// If the value is not convertible to the type, the original value is returned.
//
// If the pointer of `v` is invalid, a new value of type `t` is created, and the pointer is set to it, then the pointer is returned.
func RConvert(v *reflect.Value, t reflect.Type) (*reflect.Value, bool) {
	var original = *v
	if !v.IsValid() {
		var z = reflect.New(t)
		*v = z
		return v, true
	}
	if v.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr {
		*v = v.Elem()
	} else if v.Kind() != reflect.Ptr && t.Kind() == reflect.Ptr {
		var z = reflect.New(v.Type())
		z.Elem().Set(*v)
		*v = z
	}
	if v.Type().AssignableTo(t) {
		return v, true
	}
	if v.CanConvert(t) {
		*v = v.Convert(t)
		return v, true
	}
	*v = original
	return v, false
}

func rSet(src, dst *reflect.Value, isPointer bool) {
	if isPointer {
		if dst.IsZero() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst.Elem().Set(src.Elem())
	} else {
		dst.Set(*src)
	}
}

// RSet sets a value from one reflect.Value to another.
//
// If the destination value is not settable, this function will return false.
//
// If the source value is not immediately assignable to the destination value, and the convert parameter is true,
// the source value will be converted to the destination value's type.
//
// If the source value is not immediately assignable to the destination value, and the convert parameter is false,
// this function will return false.
func RSet(src, dst *reflect.Value, convert bool) (canset bool) {
	if !src.IsValid() || !dst.IsValid() {
		return false
	}
	if !dst.CanSet() {
		return false
	}
	var isPointer = dst.Kind() == reflect.Ptr
	var isImmediatelyAssignable = src.Type().AssignableTo(dst.Type())
	if isImmediatelyAssignable {
		rSet(src, dst, isPointer)
		return true
	}
	if convert {
		src, canset = RConvert(src, dst.Type())
		if !canset {
			return false
		}
	}
	rSet(src, dst, isPointer)
	return true
}
