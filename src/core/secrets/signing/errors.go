package signing

import (
	"github.com/Nigel2392/go-django/src/core/secrets/algorithm"
)

var (
	ErrSigning                   = algorithm.ErrSigning
	ErrBadSignature              = algorithm.ErrBadSignature
	ErrSignatureMismatch         = algorithm.ErrSignatureMismatch
	ErrSignatureExpired          = algorithm.ErrSignatureExpired
	ErrUnknownSignatureAlgorithm = algorithm.ErrUnknownSignatureAlgorithm
)
