package except

import (
	"reflect"
	"slices"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/errs"
)

func GetServerError(err error) error {
	switch e := err.(type) {
	case *HttpError:
		return e
	case interface{ Unwrap() error }:
		return GetServerError(e.Unwrap())
	case interface{ Unwrap() []error }:
		for _, e := range e.Unwrap() {
			if e = GetServerError(e); e != nil {
				return e
			}
		}
	}
	return nil
}

func Fail(code Code, msg any, args ...interface{}) (err error) {
	err = &HttpError{
		Code:    code,
		Message: errs.Convert(msg, errs.ErrUnknown, args...),
	}

	return assert.Fail(err)
}

// Assert asserts that the condition is true
// if the condition is false, it panics with the message
func Assert(cond interface{}, code Code, msg any, args ...interface{}) error {
	var rTyp, rVal = reflect.TypeOf(cond), reflect.ValueOf(cond)
	if rTyp.Kind() == reflect.Func && !rVal.IsNil() {
		rVal = rVal.Call(nil)[0]
		rTyp = rVal.Type()
	}

	if rTyp.Kind() == reflect.Bool {
		if !rVal.Bool() {
			return Fail(code, msg, args...)
		}
	}

	if rVal.Kind() == reflect.Ptr && rVal.IsNil() {
		return Fail(code, msg, args...)
	}

	if !rVal.IsValid() || rVal.IsZero() {
		return Fail(code, msg, args...)
	}

	return nil
}

// Equal asserts that the two values are equal
// if the two values are not equal, it panics with the message
func AssertEqual[T comparable](a, b T, code Code, msg any, args ...interface{}) error {
	return Assert(a == b, code, msg, args...)
}

// Contains asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func AssertContains[T comparable](v T, a []T, code Code, msg any, args ...interface{}) error {
	return AssertContainsFunc(a, func(x T) bool { return x == v }, code, msg, args...)
}

// ContainsFunc asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func AssertContainsFunc[T any](a []T, f func(T) (in bool), code Code, msg any, args ...interface{}) error {
	return Assert(slices.ContainsFunc(a, f), code, msg, args...)
}
