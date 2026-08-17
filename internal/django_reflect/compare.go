package django_reflect

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"reflect"
	re "reflect"
	"unsafe"

	"github.com/Nigel2392/go-django/internal/bitch"
	"github.com/Nigel2392/goldcrest"
)

type FLAG_EQ = bitch.Flag

type equalsStep = func(*EqStepState) (eq bool, ok bool)

var (
	_, _, _, _, _, _, _, _ equalsStep = EQ_ISZEROER,
		EQ_DRIVER_VALUER,
		EQ_BYTES_RUNES,
		EQ_NIL_ZEROVALS,
		EQ_DEREF_POINTERS,
		EQ_TYPES_OPT_CNV,
		EQ_UNDERLYING_KIND,
		EQ_NIL_LEN_ZERO

	_EQ_STEPS = []equalsStep{
		EQ_ISZEROER,        // 100
		EQ_DRIVER_VALUER,   // 200
		EQ_BYTES_RUNES,     // 300
		EQ_NIL_ZEROVALS,    // 400
		EQ_DEREF_POINTERS,  // 500
		EQ_TYPES_OPT_CNV,   // 600
		EQ_UNDERLYING_KIND, // 700
		EQ_NIL_LEN_ZERO,    // 800
	}
)

func init() {
	for i, step := range _EQ_STEPS {
		RegisterCompareStep((i+1)*100, step)
	}
}

func RegisterCompareStep(order int, step func(*EqStepState) (eq bool, ok bool)) {
	goldcrest.Register(_EQ_HOOK, order, step)
}

const (
	_EQ_HOOK = "django_reflect.Equals"

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

		As mentioned, this is only valuable in extremely rare situations, which is why
		it isn't included by default.
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

type EqStepState struct {
	A, B   any
	V1, V2 reflect.Value
	Opts   FLAG_EQ

	steps []equalsStep
}

func (e *EqStepState) Equals(a, b any) bool {
	return equalsStepped(a, b, e.Opts, e.steps)
}

// Equals checks if a is equal to b.
//
// This does not nescessarily mean a == b,
// see the EQ_FLAGS documentation above for more details.
//
// Custom comparison operations can be added with [RegisterCompareStep]
func Equals(a, b any, opts ...any) bool {
	var flags FLAG_EQ
	var steps []equalsStep
	for i, op := range opts {
		switch op := op.(type) {
		case FLAG_EQ:
			flags |= op
		case []FLAG_EQ:
			for _, f := range op {
				flags |= f
			}
		case equalsStep:
			steps = append(steps, op)
		case []equalsStep:
			steps = append(steps, op...)
		default:
			panic(fmt.Errorf(
				"[%d]: option %T is not of type FLAG_EQ or func(*EqStepState) (bool, bool), or a slice of either.",
				i, op,
			))
		}
	}

	if len(steps) == 0 {
		steps = goldcrest.Get[equalsStep](
			_EQ_HOOK,
		)
	}

	return equalsStepped(a, b, flags, steps)
}

func equalsStepped(a, b any, opts FLAG_EQ, steps []equalsStep) bool {
	// cant ignore this one!
	if a == nil && b == nil {
		return true
	}

	if !bitch.Is(opts, EQ_IGNORE_NIL) && (a == nil || b == nil) {
		return (a == nil) == (b == nil)
	}

	// a.Equals(b) || b.Equals(a)
	if e, ok := a.(EqualityChecker); ok {
		return e.Equal(b)
	}

	if e, ok := b.(EqualityChecker); ok {
		return e.Equal(a)
	}
	if e, ok := a.(EqualityChecker2); ok {
		return e.Equals(b)
	}

	if e, ok := b.(EqualityChecker2); ok {
		return e.Equals(a)
	}

	var state = &EqStepState{
		A:     a,
		B:     b,
		V1:    reflect.ValueOf(a),
		V2:    reflect.ValueOf(b),
		Opts:  opts,
		steps: steps,
	}

	// safe because state does not live past this function
	statePtr := (*EqStepState)(noescape(unsafe.Pointer(state)))

	for _, step := range steps {
		eq, ok := step(statePtr)
		if ok {
			return eq
		}
	}

	return re.DeepEqual(state.V1.Interface(), state.V2.Interface())
}

func EQ_ISZEROER(state *EqStepState) (eq bool, ok bool) {
	if !bitch.Is(state.Opts, EQ_ZEROS) {
		return false, false
	}

	if a, ok := state.A.(isZeroer); ok {
		if b, ok := state.B.(isZeroer); ok {
			return (a == nil || a.IsZero()) == (b == nil || b.IsZero()), true
		}
	}
	return false, false
}

func EQ_DRIVER_VALUER(state *EqStepState) (eq bool, ok bool) {
	if !bitch.Is(state.Opts, EQ_DRIVER_VALUE) {
		return false, false
	}

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
		return state.Equals(state.A, state.B), true
	}

	return false, false
}

