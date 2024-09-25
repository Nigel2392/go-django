package errs

import (
	"encoding/json"
	"errors"
	"fmt"
)

type DjangoError interface {
	error
	Is(error) bool
	DjangoError() // marker method
}

type ValidationError[T comparable] struct {
	Name T
	Err  error
}

func NewValidationError[T comparable](name T, err any) ValidationError[T] {

	switch e := err.(type) {
	case error:
		return ValidationError[T]{Name: name, Err: e}
	case string:
		return ValidationError[T]{Name: name, Err: errors.New(e)}
	default:
		return ValidationError[T]{Name: name, Err: fmt.Errorf("%v", e)}
	}
}

func (e ValidationError[T]) Is(other error) bool {
	switch otherErr := other.(type) {
	case *ValidationError[T]:
		return errors.Is(e.Err, otherErr.Err) && otherErr.Name == e.Name
	case ValidationError[T]:
		return errors.Is(e.Err, otherErr.Err) && otherErr.Name == e.Name
	case Error:
		return errors.Is(e.Err, otherErr)
	}
	return errors.Is(e.Err, other)
}

func (e ValidationError[T]) Error() string {
	return e.Err.Error()
}

func (e ValidationError[T]) DjangoError() {}

func (e ValidationError[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Err)
}

func Errors[T comparable](m []ValidationError[T]) []error {
	var errs = make([]error, 0, len(m))
	for _, v := range m {
		errs = append(errs, v)
	}
	return errs
}
