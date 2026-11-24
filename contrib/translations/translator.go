package translations

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/Nigel2392/go-django/internal/sets"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
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

func (t *Translator) BestLocaleMatch(ctx context.Context, locales ...language.Tag) (language.Tag, language.Confidence) {
	configured, err := configuredLocaleNames(django.Global.Settings)
	if err != nil {
		return language.Und, language.No
	}

	allowedLocales := make([]language.Tag, 0, len(configured))
	for _, conf := range configured {
		tag, err := language.Parse(conf)
		if err != nil {
			logger.Errorf(
				"Failed to parse locale %q",
				conf,
			)
		}

		allowedLocales = append(allowedLocales, tag)
	}

	if len(allowedLocales) == 0 {
		allowedLocales = []language.Tag{language.English}
	}

	var matcher = language.NewMatcher(djangoapps.CONFIG.Get(
		APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE,
		allowedLocales,
	))

	tag, _, conf := matcher.Match(locales...)
	return tag, conf
}

func (t *Translator) IsAcceptedLocale(ctx context.Context, locales ...language.Tag) bool {
	var checks = sets.NewOrderedSet[string]()
	for _, locale := range locales {
		checks.Add(localeChecks(locale)...)
	}

	var values = checks.Values()
	var app, ok = djangoapps.AppFromContext(ctx)
	if !ok && translatorApp != nil {
		app = translatorApp
	}

	if app != nil && t.appTranslations != nil {
		appTranslations, ok := t.appTranslations[app.Name()]
		if !ok {
			goto checkTranslatorHeader
		}

		for _, check := range values {
			translations, ok := appTranslations[check][check]
			if ok && len(translations) > 0 {
				return true
			}
		}
	}

checkTranslatorHeader:
	for _, check := range values {
		translations, ok := t.translations[check][check]
		if ok && len(translations) > 0 {
			return true
		}
	}

	_, confidence := t.BestLocaleMatch(ctx, locales...)
	return confidence != language.No
}

func (t *Translator) Translate(ctx context.Context, v string) string {

	if v == "" {
		return v
	}

	var (
		locale = trans.LocaleFromContext(ctx)
		checks = localeChecks(locale)
	)

	var app, ok = djangoapps.AppFromContext(ctx)
	if !ok && translatorApp != nil {
		app = translatorApp
	}

	if app != nil && t.appTranslations != nil {
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

	fallback := singular
	if n != 1 {
		fallback = plural
	}
	if t.hdr == nil || t.hdr.hdr == nil || t.hdr.hdr.Locales == nil {
		return fallback
	}

	locale := trans.LocaleFromContext(ctx)
	key := getHashForPluralTexts(singular, plural)
	hasRule := false
	for _, check := range localeChecks(locale) {
		if _, ok := t.hdr.hdr.Locales.Get(check); !ok {
			continue
		}

		index, err := t.hdr.pluralIndex(check, n)
		if !hasRule {
			hasRule = true
			fallback = singular
			if err == nil && index != 0 {
				fallback = plural
			}
		}
		if err != nil {
			logger.Errorf(
				"Failed to get plural index for locale '%s' and count %d: %v",
				check, n, err,
			)
			continue
		}

		translation, found, err := getTranslationFromMap(t.translations, []string{check}, key, index)
		if err != nil {
			logger.Errorf(
				"Failed to get translation for locale '%s', plural index %d and hash '%s': %v",
				check, index, key, err,
			)
		}
		if found {
			return translation
		}
	}

	return fallback
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

	if t.hdr == nil || t.hdr.hdr == nil || t.hdr.hdr.Locales == nil {
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
		if django.Global != nil && django.Global.Settings != nil {
			configured, _ := django.Global.Settings.Get(APPVAR_TRANSLATIONS_LOCALES)
			switch values := configured.(type) {
			case []language.Tag:
				locales = append(locales, values...)
			case []string:
				for _, value := range values {
					if tag, err := language.Parse(value); err == nil {
						locales = append(locales, tag)
					}
				}
			}
		}
	}

	var t, ok = trans.DefaultBackend.(*Translator)
	if !ok {
		logger.Errorf(
			"Default translation backend is not a Translator, cannot check translation existence for %q",
			v,
		)
		return v, false
	}
	if t == nil || t.hdr == nil || t.hdr.hdr == nil || t.hdr.hdr.Locales == nil {
		return v, false
	}

	for _, locale := range locales {
		for _, check := range localeChecks(locale) {
			if _, ok := t.hdr.hdr.Locales.Get(check); !ok {
				continue
			}

			if forms := t.translations[check][v]; len(forms) > 0 && forms[0] != "" {
				return forms[0], true
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
	var firstErr error
	for _, check := range checks {
		forms, ok := localeMap[check][v]
		if !ok {
			continue
		}

		if idx < 0 || idx >= len(forms) {
			if firstErr == nil {
				firstErr = fmt.Errorf("index %d out of bounds for translations with %d entries", idx, len(forms))
			}
			continue
		}

		if forms[idx] != "" {
			return forms[idx], true, nil
		}
	}

	return v, false, firstErr
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
