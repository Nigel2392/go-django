package translations

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/elliotchance/orderedmap/v2"
	"golang.org/x/text/language"
)

func testHeader(locales map[string]TranslationHeaderLocale) *FileTranslationsHeader {
	header := &FileTranslationsHeader{Locales: orderedmap.NewOrderedMap[string, TranslationHeaderLocale]()}
	for locale, rule := range locales {
		header.Locales.Set(locale, rule)
	}
	return header
}

func TestLocaleChecks(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want []string
	}{
		{"nl", []string{"nl", "NL"}},
		{"nl-BE", []string{"nl-BE", "nl", "BE"}},
		{"zh-Hant-TW", []string{"zh-Hant-TW", "zh", "TW"}},
		{"und", []string{"und", "en", "US"}},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			if got := localeChecks(language.MustParse(tc.tag)); !slices.Equal(got, tc.want) {
				t.Errorf("localeChecks(%q) = %v, want %v", tc.tag, got, tc.want)
			}
		})
	}
}

func TestTranslatorTranslate(t *testing.T) {
	backend := NewTranslator(nil, map[string]map[string][]string{
		"nl":    {"Hello": {"Hallo"}, "Blank": {""}, "Broken": {}, "Hello %s": {"Hallo %s"}},
		"nl-BE": {"Hello": {"Dag"}},
		"BE":    {"Hello": {"Bonjour"}},
	}, map[string]map[string]map[string][]string{
		"shop": {"nl": {"Hello": {"Welkom"}, "Broken app": {}}},
	})
	for _, tc := range []struct {
		name, locale, app, input, want string
		args                           []any
	}{
		{name: "base language", locale: "nl-NL", app: "other", input: "Hello", want: "Hallo"},
		{name: "exact locale", locale: "nl-BE", app: "other", input: "Hello", want: "Dag"},
		{name: "region", locale: "fr-BE", app: "other", input: "Hello", want: "Bonjour"},
		{name: "app overrides global", locale: "nl", app: "shop", input: "Hello", want: "Welkom"},
		{name: "app falls back to global", locale: "nl", app: "shop", input: "Hello %s", args: []any{"Ada"}, want: "Hallo Ada"},
		{name: "invalid app entry", locale: "nl", app: "shop", input: "Broken app", want: "Broken app"},
		{name: "invalid global entry", locale: "nl", app: "other", input: "Broken", want: "Broken"},
		{name: "blank translation", locale: "nl", app: "other", input: "Blank", want: "Blank"},
		{name: "unknown text", locale: "nl", app: "other", input: "Missing", want: "Missing"},
		{name: "unknown locale", locale: "ja", app: "other", input: "Hello", want: "Hello"},
		{name: "empty text", locale: "nl", app: "shop"},
		{name: "untranslated format", locale: "ja", app: "other", input: "%d results", args: []any{3}, want: "3 results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), language.MustParse(tc.locale))
			ctx = djangoapps.ContextWithApp(ctx, apps.NewAppConfig(tc.app))
			if got := backend.Translatef(ctx, tc.input, tc.args...); got != tc.want {
				t.Errorf("Translatef(%q, %v) = %q, want %q", tc.input, tc.args, got, tc.want)
			}
		})
	}
}

func TestGetTranslationFromMap(t *testing.T) {
	catalog := map[string]map[string][]string{"nl": {"item": {"artikel", "artikelen"}}, "fr": {"empty": {}}}
	for _, tc := range []struct {
		name    string
		locales []string
		key     string
		index   int
		want    string
		found   bool
		wantErr bool
	}{
		{"first available locale", []string{"nl-BE", "nl", "fr"}, "item", 1, "artikelen", true, false},
		{"missing locale", []string{"en"}, "item", 0, "item", false, false},
		{"missing key", []string{"nl"}, "missing", 0, "missing", false, false},
		{"negative index", []string{"nl"}, "item", -1, "item", false, true},
		{"index past last form", []string{"nl"}, "item", 2, "item", false, true},
		{"no forms", []string{"fr"}, "empty", 0, "empty", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, found, err := getTranslationFromMap(catalog, tc.locales, tc.key, tc.index)
			if got != tc.want || found != tc.found || (err != nil) != tc.wantErr {
				t.Errorf("lookup = (%q, %t, %v), want (%q, %t, error: %t)", got, found, err, tc.want, tc.found, tc.wantErr)
			}
		})
	}
}

