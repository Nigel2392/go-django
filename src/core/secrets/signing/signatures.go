package signing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

var signature_algos = make(map[string]Algorithm)

func init() {
	var name = "sha256"

	RegisterSignatureAlgos(&SignatureAlgorithm{
		Algorithm: name,
		SignatureFn: func(ctx context.Context, data, salt, key []byte) ([]byte, error) {
			var h = hmac.New(sha256.New, key)
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
			var h = hmac.New(sha256.New, key)
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
			if !hmac.Equal(h.Sum(nil), signature) {
				return ErrSignatureMismatch
			}
			return nil
		},
	})
}

func getSignatureAlgo(name string) (Algorithm, bool) {
	algo, ok := signature_algos[name]
	return algo, ok
}

func RegisterSignatureAlgos(algo ...Algorithm) {
	for _, a := range algo {
		signature_algos[a.Name()] = a
	}
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
