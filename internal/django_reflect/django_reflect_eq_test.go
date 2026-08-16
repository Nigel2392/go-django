package django_reflect

import (
	"database/sql/driver"
	"testing"
)

type customString string
type customInt int

type eqChecker struct {
	val int
}

func (c eqChecker) Equal(other any) bool {
	if o, ok := other.(eqChecker); ok {
		return c.val == o.val
	}
	return false
}

type eqChecker2 struct {
	val string
}

func (c eqChecker2) Equals(other any) bool {
	if o, ok := other.(eqChecker2); ok {
		return c.val == o.val
	}
	return false
}

type eqCheckerPtr struct {
	val int
}

func (c *eqCheckerPtr) Equal(other any) bool {
	if c == nil {
		if other == nil {
			return true
		}
		if o, ok := other.(*eqCheckerPtr); ok {
			return o == nil
		}
		return false
	}
	if o, ok := other.(*eqCheckerPtr); ok {
		if o == nil {
			return false
		}
		return c.val == o.val
	}
	return false
}

type zeroChecker struct {
	isZero bool
}

func (z zeroChecker) IsZero() bool {
	return z.isZero
}

type valuerStruct struct {
	val string
}

func (v valuerStruct) Value() (driver.Value, error) {
	return v.val, nil
}

type valuerStructInt struct{ val int }

func (v valuerStructInt) Value() (driver.Value, error) { return int64(v.val), nil }

type valuerStructBool struct{ val bool }

func (v valuerStructBool) Value() (driver.Value, error) { return v.val, nil }

type valuerStructFloat struct{ val float64 }

func (v valuerStructFloat) Value() (driver.Value, error) { return v.val, nil }

type valuerStructPtr struct{ val *int }

func (v *valuerStructPtr) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	if v.val == nil {
		return nil, nil
	}
	return int64(*v.val), nil
}

type valuerStructErr struct{}

func (v valuerStructErr) Value() (driver.Value, error) {
	return nil, ErrTypeMismatch
}

func ptr[T any](v T) *T {
	return &v
}

