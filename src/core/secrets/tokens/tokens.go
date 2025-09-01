package tokens

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/secrets"
	"github.com/Nigel2392/go-django/src/core/secrets/algorithm"
)

func iterParts(parts []any) iter.Seq2[string, error] {
	return iter.Seq2[string, error](func(yield func(string, error) bool) {
		for _, p := range parts {
			var value string

			if v, ok := p.(time.Time); ok {
				if !yield(strconv.FormatInt(v.Unix(), 10), nil) {
					return
				}
				continue
			}

			var rVal = reflect.ValueOf(p)
			switch rVal.Kind() {
			case reflect.String:
				value = rVal.String()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				value = strconv.FormatInt(rVal.Int(), 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				value = strconv.FormatUint(rVal.Uint(), 10)
			case reflect.Float32, reflect.Float64:
				value = strconv.FormatFloat(rVal.Float(), 'f', -1, 64)
			case reflect.Bool:
				value = strconv.FormatBool(rVal.Bool())
			default:
				if !yield("", fmt.Errorf("unsupported part type: %T", p)) {
					return
				}
			}

			if !yield(value, nil) {
				return
			}
		}
	})
}

type ObjectPartsProvider interface {
	TokenParts(obj any) ([]any, error)
}

type PartsProvider interface {
	// The only valid part types are:
	// - string
	// - number (int, float, uint, etc..)
	// - boolean
	// - time.Time
	TokenParts() ([]any, error)
}

type TokenGenerator interface {
	MakeToken(ctx context.Context, obj any) (string, error)
	CheckToken(ctx context.Context, obj any, token string) (bool, error)
}

type ResetTokenGenerator struct {
	KeySalt         string
	Secret          secrets.SecretKey
	SecretFallbacks []secrets.SecretKey
	Algorithm       algorithm.Algorithm
	Provider        ObjectPartsProvider
	MaxAge          time.Duration
}

func NewResetTokenGenerator(algo string, keySalt string, secret secrets.SecretKey, fallbacks []secrets.SecretKey, maxAge time.Duration, provider ObjectPartsProvider) *ResetTokenGenerator {
	var algorithm, ok = algorithm.GetSignatureAlgo(algo)
	if !ok {
		panic("unknown algorithm: " + algo)
	}

	if provider == nil {
		provider = &ifacePartsProvider{}
	}

	return &ResetTokenGenerator{
		Algorithm:       algorithm,
		KeySalt:         keySalt,
		Secret:          secret,
		SecretFallbacks: fallbacks,
		MaxAge:          maxAge,
		Provider:        provider,
	}
}

func (g *ResetTokenGenerator) MakeToken(ctx context.Context, obj any) (string, error) {
	return g.MakeTokenWithTimestamp(ctx, obj, time.Now().Unix())
}

func (g *ResetTokenGenerator) CheckToken(ctx context.Context, obj any, token string) (valid bool, err error) {
	if obj == nil && token == "" {
		return false, nil
	}

	var timestamp, sig string
	var parts = strings.SplitN(token, "-", 2)
	if len(parts) != 2 {
		return false, algorithm.ErrBadSignature.Wrap(
			"invalid token format",
		)
	}

	timestamp = parts[0]
	sig = parts[1]

	var ts int64
	if ts, err = strconv.ParseInt(timestamp, 10, 64); err != nil {
		return false, algorithm.ErrBadSignature.Wrap(
			"invalid timestamp format",
		)
	}

	var sigBytes []byte
	if sigBytes, err = algorithm.DecodeString(sig); err != nil {
		return false, algorithm.ErrBadSignature.Wrap(
			"invalid signature format",
		)
	}

	data, err := g.MakeHashValue(obj, ts)
	if err != nil {
		return false, err
	}

	var errList = make([]error, 0, 1+len(g.SecretFallbacks))
	for _, secret := range append([]secrets.SecretKey{g.Secret}, g.SecretFallbacks...) {

		if err = g.Algorithm.Verify(ctx, []byte(data), sigBytes, []byte(g.KeySalt), secret); err != nil {
			errList = append(errList, err)
			continue
		}

		valid = true
		break
	}

	if len(errList) > 0 && !valid {
		return false, errors.Join(errList...)
	}

	if g.MaxAge > 0 {
		var age = time.Since(time.Unix(ts, 0))
		if age > g.MaxAge {
			return false, algorithm.ErrBadSignature.Wrap(
				"token has expired",
			)
		}
	}

	return valid, nil
}

func (g *ResetTokenGenerator) MakeTokenWithTimestamp(ctx context.Context, obj any, timestamp int64) (string, error) {
	var hashValue, err = g.MakeHashValue(obj, timestamp)
	if err != nil {
		return "", err
	}

	sig, err := g.Algorithm.Signature(ctx, []byte(hashValue), []byte(g.KeySalt), g.Secret)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d-%s", timestamp, algorithm.EncodeToString(sig)), nil
}

func (g *ResetTokenGenerator) MakeHashValue(obj any, timestamp int64) (string, error) {
	var sb = new(strings.Builder)

	var parts, err = g.Provider.TokenParts(obj)
	if err != nil {
		return "", err
	}

	var index int
	for part, err := range iterParts(append(parts, timestamp)) {
		if err != nil {
			return "", err
		}

		sb.WriteString(part)
		index++
	}

	return sb.String(), nil
}
