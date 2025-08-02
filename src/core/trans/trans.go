package trans

import (
	"context"
	"fmt"
	"strings"
	"time"
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

func Time(ctx context.Context, t time.Time, format string) Translation {
	var (
		timeInfo   = newTimeInfo(t)
		text       strings.Builder
		formatting bool
		flag       bool
	)

	for i := 0; i < len(format); i++ {
		if format[i] == '%' && !formatting {
			formatting = true
			flag = false
			continue
		}

		if format[i] == '%' && formatting {
			formatting = false
		}

		if !formatting {
			text.WriteByte(format[i])
			continue
		}

		if format[i] == '-' {
			flag = true
			continue
		}

		// currently formatting
		var formatKey string
		if flag {
			formatKey = string([]byte{'-', format[i]})
		} else {
			formatKey = string(format[i])
		}

		var formatFunc, ok = formatMap[formatKey]
		if !ok {
			panic(fmt.Errorf(
				"unknown format specifier %s in time format %s", formatKey, format,
			))
		}

		var translated = formatFunc(ctx, timeInfo)
		text.WriteString(translated)
		formatting = false
	}

	return Translation(text.String())
}

func LocaleFromContext(ctx context.Context) Locale {
	if DefaultBackend == nil {
		return ""
	}
	return DefaultBackend.Locale(ctx)
}
