package signing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/secrets/algorithm"
)

var _ Signer = (*BaseSigner)(nil)

type BaseSigner struct {
	sep          string
	key          []byte
	salt         []byte
	algo         string
	fallbackKeys [][]byte
	sig          algorithm.Algorithm
}

func NewBaseSigner(key []byte, sep string, salt []byte, algo string, fallbackKeys [][]byte) *BaseSigner {
	if len(key) == 0 || len(salt) == 0 {
		panic("Key and salt must be provided to BaseSigner")
	}

	if unsafe_sep_re.MatchString(sep) {
		panic("Unsafe separator provided to BaseSigner")
	}

	return &BaseSigner{
		key:          key,
		sep:          sep,
		salt:         salt,
		algo:         algo,
		fallbackKeys: fallbackKeys,
	}
}

func (s *BaseSigner) setup() error {
	if len(s.key) == 0 || len(s.salt) == 0 {
		panic("Key and salt must be provided to BaseSigner")
	}

	if s.algo == "" {
		s.algo = "sha256"
	}

	if s.sep == "" {
		s.sep = ":"
	}

	if s.sig == nil {

		var ok bool
		s.sig, ok = algorithm.GetSignatureAlgo(s.algo)
		if !ok {
			return ErrUnknownSignatureAlgorithm
		}
	}

	return nil
}

func (s *BaseSigner) Separator() string {
	return s.sep
}

func (s *BaseSigner) Salt() []byte {
	return s.salt
}

func (s *BaseSigner) Key() []byte {
	return s.key
}

func (s *BaseSigner) Algorithm() algorithm.Algorithm {
	if err := s.setup(); err != nil {
		return nil
	}
	return s.sig
}

func (s *BaseSigner) FallbackKeys() [][]byte {
	return s.fallbackKeys
}

func (s *BaseSigner) Signature(ctx context.Context, data []byte) ([]byte, error) {
	if err := s.setup(); err != nil {
		return nil, err
	}

	return s.sig.Signature(ctx, data, s.salt, s.key)
}

func (s *BaseSigner) Sign(ctx context.Context, data []byte) (string, error) {
	if err := s.setup(); err != nil {
		return "", err
	}

	signature, err := s.sig.Signature(ctx, data, s.salt, s.key)
	if err != nil {
		return "", err
	}

	var signed = fmt.Sprintf(
		"%s%s%s",
		string(data),
		s.sep,
		algorithm.EncodeToString(signature),
	)

	return signed, nil
}

func (s *BaseSigner) Unsign(ctx context.Context, signedData string) ([]byte, error) {
	if err := s.setup(); err != nil {
		return nil, err
	}

	var lastIdx = strings.LastIndex(signedData, s.sep)
	if lastIdx == -1 {
		return nil, ErrBadSignature
	}

	if lastIdx+1 >= len(signedData) {
		return nil, ErrBadSignature
	}

	var data = []byte(signedData[:lastIdx])
	var signatureB64 = signedData[lastIdx+len(s.sep):]
	var signature, err = algorithm.DecodeString(signatureB64)
	if err != nil {
		return nil, ErrBadSignature
	}

	for _, key := range append([][]byte{s.key}, s.fallbackKeys...) {
		if err := s.sig.Verify(ctx, data, signature, s.salt, key); err == nil {
			return data, nil
		}
	}

	return nil, ErrBadSignature.Wrapf(
		"signature %q does not match",
		signatureB64,
	)
}

type TimestampSigner struct {
	Signer
	MaxAge time.Duration
}

func NewTimestampSigner(key []byte, sep string, salt []byte, algo string, fallbackKeys [][]byte, maxAge time.Duration) *TimestampSigner {
	return &TimestampSigner{
		Signer: NewBaseSigner(key, sep, salt, algo, fallbackKeys),
		MaxAge: maxAge,
	}
}

func (s *TimestampSigner) Sign(ctx context.Context, data []byte) (string, error) {
	var timestamp = time.Now().Unix()
	var dataWithTimestamp = []byte(fmt.Sprintf(
		"%s%s%x",
		data,
		s.Separator(),
		timestamp,
	))

	// Sign the data with the base signer
	return s.Signer.Sign(ctx, dataWithTimestamp)
}

func (s *TimestampSigner) Unsign(ctx context.Context, signedData string, maxAge time.Duration) ([]byte, error) {
	var dataWithTimestamp, err = s.Signer.Unsign(ctx, signedData)
	if err != nil {
		return nil, err
	}

	var lastIdx = strings.LastIndex(string(dataWithTimestamp), s.Separator())
	if lastIdx == -1 {
		return nil, ErrBadSignature
	}

	if lastIdx+1 >= len(dataWithTimestamp) {
		return nil, ErrBadSignature
	}

	var data = dataWithTimestamp[:lastIdx]
	var timestampHex = dataWithTimestamp[lastIdx+1:]
	var timestampInt int64
	_, err = fmt.Sscanf(string(timestampHex), "%x", &timestampInt)
	if err != nil {
		return nil, ErrBadSignature.Wrapf(
			"invalid timestamp %q: %s",
			timestampHex, err,
		)
	}

	var timestamp = time.Unix(timestampInt, 0)
	if maxAge > 0 && time.Since(timestamp) > maxAge {
		return nil, ErrSignatureExpired.Wrapf(
			"signature expired: %s > %s",
			time.Since(timestamp),
			maxAge,
		)
	}

	return data, nil
}
