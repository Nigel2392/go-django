# Secrets, Signing, Encryption, and Tokens

The `github.com/Nigel2392/go-django/src/core/secrets` package and its five subpackages provide cryptographic signing, authenticated encryption, key rotation, password-reset style token generation, and sensitive-field detection.

---

## Core Package (`src/core/secrets`)

### Key Types and Settings

```go
package secrets

type SecretKey []byte

func (s SecretKey) String() string
func (s SecretKey) Bytes() []byte
func (s SecretKey) IsZero() bool

const (
	APPVAR_SECRET_KEY           = "APPVAR_SECRET_KEY"           // string
	APPVAR_SECRET_KEY_FALLBACKS = "APPVAR_SECRET_KEY_FALLBACKS" // []secrets.SecretKey
	APPVAR_SIGNER_BACKEND       = "APPVAR_SIGNER_BACKEND"       // signing.Signer
	APPVAR_ENCRYPTION_BACKEND   = "APPVAR_ENCRYPTION_BACKEND"   // encryption.Encryption
)

func SECRET_KEY() SecretKey
func SECRET_KEY_FALLBACKS() []SecretKey
func SIGNER_BACKEND() signing.Signer
func ENCRYPTION_BACKEND() (encryption.Encryption, error)
```

- **`APPVAR_SECRET_KEY`**: Must be a `string`. At startup, `secrets` registers a `TagSettings` check (`secrets.missing.key`) that fails with `checks.CRITICAL` if `APPVAR_SECRET_KEY` is not set (printing a freshly generated 32-character key in `Hint`). Calling `SECRET_KEY()` without `APPVAR_SECRET_KEY` configured panics via `assert.Fail`.
- **`APPVAR_SECRET_KEY_FALLBACKS`**: Must be configured as **`[]secrets.SecretKey`** (not `[]string`), because `SECRET_KEY_FALLBACKS()` calls `django.ConfigGet(..., make([]SecretKey, 0))`. Passing `[]string` will silently fall back to an empty slice.
- **`SIGNER_BACKEND()`**: Returns the configured `signing.Signer`, or constructs a default `signing.BaseSigner` using `SECRET_KEY()`, separator `":"`, salt `"godjango.secrets"`, algorithm `"sha256"`, and `SECRET_KEY_FALLBACKS()`.
- **`ENCRYPTION_BACKEND()`**: Returns the configured `encryption.Encryption` backend, or lazily builds `encryption.DEFAULT` (`encryption.AES`) with `SECRET_KEY()` and `SECRET_KEY_FALLBACKS()` and caches it into `django.Global.Settings` under `APPVAR_ENCRYPTION_BACKEND`.

### Top-Level Convenience Functions

```go
func Encrypt(ctx context.Context, data []byte) ([]byte, error)
func Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
func Sign(ctx context.Context, data []byte) (string, error)
func Unsign(ctx context.Context, signed string) ([]byte, error)
func SignObject(ctx context.Context, obj interface{}) (string, error)
func UnsignObject(ctx context.Context, signed string, obj interface{}) error
```

```go
package main

import (
	"context"
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/secrets"
)

func main() {
	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			// 32-byte key required for AES-256-GCM encryption
			secrets.APPVAR_SECRET_KEY: "0123456789abcdef0123456789abcdef",
			secrets.APPVAR_SECRET_KEY_FALLBACKS: []secrets.SecretKey{
				secrets.SecretKey("fedcba9876543210fedcba9876543210"),
			},
		}),
	)
	if err := app.Initialize(); err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Encrypt and decrypt bytes
	ciphertext, err := secrets.Encrypt(ctx, []byte("sensitive payload"))
	if err != nil {
		panic(err)
	}
	plaintext, err := secrets.Decrypt(ctx, ciphertext)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(plaintext))

	// Sign and verify structured JSON objects
	token, err := secrets.SignObject(ctx, map[string]any{"user_id": 42})
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err := secrets.UnsignObject(ctx, token, &out); err != nil {
		panic(err)
	}
	fmt.Println(out["user_id"])
}
```

---

## Signature Algorithms (`src/core/secrets/algorithm`)

```go
import "github.com/Nigel2392/go-django/src/core/secrets/algorithm"

type Algorithm interface {
	Name() string
	Signature(ctx context.Context, data, salt, key []byte) ([]byte, error)
	Verify(ctx context.Context, data, signature, salt, key []byte) error
}

type SignatureAlgorithm struct {
	Algorithm   string
	SignatureFn func(ctx context.Context, data, salt, key []byte) ([]byte, error)
	VerifyFn    func(ctx context.Context, data, signature []byte, salt, key []byte) error
}

func NewSignatureAlgorithm(name string, hashFunc func() hash.Hash) Algorithm
func EncodeToString(data []byte) string
func DecodeString(s string) ([]byte, error)
func GetSignatureAlgo(name string) (Algorithm, bool)
func RegisterSignatureAlgos(algo ...Algorithm)
```

