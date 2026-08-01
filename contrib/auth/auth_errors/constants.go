package autherrors

import "github.com/Nigel2392/go-django/src/core/errs"

const (
	// Auth errors
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
	ErrPasswordInvalid errs.Error = "password is not valid"
	ErrPwdHashMismatch errs.Error = ErrPasswordInvalid
	ErrPwdNoMatch      errs.Error = "passwords do not match"
	ErrGenericAuthFail errs.Error = "authentication failed"
	ErrNoSession       errs.Error = "no session found"
)
