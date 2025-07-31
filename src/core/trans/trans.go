package trans

import (
	"context"
)

var DefaultBackend TranslationBackend = &SprintBackend{}

const PACKAGE_PATH = "github.com/Nigel2392/go-django/src/core/trans"

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
	Pluralize(ctx context.Context, singular, plural Untranslated, n int) Translation
	Pluralizef(ctx context.Context, singular, plural Untranslated, n int, args ...any) Translation
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

func P(ctx context.Context, singular, plural Untranslated, n int, args ...any) Translation {
	if len(args) == 0 {
		return DefaultBackend.Pluralize(ctx, singular, plural, n)
	}
	return DefaultBackend.Pluralizef(ctx, singular, plural, n, args...)
}

func LocaleFromContext(ctx context.Context) Locale {
	if DefaultBackend == nil {
		return ""
	}
	return DefaultBackend.Locale(ctx)
}
