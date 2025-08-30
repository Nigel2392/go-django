package secrets

import (
	"context"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/secrets/signing"
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
	APPVAR_SIGNER_BACKEND = "APPVAR_SIGNER_BACKEND" // internals/signing.Signer
)

var _ = checks.Register(checks.TagSettings, func(ctx context.Context, app any, settings Settings) (messages []checks.Message) {
	if _, ok := settings.Get(APPVAR_SECRET_KEY); !ok {
		messages = append(messages, checks.Message{
			Type:   checks.CRITICAL,
			Object: settings,
			ID:     "secrets.missing.key",
			Text:   "The APPVAR_SECRET_KEY option is not present in the settings",
			Hint:   "You must set the APPVAR_SECRET_KEY setting to a unique, unpredictable value in the application's settings",
		})
	}
	return
})

func SECRET_KEY() SecretKey {
	var key, ok = django.ConfigGetOK[string](
		django.Global.Settings,
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
	var fallbacks = django.ConfigGet(
		django.Global.Settings,
		APPVAR_SECRET_KEY_FALLBACKS,
		make([]SecretKey, 0),
	)
	return fallbacks
}

func SIGNER_BACKEND() signing.Signer {
	var signer, ok = django.ConfigGetOK[signing.Signer](
		django.Global.Settings,
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

func keyBytes(fallbacks []SecretKey) [][]byte {
	var keys = make([][]byte, 0, len(fallbacks))
	for _, key := range fallbacks {
		keys = append(keys, key.Bytes())
	}
	return keys
}
