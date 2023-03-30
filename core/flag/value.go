package flag

import (
	"fmt"
	"reflect"
	"strconv"
)

// Default value for a flag.
type Value interface {
	// String returns the string representation of the value.
	// Beware! This will convert the value to a string if it is not a string.
	String() string
	// Int returns the value as an integer.
	Int() int
	// Uint returns the value as an unsigned integer.
	Uint() uint
	// Float returns the value as a float.
	Float() float64
	// Bool returns the value as a boolean.
	Bool() bool
	// IsZero returns true if the value is zero.
	IsZero() bool
	// Any returns the value as an interface{}.
	Any() interface{}
	// Set sets the value.
	Set(s string) error
}

type value struct {
	value interface{}
}

func (v value) String() string {
	switch val := dePtr(v.value).(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	}
	return ""
}

func (v value) Int() int {
	switch val := dePtr(v.value).(type) {
	case int:
		return val
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint64:
		return int(val)
	case float64:
		return int(val)
	}
	return 0
}

func (v value) Uint() uint {
	switch val := dePtr(v.value).(type) {
	case int:
		return uint(val)
	case int64:
		return uint(val)
	case uint:
		return uint(val)
	case uint64:
		return uint(val)
	case float64:
		return uint(val)
	}
	return 0
}

func (v value) Float() float64 {
	switch val := dePtr(v.value).(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case float64:
		return float64(val)
	}
	return 0
}

func (v value) Bool() bool {
	switch val := dePtr(v.value).(type) {
	case bool:
		return val
	}
	return false
}

func (v value) IsZero() bool {
	return v.value == nil || v.value == reflect.Zero(reflect.TypeOf(v.value)).Interface()
}

func (v *value) Set(s string) error {
	var err error
	switch v.value.(type) {
	case string:
		v.value = s
	case int:
		v.value, err = strconv.Atoi(s)
	case int64:
		v.value, err = strconv.ParseInt(s, 10, 64)
	case uint:
		v.value, err = strconv.ParseUint(s, 10, 64)
	case uint64:
		v.value, err = strconv.ParseUint(s, 10, 64)
	case float64:
		v.value, err = strconv.ParseFloat(s, 64)
	case bool:
		v.value, err = strconv.ParseBool(s)
	default:
		//lint:ignore ST1005 I like capitalized error strings.
		return fmt.Errorf("Unsupported type: %s", reflect.TypeOf(v.value))
	}
	return err
}

func (v value) Any() interface{} {
	return v.value
}
