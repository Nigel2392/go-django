package signing_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/secrets/signing"
)

func TestSignature_DifferentKeysProduceDifferentSigs(t *testing.T) {
	ctx := context.Background()
	signer1 := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "sha256", nil)
	signer2 := signing.NewBaseSigner([]byte("predictable-secret2"), ":", []byte("salt"), "sha256", nil)

	tests := [][]byte{
		[]byte("hello"),
		[]byte("3098247:529:087:"),
		[]byte("\u2019"),
	}

	for _, in := range tests {
		sig1, err := signer1.Signature(ctx, in)
		if err != nil {
			t.Fatalf("signature() err: %v", err)
		}
		sig2, err := signer2.Signature(ctx, in)
		if err != nil {
			t.Fatalf("signature() err: %v", err)
		}
		if string(sig1) == string(sig2) {
			t.Fatalf("expected different signatures for different keys; got equal for %q", in)
		}
	}
}

func TestSignature_WithSaltAffectsSignature(t *testing.T) {
	ctx := context.Background()
	s1 := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("extra-salt"), "sha256", nil)
	s2 := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("other-salt"), "sha256", nil)

	in := []byte("hello")
	sig1, err := s1.Signature(ctx, in)
	if err != nil {
		t.Fatalf("signature err: %v", err)
	}
	sig2, err := s2.Signature(ctx, in)
	if err != nil {
		t.Fatalf("signature err: %v", err)
	}
	if string(sig1) == string(sig2) {
		t.Fatalf("expected different signatures for different salts")
	}
}

func TestInvalidAlgorithm(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "whatever", nil)
	_, err := signer.Sign(ctx, []byte("hello"))
	if !errors.Is(err, signing.ErrUnknownSignatureAlgorithm) {
		t.Fatalf("expected ErrUnknownSignatureAlgorithm, got %v", err)
	}
}

func TestSignUnsign_Reversible(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "sha256", nil)

	examples := []string{
		"q;wjmbk;wkmb",
		"3098247529087",
		"3098247:529:087:",
		"jkw osanteuh ,rcuh nthu aou oauh ,ud du",
		"\u2019",
	}

	for _, ex := range examples {
		signed, err := signer.Sign(ctx, []byte(ex))
		if err != nil {
			t.Fatalf("sign err: %v", err)
		}
		if signed == ex {
			t.Fatalf("expected signed string to differ from input")
		}
		raw, err := signer.Unsign(ctx, signed)
		if err != nil {
			t.Fatalf("unsign err: %v", err)
		}
		if ex != string(raw) {
			t.Fatalf("unsign mismatch: want %q, got %q", ex, string(raw))
		}
	}
}

func TestSignUnsign_NonStringLikeValues(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "sha256", nil)

	// Mirror Python intent by round-tripping textual forms of non-strings.
	values := []any{
		123,
		1.23,
		true,
		time.Now().Truncate(24 * time.Hour).Format("2006-01-02"),
	}

	for _, v := range values {
		val := fmt.Sprint(v)
		signed, err := signer.Sign(ctx, []byte(val))
		if err != nil {
			t.Fatalf("sign err: %v", err)
		}
		if signed == val {
			t.Fatalf("expected signed string to differ from input")
		}
		raw, err := signer.Unsign(ctx, signed)
		if err != nil {
			t.Fatalf("unsign err: %v", err)
		}
		if string(raw) != val {
			t.Fatalf("unsign mismatch: want %q, got %q", val, string(raw))
		}
	}
}

func TestUnsign_DetectsTampering(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "sha256", nil)

	value := "Another string"
	signed, err := signer.Sign(ctx, []byte(value))
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}

	// Control: valid
	got, err := signer.Unsign(ctx, signed)
	if err != nil || string(got) != value {
		t.Fatalf("control unsign failed: %v, %q", err, string(got))
	}

	transforms := []func(string) string{
		func(s string) string { return strings.ToUpper(s) },
		func(s string) string { return s + "a" },
		func(s string) string { return "a" + s[1:] },
		func(s string) string { return strings.ReplaceAll(s, ":", "") },
	}

	for i, tf := range transforms {
		if _, err := signer.Unsign(ctx, tf(signed)); !errors.Is(err, signing.ErrBadSignature) {
			t.Fatalf("transform %d: expected ErrBadSignature, got %v", i, err)
		}
	}
}

func TestUnsign_NoSeparator(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-secret"), ":", []byte("salt"), "sha256", nil)
	_, err := signer.Unsign(ctx, "no-sep-here")
	if !errors.Is(err, signing.ErrBadSignature) {
		t.Fatalf("expected ErrBadSignature when separator missing, got %v", err)
	}
}

