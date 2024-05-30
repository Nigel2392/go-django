package blocks

import (
	"fmt"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/errs"
)

type BaseBlockValidationError[T comparable] struct {
	Errors         map[T][]error
	NonBlockErrors []error
}

func NewBlockErrors[T comparable](errors ...error) *BaseBlockValidationError[T] {

	var v = &BaseBlockValidationError[T]{
		Errors:         make(map[T][]error),
		NonBlockErrors: make([]error, 0),
	}
	for _, err := range errors {
		switch e := err.(type) {
		case errs.ValidationError[T]:
			v.AddError(e.Name, e)
		case *errs.ValidationError[T]:
			v.AddError(e.Name, e)
		case *BaseBlockValidationError[T]:
			for k, e := range e.Errors {
				v.AddError(k, e...)
			}
			v.NonBlockErrors = append(v.NonBlockErrors, e.NonBlockErrors...)
		default:
			v.AddNonBlockError(e)
		}
	}
	return v
}

func (m *BaseBlockValidationError[T]) HasErrors() bool {
	return len(m.Errors) != 0 || len(m.NonBlockErrors) != 0
}

func (m *BaseBlockValidationError[T]) AddError(key T, err ...error) *BaseBlockValidationError[T] {
	if _, ok := m.Errors[key]; !ok {
		m.Errors[key] = make([]error, 0)
	}

	assert.False(len(err) == 0, "error must not be empty")

	m.Errors[key] = append(m.Errors[key], err...)
	return m
}

func (m *BaseBlockValidationError[T]) AddNonBlockError(err error) {
	m.NonBlockErrors = append(m.NonBlockErrors, err)
}

func (m *BaseBlockValidationError[T]) Get(key T) []error {
	if errs, ok := m.Errors[key]; ok {
		return errs
	}
	return nil
}

func (m *BaseBlockValidationError[T]) Error() string {
	if len(m.Errors) == 0 && len(m.NonBlockErrors) == 0 {
		return ""
	}

	if len(m.Errors) == 0 {
		return fmt.Sprintf("%d non-field errors occurred when validating", len(m.Errors))
	}

	return fmt.Sprintf("%d errors occurred when validating", len(m.Errors)+len(m.NonBlockErrors))
}