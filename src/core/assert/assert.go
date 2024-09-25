package assert

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/src/core/errs"
)

type AssertError = errs.Error

var (
	Panic func(error) = func(err error) { panic(err) }

	LogOnError func(error)

	AssertionFailedError AssertError = "assertion failed"
)

// FailFunc is a function that is called when an assertion fails
// it is used to customize the behavior of the assertion
func FailFunc(failFn func(err error) error, msg any, args ...interface{}) error {

	var m string
	var mErr = &errs.MultiError{
		Errors: make([]error, 0),
	}
	if len(args) > 0 {
		switch e := msg.(type) {
		case string:
			m = fmt.Sprintf(e, args...)
		case error:
			mErr.Append(e)
			m = fmt.Sprint(args...)
		default:
			m = fmt.Sprint(append([]interface{}{e}, args...)...)
		}
	} else {
		switch msg := msg.(type) {
		case error:
			mErr.Append(msg)
		case string:
			m = msg
		default:
			m = fmt.Sprint(msg)
		}
	}

	var err error
	if len(mErr.Errors) == 0 {
		err = AssertionFailedError
	} else {
		mErr.Errors = append([]error{AssertionFailedError}, mErr.Errors...)
		err = mErr
	}

	if m != "" {
		err = errs.Wrap(
			err, m,
		)
	}

	return failFn(err)
}

// fail is the default function that is called when an assertion fails
func fail(err error) error {
	if Panic != nil {
		Panic(err)
	}
	if LogOnError != nil {
		LogOnError(err)
	}
	return err
}

// Fail is called when an assertion fails
// It is used to either return an error if PanicEnabled is false
// or panic if PanicEnabled is true
func Fail(msg any, args ...interface{}) error {
	return FailFunc(fail, msg, args...)
}

// Assert asserts that the condition is true
// if the condition is false, it panics with the message
func Assert(cond bool, msg any, args ...interface{}) error {
	if !cond {
		return Fail(msg, args...)
	}
	return nil
}

// AssertFunc asserts that the condition is true
// if the condition is false, it panics with the message
func AssertFunc(fail func(err error) error, cond bool, msg any, args ...interface{}) error {
	if !cond {
		return FailFunc(fail, msg, args...)
	}
	return nil
}

// Equal asserts that the two values are equal
// if the two values are not equal, it panics with the message
func Equal[T comparable](a, b T, msg any, args ...interface{}) error {
	return Assert(a == b, msg, args...)
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
