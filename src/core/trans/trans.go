package trans

import (
	"context"
)

var DefaultBackend TranslationBackend = &SprintBackend{}

type (
	Locale             = string
	Translation        = string
	Untranslated       = string
	TranslationTextMap = map[Untranslated]Translation
	LocaleMap          = map[Locale]TranslationTextMap
)

type TranslationBackend interface {
	Translate(ctx context.Context, v Untranslated) Translation
	Translatef(ctx context.Context, v Untranslated, args ...any) Translation
	Locale(ctx context.Context) Locale
}

func S(v Untranslated, args ...any) func(ctx context.Context) Translation {
	return func(ctx context.Context) Translation {
		return T(ctx, v, args...)
	}
}

func T(ctx context.Context, v Untranslated, args ...any) Translation {
	if len(args) == 0 {
		return DefaultBackend.Translate(ctx, v)
	}
	return DefaultBackend.Translatef(ctx, v, args...)
}

func LocaleFromContext(ctx context.Context) Locale {
	if DefaultBackend == nil {
		return ""
	}
	return DefaultBackend.Locale(ctx)
}

var appTranslationsRegistry = make(map[string]LocaleMap)
