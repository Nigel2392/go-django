package signing

import (
	"context"
	"encoding/base64"
	"regexp"
)

type Algorithm interface {
	Name() string
	Signature(ctx context.Context, data, salt, key []byte) ([]byte, error)
	Verify(ctx context.Context, data, signature, salt, key []byte) error
}

type Signer interface {
	Separator() string
	Salt() []byte
	Key() []byte
	Algorithm() Algorithm
	FallbackKeys() [][]byte
	Sign(ctx context.Context, data []byte) (string, error)
	Unsign(ctx context.Context, signature string) ([]byte, error)
}

var unsafe_sep_re = regexp.MustCompile("^[A-z0-9-_=-]*$")
var safeB64Encoding = base64.
	NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_").
	WithPadding(base64.NoPadding)
