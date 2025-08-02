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

// S is a shortcut for translating a string with the default backend.
//
// It returns a function that can be used in templates to translate the string.
// The function takes a context and returns a Translation (alias for string).
func S(v Untranslated, args ...any) func(ctx context.Context) Translation {
	return func(ctx context.Context) Translation {
		return T(ctx, v, args...)
	}
}

// T translates a string with the default backend.
//
// It returns a Translation (alias for string) that can be used in templates.
func T(ctx context.Context, v Untranslated, args ...any) Translation {
	if len(args) == 0 {
		return DefaultBackend.Translate(ctx, v)
	}
	return DefaultBackend.Translatef(ctx, v, args...)
}

// P is a shortcut for pluralizing a string with the default backend.
// It returns a function that can be used in templates to pluralize the string.
// The function takes a context, singular and plural forms, and a count.
// It returns a Translation (alias for string) that can be used in templates.
func P(ctx context.Context, singular, plural Untranslated, n int, args ...any) Translation {
	if len(args) == 0 {
		return DefaultBackend.Pluralize(ctx, singular, plural, n)
	}
	return DefaultBackend.Pluralizef(ctx, singular, plural, n, args...)
}

// Time formats a time.Time value into a Translation (alias for string) using the specified format.
//
// It parses the format string and replaces format specifiers with their corresponding values.
// The format codes are as follows:
//
// - %a 	short weekday na	(e.g., "Mon")
// - %A 	full weekday nam	(e.g., "Monday")
// - %w 	weekday number 		(1-7, Monday is 1, Sunday is 7)
// - %b 	short month name	(e.g., "Jan")
// - %B 	full month name 	(e.g., "January")
// - %m 	month number 		(01-12)
// - %-m	month number 		(1-12)
// - %d 	day of the month	(01-31)
// - %-d	day of the month	(1-31)
// - %y 	year 				(4 digits, e.g., 2023)
// - %-y	year 				(2 digits, e.g., 23)
// - %Y 	year 				(4 digits, e.g., 2023)
// - %-Y	year 				(2 digits, e.g., 23)
// - %H 	hour 				(00-23)
// - %-H	hour 				(0-23)
// - %I 	hour 				(01-12)
// - %-I	hour 				(1-12)
// - %M 	minute 				(00-59)
// - %-M	minute 				(0-59)
// - %S 	second 				(00-59)
// - %-S	second 				(0-59)
// - %f 	milliseconds 		(000-999)
// - %-f	milliseconds 		(0-999)
// - %F 	microseconds 		(000000-999999)
// - %-F	microseconds 		(0-999999)
// - %z 	timezone offset 	(e.g., "+02:00")
// - %Z 	timezone name 		(e.g., "UTC")
// - %j 	day of the year 	(001-366)
// - %U 	week number 		(00-53, Sunday as first day of week)
// - %W 	week number 		(00-53, Monday as first day of week)
// - %p 	AM/PM
// - %% 	an escaped percent sign
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

// LocaleFromContext retrieves the locale from the current context.
// If the default backend is not set, it returns an empty string.
func LocaleFromContext(ctx context.Context) Locale {
	if DefaultBackend == nil {
		return ""
	}
	return DefaultBackend.Locale(ctx)
}
