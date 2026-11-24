package secrets

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"unsafe"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/secrets/encryption"
	"github.com/Nigel2392/go-django/src/core/secrets/signing"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
)

type Settings interface {
	Get(key string) (any, bool)
}

const (
	// Secret key for the application
	APPVAR_SECRET_KEY = "APPVAR_SECRET_KEY" // string

	// Secret key fallbacks for the application
	APPVAR_SECRET_KEY_FALLBACKS = "APPVAR_SECRET_KEY_FALLBACKS" // []string

	// Signer backend for the application
	APPVAR_SIGNER_BACKEND = "APPVAR_SIGNER_BACKEND" // [signing.Signer]

	// Encryption backend for the application
	APPVAR_ENCRYPTION_BACKEND = "APPVAR_ENCRYPTION_BACKEND"
)

var _ = checks.Register(checks.TagSettings, func(ctx context.Context, app any, settings Settings) (messages []checks.Message) {
	if _, ok := settings.Get(APPVAR_SECRET_KEY); !ok {

		var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+=[]{}|,.?;:"

		b := make([]byte, 32)
		max := big.NewInt(int64(len(letterRunes)))
		for i := range b {
			n, err := rand.Int(rand.Reader, max)
			if err != nil {
				panic("failed to generate secret key")
			}
			b[i] = letterRunes[n.Int64()]
		}

		messages = append(messages, checks.Message{
			Type:   checks.CRITICAL,
			Object: settings,
			ID:     "secrets.missing.key",
			Text:   "The APPVAR_SECRET_KEY option is not present in the settings",
			Hint: fmt.Sprintf(
				"You must set the APPVAR_SECRET_KEY setting to a unique, unpredictable value in the application's settings. We generated one for you! [%v]",
				string(b),
			),
		})
	}
	return
})

func SECRET_KEY() SecretKey {
	var key, ok = djangoapps.CONFIG.GetOK[string](
		APPVAR_SECRET_KEY,
	)
	if !ok {
		assert.Fail(
			"Missing APPVAR_SECRET_KEY in settings",
		)
	}
	return SecretKey(key)
}

func SECRET_KEY_FALLBACKS() []SecretKey {
	var fallbacks = djangoapps.CONFIG.Get(
		APPVAR_SECRET_KEY_FALLBACKS,
		make([]SecretKey, 0),
	)
	return fallbacks
}

func SIGNER_BACKEND() signing.Signer {
	var signer, ok = djangoapps.CONFIG.GetOK[signing.Signer](
		APPVAR_SIGNER_BACKEND,
	)
	if !ok {
		signer = signing.NewBaseSigner(
			SECRET_KEY().Bytes(), ":", []byte("godjango.secrets"), "sha256",
			keyBytes(SECRET_KEY_FALLBACKS()),
		)
	}
	return signer
}

func ENCRYPTION_BACKEND() (enc encryption.Encryption, err error) {
	var ok bool
	enc, ok = djangoapps.CONFIG.GetOK[encryption.Encryption](
		APPVAR_ENCRYPTION_BACKEND,
	)
	if !ok {
		enc, err = encryption.GetCrypto(
			encryption.DEFAULT,
			SECRET_KEY().Bytes(),
			keyBytes(SECRET_KEY_FALLBACKS()),
		)

		// probably safe to set the default backend
		django.Global.Settings.Set(
			APPVAR_ENCRYPTION_BACKEND, enc,
		)
	}
	return enc, err
}

func keyBytes(fallbacks []SecretKey) [][]byte {
	return *(*[][]byte)(unsafe.Pointer(&fallbacks))
}
