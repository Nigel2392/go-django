package algorithm

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"hash"
)

var safeB64Encoding = base64.
	NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_").
	WithPadding(base64.NoPadding)

type Algorithm interface {
	Name() string
	Signature(ctx context.Context, data, salt, key []byte) ([]byte, error)
	Verify(ctx context.Context, data, signature, salt, key []byte) error
}

func EncodeToString(data []byte) string {
	return safeB64Encoding.EncodeToString(data)
}

func DecodeString(s string) ([]byte, error) {
	return safeB64Encoding.DecodeString(s)
}

type SignatureAlgorithm struct {
	Algorithm   string
	SignatureFn func(ctx context.Context, data, salt, key []byte) ([]byte, error)
	VerifyFn    func(ctx context.Context, data, signature []byte, salt, key []byte) error
}

func (s *SignatureAlgorithm) Name() string {
	return s.Algorithm
}

func (s *SignatureAlgorithm) Signature(ctx context.Context, data, salt, key []byte) ([]byte, error) {
	return s.SignatureFn(ctx, data, salt, key)
}

func (s *SignatureAlgorithm) Verify(ctx context.Context, data, signature []byte, salt, key []byte) error {
	return s.VerifyFn(ctx, data, signature, salt, key)
}

func NewSignatureAlgorithm(name string, hashFunc func() hash.Hash) Algorithm {
	return &SignatureAlgorithm{
		Algorithm: name,
		SignatureFn: func(ctx context.Context, data, salt, key []byte) ([]byte, error) {
			var h = hmac.New(hashFunc, key)
			if len(salt) > 0 {
				if _, err := h.Write(salt); err != nil {
					return nil, ErrSigning.WithCause(fmt.Errorf(
						"failed to write salt to %q hash: %w",
						name, err,
					))
				}
			}
			if _, err := h.Write(data); err != nil {
				return nil, ErrSigning.WithCause(fmt.Errorf(
					"failed to write data to %q hash: %w",
					name, err,
				))
			}
			return h.Sum(nil), nil
		},
		VerifyFn: func(ctx context.Context, data, signature []byte, salt, key []byte) error {
			var h = hmac.New(hashFunc, key)
			if len(salt) > 0 {
				if _, err := h.Write(salt); err != nil {
					return ErrSigning.WithCause(fmt.Errorf(
						"failed to write salt to %q hash: %w",
						name, err,
					))
				}
			}
			if _, err := h.Write(data); err != nil {
				return ErrSigning.WithCause(fmt.Errorf(
					"failed to write data to %q hash: %w",
					name, err,
				))
			}
			var sum = h.Sum(nil)
			if !hmac.Equal(sum, signature) {
				return ErrSignatureMismatch.Wrapf(
					"signature mismatch for %q hash: %x != %x",
					name, sum, signature,
				)
			}
			return nil
		},
	}
}
