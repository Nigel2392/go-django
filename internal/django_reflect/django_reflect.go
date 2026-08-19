package django_reflect

import (
	re "reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

func init() {
	for k, v := range unsafeKinds {
		for k2, rev := range v {
			if !rev {
				continue
			}

			if _, ok := unsafeKinds[k2]; !ok {
				unsafeKinds[k2] = make(map[re.Kind]bool)
			}
			unsafeKinds[k2][k] = true
		}
	}
}

var unsafeKinds = map[re.Kind]map[re.Kind]bool{
	re.String: {
		re.Int:    true,
		re.Int8:   true,
		re.Int16:  true,
		re.Int32:  true,
		re.Int64:  true,
		re.Uint:   true,
		re.Uint8:  true,
		re.Uint16: true,
		re.Uint32: true,
		re.Uint64: true,
	},
	re.Slice: {
		re.Array: false,
	},
}

func isSafeConversion(from, to re.Type) bool {
	if from.Kind() == to.Kind() {
		return true
	}
	if unsafeTo, ok := unsafeKinds[from.Kind()]; ok {
		if _, ok := unsafeTo[to.Kind()]; ok {
			return false
		}
	}
	return true
}

func ConvertToType(value re.Value, targetType re.Type) (re.Value, error) {
	var assignableToParam = value.Type().AssignableTo(targetType)
	var convertibleToParam = isSafeConversion(value.Type(), targetType) && value.Type().ConvertibleTo(targetType)
	if !assignableToParam && !convertibleToParam {
		return re.Value{}, errors.TypeMismatch.Wrapf(
			"cannot convert type %s to %s",
			value.Type(), targetType,
		)
	}

	if convertibleToParam && !assignableToParam {
		value = value.Convert(targetType)
	}

	if !value.Type().AssignableTo(targetType) {
		return re.Value{}, errors.TypeMismatch.Wrapf(
			"cannot assign type %s to %s",
			value.Type(), targetType,
		)
	}

	return value, nil
}

// RConvert converts a re.Value to a different type.
//
// If the value is not convertible to the type, the original value is returned.
//
// If the pointer of `v` is invalid, a new value of type `t` is created, and the pointer is set to it, then the pointer is returned.
func RConvert(v *re.Value, t re.Type) (*re.Value, bool) {
	if !v.IsValid() {
		z := re.New(t)
		*v = z
		return v, true
	}

	if v.Type() == t {
		return v, true
	}

	if t.Kind() == re.Interface && v.Type().Implements(t) {
		// return the value as an interface
		var z = re.New(t)
		z.Elem().Set(*v)
		*v = z.Elem()
		return v, true
	}

	// Handle pointer-to-value or value-to-pointer
	if v.Kind() == re.Ptr && t.Kind() != re.Ptr {
		if v.IsNil() {
			*v = re.New(t).Elem()
		} else {
			*v = v.Elem()
		}
	} else if v.Kind() != re.Ptr && t.Kind() == re.Ptr {
		ptr := re.New(v.Type())
		ptr.Elem().Set(*v)
		*v = ptr
	}

	if v.Type().AssignableTo(t) || v.CanConvert(t) {
		*v = v.Convert(t)
		return v, true
	}

	return v, false
}

// RSet sets a value from one re.Value to another.
//
// If the destination value is not settable, this function will return false.
//
// If the source value is not immediately assignable to the destination value, and the convert parameter is true,
// the source value will be converted to the destination value's type.
//
// If the source value is not immediately assignable to the destination value, and the convert parameter is false,
// this function will return false.
func RSet(src, dst *re.Value, convert bool) bool {
	if !src.IsValid() || !dst.IsValid() || !dst.CanSet() {
		return false
	}

	// Direct pointer assignment if types match
	if src.Type() == dst.Type() && src.Kind() == re.Ptr {
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

type IsZeroer interface {
	IsZero() bool
}

var _isZeroerType = re.TypeOf((*IsZeroer)(nil)).Elem()

func ReflectValue(value interface{}) re.Value {
	var rv re.Value
	switch v := value.(type) {
	case re.Value:
		rv = v
	case *re.Value:
		if v == nil {
			return re.Value{}
		}
		rv = *v
	default:
		rv = re.ValueOf(value)
	}
	return rv
}

func IsZero(value interface{}) bool {
	if value == nil {
		return true
	}

	if zeroer, ok := value.(IsZeroer); ok {
		return zeroer.IsZero()
	}

	var rv = ReflectValue(value)
	if !rv.IsValid() {
		return true
	}

	if rv.Kind() == re.Interface && rv.IsNil() {
		return true
	}

	if rv.Kind() == re.Ptr && rv.IsNil() {
		return true
	}

	// check if either the pointer to the value or the value itself implements isZeroer
	if rv.Type().Implements(_isZeroerType) {
		return rv.Interface().(IsZeroer).IsZero()
	} else if rv.Kind() == re.Ptr && !rv.IsNil() && rv.Elem().Type().Implements(_isZeroerType) {
		return rv.Elem().Interface().(IsZeroer).IsZero()
	}

	switch rv.Kind() {
	case re.Bool:
		return !rv.Bool()
	case re.Int, re.Int8, re.Int16, re.Int32, re.Int64:
		return rv.Int() == 0
	case re.Uint, re.Uint8, re.Uint16, re.Uint32, re.Uint64, re.Uintptr:
		return rv.Uint() == 0
	case re.Float32, re.Float64:
		return rv.Float() == 0
	case re.Complex64, re.Complex128:
		return rv.Complex() == 0
	case re.Ptr:
		if !rv.IsValid() || rv.IsNil() {
			return true
		}
		return IsZero(rv.Elem().Interface())
	case re.String:
		return rv.String() == ""
	case re.Slice, re.Array:
		if rv.Len() == 0 {
			return true
		}

		for i := 0; i < rv.Len(); i++ {
			if !IsZero(rv.Index(i).Interface()) {
				return false
			}
		}
	case re.Map:
		return rv.Len() == 0
	}

	return re.DeepEqual(rv.Interface(), re.Zero(rv.Type()).Interface())
}