func TestCustomSeparators_Work(t *testing.T) {
	ctx := context.Background()
	separators := []string{"/", "*sep*", ","}

	for _, sep := range separators {
		signer := signing.NewBaseSigner([]byte("predictable-secret"), sep, []byte("salt"), "sha256", nil)
		signed, err := signer.Sign(ctx, []byte("foo"))
		if err != nil {
			t.Fatalf("sign err (sep=%q): %v", sep, err)
		}
		val, err := signer.Unsign(ctx, signed)
		if err != nil {
			t.Fatalf("unsign err (sep=%q): %v: %s", sep, err, signed)
		}
		if string(val) != "foo" {
			t.Fatalf("unsign mismatch (sep=%q): got %q", sep, string(val))
		}
	}
}

func TestVerifyWithNonDefaultKey_UsesFallbacks(t *testing.T) {
	ctx := context.Background()

	oldSigner := signing.NewBaseSigner([]byte("secret"), ":", []byte("salt"), "sha256", nil)
	newSigner := signing.NewBaseSigner([]byte("newsecret"), ":", []byte("salt"), "sha256",
		[][]byte{[]byte("othersecret"), []byte("secret")},
	)

	signed, err := oldSigner.Sign(ctx, []byte("abc"))
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}

	out, err := newSigner.Unsign(ctx, signed)
	if err != nil {
		t.Fatalf("unsign err: %v", err)
	}
	if string(out) != "abc" {
		t.Fatalf("unsign mismatch: got %q", string(out))
	}
}

func TestSignUnsignMultipleKeys_DefaultKeyAlsoValid(t *testing.T) {
	ctx := context.Background()

	signer := signing.NewBaseSigner([]byte("secret"), ":", []byte("salt"), "sha256",
		[][]byte{[]byte("oldsecret")},
	)

	signed, err := signer.Sign(ctx, []byte("abc"))
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}
	out, err := signer.Unsign(ctx, signed)
	if err != nil {
		t.Fatalf("unsign err: %v", err)
	}
	if string(out) != "abc" {
		t.Fatalf("unsign mismatch: got %q", string(out))
	}
}

func TestTimestampSigner_BasicAndExpiry(t *testing.T) {
	ctx := context.Background()
	value := []byte("hello")

	tsSigner := signing.NewTimestampSigner([]byte("predictable-key"), ":", []byte("salt"), "sha256", nil, 0)
	baseSigner := signing.NewBaseSigner([]byte("predictable-key"), ":", []byte("salt"), "sha256", nil)

	signedTS, err := tsSigner.Sign(ctx, value)
	if err != nil {
		t.Fatalf("timestamp sign err: %v", err)
	}
	signedBase, err := baseSigner.Sign(ctx, value)
	if err != nil {
		t.Fatalf("base sign err: %v", err)
	}

	// They should differ because timestamp is appended before signing.
	if signedTS == signedBase {
		t.Fatalf("expected timestamped signature to differ from base signature")
	}

	// Immediately valid when maxAge allows it.
	out, err := tsSigner.Unsign(ctx, signedTS, 5*time.Second)
	if err != nil {
		t.Fatalf("unsign timestamp err: %v", err)
	}
	if string(out) != string(value) {
		t.Fatalf("unsign timestamp mismatch: got %q", string(out))
	}

	// Expiry: wait just beyond a very small maxAge.
	short := 200 * time.Millisecond
	time.Sleep(short + 100*time.Millisecond)

	_, err = tsSigner.Unsign(ctx, signedTS, short)
	if !errors.Is(err, signing.ErrSignatureExpired) {
		t.Fatalf("expected ErrSignatureExpired, got %v", err)
	}
}

func TestBinaryKey_Works(t *testing.T) {
	ctx := context.Background()
	binaryKey := []byte{0xe7} // non-ASCII key

	s := signing.NewBaseSigner(binaryKey, ":", []byte("salt"), "sha256", nil)
	signed, err := s.Sign(ctx, []byte("foo"))
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}
	out, err := s.Unsign(ctx, signed)
	if err != nil {
		t.Fatalf("unsign err: %v", err)
	}
	if string(out) != "foo" {
		t.Fatalf("unsign mismatch with binary key: got %q", string(out))
	}
}

type SignableObject struct {
	ID   int            `json:"id"`
	Val  string         `json:"val"`
	Data map[string]any `json:"data"`
}

func TestSignObject_UnsignObject(t *testing.T) {
	ctx := context.Background()
	signer := signing.NewBaseSigner([]byte("predictable-key"), ":", []byte("salt"), "sha256", nil)

	obj := SignableObject{
		ID:   1,
		Val:  "test",
		Data: map[string]any{"foo": "bar"},
	}

	signed, err := signing.SignObject(ctx, signer, obj)
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}

	var unsignd SignableObject
	if err := signing.UnsignObject(ctx, signer, signed, &unsignd); err != nil {
		t.Fatalf("unsign err: %v", err)
	}

	// Ensure the original object and the unsign result are identical.
	if !reflect.DeepEqual(obj, unsignd) {
		t.Fatalf("unsign mismatch: got %+v", unsignd)
	}
}
