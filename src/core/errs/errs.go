package errs

type Error string

func (e Error) Error() string {
	return string(e)
}

func (e Error) Is(other error) bool {
	switch otherErr := other.(type) {
	case Error:
		return e == otherErr
	case interface{ Is(error) bool }:
		return otherErr.Is(e)
	}
	return false
}
