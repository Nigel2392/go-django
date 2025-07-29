package translations

import (
	"context"
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

var _ trans.TranslationBackend = &Translator{}

type localeContextKey struct{}

type Translator struct {
	translations    map[trans.Locale]map[trans.Untranslated]trans.Translation
	appTranslations map[string]trans.LocaleMap
}

func (t *Translator) Translate(ctx context.Context, v string) string {

	if v == "" {
		return v
	}

	var (
		locale = LocaleFromContext(ctx)
		checks = localeChecks(locale)
	)

	var app, ok = django.AppFromContext(ctx)
	if ok {
		logger.Debugf("Translating %q in app %s", v, app.Name())
	}

	appTranslations, ok := t.appTranslations[app.Name()]
	if ok {
		var t, ok = getTranslationFromMap(appTranslations, checks, v)
		if ok && t != "" {
			return t
		}

		logger.Debugf(
			"Translation for app %s, locale '%s' and key '%s' not found, checking global translations",
			app.Name(), locale.String(), v,
		)
	}

	var translation trans.Translation
	if translation, ok = getTranslationFromMap(t.translations, checks, v); ok && translation != "" {
		return translation
	}

	if translation == "" {
		logger.Debugf(
			"Translation for locale '%s' and key '%s' is empty, returning original value: %q",
			locale.String(), v, v,
		)
	} else {
		logger.Debugf(
			"Translation for locale '%s' and key '%s' not found, returning original value: %q",
			locale.String(), v, v,
		)
	}

	return v
}

func (t *Translator) Translatef(ctx context.Context, v string, args ...any) string {
	if len(args) == 0 {
		return t.Translate(ctx, v)
	}
	return fmt.Sprintf(t.Translate(ctx, v), args...)
}

func (t *Translator) Pluralize(ctx context.Context, singular, plural string, n int) string {
	if n == 1 {
		return t.Translate(ctx, singular)
	}
	return t.Translate(ctx, plural)
}

func (t *Translator) Pluralizef(ctx context.Context, singular, plural string, n int, args ...any) string {
	if n == 1 {
		return t.Translatef(ctx, singular, args...)
	}
	return t.Translatef(ctx, plural, args...)
}

func (t *Translator) Locale(ctx context.Context) string {
	if locale, ok := ctx.Value(localeContextKey{}).(language.Tag); ok {
		return locale.String()
	}

	return django.ConfigGet(
		django.Global.Settings,
		APPVAR_TRANSLATIONS_DEFAULT_LOCALE,
		language.English,
	).String()
}

func ContextWithLocale(ctx context.Context, locale language.Tag) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

func LocaleFromContext(ctx context.Context) language.Tag {
	if locale, ok := ctx.Value(localeContextKey{}).(language.Tag); ok {
		return locale
	}

	return django.ConfigGet(
		django.Global.Settings,
		APPVAR_TRANSLATIONS_DEFAULT_LOCALE,
		language.English,
	)
}

func Translate(v string, locales ...language.Tag) (string, bool) {
	if v == "" {
		return v, false
	}

	if len(locales) == 0 {
		locales = []language.Tag{django.ConfigGet(
			django.Global.Settings,
			APPVAR_TRANSLATIONS_DEFAULT_LOCALE,
			language.English,
		)}

		var settingsLocales = django.ConfigGet(
			django.Global.Settings,
			APPVAR_TRANSLATIONS_LOCALES,
			[]language.Tag{},
		)

		locales = append(
			locales,
			settingsLocales...,
		)
	}

	var t, ok = trans.DefaultBackend.(*Translator)
	if !ok {
		logger.Errorf(
			"Default translation backend is not a Translator, cannot check translation existence for %q",
			v,
		)
		return v, false
	}

	for _, locale := range locales {
		for _, check := range localeChecks(locale) {
			var translations, ok = t.translations[check]
			if !ok {
				continue
			}

			if t, ok := translations[v]; ok && t != "" {
				return t, true
			}
		}
	}

	return v, false
}

func getTranslationFromMap(localeMap map[trans.Locale]map[trans.Untranslated]trans.Translation, checks []string, v string) (trans.Translation, bool) {
	var (
		translations map[trans.Untranslated]trans.Translation
		ok           bool
	)

	for _, check := range checks {
		if translations, ok = localeMap[check]; ok {
			break
		}
	}
	if !ok {
		return v, false
	}

	translation, ok := translations[v]
	return translation, ok
}

func localeChecks(locale language.Tag) []string {
	var (
		localeStr   = locale.String()
		base, baseC = locale.Base()
		reg, regC   = locale.Region()
		baseStr     = base.String()
		regStr      = reg.String()
		checks      = []string{
			localeStr,
		}
	)

	if baseC > language.No && baseStr != "" && baseStr != localeStr {
		checks = append(checks, baseStr)
	}

	if regC > language.No && regStr != "" && regStr != localeStr {
		checks = append(checks, regStr)
	}

	return checks
}
