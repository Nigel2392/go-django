package django_reflect

import (
	"bytes"
	"database/sql/driver"
	"reflect"
	re "reflect"

	"github.com/Nigel2392/go-django/internal/bitch"
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

type isZeroer interface {
	IsZero() bool
}

var _isZeroerType = re.TypeOf((*isZeroer)(nil)).Elem()

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
		return rv.Interface().(isZeroer).IsZero()
	} else if rv.Kind() == re.Ptr && !rv.IsNil() && rv.Elem().Type().Implements(_isZeroerType) {
		return rv.Elem().Interface().(isZeroer).IsZero()
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

type FLAG_EQ = bitch.Flag

const (
	EQ_NONE ScanFlag = 0

	// convert x and y to driver.Value
	EQ_DRIVER_VALUE ScanFlag = 1 << iota

	// check for IsZero method on both
	// special case when x and y are of kinds Array, Slice, Map, Chan:
	// 	 x == nil || y == nil is ignored and only length is checked.
	// 	 go psuedocode: len(x) == 0 && len(y) == 0
	EQ_ZEROS

	// if x is a pointer and y isn't, dereference x and vice versa
	//   psuedocode: (*a == b || a == *b) -> a == b
	EQ_IGNORE_PTR

	/*
		ignores the first check performed by [equals], this /CAN/ be useful in certain situations
		for example, the following struct is defined:

		```
			type myStruct{value int}

			func (m *mystruct) Equals(other any) bool {
				// ...
				if o, ok := other.(*myStruct); ok {
					if m == nil || other == nil {
						return m == nil && (o == nil || o.value == 0) || o == nil && (m == nil || m.value == 0)
					}

					return m.value == o.value
				}
				// ...
				return false
			}
		```

		Would mean the following call is true:

		```
			Equals((*mystruct)(nil), &myStruct{value: 0}, EQ_IGNORE_NIL) == true
		```
	*/
	EQ_IGNORE_NIL

	// auto-convert types
	//   psuedocode: int8(5) == int64(5)
	EQ_TYPE_CONVERT

	// include all above conversions
	EQ_DFLT = EQ_DRIVER_VALUE |
		EQ_ZEROS |
		EQ_IGNORE_PTR |
		EQ_TYPE_CONVERT
)

type EqualityChecker interface {
	Equal(other any) bool
}

type EqualityChecker2 interface {
	Equals(other any) bool
}

func equalityCheckerEquals(a, b any) (eq bool, ok bool) {
	if e, ok := a.(EqualityChecker); ok {
		return e.Equal(b), true
	}

	if e, ok := b.(EqualityChecker); ok {
		return e.Equal(a), true
	}
	if e, ok := a.(EqualityChecker2); ok {
		return e.Equals(b), true
	}

	if e, ok := b.(EqualityChecker2); ok {
		return e.Equals(a), true
	}

	return false, false
}

// Equals checks if a is equal to be.
//
// This does not nescessarily mean a == b.
func Equals(a, b any, flags ...FLAG_EQ) bool {
	var opts FLAG_EQ
	switch len(flags) {
	case 0:
	case 1:
		opts = flags[0]
	default:
		for _, f := range flags {
			opts |= f
		}
	}

	return equals(a, b, opts)
}

type EqStepState struct {
	A, B   any
	V1, V2 reflect.Value
	Opts   FLAG_EQ
}

type equalsStep = func(*EqStepState) (eq bool, ok bool)

var (
	_ equalsStep = stepIsZeroer
	_ equalsStep = stepCmpDriverValue
	_ equalsStep = stepCmpBytesRunes
	_ equalsStep = stepDerefPointers
)

func stepIsZeroer(state *EqStepState) (eq bool, ok bool) {
	if a, ok := state.A.(isZeroer); ok {
		if b, ok := state.B.(isZeroer); ok {
			return (a == nil || a.IsZero()) == (b == nil || b.IsZero()), true
		}
	}
	return false, false
}

func stepCmpDriverValue(state *EqStepState) (eq bool, ok bool) {
	var dv bool

	// get [a] driver.Value
	if e, ok := state.A.(driver.Valuer); ok && state.A != nil {
		nA, err := e.Value()
		if err != nil {
			return false, false
		}

		dv = true
		state.A = nA
	}

	// get [b] driver.Value
	if e, ok := state.B.(driver.Valuer); ok && state.B != nil {
		nB, err := e.Value()
		if err != nil {
			return false, false
		}

		dv = true
		state.B = nB
	}

	if dv {
		// compare again if EITHER was converted to value
		return equals(state.A, state.B, state.Opts), true
	}

	return false, false
}

func stepCmpBytesRunes(state *EqStepState) (eq bool, ok bool) {
	// compare byte and rune slice types
	if !(state.V1.Kind() == re.Slice && state.V2.Kind() == re.Slice) {
		return false, false
	}

	v1TE := state.V1.Type().Elem()
	v2TE := state.V2.Type().Elem()

	// Womp womp.
	switch {
	case (v1TE.Kind() == re.Uint8 || v1TE.Kind() == re.Uint32) &&
		(v2TE.Kind() == re.Uint8 || v2TE.Kind() == re.Uint32) &&
		(state.V1.IsNil() || state.V2.IsNil()):

		// EQ_ZEROS will not matter for byte or rune slices.
		return state.V1.IsNil() == state.V2.IsNil(), true

	case v1TE.Kind() == re.Uint8 && v2TE.Kind() == re.Uint8:
		// []byte == []byte
		return bytes.Equal(state.V1.Bytes(), state.V2.Bytes()), true

	case v1TE.Kind() == re.Uint32 && v2TE.Kind() == re.Uint32:
		// []rune == []rune
		return string(*(*[]rune)(state.V1.UnsafePointer())) == string(*(*[]rune)(state.V2.UnsafePointer())), true

	case v1TE.Kind() == re.Uint8 && v2TE.Kind() == re.Uint32:
		// []byte == []rune
		return string(state.V1.Bytes()) == string(*(*[]rune)(state.V2.UnsafePointer())), true

	case v2TE.Kind() == re.Uint8 && v1TE.Kind() == re.Uint32:
		// []rune == []byte
		return string(*(*[]rune)(state.V1.UnsafePointer())) == string(state.V1.Bytes()), true
	}

	return false, false
}

func stepDerefPointers(state *EqStepState) (eq bool, retEq bool) {
	for state.V1.Kind() == re.Pointer || state.V2.Kind() == re.Pointer {

		var (
			v1Nil = state.V1.Kind() == re.Pointer && state.V1.IsNil()
			v2Nil = state.V2.Kind() == re.Pointer && state.V2.IsNil()
		)

		if state.V1.Kind() == re.Pointer && !v1Nil {
			state.V1 = state.V1.Elem()
		}
		if state.V2.Kind() == re.Pointer && !v2Nil {
			state.V2 = state.V2.Elem()
		}

		if v1Nil || v2Nil {
			break
		}
	}

	return false, false
}

func equals(a, b any, opts FLAG_EQ) bool {
	// cant ignore this one!
	if a == nil && b == nil {
		return true
	}

	if !bitch.Is(opts, EQ_IGNORE_NIL) && (a == nil || b == nil) {
		return (a == nil) == (b == nil)
	}

	// a.Equals(b) || b.Equals(a)
	var eq, ok = equalityCheckerEquals(a, b)
	if ok {
		return eq
	}

	// a.IsZero() == b.IsZero()
	if bitch.Is(opts, EQ_ZEROS) {
		if a, ok := a.(isZeroer); ok {
			if b, ok := b.(isZeroer); ok {
				return (a == nil || a.IsZero()) == (b == nil || b.IsZero())
			}
		}
	}

	// Equals(a.Value(), b.Value(), opts)
	if bitch.Is(opts, EQ_DRIVER_VALUE) {
		var dv bool

		// get [a] driver.Value
		if e, ok := a.(driver.Valuer); ok && a != nil {
			nA, err := e.Value()
			if err != nil {
				goto reflectCompare
			}

			dv = true
			a = nA
		}

		// get [b] driver.Value
		if e, ok := b.(driver.Valuer); ok && b != nil {
			nB, err := e.Value()
			if err != nil {
				goto reflectCompare
			}

			dv = true
			b = nB
		}

		if dv {
			// compare again if EITHER was converted to value
			return equals(a, b, opts)
		}
	}

	//	if aB, ok := a.([]byte); ok && a != nil {
	//		a = string(aB)
	//	}
	//
	//	if bB, ok := b.([]byte); ok && b != nil {
	//		b = string(bB)
	//	}
	//
	//	if aB, ok := a.([]rune); ok && a != nil {
	//		a = string(aB)
	//	}
	//
	//	if bB, ok := b.([]rune); ok && b != nil {
	//		b = string(bB)
	//	}

reflectCompare:
	var (
		v1 = re.ValueOf(a)
		v2 = re.ValueOf(b)
	)

	if v1.Comparable() && v2.Comparable() && a == b {
		return true
	}

	// compare byte and rune slice types
	if v1.Kind() == re.Slice && v2.Kind() == re.Slice {
		v1TE := v1.Type().Elem()
		v2TE := v2.Type().Elem()

		// Womp womp.
		switch {
		case (v1TE.Kind() == re.Uint8 || v1TE.Kind() == re.Uint32) &&
			(v2TE.Kind() == re.Uint8 || v2TE.Kind() == re.Uint32) &&
			(v1.IsNil() || v2.IsNil()):

			// EQ_ZEROS will not matter for byte or rune slices.
			return v1.IsNil() == v2.IsNil()

		case v1TE.Kind() == re.Uint8 && v2TE.Kind() == re.Uint8:
			// []byte == []byte
			return bytes.Equal(v1.Bytes(), v2.Bytes())

		case v1TE.Kind() == re.Uint32 && v2TE.Kind() == re.Uint32:
			// []rune == []rune
			return string(*(*[]rune)(v1.UnsafePointer())) == string(*(*[]rune)(v2.UnsafePointer()))

		case v1TE.Kind() == re.Uint8 && v2TE.Kind() == re.Uint32:
			// []byte == []rune
			return string(v1.Bytes()) == string(*(*[]rune)(v2.UnsafePointer()))

		case v2TE.Kind() == re.Uint8 && v1TE.Kind() == re.Uint32:
			// []rune == []byte
			return string(*(*[]rune)(v1.UnsafePointer())) == string(v1.Bytes())
		}
	}

	// (*a == b || a == *b) -> a == b
	for bitch.Is(opts, EQ_IGNORE_PTR) && (v1.Kind() == re.Pointer || v2.Kind() == re.Pointer) {

		var (
			v1Nil = v1.Kind() == re.Pointer && v1.IsNil()
			v2Nil = v2.Kind() == re.Pointer && v2.IsNil()
		)

		if v1.Kind() == re.Pointer && !v1Nil {
			v1 = v1.Elem()
		}
		if v2.Kind() == re.Pointer && !v2Nil {
			v2 = v2.Elem()
		}

		if v1Nil || v2Nil {
			break
		}
	}

	switch {
	case v1.Kind() == re.Invalid && v2.Kind() == re.Invalid:
		// this shouldnt really happen due to the initial nil == nil check
		return true

	case v1.Kind() == re.Invalid || v2.Kind() == re.Invalid:
		z, ok := _eq__nilLenKindIsZero(opts, v1, v2)
		if ok {
			return z
		}

		if v1.Kind() != re.Invalid && _eq__canNil(v1.Kind()) {
			return v1.IsNil() == (v2.Kind() == re.Invalid)
		}

		if v2.Kind() != re.Invalid && _eq__canNil(v2.Kind()) {
			return v2.IsNil() == (v1.Kind() == re.Invalid)
		}

		if !bitch.Is(opts, EQ_ZEROS) {
			return false
		}

		// i.e. 0 == nil == true
		return (v1.Kind() != re.Invalid && v1.IsZero() == (v2.Kind() == re.Invalid)) ||
			(v2.Kind() != re.Invalid && v2.IsZero() == (v1.Kind() == re.Invalid))
	}

	var (
		v1T = v1.Type()
		v2T = v2.Type()
	)
	if v1T != v2T {
		// dont convert types, return
		if !bitch.Is(opts, EQ_TYPE_CONVERT) {
			return false
		}

		// convert types (example, int8 -> int64)
		if v1.Kind() != v2.Kind() {
			switch {
			case isSafeConversion(v1T, v2T) && v1T.ConvertibleTo(v2T):
				v1 = v1.Convert(v2T)
				v1T = v2T
			case isSafeConversion(v2T, v1T) && v2T.ConvertibleTo(v1T):
				v2 = v2.Convert(v1T)
				v2T = v1T
			}
		}
	}

	// no type mismatch possible unless convert flag is set, handled above
	// compare underlying values (drivers.String("str") == string("str"))
	if v1.Kind() == v2.Kind() {
		switch v1.Kind() {
		case re.String:
			return v1.String() == v2.String()
		case re.Int, re.Int8, re.Int16, re.Int32, re.Int64:
			return v1.Int() == v2.Int()
		case re.Uint, re.Uint8, re.Uint16, re.Uint32, re.Uint64, re.Uintptr:
			return v1.Uint() == v2.Uint()
		case re.Bool:
			return v1.Bool() == v2.Bool()
		case re.Float32, re.Float64:
			return v1.Float() == v2.Float()
		case re.Complex64, re.Complex128:
			return v1.Complex() == v2.Complex()
		}
	}

	if isZero, ok := _eq__nilLenKindIsZero(opts, v1, v2); ok {
		return isZero
	}

	return re.DeepEqual(v1.Interface(), v2.Interface())
}

func _eq__nilLenKindIsZero(opts FLAG_EQ, a, b re.Value) (bothZero, ok bool) {
	if !bitch.Is(opts, EQ_ZEROS) {
		return false, false
	}

	if !_eq__nilLenKind(a.Kind(), b.Kind()) {
		return false, false
	}

	bothZero = (a.Kind() == re.Invalid || (a.Kind() != re.Array && a.IsNil()) || a.Len() == 0) &&
		(b.Kind() == re.Invalid || (b.Kind() != re.Array && b.IsNil()) || b.Len() == 0)

	return bothZero, true
}

func _eq__nilLenKind(a, b re.Kind) (yes bool) {
	yes = _eq__isNilLenKind(a) || _eq__isNilLenKind(b)
	return yes && a == b || (a == re.Array || a == re.Slice) && (b == re.Array || b == re.Slice)
}

func _eq__isNilLenKind(k re.Kind) bool {
	return (k == re.Array ||
		k == re.Slice ||
		k == re.Map ||
		k == re.Chan)
}

func _eq__canNil(k re.Kind) bool {
	return (k == re.Slice ||
		k == re.Chan ||
		k == re.Func ||
		k == re.Interface ||
		k == re.Map ||
		k == re.Pointer)

}
