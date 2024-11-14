package django_reflect

import "reflect"

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

	if v.CanConvert(t) {
		*v = v.Convert(t)
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
