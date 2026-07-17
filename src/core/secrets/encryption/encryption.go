package encryption

import (
	"context"
)

type Encryption interface {
	Encrypt(ctx context.Context, data []byte) ([]byte, error)
	Decrypt(ctx context.Context, data []byte) ([]byte, error)
}

const (
	DEFAULT = "encryption.default.engine"
	AES     = "encryption.default.AES"
)

var _ = RegisterCrypto(DEFAULT, func(key []byte, fallbackKeys [][]byte) (Encryption, error) {
	return newBaseEncryption(key, fallbackKeys)
})

var _ = RegisterCrypto(AES, func(key []byte, fallbackKeys [][]byte) (Encryption, error) {
	return newBaseEncryption(key, fallbackKeys)
})
