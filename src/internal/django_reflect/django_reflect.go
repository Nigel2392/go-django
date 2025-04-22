package django_reflect

import "reflect"

// RConvert converts a reflect.Value to a different type.
//
// If the value is not convertible to the type, the original value is returned.
//
// If the pointer of `v` is invalid, a new value of type `t` is created, and the pointer is set to it, then the pointer is returned.
func RConvert(v *reflect.Value, t reflect.Type) (*reflect.Value, bool) {
	if !v.IsValid() {
		z := reflect.New(t)
		*v = z
		return v, true
	}

	if v.Type() == t {
		return v, true
	}

	// Handle pointer-to-value or value-to-pointer
	if v.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr {
		if v.IsNil() {
			*v = reflect.New(t).Elem()
		} else {
			*v = v.Elem()
		}
	} else if v.Kind() != reflect.Ptr && t.Kind() == reflect.Ptr {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(*v)
		*v = ptr
	}

	if v.Type().AssignableTo(t) || v.CanConvert(t) {
		*v = v.Convert(t)
		return v, true
	}

	return v, false
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
func RSet(src, dst *reflect.Value, convert bool) bool {
	if !src.IsValid() || !dst.IsValid() || !dst.CanSet() {
		return false
	}

	// Direct pointer assignment if types match
	if src.Type() == dst.Type() && src.Kind() == reflect.Ptr {
		dst.Set(*src)
		return true
	}

	if src.Type().AssignableTo(dst.Type()) {
		dst.Set(*src)
		return true
	}

	if convert {
		if conv, ok := RConvert(src, dst.Type()); ok {
			dst.Set(*conv)
			return true
		}
	}

	return false
}
