package trans

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/deconstruct"
	"golang.org/x/text/language"
)

func init() {
	deconstruct.Constructors.Register("django.translations.Lazy", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		return Lazy{Text: kwargs["text"].(string), Args: kwargs["args"].([]any)}, nil
	})
}

var DefaultBackend TranslationBackend = &SprintBackend{}

const PACKAGE_PATH = "github.com/Nigel2392/go-django/src/core/trans"

var TRANSLATIONS_DEFAULT_LOCALE = language.English

type (
	Locale        = string
	Translation   = string
	Untranslated  = string
	TranslateFunc func(ctx context.Context) string
	Lazy          struct {
		Text string
		Args []any
	}

	Translatable any // Lazy | TranslateFunc
)

func L(text string, args ...any) Lazy {
	return Lazy{
		Text: text,
		Args: args,
	}
}

func (l Lazy) T(ctx context.Context) string {
	return T(ctx, l.Text, l.Args...)
}

func (l Lazy) Deconstruct() (string, []any, map[string]any) {
	return "django.translations.Lazy", []any{}, map[string]any{
		"args": l.Args,
		"text": l.Text,
	}
}

type TranslationBackend interface {
	Translate(ctx context.Context, v Untranslated) Translation
	Translatef(ctx context.Context, v Untranslated, args ...any) Translation
	Pluralize(ctx context.Context, singular, plural Untranslated, n int) Translation
	Pluralizef(ctx context.Context, singular, plural Untranslated, n int, args ...any) Translation
	TimeFormat(ctx context.Context, short bool) Translation
}

// S is a shortcut for translating a string with the default backend.
//
// It returns a function that can be used in templates to translate the string.
// The function takes a context and returns a Translation (alias for string).
func S(v Untranslated, args ...any) TranslateFunc {
	return func(ctx context.Context) Translation {
		return T(ctx, v, args...)
	}
}

// SP is a shortcut for pluralizing a string with the default backend.
//
// It returns a function that can be used in templates to pluralize the string.
// The function takes a context, singular and plural forms, and a count.
// It returns a Translation (alias for string) that can be used in templates.
func SP(singular, plural Untranslated, n any, args ...any) TranslateFunc {
	return func(ctx context.Context) Translation {
		return P(ctx, singular, plural, n, args...)
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
func P(ctx context.Context, singular, plural Untranslated, n any, args ...any) Translation {

	var val int64
	var rV = reflect.ValueOf(n)
	switch rV.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = rV.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val = int64(rV.Uint())
	case reflect.Slice, reflect.Array, reflect.Map:
		val = int64(rV.Len())
	case reflect.String:
		v, err := strconv.ParseInt(rV.String(), 10, 64)
		if err != nil {
			panic(fmt.Errorf("failed to parse string %q as int: %w", rV.String(), err))
		}
		val = v
	default:
		panic(fmt.Errorf("unsupported type %s for pluralization", rV.Kind()))
	}

	if len(args) == 0 {
		return DefaultBackend.Pluralize(ctx, singular, plural, int(val))
	}
	return DefaultBackend.Pluralizef(ctx, singular, plural, int(val), args...)
}

func GetTextFunc(v any) (fn TranslateFunc) {
	if v == nil {
		return nil
	}

	switch v := v.(type) {
	case string:
		fn = S(v)
	case func() string:
		fn = func(_ context.Context) string { return v() }
	case func(ctx context.Context) string:
		fn = v
	case TranslateFunc:
		fn = v
	case Lazy:
		fn = v.T
	default:
		panic(fmt.Errorf(
			"unsupported type %T for translation", v,
		))
	}

	return fn
}

func GetText(ctx context.Context, v any) (string, bool) {
	var fn = GetTextFunc(v)
	if fn == nil {
		return "", false
	}

	return fn(ctx), true
}

const (
	SHORT_TIME_FORMAT = "SHORT_TIME_FORMAT"
	LONG_TIME_FORMAT  = "LONG_TIME_FORMAT"
)

// Time formats a time.Time value into a Translation (alias for string) using the specified format.
//
// It parses the format string and replaces format specifiers with their corresponding values.
//
// Values such as weekday names and month names are localized based on the current context's locale.
//
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
		timeInfo = newTimeInfo(t)
		flags    = []byte{
			'-',
			'!',
		}

		text           strings.Builder
		formatting     bool
		currentFlags   []byte
		currentFlagMap = make(map[byte]bool, len(flags))
	)

	switch format {
	case SHORT_TIME_FORMAT:
		format = DefaultBackend.TimeFormat(ctx, true)
	case LONG_TIME_FORMAT:
		format = DefaultBackend.TimeFormat(ctx, false)
	}

	for i := 0; i < len(format); i++ {
		if format[i] == '%' && !formatting {
			formatting = true
			currentFlags = make([]byte, 0, len(flags))
			clear(currentFlagMap)
			continue
		}

		if format[i] == '%' && formatting {

			// panic if we have something like %..%
			if len(currentFlags) > 0 {
				panic(fmt.Errorf(
					"unexpected %% in time format %s at position %d", format, i,
				))
			}

			// set formatting to false if we have %%
			// this is used to escape the percent sign
			formatting = false
		}

		if !formatting {
			text.WriteByte(format[i])
			continue
		}

		if slices.Contains(flags, format[i]) {

			if _, ok := currentFlagMap[format[i]]; ok {
				panic(fmt.Errorf(
					"duplicate flag %s in time format %s", string(format[i]), format,
				))
			}

			currentFlags = append(currentFlags, format[i])
			currentFlagMap[format[i]] = true
			continue
		}

		// currently formatting
		var formatter, ok = formatMap[format[i]]
		if !ok {
			panic(fmt.Errorf(
				"unknown format specifier %s in time format %s", string(format[i]), format,
			))
		}

		// call the format function with the current context, time info and specified flags
		for _, flag := range currentFlags {
			if !formatter.supportsFlag(flag) {
				panic(fmt.Errorf(
					"format specifier %s does not support flag %s in time format %s",
					string(format[i]), string(flag), format,
				))
			}
		}

		var translated = formatter.format(ctx, timeInfo, currentFlagMap)
		text.WriteString(translated)

		// reset formatting state
		formatting = false
		currentFlags = make([]byte, 0, len(flags))
		clear(currentFlagMap)
	}

	return text.String()
}