func TestEquals(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		flags    []FLAG_EQ
		expected bool
	}{
		// EQ_NONE (15 tests)
		{"None_same_int", 1, 1, []FLAG_EQ{EQ_NONE}, true},
		{"None_diff_int", 1, 2, []FLAG_EQ{EQ_NONE}, false},
		{"None_same_string", "a", "a", []FLAG_EQ{EQ_NONE}, true},
		{"None_diff_string", "a", "b", []FLAG_EQ{EQ_NONE}, false},
		{"None_diff_type_int_int64", 1, int64(1), []FLAG_EQ{EQ_NONE}, false},
		{"None_diff_type_int_string", 1, "1", []FLAG_EQ{EQ_NONE}, false},
		{"None_nil_nil", nil, nil, []FLAG_EQ{EQ_NONE}, true},
		{"None_nil_val", nil, 1, []FLAG_EQ{EQ_NONE}, false},
		{"None_val_nil", 1, nil, []FLAG_EQ{EQ_NONE}, false},
		{"None_ptr_val_same", ptr(1), 1, []FLAG_EQ{EQ_NONE}, false},
		{"None_val_ptr_same", 1, ptr(1), []FLAG_EQ{EQ_NONE}, false},
		{"None_same_struct", valuerStruct{"a"}, valuerStruct{"a"}, []FLAG_EQ{EQ_NONE}, true},
		{"None_diff_struct", valuerStruct{"a"}, valuerStruct{"b"}, []FLAG_EQ{EQ_NONE}, false},
		{"None_EqualityChecker", eqChecker{1}, eqChecker{1}, []FLAG_EQ{EQ_NONE}, true},
		{"None_EqualityChecker2", eqChecker2{"a"}, eqChecker2{"a"}, []FLAG_EQ{EQ_NONE}, true},

		// EQ_TYPE_CONVERT (15 tests)
		{"TypeConvert_int_int64", 1, int64(1), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_int_int64_diff", 1, int64(2), []FLAG_EQ{EQ_TYPE_CONVERT}, false},
		{"TypeConvert_int8_int32", int8(5), int32(5), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_uint_int", uint(5), int(5), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_float32_float64", float32(1.5), float64(1.5), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_float64_int", float64(1.0), int(1), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_float64_int_truncation", float64(1.5), int(1), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_customInt", customInt(1), 1, []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_customString", customString("test"), "test", []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_customString_diff", customString("test"), "test2", []FLAG_EQ{EQ_TYPE_CONVERT}, false},
		{"TypeConvert_string_bytes", "test", []byte("test"), []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_bytes_string", []byte("test"), "test", []FLAG_EQ{EQ_TYPE_CONVERT}, true},
		{"TypeConvert_string_int", "1", 1, []FLAG_EQ{EQ_TYPE_CONVERT}, false}, // Unsafe conversion
		{"TypeConvert_int_string", 1, "1", []FLAG_EQ{EQ_TYPE_CONVERT}, false}, // Unsafe conversion
		{"TypeConvert_slice_array", []int{1, 2}, [2]int{1, 2}, []FLAG_EQ{EQ_TYPE_CONVERT}, false},

		// EQ_IGNORE_PTR (15 tests)
		{"IgnorePtr_val_ptr", 1, ptr(1), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_ptr_val", ptr(1), 1, []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_ptr_ptr", ptr(1), ptr(1), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_diff", ptr(1), 2, []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_diff2", 1, ptr(2), []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_double_ptr_val", ptr(ptr(1)), ptr(1), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_struct_ptr", valuerStruct{"a"}, ptr(valuerStruct{"a"}), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_struct_ptr_diff", valuerStruct{"a"}, ptr(valuerStruct{"b"}), []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_string_ptr", "test", ptr("test"), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_string_ptr_diff", "test", ptr("test2"), []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_nil_ptr_val", (*int)(nil), 0, []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_val_nil_ptr", 0, (*int)(nil), []FLAG_EQ{EQ_IGNORE_PTR}, false},
		{"IgnorePtr_nil_ptr_nil_ptr", (*int)(nil), (*int)(nil), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_bool_ptr", true, ptr(true), []FLAG_EQ{EQ_IGNORE_PTR}, true},
		{"IgnorePtr_float_ptr", 1.5, ptr(1.5), []FLAG_EQ{EQ_IGNORE_PTR}, true},

		// EQ_ZEROS (15 tests)
		{"Zeros_interface_both_zero", zeroChecker{true}, zeroChecker{true}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_interface_one_zero", zeroChecker{true}, zeroChecker{false}, []FLAG_EQ{EQ_ZEROS}, false},
		{"Zeros_interface_both_not_zero", zeroChecker{false}, zeroChecker{false}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_slice_nil_empty", []int(nil), []int{}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_slice_empty_empty", []int{}, []int{}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_slice_nil_nil", []int(nil), []int(nil), []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_slice_nil_not_empty", []int(nil), []int{1}, []FLAG_EQ{EQ_ZEROS}, false},
		{"Zeros_map_nil_empty", map[string]int(nil), map[string]int{}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_map_empty_empty", map[string]int{}, map[string]int{}, []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_map_nil_nil", map[string]int(nil), map[string]int(nil), []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_map_nil_not_empty", map[string]int(nil), map[string]int{"a": 1}, []FLAG_EQ{EQ_ZEROS}, false},
		{"Zeros_chan_nil_empty", (chan int)(nil), make(chan int, 0), []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_chan_empty_empty", make(chan int, 0), make(chan int, 0), []FLAG_EQ{EQ_ZEROS}, true},
		{"Zeros_chan_nil_not_empty", (chan int)(nil), func() chan int { c := make(chan int, 1); c <- 1; return c }(), []FLAG_EQ{EQ_ZEROS}, false},
		{"Zeros_int_int", 0, 0, []FLAG_EQ{EQ_ZEROS}, true},

		// EQ_DRIVER_VALUE (15 tests)
		{"DriverValue_same", valuerStruct{"test"}, "test", []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_both_valuer", valuerStruct{"test"}, valuerStruct{"test"}, []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_diff", valuerStruct{"test"}, "test2", []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_diff2", "test2", valuerStruct{"test"}, []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_same_valuer_valuer", valuerStruct{"a"}, valuerStruct{"a"}, []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_diff_valuer_valuer", valuerStruct{"a"}, valuerStruct{"b"}, []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_int", valuerStructInt{5}, int64(5), []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_int_diff", valuerStructInt{5}, int64(6), []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_bool", valuerStructBool{true}, true, []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_bool_diff", valuerStructBool{true}, false, []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_float", valuerStructFloat{1.5}, 1.5, []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_float_diff", valuerStructFloat{1.5}, 2.5, []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_err_fallback", valuerStructErr{}, valuerStructErr{}, []FLAG_EQ{EQ_DRIVER_VALUE}, true},
		{"DriverValue_err_fallback_diff", valuerStructErr{}, 1, []FLAG_EQ{EQ_DRIVER_VALUE}, false},
		{"DriverValue_nil_valuer", (*valuerStructPtr)(nil), (*valuerStructPtr)(nil), []FLAG_EQ{EQ_DRIVER_VALUE}, true},

		// EQ_IGNORE_NIL (15 tests)
		{"IgnoreNil_typed_untyped", (*int)(nil), nil, []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_untyped_untyped", nil, nil, []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_typed_typed", (*int)(nil), (*int)(nil), []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_typed_diff_typed", (*int)(nil), (*float64)(nil), []FLAG_EQ{EQ_IGNORE_NIL}, false},
		{"IgnoreNil_eqcheckerptr_nil_nil", (*eqCheckerPtr)(nil), (*eqCheckerPtr)(nil), []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_eqcheckerptr_nil_val", (*eqCheckerPtr)(nil), &eqCheckerPtr{1}, []FLAG_EQ{EQ_IGNORE_NIL}, false},
		{"IgnoreNil_eqcheckerptr_val_nil", &eqCheckerPtr{1}, (*eqCheckerPtr)(nil), []FLAG_EQ{EQ_IGNORE_NIL}, false},
		{"IgnoreNil_int_int", 1, 1, []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_int_diff", 1, 2, []FLAG_EQ{EQ_IGNORE_NIL}, false},
		{"IgnoreNil_string_string", "a", "a", []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_string_diff", "a", "b", []FLAG_EQ{EQ_IGNORE_NIL}, false},
		{"IgnoreNil_slice_nil_nil", []int(nil), []int(nil), []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_slice_empty_empty", []int{}, []int{}, []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_map_nil_nil", map[int]int(nil), map[int]int(nil), []FLAG_EQ{EQ_IGNORE_NIL}, true},
		{"IgnoreNil_map_empty_empty", map[int]int{}, map[int]int{}, []FLAG_EQ{EQ_IGNORE_NIL}, true},

		// EQ_DFLT (15 tests)
		{"All_mixed", valuerStruct{"test"}, ptr("test"), []FLAG_EQ{EQ_DFLT}, true},
		{"All_mixed_diff", valuerStruct{"test"}, ptr("test2"), []FLAG_EQ{EQ_DFLT}, false},
		{"All_int_custom_ptr", ptr(customInt(5)), int64(5), []FLAG_EQ{EQ_DFLT}, true},
		{"All_valuerInt_customIntPtr", valuerStructInt{5}, ptr(customInt(5)), []FLAG_EQ{EQ_DFLT}, true},
		{"All_zeros_slice", []int(nil), []int{}, []FLAG_EQ{EQ_DFLT}, true},
		{"All_zeros_interface", zeroChecker{true}, zeroChecker{true}, []FLAG_EQ{EQ_DFLT}, true},
		{"All_ignoreptr_typeconv", ptr(1), int64(1), []FLAG_EQ{EQ_DFLT}, true},
		{"All_ignoreptr_typeconv_diff", ptr(1), int64(2), []FLAG_EQ{EQ_DFLT}, false},
		{"All_valuer_typeconv", valuerStructInt{5}, int64(5), []FLAG_EQ{EQ_DFLT}, true},
		{"All_valuer_typeconv_diff", valuerStructInt{5}, int64(6), []FLAG_EQ{EQ_DFLT}, false},
		{"All_valuer_typeconv_ignoreptr", valuerStructInt{5}, ptr(int64(5)), []FLAG_EQ{EQ_DFLT}, true},
		{"All_valuer_typeconv_ignoreptr_diff", valuerStructInt{5}, ptr(int64(6)), []FLAG_EQ{EQ_DFLT}, false},
		{"All_zeros_map", map[string]int(nil), map[string]int{}, []FLAG_EQ{EQ_DFLT}, true},
		{"All_typed_nil_val", (*int)(nil), 0, []FLAG_EQ{EQ_DFLT}, false},
		{"All_eqchecker2", eqChecker2{"a"}, eqChecker2{"a"}, []FLAG_EQ{EQ_DFLT}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Equals(tt.a, tt.b, tt.flags...)
			if result != tt.expected {
				t.Errorf("%s: Equals(%v, %v, %v) = %v, expected %v", tt.name, tt.a, tt.b, tt.flags, result, tt.expected)
			}
		})
	}
}
