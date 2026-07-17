package encryption

import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

const (
	ErrorCodeEncryptionError errors.GoCode = "encryption.encryption.error"
	ErrorCodeDecryptionError errors.GoCode = "encryption.decryption.error"
)

var (
	ErrEncryption = errors.New(
		ErrorCodeEncryptionError,
		"error encrypting data",
	)
	ErrDecryption = errors.New(
		ErrorCodeDecryptionError,
		"error decrypting data",
	)
)
