package auth

import (
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/except"
)

var _ except.ServerError = (*authenticationError)(nil)

type authenticationError struct {
	Message string
	Status  int
}

// Panic and raise an authentication error
// We have a hook setup to catch any authentication errors and redirect to the login page
func Fail(code int, msg string) {

	assert.True(
		code == 0 || code >= 400 && code < 600,
		"Invalid status code %d, must be between 400 and 599", code,
	)

	assert.True(
		msg != "",
		"Message must not be empty",
	)

	if code == 0 {
		code = 401
	}

	panic(&authenticationError{
		Message: msg,
		Status:  code,
	})
}

func (e *authenticationError) Error() string {
	return e.Message
}

func (e *authenticationError) StatusCode() int {
	return e.Status
}

func (e *authenticationError) UserMessage() string {
	return e.Message
}
