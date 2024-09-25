package errs

import (
	"errors"
	"strings"
)

type WithMessage struct {
	cause error
	msg   string
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return &WithMessage{
		cause: err,
		msg:   message,
	}
}

func (w *WithMessage) Error() string {
	var b = new(strings.Builder)
	var errStr = w.cause.Error()
	b.Grow(len(errStr) + len(w.msg) + 2)
	b.WriteString(w.msg)
	b.WriteString(": ")
	b.WriteString(errStr)
	return b.String()
}

func (w *WithMessage) Is(other error) bool {
	switch otherErr := other.(type) {
	case *WithMessage:
		return errors.Is(w.cause, otherErr.cause) && w.msg == otherErr.msg
	}
	return errors.Is(w.cause, other)
}

func (w *WithMessage) Unwrap() error {
	return w.cause
}