- **`NewSignatureAlgorithm`**: Wraps a `hash.Hash` constructor (such as `sha256.New` or `sha512.New`) using HMAC and constant-time comparison (`hmac.Equal`). `"sha256"` is registered automatically during package `init()`.
- **`EncodeToString` / `DecodeString`**: URL-safe base64 encoding (`A-Z`, `a-z`, `0-9`, `-`, `_`) with padding disabled (`base64.NoPadding`).
- **Errors and codes**:
  - `ErrSigning` (`ErrorCodeSigningError = "signing.signature.error"`)
  - `ErrBadSignature` (`ErrorCodeBadSignature = "signing.signature.bad"`)
  - `ErrSignatureMismatch` (`ErrorCodeSignatureMisMatch = "signing.signature.mismatch"`)
  - `ErrSignatureExpired` (`ErrorCodeSignatureExpired = "signing.signature.expired"`)
  - `ErrUnknownSignatureAlgorithm` (`ErrorSignerAlgorithm = "signing.signer.algorithm"`)

---

## Encryption (`src/core/secrets/encryption`)

```go
import "github.com/Nigel2392/go-django/src/core/secrets/encryption"

type Encryption interface {
	Encrypt(ctx context.Context, data []byte) ([]byte, error)
	Decrypt(ctx context.Context, data []byte) ([]byte, error)
}

const (
	DEFAULT = "encryption.default.engine"
	AES     = "encryption.default.AES"
)

var (
	ErrEncryption = errors.New(ErrorCodeEncryptionError, "error encrypting data")
	ErrDecryption = errors.New(ErrorCodeDecryptionError, "error decrypting data")
)

func GetCrypto(which string, key []byte, fallbackKeys [][]byte) (Encryption, error)
func RegisterCrypto(which string, newEncryption func(key []byte, fallbackKeys [][]byte) (Encryption, error)) (overwritten bool)
func ContextWithAdditionalData(c context.Context, additionalData []byte) context.Context
func AdditionalDataFromContext(c context.Context) ([]byte, bool)
```

### AES-GCM Details and Key Rotation

Both `DEFAULT` and `AES` use AES-GCM:

- **Key lengths**: `aes.NewCipher` requires every key (`key` and every entry in `fallbackKeys`) to be **16, 24, or 32 bytes** (AES-128, AES-192, or AES-256). Any other length returns `ErrEncryption` wrapping `aes.KeySizeError`.
- **Payload format**: `[4-byte FNV-32a key ID][Nonce][Ciphertext]`. During `Decrypt`, the leading 4-byte key ID selects the matching primary or fallback cipher so rotated keys continue decrypting older ciphertexts without trial decryption.
- **Additional Authenticated Data (AAD)**: Pass `encryption.ContextWithAdditionalData(ctx, aad)` to both `Encrypt` and `Decrypt` to bind extra context (such as a user or tenant ID) to the authentication tag.
- **Unknown engine**: `GetCrypto` returns `(nil, nil)` if `which` is not registered.

---

## Signing (`src/core/secrets/signing`)

```go
import "github.com/Nigel2392/go-django/src/core/secrets/signing"

type Signer interface {
	Separator() string
	Salt() []byte
	Key() []byte
	FallbackKeys() [][]byte
	Sign(ctx context.Context, data []byte) (string, error)
	Unsign(ctx context.Context, signature string) ([]byte, error)
	Algorithm() algorithm.Algorithm
}
```

### `BaseSigner` and `TimestampSigner`

```go
type BaseSigner struct { /* unexported fields */ }

func NewBaseSigner(key []byte, sep string, salt []byte, algo string, fallbackKeys [][]byte) *BaseSigner
func (s *BaseSigner) Signature(ctx context.Context, data []byte) ([]byte, error)
func (s *BaseSigner) Sign(ctx context.Context, data []byte) (string, error)
func (s *BaseSigner) Unsign(ctx context.Context, signedData string) ([]byte, error)
// Plus Separator, Salt, Key, Algorithm, FallbackKeys

type TimestampSigner struct {
	Signer
	MaxAge time.Duration
}

func NewTimestampSigner(key []byte, sep string, salt []byte, algo string, fallbackKeys [][]byte, maxAge time.Duration) *TimestampSigner
func (s *TimestampSigner) Sign(ctx context.Context, data []byte) (string, error)
func (s *TimestampSigner) Unsign(ctx context.Context, signedData string) ([]byte, error)
```