func EQ_BYTES_RUNES(state *EqStepState) (eq bool, ok bool) {
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

func EQ_NIL_ZEROVALS(state *EqStepState) (eq bool, retEq bool) {
	switch {
	case state.V1.Kind() == re.Invalid && state.V2.Kind() == re.Invalid:
		// this shouldnt really happen due to the initial nil == nil check
		return true, true

	case state.V1.Kind() == re.Invalid || state.V2.Kind() == re.Invalid:
		z, ok := _eq__nilLenKindIsZero(state.Opts, state.V1, state.V2)
		if ok {
			return z, true
		}

		if state.V1.Kind() != re.Invalid && _eq__canNil(state.V1.Kind()) {
			return state.V1.IsNil() == (state.V2.Kind() == re.Invalid), true
		}

		if state.V2.Kind() != re.Invalid && _eq__canNil(state.V2.Kind()) {
			return state.V2.IsNil() == (state.V1.Kind() == re.Invalid), true
		}

		if !bitch.Is(state.Opts, EQ_ZEROS) {
			return false, true
		}

		// i.e. 0 == nil == true
		return (state.V1.Kind() != re.Invalid && state.V1.IsZero() == (state.V2.Kind() == re.Invalid)) ||
			(state.V2.Kind() != re.Invalid && state.V2.IsZero() == (state.V1.Kind() == re.Invalid)), true
	}

	return false, false
}

func EQ_DEREF_POINTERS(state *EqStepState) (eq bool, retEq bool) {
	if !bitch.Is(state.Opts, EQ_IGNORE_PTR) {
		return false, false
	}

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

func EQ_TYPES_OPT_CNV(state *EqStepState) (eq, retEq bool) {
	if v1T, v2T := state.V1.Type(), state.V2.Type(); v1T != v2T {
		// dont convert types, return
		if !bitch.Is(state.Opts, EQ_TYPE_CONVERT) {
			return false, true
		}

		// convert types (example, int8 -> int64)
		if state.V1.Kind() != state.V2.Kind() {
			switch {
			case isSafeConversion(v1T, v2T) && v1T.ConvertibleTo(v2T):
				state.V1 = state.V1.Convert(v2T)
			case isSafeConversion(v2T, v1T) && v2T.ConvertibleTo(v1T):
				state.V2 = state.V2.Convert(v1T)
			}
		}
	}
	return false, false
}

func EQ_UNDERLYING_KIND(state *EqStepState) (eq, retEq bool) {
	// no type mismatch possible unless convert flag is set, handled above
	// compare underlying values (drivers.String("str") == string("str"))
	if state.V1.Kind() == state.V2.Kind() {
		switch state.V1.Kind() {
		case re.String:
			return state.V1.String() == state.V2.String(), true
		case re.Int, re.Int8, re.Int16, re.Int32, re.Int64:
			return state.V1.Int() == state.V2.Int(), true
		case re.Uint, re.Uint8, re.Uint16, re.Uint32, re.Uint64, re.Uintptr:
			return state.V1.Uint() == state.V2.Uint(), true
		case re.Bool:
			return state.V1.Bool() == state.V2.Bool(), true
		case re.Float32, re.Float64:
			return state.V1.Float() == state.V2.Float(), true
		case re.Complex64, re.Complex128:
			return state.V1.Complex() == state.V2.Complex(), true
		}
	}
	return false, false
}

func EQ_NIL_LEN_ZERO(state *EqStepState) (eq, retEq bool) {
	if isZero, ok := _eq__nilLenKindIsZero(state.Opts, state.V1, state.V2); ok {
		return isZero, true
	}
	return false, false
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