func TestTranslatorPluralize(t *testing.T) {
	header := testHeader(map[string]TranslationHeaderLocale{
		"nl": {PluralRule: "n != 1"},
		"fr": {PluralRule: "n > 1"},
		"ru": {PluralRule: "(n % 10 == 1 && n % 100 != 11) ? 0 : ((n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) ? 1 : 2)"},
		"de": {PluralRule: "unknown + n"},
		"it": {PluralRule: "2"},
	})
	key := getHashForPluralTexts("file", "files")
	backend := NewTranslator(header, map[string]map[string][]string{
		"nl": {key: {"bestand", "bestanden"}},
		"ru": {key: {"файл", "файла", "файлов"}},
		"it": {key: {"file"}},
	}, nil)

	for _, tc := range []struct {
		name, locale string
		count        int
		want         string
	}{
		{"regional singular", "nl-BE", 1, "bestand"},
		{"translated zero", "nl", 0, "bestanden"},
		{"translated plural", "nl", 2, "bestanden"},
		{"three forms one", "ru", 21, "файл"},
		{"three forms few", "ru", 22, "файла"},
		{"three forms many", "ru", 11, "файлов"},
		{"locale fallback singular at zero", "fr", 0, "file"},
		{"locale fallback plural", "fr", 2, "files"},
		{"default fallback singular", "ja", 1, "file"},
		{"default fallback plural at zero", "ja", 0, "files"},
		{"invalid rule evaluation", "de", 2, "file"},
		{"missing translated form", "it", 2, "files"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), language.MustParse(tc.locale))
			if got := backend.Pluralize(ctx, "file", "files", tc.count); got != tc.want {
				t.Errorf("Pluralize(%d) = %q, want %q", tc.count, got, tc.want)
			}
		})
	}

	ctx := trans.ContextWithLocale(t.Context(), language.Dutch)
	if got := backend.Pluralize(ctx, "", "", 2); got != "" {
		t.Errorf("Pluralize with empty forms = %q", got)
	}

	formatted := NewTranslator(header, map[string]map[string][]string{
		"nl": {getHashForPluralTexts("%d file", "%d files"): {"%d bestand", "%d bestanden"}},
	}, nil)
	if got := formatted.Pluralizef(ctx, "%d file", "%d files", 3, 3); got != "3 bestanden" {
		t.Errorf("Pluralizef = %q, want 3 bestanden", got)
	}
}

func TestTranslatorTimeFormat(t *testing.T) {
	backend := NewTranslator(testHeader(map[string]TranslationHeaderLocale{
		"nl-BE": {ShortTimeFormat: "%d/%m/%Y"},
		"nl":    {ShortTimeFormat: "%d-%m-%Y", LongTimeFormat: "%A %B"},
	}), nil, nil)
	for _, tc := range []struct {
		name, locale string
		short        bool
		want         string
	}{
		{"exact short", "nl-BE", true, "%d/%m/%Y"},
		{"base long", "nl-BE", false, "%A %B"},
		{"base short", "nl-NL", true, "%d-%m-%Y"},
		{"default short", "ja", true, "%Y-%m-%d %H:%M:%S"},
		{"default long", "ja", false, "%A, %d %B %Y %H:%M:%S"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), language.MustParse(tc.locale))
			if got := backend.TimeFormat(ctx, tc.short); got != tc.want {
				t.Errorf("TimeFormat(short=%t) = %q, want %q", tc.short, got, tc.want)
			}
		})
	}
	if got := NewTranslator(nil, nil, nil).TimeFormat(context.Background(), true); got != "%Y-%m-%d %H:%M:%S" {
		t.Errorf("TimeFormat with nil header = %q", got)
	}
}

func TestTranslateReportsCatalogMembership(t *testing.T) {
	previous := trans.DefaultBackend
	t.Cleanup(func() { trans.DefaultBackend = previous })
	trans.DefaultBackend = NewTranslator(testHeader(map[string]TranslationHeaderLocale{
		"nl": {}, "fr": {},
	}), map[string]map[string][]string{
		"nl": {"Hello": {"Hallo"}, "Blank": {""}},
		"fr": {"Hello": {"Bonjour"}},
		"de": {"Hello": {"Hallo auf Deutsch"}},
	}, nil)

	for _, tc := range []struct {
		name, input, want string
		locales           []language.Tag
		found             bool
	}{
		{"empty input", "", "", []language.Tag{language.Dutch}, false},
		{"first available locale", "Hello", "Bonjour", []language.Tag{language.Japanese, language.French, language.Dutch}, true},
		{"base language", "Hello", "Hallo", []language.Tag{language.MustParse("nl-BE")}, true},
		{"blank translation", "Blank", "Blank", []language.Tag{language.Dutch}, false},
		{"header missing", "Hello", "Hello", []language.Tag{language.German}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, found := Translate(tc.input, tc.locales...); got != tc.want || found != tc.found {
				t.Errorf("Translate(%q) = (%q, %t), want (%q, %t)", tc.input, got, found, tc.want, tc.found)
			}
		})
	}

	trans.DefaultBackend = nil
	if got, found := Translate("Hello", language.Dutch); got != "Hello" || found {
		t.Errorf("Translate with no backend = (%q, %t), want (Hello, false)", got, found)
	}
}

func TestPluralIndex(t *testing.T) {
	for _, tc := range []struct {
		name, rule  string
		count, want int
		wantErr     string
	}{
		{"boolean singular", "n != 1", 1, 0, ""},
		{"boolean plural", "n != 1", 0, 1, ""},
		{"numeric result", "n % 3", 5, 2, ""},
		{"numeric string", "'2'", 1, 2, ""},
		{"non-numeric string", "'few'", 1, 0, "invalid syntax"},
		{"unknown variable", "missing + n", 1, 0, "failed to evaluate plural rule for locale nl"},
		{"no rule singular", "", 1, 0, ""},
		{"no rule plural", "", 3, 1, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			locales := map[string]TranslationHeaderLocale{}
			if tc.rule != "" {
				locales["nl"] = TranslationHeaderLocale{PluralRule: tc.rule}
			}
			header := newTranslationHeader(testHeader(locales))
			got, err := header.pluralIndex("nl", tc.count)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("pluralIndex error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("pluralIndex = (%d, %v), want (%d, nil)", got, err, tc.want)
			}
		})
	}
}