`signing` re-exports `ErrSigning`, `ErrBadSignature`, `ErrSignatureMismatch`, `ErrSignatureExpired`, and `ErrUnknownSignatureAlgorithm` from `algorithm`. `BaseSigner.Unsign` tries the primary key followed by `fallbackKeys`. `TimestampSigner.Unsign` returns `ErrSignatureExpired` if `MaxAge > 0 && time.Since(timestamp) > MaxAge`.

### Object Signing

```go
type Encoder interface {
	Encode(v any) error
}

type Decoder interface {
	Decode(v any) error
}

var (
	DefaultEncoder = JSONEncoder
	DefaultDecoder = JSONDecoder
)

func SignObject(ctx context.Context, signer Signer, obj interface{}) (string, error)
func UnsignObject(ctx context.Context, signer Signer, signed string, obj interface{}) error
func SignObjectEncoder(ctx context.Context, signer Signer, obj interface{}, encoder func(w io.Writer) Encoder) (string, error)
func UnsignObjectDecoder(ctx context.Context, signer Signer, signed string, obj interface{}, decoder func(r io.Reader) Decoder) error
func JSONEncoder(w io.Writer) Encoder
func JSONDecoder(r io.Reader) Decoder
```

### Signing Caveats

- **Non-empty keys/salt and unsafe separators**: `NewBaseSigner` panics if `len(key) == 0 || len(salt) == 0` or if `sep` matches `^[A-z0-9-_=-]*$` (including an empty string `""` or base64url characters). Always pass a non-alphanumeric separator such as `":"` or `"."`.
- **1-byte separator assumption in `TimestampSigner`**: `TimestampSigner.Unsign` extracts the timestamp using `dataWithTimestamp[lastIdx+1:]`, which assumes a single-byte separator such as `":"` or `"."`. Multi-byte separators will fail timestamp parsing.

---

## Tokens (`src/core/secrets/tokens`)

The `tokens` package generates and validates stateless tokens bound to an object's current state (such as a user ID, password hash timestamp, or last login time):

```go
import "github.com/Nigel2392/go-django/src/core/secrets/tokens"

type ObjectPartsProvider interface {
	TokenParts(obj any) ([]any, error)
}

type PartsProvider interface {
	// Valid part types: string, numeric types, bool, time.Time
	TokenParts() ([]any, error)
}

type PartsProviderFunc[T any] func(obj T) ([]any, error)

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

func NewResetTokenGenerator(
	algo string,
	keySalt string,
	secret secrets.SecretKey,
	fallbacks []secrets.SecretKey,
	maxAge time.Duration,
	provider ObjectPartsProvider,
) *ResetTokenGenerator

func (g *ResetTokenGenerator) MakeToken(ctx context.Context, obj any) (string, error)
func (g *ResetTokenGenerator) CheckToken(ctx context.Context, obj any, token string) (valid bool, err error)
func (g *ResetTokenGenerator) MakeTokenWithTimestamp(ctx context.Context, obj any, timestamp int64) (string, error)
func (g *ResetTokenGenerator) MakeHashValue(obj any, timestamp int64) (string, error)
```

If `provider == nil`, `NewResetTokenGenerator` expects the target object to implement `PartsProvider`. `NewResetTokenGenerator` panics if `algo` is not registered in `algorithm`.

```go
gen := tokens.NewResetTokenGenerator(
	"sha256",
	"myapp.password.reset",
	secrets.SECRET_KEY(),
	secrets.SECRET_KEY_FALLBACKS(),
	24*time.Hour,
	tokens.PartsProviderFunc[*User](func(u *User) ([]any, error) {
		return []any{u.ID, u.PasswordHash, u.LastLogin}, nil
	}),
)

token, err := gen.MakeToken(ctx, user)
valid, err := gen.CheckToken(ctx, user, token)
```

### Token Caveat

When `g.MaxAge > 0` and a token has expired, `ResetTokenGenerator.CheckToken` returns `false, algorithm.ErrBadSignature.Wrap("token has expired")`. Check `errors.Is(err, algorithm.ErrBadSignature)` rather than `algorithm.ErrSignatureExpired`.

---

## Sensitive Field Detection (`src/core/secrets/safety`)

The `safety` package helps prevent accidental exposure of secret values in logs, error reports, or debug UIs:

```go
import "github.com/Nigel2392/go-django/src/core/secrets/safety"

type SecretChecker func(ctx context.Context, fieldName string) (isSecret bool)

func RegisterChecker(checker SecretChecker)
func IsSecretField(ctx context.Context, fieldName string) bool
```

During `init()`, `safety` registers a default checker that strips non-alphanumeric/dot characters (`[^a-zA-Z0-9.]`), lowercases the field name, and checks a built-in dictionary of common credential names (such as `password`, `token`, `apikey`, `secret`, `dbpassword`, `jwtsecret`, `sessionid`, and `csrftoken`). Use `RegisterChecker` to flag additional application-specific field names.
