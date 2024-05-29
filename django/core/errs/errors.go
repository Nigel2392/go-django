package errs

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Name string
	Err  error
}

func NewValidationError(name string, err any) ValidationError {

	switch e := err.(type) {
	case error:
		return ValidationError{Name: name, Err: e}
	case string:
		return ValidationError{Name: name, Err: errors.New(e)}
	default:
		return ValidationError{Name: name, Err: fmt.Errorf("%v", e)}
	}
}

func (e ValidationError) Is(other error) bool {
	switch otherErr := other.(type) {
	case *ValidationError:
		return errors.Is(e.Err, otherErr.Err) && otherErr.Name == e.Name
	case ValidationError:
		return errors.Is(e.Err, otherErr.Err) && otherErr.Name == e.Name
	case Error:
		return errors.Is(e.Err, otherErr)
	}
	return errors.Is(e.Err, other)
}

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func Errors(m []ValidationError) []error {
	var errs = make([]error, 0, len(m))
	for _, v := range m {
		errs = append(errs, v)
	}
	return errs
}

const (
	ErrFieldRequired Error = "Required field cannot be empty"
	ErrInvalidSyntax Error = "Invalid syntax for value"
	ErrInvalidType   Error = "Invalid type provided"
)
