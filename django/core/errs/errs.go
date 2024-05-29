package errs

import "errors"

type Error string

func (e Error) Error() string {
	return string(e)
}

func (e Error) Is(other error) bool {
	switch otherErr := other.(type) {
	case *ValidationError:
		return errors.Is(e, otherErr.Err)
	case ValidationError:
		return errors.Is(e, otherErr.Err)
	case Error:
		return e == otherErr
	}
	return false
}
