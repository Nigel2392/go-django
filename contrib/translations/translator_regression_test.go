package translations

import (
	"fmt"
	"sync"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"golang.org/x/text/language"
)

func TestTranslatorFallsBackForPartialRegionalCatalogs(t *testing.T) {
	const message = "Open"
	ctx := trans.ContextWithLocale(t.Context(), language.MustParse("nl-BE"))
	for _, tc := range []struct {
		name     string
		regional []string
		base     string
		want     string
	}{
		{name: "regional translation takes precedence", regional: []string{"Openen"}, base: "Openmaken", want: "Openen"},
		{name: "missing regional key uses base language", base: "Openmaken", want: "Openmaken"},
		{name: "blank regional entry uses base language", regional: []string{""}, base: "Openmaken", want: "Openmaken"},
		{name: "missing base key uses region", want: "Open region"},
		{name: "blank base entry uses region", want: "Open region"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			translations := map[string]map[string][]string{
				"nl-BE": {"Close": {"Sluiten"}},
				"nl":    {"Open": {tc.base}},
				"BE":    {"Open": {"Open region"}},
			}
			if tc.regional != nil {
				translations["nl-BE"][message] = tc.regional
			}
			if tc.name == "missing base key uses region" {
				delete(translations["nl"], message)
			}
			translator := NewTranslator(nil, translations, nil)
			if got := translator.Translate(ctx, message); got != tc.want {
				t.Errorf("Translate(%q) = %q, want %q", message, got, tc.want)
			}
		})
	}
}

func TestTranslatorAppCatalogLookup(t *testing.T) {
	setTestTranslationsApp(t, nil)
	ctx := trans.ContextWithLocale(t.Context(), language.MustParse("nl-BE"))
	global := map[string]map[string][]string{"nl": {"Hello": {"Hallo"}}}
	appTranslations := map[string]map[string]map[string][]string{
		"shop": {
			"nl-BE": {"Local only": {"Enkel hier"}},
			"nl":    {"Hello": {"Welkom"}},
		},
	}
	translator := NewTranslator(nil, global, appTranslations)
	t.Run("standalone translator without app context", func(t *testing.T) {
		if got := translator.Translate(ctx, "Hello"); got != "Hallo" {
			t.Errorf("Translate = %q, want Hallo", got)
		}
	})
	t.Run("app regional catalog misses key", func(t *testing.T) {
		ctx := djangoapps.ContextWithApp(ctx, apps.NewAppConfig("shop"))
		if got := translator.Translate(ctx, "Hello"); got != "Welkom" {
			t.Errorf("Translate = %q, want Welkom", got)
		}
	})
	t.Run("framework app used when context has no app", func(t *testing.T) {
		setTestTranslationsApp(t, &TranslationsAppConfig{AppConfig: apps.NewAppConfig("shop")})
		if got := translator.Translate(ctx, "Hello"); got != "Welkom" {
			t.Errorf("Translate = %q, want Welkom", got)
		}
	})
}

func TestTranslatorPluralRegionalFallback(t *testing.T) {
	const singular, plural = "item", "items"
	key := getHashForPluralTexts(singular, plural)
	regional := TranslationHeaderLocale{NumPluralForms: 2, PluralRule: "n > 1"}
	base := TranslationHeaderLocale{NumPluralForms: 2, PluralRule: "n != 1"}
	ctx := trans.ContextWithLocale(t.Context(), language.MustParse("fr-CA"))
	for _, tc := range []struct {
		name           string
		exact, base    []string
		regionalHeader TranslationHeaderLocale
		count          int
		want           string
	}{
		{"exact rule and form", []string{"objet", "objets"}, []string{"article", "articles"}, regional, 0, "objet"},
		{"base rule after missing regional message", nil, []string{"article", "articles"}, regional, 0, "articles"},
		{"base rule after empty regional form", []string{"", "objets"}, []string{"article", "articles"}, regional, 0, "articles"},
		{"base rule after truncated regional forms", []string{"objet"}, []string{"article", "articles"}, regional, 2, "articles"},
		{"base rule after empty regional list", []string{}, []string{"article", "articles"}, regional, 1, "article"},
		{"base rule after invalid regional rule", []string{"objet", "objets"}, []string{"article", "articles"}, TranslationHeaderLocale{PluralRule: "n <"}, 2, "articles"},
		{"source selected by regional rule", nil, nil, regional, 0, singular},
		{"source selected by base when regional header is absent", []string{"objet", "objets"}, nil, TranslationHeaderLocale{}, 0, plural},
		{"blank forms fall back to source", []string{"objet", ""}, []string{"article", ""}, regional, 4, plural},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string]TranslationHeaderLocale{"fr": base}
			if tc.regionalHeader.PluralRule != "" {
				headers["fr-CA"] = tc.regionalHeader
			}
			translations := map[string]map[string][]string{
				"fr-CA": {"Other": {"Autre"}},
				"fr":    {"Other": {"Autre"}},
			}
			if tc.exact != nil {
				translations["fr-CA"][key] = tc.exact
			}
			if tc.base != nil {
				translations["fr"][key] = tc.base
			}
			translator := NewTranslator(testHeader(headers), translations, nil)
			if got := translator.Pluralize(ctx, singular, plural, tc.count); got != tc.want {
				t.Errorf("Pluralize(%d) = %q, want %q", tc.count, got, tc.want)
			}
		})
	}
}

