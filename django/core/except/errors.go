package except

import (
	"errors"
	"strconv"
	"strings"
)

type (
	Code      = int
	HttpError struct {
		Message error
		Code    Code
	}
)

const (
	// Everything went smoothly
	Code200 Code = 200 // OK

	// Redirects / Cached
	Code301 Code = 301 // Moved Permanently
	Code302 Code = 302 // Found
	Code303 Code = 303 // See Other
	Code304 Code = 304 // Not Modified

	// Client Errors
	Code400 Code = 400 // Bad Request
	Code401 Code = 401 // Unauthorized
	Code402 Code = 402 // Payment Required
	Code403 Code = 403 // Forbidden
	Code404 Code = 404 // Not Found
	Code405 Code = 405 // Method Not Allowed
	Code406 Code = 406 // Not Acceptable
	Code407 Code = 407 // Proxy Authentication Required
	Code408 Code = 408 // Request Timeout
	Code409 Code = 409 // Conflict
	Code410 Code = 410 // Gone
	Code411 Code = 411 // Length Required
	Code412 Code = 412 // Precondition Failed
	Code413 Code = 413 // Payload Too Large
	Code414 Code = 414 // URI Too Long
	Code415 Code = 415 // Unsupported Media Type

	// Server Errors
	Code500 Code = 500 // Internal Server Error
	Code501 Code = 501 // Not Implemented
	Code502 Code = 502 // Bad Gateway
	Code503 Code = 503 // Service Unavailable
	Code504 Code = 504 // Gateway Timeout
	Code505 Code = 505 // HTTP Version Not Supported
	Code511 Code = 511 // Network Authentication Required

)

func ServerError(code Code, msg error) *HttpError {
	return &HttpError{
		Message: msg,
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
