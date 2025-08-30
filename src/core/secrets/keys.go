package secrets

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/secrets/signing"
)

type SecretKey []byte

func (s SecretKey) String() string {
	return string(s)
}

func (s SecretKey) Bytes() []byte {
	return []byte(s)
}

func (s SecretKey) IsZero() bool {
	if len(s) == 0 {
		return true
	}
	for _, b := range s {
		if b != 0 {
			return false
		}
	}
	return true
}

func (s SecretKey) Sign(ctx context.Context, data []byte) (string, error) {
	return SIGNER_BACKEND().Sign(ctx, data)
}

func (s SecretKey) Unsign(ctx context.Context, signed string) ([]byte, error) {
	return SIGNER_BACKEND().Unsign(ctx, signed)
}

func (s SecretKey) SignObject(ctx context.Context, obj interface{}) (string, error) {
	return signing.SignObject(ctx, SIGNER_BACKEND(), obj)
}

func (s SecretKey) UnsignObject(ctx context.Context, signed string, obj interface{}) error {
	return signing.UnsignObject(ctx, SIGNER_BACKEND(), signed, obj)
}