func TestPluralizeWithoutCatalogHeader(t *testing.T) {
	translators := map[string]*Translator{
		"zero value":       {},
		"nil header":       NewTranslator(nil, nil, nil),
		"nil locale map":   NewTranslator(&FileTranslationsHeader{}, nil, nil),
		"empty locale map": NewTranslator(testHeader(nil), nil, nil),
		"locale with no plural rule": NewTranslator(testHeader(map[string]TranslationHeaderLocale{
			"en": {NumPluralForms: 2},
		}), nil, nil),
	}
	ctx := trans.ContextWithLocale(t.Context(), language.English)
	for name, translator := range translators {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				count int
				want  string
			}{
				{-1, "files"},
				{0, "files"},
				{1, "file"},
				{2, "files"},
			} {
				t.Run(fmt.Sprint(tc.count), func(t *testing.T) {
					if got := translator.Pluralize(ctx, "file", "files", tc.count); got != tc.want {
						t.Errorf("Pluralize(%d) = %q, want %q", tc.count, got, tc.want)
					}
				})
			}
		})
	}
}

func TestMalformedPluralRuleDoesNotAffectOtherLocales(t *testing.T) {
	header := testHeader(map[string]TranslationHeaderLocale{
		"nl": {NumPluralForms: 2, PluralRule: "n != 1"},
		"de": {NumPluralForms: 2, PluralRule: "n <"},
	})
	translator := NewTranslator(header, map[string]map[string][]string{
		"nl": {getHashForPluralTexts("file", "files"): {"bestand", "bestanden"}},
	}, nil)
	for _, tc := range []struct {
		locale language.Tag
		want   string
	}{
		{language.Dutch, "bestanden"},
		{language.German, "file"},
		{language.Dutch, "bestanden"},
	} {
		ctx := trans.ContextWithLocale(t.Context(), tc.locale)
		if got := translator.Pluralize(ctx, "file", "files", 2); got != tc.want {
			t.Errorf("Pluralize(%s) = %q, want %q", tc.locale, got, tc.want)
		}
	}
}

func TestTranslatorPluralizeConcurrentInitialLookup(t *testing.T) {
	header := testHeader(map[string]TranslationHeaderLocale{
		"nl": {NumPluralForms: 2, PluralRule: "n != 1"},
		"fr": {NumPluralForms: 2, PluralRule: "n > 1"},
	})
	key := getHashForPluralTexts("file", "files")
	translator := NewTranslator(header, map[string]map[string][]string{
		"nl": {key: {"bestand", "bestanden"}},
		"fr": {key: {"fichier", "fichiers"}},
	}, nil)
	start := make(chan struct{})
	var requests sync.WaitGroup
	for range 32 {
		requests.Go(func() {
			<-start
			for _, tc := range []struct {
				locale language.Tag
				count  int
				want   string
			}{
				{language.Dutch, 0, "bestanden"},
				{language.Dutch, 1, "bestand"},
				{language.French, 0, "fichier"},
				{language.French, 2, "fichiers"},
			} {
				ctx := trans.ContextWithLocale(t.Context(), tc.locale)
				if got := translator.Pluralize(ctx, "file", "files", tc.count); got != tc.want {
					t.Errorf("Pluralize(%s, %d) = %q, want %q", tc.locale, tc.count, got, tc.want)
				}
			}
		})
	}
	close(start)
	requests.Wait()
}

func TestTranslateLocaleDefaultsAndMalformedEntries(t *testing.T) {
	oldBackend, oldLocale := trans.DefaultBackend, trans.TRANSLATIONS_DEFAULT_LOCALE
	t.Cleanup(func() {
		trans.DefaultBackend, trans.TRANSLATIONS_DEFAULT_LOCALE = oldBackend, oldLocale
	})
	trans.TRANSLATIONS_DEFAULT_LOCALE = language.English
	trans.DefaultBackend = NewTranslator(testHeader(map[string]TranslationHeaderLocale{
		"en": {}, "nl": {}, "fr": {},
	}), map[string]map[string][]string{
		"en": {"Open": {"Open in English"}, "Blank": {}},
		"nl": {"Open": {"Openen"}, "Blank": {"Gevuld"}, "Close": {"Sluiten"}},
		"fr": {"Close": {"Fermer"}},
	}, nil)

	for _, tc := range []struct {
		name       string
		configured any
		key, want  string
		locales    []language.Tag
		found      bool
	}{
		{name: "default locale first", configured: []string{"nl"}, key: "Open", want: "Open in English", found: true},
		{name: "configured locale strings", configured: []string{"bad_tag!", "nl", "fr"}, key: "Close", want: "Sluiten", found: true},
		{name: "configured locale tags", configured: []language.Tag{language.French, language.Dutch}, key: "Close", want: "Fermer", found: true},
		{name: "explicit locales override settings", configured: []string{"nl"}, key: "Close", want: "Fermer", locales: []language.Tag{language.French}, found: true},
		{name: "empty forms skip to next locale", configured: []string{"nl"}, key: "Blank", want: "Gevuld", found: true},
		{name: "missing locale", configured: []string{"fr"}, key: "Unknown", want: "Unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setTestDjango(t, django.Config(map[string]any{APPVAR_TRANSLATIONS_LOCALES: tc.configured}))
			if got, found := Translate(tc.key, tc.locales...); got != tc.want || found != tc.found {
				t.Errorf("Translate(%q) = (%q, %t), want (%q, %t)", tc.key, got, found, tc.want, tc.found)
			}
		})
	}

	t.Run("without a django application", func(t *testing.T) {
		previous := django.Global
		t.Cleanup(func() { django.Global = previous })
		django.Global = nil
		if got, ok := Translate("Open"); got != "Open in English" || !ok {
			t.Errorf("Translate = (%q, %t), want (Open in English, true)", got, ok)
		}
	})

	t.Run("without catalog header", func(t *testing.T) {
		trans.DefaultBackend = NewTranslator(nil, nil, nil)
		if got, ok := Translate("Open", language.English); got != "Open" || ok {
			t.Errorf("Translate = (%q, %t), want (Open, false)", got, ok)
		}
	})
}
