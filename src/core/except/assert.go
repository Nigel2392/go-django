package except

import (
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
)

type ServerError interface {
	error
	StatusCode() int
	UserMessage() string
}

func GetServerError(err error) ServerError {
	switch e := err.(type) {
	case ServerError:
		return e
	case interface{ Unwrap() error }:
		return GetServerError(e.Unwrap())
	case interface{ Unwrap() []error }:
		for _, e := range e.Unwrap() {
			if serverErr := GetServerError(e); serverErr != nil {
				return serverErr
			}
		}
	}
	return nil
}

func Fail(code int, msg any, args ...interface{}) (err error) {
	err = &HttpError{
		Code:    code,
		Message: errs.Convert(msg, errs.ErrUnknown, args...),
	}

	return assert.Fail(err)
}

func isBoolFunc(t reflect.Type) bool {
	return t.NumOut() == 1 && t.Out(0).Kind() == reflect.Bool && t.NumIn() == 0
}

// Assert asserts that the condition is true
// if the condition is false, it panics with the message
func Assert(cond interface{}, code int, msg any, args ...interface{}) error {
	var rTyp, rVal = reflect.TypeOf(cond), reflect.ValueOf(cond)
	if rTyp.Kind() == reflect.Func && !rVal.IsNil() && isBoolFunc(rTyp) {
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

// AssertNil asserts that the value is nil
// if the value is not nil, it panics with the message
func AssertNil(v any, code int, msg any, args ...interface{}) error {
	return Assert(v == nil, code, msg, args...)
}

// AssertNotNil asserts that the value is not nil
// if the value is nil, it panics with the message
func AssertNotNil(v any, code int, msg any, args ...interface{}) error {
	return Assert(v != nil, code, msg, args...)
}

// Equal asserts that the two values are equal
// if the two values are not equal, it panics with the message
func AssertEqual[T comparable](a, b T, code int, msg any, args ...interface{}) error {
	return Assert(a == b, code, msg, args...)
}

// Contains asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func AssertContains[T comparable](v T, a []T, code int, msg any, args ...interface{}) error {
	return AssertContainsFunc(a, func(x T) bool { return x == v }, code, msg, args...)
}

// ContainsFunc asserts that the value is in the slice
// if the value is not in the slice, it panics with the message
func AssertContainsFunc[T any](a []T, f func(T) (in bool), code int, msg any, args ...interface{}) error {
	return Assert(slices.ContainsFunc(a, f), code, msg, args...)
}
