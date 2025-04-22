package except

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/errs"
)

type HttpError struct {
	Message error
	Code    int
}

func NewServerError(code int, msg any, args ...any) ServerError {
	return &HttpError{
		Message: errs.Convert(msg, nil, args...),
		Code:    code,
	}
}

func (e *HttpError) Error() string {
	var b = new(strings.Builder)
	b.WriteString("ServerError")
	var codeStr = strconv.Itoa(e.Code)
	if e.Code != 0 {
		b.WriteString(" (")
		b.WriteString(codeStr)
		b.WriteString(")")
	}
	b.WriteString(": ")
	b.WriteString(e.Message.Error())
	return b.String()
}

func (e *HttpError) StatusCode() int {
	return int(e.Code)
}

func (e *HttpError) UserMessage() string {
	return e.Message.Error()
}

func (e *HttpError) Unwrap() error {
	return e.Message
}

func (e *HttpError) As(target interface{}) bool {
	return errors.As(e.Message, target)
}

func (e *HttpError) Is(other error) bool {
	if other == nil {
		return e.Message == nil && e.Code == 0
	}

	switch otherErr := other.(type) {
	case *HttpError:
		return e.Code == otherErr.Code && errors.Is(e.Message, otherErr.Message) ||
			otherErr.Code == 0 && otherErr.Message == nil
	}

	return errors.Is(e.Message, other)
}
