package auth

import (
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/except"
)

const (
	ErrPwdCasingUpper  errs.Error = "password must contain at least one uppercase letter, and at least one lowercase letter"
	ErrPwdCasingLower  errs.Error = "password must contain at least one lowercase letter, and at least one uppercase letter"
	ErrPwdDigits       errs.Error = "password must contain at least one digit, and at least one non-digit"
	ErrPwdSpaces       errs.Error = "password must not contain spaces"
	ErrPwdSpecial      errs.Error = "password must contain at least one special character"
	ErrInvalidLogin    errs.Error = "invalid value, please try again"
	ErrInvalidEmail    errs.Error = "invalid email address"
	ErrInvalidUsername errs.Error = "invalid username"
	ErrUserExists      errs.Error = "user already exists"
	ErrIsActive        errs.Error = "user account is not active"
	ErrPasswordInvalid errs.Error = "invalid password"
	ErrPwdHashMismatch errs.Error = "password is not valid"
	ErrPwdNoMatch      errs.Error = "passwords do not match"
	ErrGenericAuthFail errs.Error = "authentication failed"
	ErrNoSession       errs.Error = "no session found"
)

var _ except.ServerError = (*authenticationError)(nil)

type authenticationError struct {
	Message string
	NextURL string
	Status  int
}

// Authentication errors can be raised using auth.Fail(...)
//
// This makes sure boilerplate code for failing auth is not repeated.
//
// It also allows for a more consistent way to handle auth errors.
//
// We have a hook setup to catch any authentication errors and redirect to the login page (see hooks.go)
func Fail(code int, msg string, next ...string) {

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

	var nextURL string
	if len(next) > 0 {
		nextURL = next[0]
	}

	panic(&authenticationError{
		Message: msg,
		Status:  code,
		NextURL: nextURL,
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
