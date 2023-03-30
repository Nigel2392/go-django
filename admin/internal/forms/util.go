package forms

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/core"
	"github.com/Nigel2392/go-django/core/models/modelutils"
)

func getValue(mdl interface{}, fieldName string) string {
	var v = reflect.ValueOf(mdl)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var vInter = v.FieldByName(fieldName).Interface()
	return valueFromInterface(vInter)
}

// sliceUp creates a slice of type T from a slice of any type,
// using the struct field name to get the value
func sliceUp[T any](v reflect.Value, fieldName string) []T {
	var slc = make([]any, 0)
	for i := 0; i < v.Len(); i++ {
		var vinter = v.Index(i)
		if vinter.Kind() == reflect.Ptr {
			vinter = vinter.Elem()
		}
		var v = vinter.FieldByName(fieldName)
		if v.IsValid() {
			slc = append(slc, v.Interface())
		}
	}
	var typSlice = make([]T, len(slc))
	for i, v := range slc {
		typSlice[i] = v.(T)
	}
	return typSlice
}

func isNoneField(mdl any, field string) bool {
	var f, err = getField(mdl, field)
	if err != nil {
		return false
	}
	return f.IsZero() || f.Interface() == nil
}

func fieldValueEqual(m any, fieldname string, value string) bool {
	var f, err = getField(m, fieldname)
	if err != nil {
		return false
	}
	return valueFromInterface(f.Interface()) == value
}

func getField(m any, fieldname string) (reflect.Value, error) {
	var v = reflect.ValueOf(m)
	if !v.IsValid() {
		return reflect.Value{}, errors.New("invalid model")
	}
	v = modelutils.DePtr(v)
	var f = v.FieldByName(fieldname)
	if f.Kind() == reflect.Ptr {
		f = f.Elem()
	}
	if !f.IsValid() {
		return reflect.Value{}, errors.New("invalid field")
	}
	return f, nil
}

// Validate a model field.
//
// If the field implements FieldValidator, call its Validate method.
//
// Otherwise, return nil.
func validateModelField(field reflect.Value, v any) error {
	if field.Kind() == reflect.Ptr {
		field = field.Elem()
	}
	if field.IsValid() && field.CanInterface() {
		var fieldValidator, ok = field.Interface().(core.FieldValidator)
		if ok {
			if err := fieldValidator.Validate(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Validate if 2 reflect.Slices equal.
func slicesEqual(a, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}
	var canCompare = a.Comparable() && b.Comparable()
	for i := 0; i < a.Len(); i++ {
		var canInterface = a.Index(i).CanInterface() && b.Index(i).CanInterface()
		var isEqual = canCompare && canInterface && a.Equal(b.Index(i))
		if !isEqual {
			return false
		}
	}
	return true
}

// Get a form field by name
func (f *Form) getField(name string) *FormField {
	for _, field := range f.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

func valueFromInterface(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case time.Duration:
		return v.String()
	case fmt.Stringer:
		return v.String()
	}
	return ""
}
