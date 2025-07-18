package errors

import (
	"errors"
	"fmt"
	"reflect"
)

type Error struct {
	Code    GoCode
	Reason  error
	Message string
	Related []error
}

func New(code GoCode, message string, related ...error) Error {
	return Error{
		Code:    code,
		Message: message,
		Related: related,
	}
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
func Errorf(code GoCode, format string, args ...interface{}) error {
	return Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func (e Error) Error() string {
	var reasonStr string
	if e.Reason != nil {
		var reason = e.Reason.Error()
		var rBytes = make([]byte, len(reason)+2)
		rBytes[0] = ':'
		rBytes[1] = ' '
		copy(rBytes[2:], reason)
		reasonStr = string(rBytes)
	}

	var code = e.Code
	if code == "" {
		code = CodeUnknown
	}

	return fmt.Sprintf("%s: %s%s", code, e.Message, reasonStr)
}

func (e Error) WithCause(reason error) Error {
	if e.Reason != nil {
		return Error{
			Code:    e.Code,
			Message: e.Message,
			Reason:  errors.Join(e.Reason, reason),
			Related: e.Related,
		}
	}

	//	if diff, ok := reason.(Error); ok && diff.Code == e.Code &&
	//		// If the reason is already an Error with the same code, we can just return it.
	//		if e.Message != diff.Message && e.Message != "" {
	//			if diff.Reason == nil {
	//				diff.Reason = errors.New(e.Message)
	//			} else {
	//				diff.Reason = Wrap(diff.Reason, e.Message)
	//			}
	//		}
	//		return diff
	//	}

	return Error{
		Code:    e.Code,
		Message: e.Message,
		Reason:  reason,
		Related: e.Related,
	}
}

func (e Error) Wrap(message string) Error {
	return Error{
		Code:    e.Code,
		Message: message,
		Reason:  e.Reason,
		Related: e.Related,
	}
}

func (e Error) Wrapf(format string, args ...any) Error {
	return Error{
		Code:    e.Code,
		Message: fmt.Sprintf(format, args...),
		Reason:  e.Reason,
		Related: e.Related,
	}
}

func (e Error) equals(other Error) bool {
	// If the codes are the same, we consider them equal.
	if e.Code == other.Code && e.Code != "" {
		return true
	}
	return e.Message == other.Message
}

func (e Error) Is(chk error) bool {
	if e2, ok := chk.(Error); ok && e.equals(e2) {
		return true
	}
	if chk == nil {
		return false
	}

	if errors.Is(chk, e.Reason) {
		return true
	}

	for _, rel := range e.Related {
		if errors.Is(chk, rel) {
			return true
		}
	}

	return false
}

func (e Error) Unwrap() error {
	return e.Reason
}

func (e Error) Cause() error {
	return e.Reason
}

type DatabaseError interface {
	Error() string
	Code() DBCode
	Reason() error
	WithCause(otherErr error) DatabaseError
	Wrap(message string) DatabaseError
	Wrapf(format string, args ...any) DatabaseError
}

type databaseError struct {
	code    DBCode
	message string
	reason  error
	related []error

	// remember the original error pointer
	original uintptr
}

func InvalidDatabaseError(err error) DatabaseError {
	return &databaseError{
		code:     CodeInvalid,
		message:  err.Error(),
		reason:   err,
		original: 0,
	}
}

func UnknownDatabaseError(code DBCode, message string, related ...error) DatabaseError {
	var err = new(databaseError)
	err.code = code
	err.message = message
	err.related = related
	err.original = reflect.ValueOf(err).Pointer()
	return err
}

func dbError(code DBCode, message string, related ...error) DatabaseError {
	var err = new(databaseError)
	err.code = code
	err.message = message
	err.related = related
	err.original = reflect.ValueOf(err).Pointer()
	return err
}

func (e *databaseError) Code() DBCode {
	return e.code
}

func (e *databaseError) Reason() error {
	return e.reason
}

func (e *databaseError) equals(other *databaseError) bool {
	return (e.code == other.code) || (e.original != 0 && e.original == other.original)
}

func (e *databaseError) WithCause(otherErr error) DatabaseError {

	var other = &databaseError{}
	if errors.As(otherErr, &other) && e.equals(other) {
		// if the reason is already a databaseError, we can just return it IF the codes match
		// the otherErr is almost always guaranteed to be a 'deeper' error in the stack.
		// we can just merge the reasons together.
		return &databaseError{
			code:    e.code,
			message: e.message,
			reason:  errors.Join(e.reason, other.reason),
		}
	}

	return &databaseError{
		code:    e.code,
		message: e.message,
		reason:  otherErr,
		related: e.related,
	}
}

func (e *databaseError) Error() string {
	if e.message == "" && e.reason == nil {
		return fmt.Sprintf("[%s]: <no message>", e.code)
	}

	var reasonStr string
	if e.reason != nil && !errors.Is(e.reason, e) {
		var rBytes []byte
		var reason = e.reason.Error()
		if reason == e.message {
			// If the reason is the same as the message, we don't need to add it again.
			goto returnString
		}

		rBytes = make([]byte, len(reason)+2)
		rBytes[0], rBytes[1] = ',', ' '
		copy(rBytes[2:], reason)
		reasonStr = string(rBytes)
	}

returnString:
	return fmt.Sprintf("[%s]: %s%s", e.code, e.message, reasonStr)
}

func (e *databaseError) Wrap(message string) DatabaseError {
	return &databaseError{
		code:    e.code,
		message: message,
		reason:  e.reason,
		related: e.related,
	}
}

func (e *databaseError) Wrapf(format string, args ...any) DatabaseError {
	return &databaseError{
		code:    e.code,
		message: fmt.Sprintf(format, args...),
		reason:  e.reason,
		related: e.related,
	}
}

func (e *databaseError) Is(chk error) bool {
	if e2, ok := chk.(*databaseError); ok && e.equals(e2) {
		return true
	}

	if chk == nil {
		return false
	}

	if errors.Is(chk, e.reason) {
		return true
	}

	for _, rel := range e.related {
		if errors.Is(chk, rel) {
			return true
		}
	}

	return false
}

func (e *databaseError) Unwrap() error {
	return e.reason
}

func (e *databaseError) Cause() error {
	return e.reason
}
