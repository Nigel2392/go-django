package signing

import (
	"context"
	"regexp"

	"github.com/Nigel2392/go-django/src/core/secrets/algorithm"
)

type Signer interface {
	Separator() string
	Salt() []byte
	Key() []byte
	FallbackKeys() [][]byte
	Sign(ctx context.Context, data []byte) (string, error)
	Unsign(ctx context.Context, signature string) ([]byte, error)
	Algorithm() algorithm.Algorithm
}

var unsafe_sep_re = regexp.MustCompile("^[A-z0-9-_=-]*$")
