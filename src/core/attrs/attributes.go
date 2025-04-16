package attrs

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
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

// SetPrimaryKey sets the primary key field of a Definer.
//
// If the primary key field is not found, this function will panic.
func SetPrimaryKey(d Definer, value interface{}) error {
	var f = d.FieldDefs().Primary()
	if f == nil {
		assert.Fail(
			"primary key not found in %T",
			d,
		)
	}

	return f.SetValue(value, false)
}

// PrimaryKey returns the primary key field of a Definer.
//
// If the primary key field is not found, this function will panic.
func PrimaryKey(d Definer) interface{} {
	var f = d.FieldDefs().Primary()
	if f == nil {
		assert.Fail(
			"primary key not found in %T",
			d,
		)
	}
	var (
		v  = f.GetValue()
		rT = reflect.TypeOf(v)
		rV = reflect.ValueOf(v)
	)

	if rT.Kind() == reflect.Ptr {
		rT = rT.Elem()
		rV = rV.Elem()
	}

	switch rT.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(rV.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rV.Uint()
	case reflect.String:
		return rV.String()
	default:

		switch v := v.(type) {
		case fmt.Stringer:
			return v.String()
		}

		assert.Fail(
			"primary key %T is not of type int, uint or string",
			v,
		)
	}
	return nil
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
		var fieldNames = fieldNames(d, nil)
		return assert.Fail(
			fmt.Sprintf("set (%T): no field named %q in %+v", d, name, fieldNames),
		)
	}

	return f.SetValue(value, force)
}
