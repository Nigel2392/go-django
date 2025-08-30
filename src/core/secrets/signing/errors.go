package signing

import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

const (
	ErrorCodeSigningError      errors.GoCode = "signing.signature.error"
	ErrorCodeBadSignature      errors.GoCode = "signing.signature.bad"
	ErrorCodeSignatureMisMatch errors.GoCode = "signing.signature.mismatch"
	ErrorCodeSignatureExpired  errors.GoCode = "signing.signature.expired"
	ErrorSignerAlgorithm       errors.GoCode = "signing.signer.algorithm"
)

var (
	ErrSigning = errors.New(
		ErrorCodeSigningError,
		"error signing data",
	)

	ErrBadSignature = errors.New(
		ErrorCodeBadSignature,
		"bad signature",
		ErrSigning,
	)

	ErrSignatureMismatch = errors.New(
		ErrorCodeSignatureMisMatch,
		"signature mismatch",
		ErrBadSignature,
	)

	ErrSignatureExpired = errors.New(
		ErrorCodeSignatureExpired,
		"signature expired",
		ErrBadSignature,
	)

	ErrUnknownSignatureAlgorithm = errors.New(
		ErrorSignerAlgorithm,
		"unknown signature algorithm",
		ErrSigning,
	)
)
