package errs

import (
	"errors"
	"strings"
)

type MultiError struct {
	Errors []error
}

func WrapErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	var errList []error = make([]error, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		errList = append(errList, err)
	}

	if len(errList) == 0 {
		return nil
	}

	if len(errList) == 1 {
		return errList[0]
	}

	return NewMultiError(errList...)
}

func NewMultiError(errs ...error) *MultiError {
	return &MultiError{Errors: errs}
}

func (m *MultiError) Append(err error) {
	if m.Errors == nil {
		m.Errors = make([]error, 0)
	}
	if err == nil {
		return
	}
	m.Errors = append(m.Errors, err)
}

func (m *MultiError) Is(target error) bool {
	for _, err := range m.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (m *MultiError) As(target interface{}) bool {
	for _, err := range m.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

func (m *MultiError) Error() string {
	var errors []string = make([]string, 0, len(m.Errors))
	for _, err := range m.Errors {

		if err == nil {
			continue
		}

		errors = append(errors, err.Error())
	}
	return strings.Join(errors, ": ")
}

func (m *MultiError) Unwrap() []error {
	return m.Errors
}
