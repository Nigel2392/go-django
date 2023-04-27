package modelutils

import (
	"errors"
	"reflect"
	"strings"
)

func GetFieldNames(m any) []string {
	var v = reflect.ValueOf(m)
	if !v.IsValid() {
		return nil
	}
	var t = v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var names = make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		var field = t.Field(i)
		if field.IsExported() {
			names = append(names, field.Name)
		}
	}
	return names
}

func GetField(m any, field string, strict bool) (any, error) {
	// fmt.Printf("GetField: %v, %v %T\n", m, field, m)
	var v = DePtr(reflect.ValueOf(m))
	if !v.IsValid() {
		// fmt.Println("GetField: ", v.Kind(), v.Type(), v.IsValid())
		return nil, errors.New("invalid value")
	}

	if strict {
		var field = v.FieldByName(field)
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), nil
		}
		return nil, errors.New("field not found")
	}

	// fmt.Println("GetField: ", v.Kind(), v.Type())
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == field && v.Type().Field(i).IsExported() {
			return v.Field(i).Interface(), nil
		}
		if v.Type().Field(i).IsExported() &&
			((strict && v.Type().Field(i).Name == field) ||
				(!strict && strings.EqualFold(v.Type().Field(i).Name, field))) {
			var f = v.Field(i)
			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
			if !f.IsValid() || !f.CanInterface() {
				return nil, errors.New("field not found")
			}
			return f.Interface(), nil
		}
		var f = v.Field(i)
		f = DePtr(f)
		if f.Kind() == reflect.Struct {
			if !f.IsValid() || !f.CanInterface() {
				continue
			}
			var v, err = GetField(f.Interface(), field, false)
			if err == nil {
				return v, nil
			}
		}
	}
	return nil, errors.New("field not found")
}

func SetField(m any, field string, value any) error {
	var v = reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		if strings.EqualFold(v.Type().Field(i).Name, field) {
			var f = v.Field(i)
			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
			switch f.Kind() {
			case reflect.String:
				f.SetString(value.(string))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				f.SetInt(value.(int64))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				f.SetUint(value.(uint64))
			case reflect.Float32, reflect.Float64:
				f.SetFloat(value.(float64))
			case reflect.Bool:
				f.SetBool(value.(bool))
			}
			return nil
		}
	}
	return errors.New("field not found")
}

// isPtr reports whether v represents a pointer.
// It returns false if v is the zero Value.
func isPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr || v.Kind() == reflect.Pointer
}

// isVld reports whether v represents a value.
// It returns false if v is the zero Value.
func isVld(v reflect.Value) bool {
	return v.IsValid()
}

// Convert a potential pointer to a value to a value
func DePtr(val any) reflect.Value {
	switch val := val.(type) {
	case reflect.Value:
		if val.IsValid() && val.IsZero() {
			var v = reflect.New(val.Type())
			val = v.Elem()
		}

		if isPtr(val) && isVld(val) {
			val = val.Elem()
		}

		if isPtr(val) && isVld(val) {
			return DePtr(val)
		}

		return val
	case reflect.Type:
		return DePtr(reflect.New(val).Elem())
	case reflect.StructField:
		return DePtr(reflect.New(val.Type).Elem())
	}
	var v = reflect.ValueOf(val)
	return DePtr(v)
}

func DePtrType(val any) reflect.Type {
	switch v := val.(type) {
	case reflect.Type:
		if v.Kind() == reflect.Ptr || v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		if v.Kind() == reflect.Ptr || v.Kind() == reflect.Pointer {
			return DePtrType(v)
		}
		return v
	case reflect.Value:
		return DePtrType(v.Type())
	case reflect.StructField:
		return DePtrType(v.Type)
	default:
		return DePtrType(reflect.TypeOf(val))
	}
}

// Get a pointer to a new value of the same type as val
func NewPtr(val any) reflect.Value {
	var v = reflect.TypeOf(val)
	v = DePtrType(v)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	return reflect.New(v)
}
