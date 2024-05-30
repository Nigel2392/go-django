package assert

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/Nigel2392/django/core/errs"
	"github.com/pkg/errors"
)

type AssertError = errs.Error

var (
	PanicEnabled bool = true

	LogOnError func(error)

	AssertionFailedError AssertError = "assertion failed"
)

// Fail will always panic unless PanicEnabled is set to false
// If PanicEnabled is false, it will return an error
func Fail(msg any, args ...interface{}) error {
	var m string
	if len(args) > 0 {
		if s, ok := msg.(string); !ok {
			m = fmt.Sprint(append([]interface{}{msg}, args...)...)
		} else {
			m = fmt.Sprintf(s, args...)
		}
	} else {
		if _, ok := msg.(string); ok {
			m = msg.(string)
		} else {
			m = fmt.Sprint(msg)
		}
	}

	var err error
	if m == "" {
		err = AssertionFailedError
	} else {
		err = errors.Wrap(
			AssertionFailedError, m,
		)
	}

	if PanicEnabled {
		panic(err)
	}

	if LogOnError != nil {
		LogOnError(err)
	}

	return err
}

// Assert asserts that the condition is true
// if the condition is false, it panics with the message
func Assert(cond bool, msg any, args ...interface{}) error {
	if !cond {
		return Fail(msg, args...)
	}
	return nil
}

// Equal asserts that the two values are equal
// if the two values are not equal, it panics with the message
func Equal[T comparable](a, b T, msg any, args ...interface{}) error {
	return Assert(a == b, msg, args...)
}

// Truthy asserts that the value is truthy
// A value is truthy if it is not an invalid value and not the zero value
func Truthy(v any, msg any, args ...interface{}) error {
	var rVal = reflect.ValueOf(v)
	return True(rVal.IsValid() && !rVal.IsZero(), msg, args...)
}

// Falsy asserts that the value is falsy
// A value is falsy if it is an invalid value or the zero value
func Falsy(v any, msg any, args ...interface{}) error {
	var rVal = reflect.ValueOf(v)
	return True(!rVal.IsValid() || rVal.IsZero(), msg, args...)
}

// Err asserts that the error is nil
// if the error is not nil, it panics with the message
func Err(err error) error {
	return True(err == nil, "expected non-nil error: %v", err)
}

// ErrNil asserts that the error is not nil
// if the error is nil, it panics with the message
func ErrNil(err error) error {
	return True(err != nil, "expected nil error")
}

// Gt asserts that the length of the slice is greater than the min
// if the length is less than or equal to the min, it panics with the message
func Gt[T any](a []T, min int, msg any, args ...interface{}) error {
	return True(len(a) > min, msg, args...)
}

// Lt asserts that the length of the slice is less than the max
// if the length is greater than or equal to the max, it panics with the message
func Lt[T any](a []T, max int, msg any, args ...interface{}) error {
	return True(len(a) < max, msg, args...)
}

// Gte asserts that the length of the slice is greater than or equal to the min
// if the length is less than the min, it panics with the message
func Gte[T any](a []T, min int, msg any, args ...interface{}) error {
	return True(len(a) >= min, msg, args...)
}

// Lte asserts that the length of the slice is less than or equal to the max
// if the length is greater than the max, it panics with the message
func Lte[T any](a []T, max int, msg any, args ...interface{}) error {
	return True(len(a) <= max, msg, args...)
}

// Contains asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func Contains[T comparable](v T, a []T, msg any, args ...interface{}) error {
	return ContainsFunc(a, func(x T) bool { return x == v }, msg, args...)
}

// ContainsFunc asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func ContainsFunc[T any](a []T, f func(T) (in bool), msg any, args ...interface{}) error {
	return Assert(slices.ContainsFunc(a, f), msg, args...)
}

// True asserts that the condition is true
// if the condition is false, it panics with the message
func True(cond bool, msg any, args ...interface{}) error {
	return Assert(cond, msg, args...)
}

// False asserts that the condition is false
// if the condition is true, it panics with the message
func False(cond bool, msg any, args ...interface{}) error {
	return Assert(!cond, msg, args...)
}
