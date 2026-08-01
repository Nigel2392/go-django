package translations

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

var _ trans.TranslationBackend = &Translator{}

type Translator struct {
	hdr             *translationHeader
	translations    map[trans.Locale]map[trans.Untranslated][]trans.Translation
	appTranslations map[string]map[trans.Locale]map[trans.Untranslated][]trans.Translation
}

func NewTranslator(hdr *FileTranslationsHeader, translations map[trans.Locale]map[trans.Untranslated][]trans.Translation, appTranslations map[string]map[trans.Locale]map[trans.Untranslated][]trans.Translation) *Translator {
	return &Translator{
		hdr:             newTranslationHeader(hdr),
		translations:    translations,
		appTranslations: appTranslations,
	}
}

func (t *Translator) Translate(ctx context.Context, v string) string {

	if v == "" {
		return v
	}

	var (
		locale = trans.LocaleFromContext(ctx)
		checks = localeChecks(locale)
	)

	var app, ok = django.AppFromContext(ctx)
	if ok {
		// logger.Debugf("Translating %q in app %s", v, app.Name())
	}

	if t.appTranslations != nil {
		appTranslations, ok := t.appTranslations[app.Name()]
		if ok {
			var t, ok, err = getTranslationFromMap(appTranslations, checks, v, 0)
			if ok && t != "" && err == nil {
				return t
			}
			if err != nil {
				logger.Errorf(
					"Failed to get translation for app %s, locale '%s' and key '%s': %v",
					app.Name(), locale.String(), v, err,
				)
				return v
			}

			logger.Debugf(
				"Translation for app %s, locale '%s' and key '%s' not found, checking global translations",
				app.Name(), locale.String(), v,
			)
		}
	}

	var (
		translation trans.Translation
		err         error
	)
	if translation, ok, err = getTranslationFromMap(t.translations, checks, v, 0); ok && translation != "" {
		return translation
	}
	if err != nil {
		logger.Errorf(
			"Failed to get translation for locale '%s' and key '%s': %v",
			locale.String(), v, err,
		)
		return v
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
	if singular == "" && plural == "" {
		return ""
	}

	var (
		locale      = trans.LocaleFromContext(ctx)
		checks      = localeChecks(locale)
		hash        = getHashForPluralTexts(singular, plural)
		foundChecks = make([]string, 0, len(checks))
	)

	// Loop over all locale checks to find the first matching translation
	//
	// If no translation is found, we will try to find the appropriate plural form
	// based on the plural rules defined in the translation header.
	//
	// If no translation header is found, we will use the default plural rule (n != 1).
	for _, check := range checks {
		if _, ok := t.hdr.hdr.Locales.Get(check); !ok {
			//	logger.Debugf(
			//		"Locale '%s' not found in translation header, using default plural rule (n != 1)",
			//		check,
			//	)
			continue
		}

		// add the check to foundChecks in case no translation is found in the first
		// loop, so we can check it again later
		foundChecks = append(foundChecks, check)

		var pluralIdx, err = t.hdr.pluralIndex(check, n)
		if err != nil {
			logger.Errorf(
				"Failed to get plural index for locale '%s' and count %d: %v",
				check, n, err,
			)
			return singular // Fallback to singular if plural index cannot be determined
		}

		translation, ok, err := getTranslationFromMap(t.translations, checks, hash, pluralIdx)
		if err != nil {
			logger.Errorf(
				"Failed to get translation for locale '%s', plural index %d and hash '%s': %v",
				check, pluralIdx, hash, err,
			)
			continue
		}
		if !ok {
			//	logger.Debugf(
			//		"Plural translation for locale '%s', plural index %d and hash '%s' not found, using default plural rule (n != 1)",
			//		check, pluralIdx, hash,
			//	)
			continue
		}

		return translation
	}

	// If we reach here, it means no translation was found for the plural form
	// Some headers may have been found however, this means that we can try to use
	// the headers' plural rules to determine the plural form.
	for _, check := range foundChecks {
		if _, ok := t.hdr.hdr.Locales.Get(check); !ok {
			//	logger.Debugf(
			//		"Locale '%s' not found in translation header, using default plural rule (n != 1)",
			//		check,
			//	)
			continue
		}

		var pluralIdx, err = t.hdr.pluralIndex(check, n)
		if err != nil {
			logger.Errorf(
				"Failed to get plural index for locale '%s' and count %d: %v",
				check, n, err,
			)
			return singular // Fallback to singular if plural index cannot be determined
		}

		if pluralIdx == 0 {
			return singular
		}

		return plural
	}

	// Fallback to the default plural rule (n != 1
	// if no plural rule was found for the locale
	if n != 1 {
		return plural
	}

	return singular
}

func (t *Translator) Pluralizef(ctx context.Context, singular, plural string, n int, args ...any) string {
	return fmt.Sprintf(t.Pluralize(ctx, singular, plural, n), args...)
}

func defaultTimeFormat(short bool) string {
	if short {
		return "%Y-%m-%d %H:%M:%S"
	}
	return "%A, %d %B %Y %H:%M:%S"
}

func (t *Translator) TimeFormat(ctx context.Context, short bool) string {

	if t.hdr == nil || t.hdr.hdr == nil {
		return defaultTimeFormat(short)
	}

	var locale = trans.LocaleFromContext(ctx)
	var checks = localeChecks(locale)
	for _, check := range checks {
		if localeHeader, ok := t.hdr.hdr.Locales.Get(check); ok {
			if short && localeHeader.ShortTimeFormat != "" {
				return localeHeader.ShortTimeFormat
			} else if !short && localeHeader.LongTimeFormat != "" {
				return localeHeader.LongTimeFormat
			}
		}
	}

	return defaultTimeFormat(short)
}

func Translate(v string, locales ...language.Tag) (string, bool) {
	if v == "" {
		return v, false
	}

	if len(locales) == 0 {
		locales = []language.Tag{trans.DefaultLocale()}

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
			if _, ok := t.hdr.hdr.Locales.Get(check); !ok {
				continue
			}

			var translations, ok = t.translations[check]
			if !ok {
				continue
			}

			if t, ok := translations[v]; ok && t[0] != "" {
				return t[0], true
			}
		}
	}

	return v, false
}

func getHashForPluralTexts(singular, plural string) string {
	var hash = fnv.New32a()
	hash.Write([]byte(singular))
	hash.Write([]byte(plural))
	return strconv.FormatUint(uint64(hash.Sum32()), 10)
}

func getTranslationFromMap(localeMap map[trans.Locale]map[trans.Untranslated][]trans.Translation, checks []string, v string, idx int) (trans.Translation, bool, error) {
	var (
		translations map[trans.Untranslated][]trans.Translation
		ok           bool
	)

	for _, check := range checks {
		if translations, ok = localeMap[check]; ok {
			break
		}
	}
	if !ok {
		return v, false, nil
	}

	translation, ok := translations[v]

	if ok && (idx < 0 || idx >= len(translation)) {
		return v, false, fmt.Errorf(
			"index %d out of bounds for translations with %d entries",
			idx, len(translation),
		)
	}

	if ok {
		return translation[idx], true, nil
	}

	return v, false, nil
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
