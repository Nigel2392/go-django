package securedb

import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

const CodeSecureValueError errors.GoCode = "SecureValueError"

var (
	ErrSecureValue = errors.New(
		CodeSecureValueError,
		"error handling secured value",
		errors.ValueError,
	)
)
