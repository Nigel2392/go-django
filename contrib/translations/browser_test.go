package translations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"slices"
	"testing"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/language"
)

// Run the response body as a classic browser script with just the globals it uses.
const browserCatalogRunner = `
const fs = require("node:fs");
const vm = require("node:vm");
const {source, expressions, existingFormatter} = JSON.parse(fs.readFileSync(0, "utf8"));
const window = {};
const context = vm.createContext({window, TextEncoder});
if (existingFormatter) {
    window.sprintf = (format, ...args) => [format, ...args].join("|");
    context.originalSprintf = window.sprintf;
}
vm.runInContext(source, context, {filename: "translations.js", timeout: 1000});
const results = expressions.map(expression => {
    try {
        return {value: vm.runInContext(expression, context, {timeout: 1000})};
    } catch (err) {
        return {error: String(err)};
    }
});
process.stdout.write(JSON.stringify(results));
`

type browserCatalogCase struct {
	name       string
	expression string
	want       any
}

func TestBuiltInPluralRulesAgreeWithJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required to compare Go and JavaScript plural rules")
	}
	counts := []int{-100, -1, 0, 1, 2, 3, 4, 5, 6, 8, 10, 11, 12, 19, 20, 21, 99, 100, 101, 102, 111, 1000, 1000000}
	encodedCounts, err := json.Marshal(counts)
	if err != nil {
		t.Fatal(err)
	}

	locales := make(map[string]TranslationHeaderLocale, len(pluralRules))
	for locale, rule := range pluralRules {
		locales[locale] = TranslationHeaderLocale{NumPluralForms: rule.l, PluralRule: rule.r}
	}
	header := newTranslationHeader(testHeader(locales))
	cases := make([]browserCatalogCase, 0, len(locales))
	for _, locale := range slices.Sorted(maps.Keys(locales)) {
		want := make([]any, 0, len(counts))
		for _, n := range counts {
			index, err := header.pluralIndex(locale, n)
			if err != nil {
				t.Fatalf("pluralIndex(%s, %d): %v", locale, n, err)
			}
			want = append(want, float64(index))
		}
		cases = append(cases, browserCatalogCase{
			name:       locale,
			expression: fmt.Sprintf("%s.map(n => Number(%s))", encodedCounts, locales[locale].PluralRule),
			want:       want,
		})
	}
	runBrowserCatalogCases(t, node, "", false, cases)
}

func TestBrowserTranslationCatalog(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required to execute the JavaScript catalog")
	}

	setTestDjango(t, django.Config(map[string]any{
		trans.APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE: []language.Tag{language.English, language.Dutch, language.Arabic, language.French, language.German},
	}))
	setTestTranslationsApp(t, nil)
	cfg := NewAppConfig().(*TranslationsAppConfig)
	cfg.translationHeader = newTranslationHeader(testHeader(map[string]TranslationHeaderLocale{
		"nl": {NumPluralForms: 2, PluralRule: "n != 1", ShortTimeFormat: "%d-%m-%Y"},
		"ar": {NumPluralForms: pluralRules["ar"].l, PluralRule: pluralRules["ar"].r},
		"fr": {NumPluralForms: 2, PluralRule: "n > 1"},
		"de": {NumPluralForms: 2, PluralRule: " ", ShortTimeFormat: "%d.%m.%Y"},
	}))
	utf8Key := getHashForPluralTexts("café ☕", "cafés ☕")
	cfg.translations = map[string]map[string][]string{
		"nl": {
			"Hello":     {"Hallo"},
			"Blank":     {""},
			"Hello %s":  {"Hoi %s"},
			"Markup":    {"</script>\n\"hallo\" & café"},
			"__proto__": {"prototypebericht"},
			utf8Key:     {"koffie ☕", "kopjes koffie ☕"},
			getHashForPluralTexts("%d file", "%d files"): {"%d bestand", "%d bestanden"},
			getHashForPluralTexts("cup", "cups"):         {"kopje"},
			getHashForPluralTexts("seat", "seats"):       {"stoel", ""},
		},
		"ar": {getHashForPluralTexts("form", "forms"): {"zero", "one", "two", "few", "many", "other"}},
		"de": {getHashForPluralTexts("book", "books"): {"Buch", "Bücher"}},
	}

	router := mux.New()
	cfg.Routing(router)
	for _, tc := range []struct {
		name, locale      string
		existingFormatter bool
		cases             []browserCatalogCase
	}{
		{
			name:   "Dutch catalog",
			locale: "nl-BE",
			cases: []browserCatalogCase{
				{"header supplies the selected time format", `window.i18n.header.short_time_format`, "%d-%m-%Y"},
				{"translated message", `window.i18n.gettext("Hello")`, "Hallo"},
				{"untranslated message", `window.i18n.gettext("Unknown")`, "Unknown"},
				{"empty message", `window.i18n.gettext("")`, ""},
				{"blank translation uses source", `window.i18n.gettext("Blank")`, "Blank"},
				{"escaped catalog text", `window.i18n.gettext("Markup")`, "</script>\n\"hallo\" & café"},
				{"ordinary message with prototype name", `window.i18n.gettext("__proto__")`, "prototypebericht"},
				{"prototype name is an own catalog entry", `Object.prototype.hasOwnProperty.call(window.i18n.translations, "__proto__")`, true},
				{"untranslated prototype name", `window.i18n.gettext("constructor")`, "constructor"},
				{"prototype entry does not create numeric messages", `window.i18n.gettext("0")`, "0"},
				{"formatted translation", `window.i18n.gettext("Hello %s", "Ada")`, "Hoi Ada"},
				{"formatted source fallback", `window.i18n.gettext("Found %d", 3)`, "Found 3"},
				{"format without arguments", `window.i18n.gettext("Hello %s")`, "Hoi %s"},
				{"signed integer", `window.i18n.gettext("%+d, %+d", 5, -5)`, "+5, -5"},
				{"float precision", `window.i18n.gettext("%+.2f, %.1f", 1.25, -2.5)`, "+1.25, -2.5"},
				{"string precision", `window.i18n.gettext("%.4s", "abcdefgh")`, "abcd"},
				{"quoted string", `window.i18n.gettext("%q", 'one"two')`, "\"one\\\"two\""},
				{"type name", `window.i18n.gettext("%T, %T, %T", [1], false, null)`, "Array, boolean, null"},
				{"value formatting", `window.i18n.gettext("%v, %#v", 42, {ok: true})`, `42, {"ok":true}`},
				{"literal percent", `window.i18n.gettext("%d%% complete", 80)`, "80% complete"},
				{"boolean rule singular", `window.i18n.pluralidx(1)`, 0},
				{"boolean rule plural", `window.i18n.pluralidx(0)`, 1},
				{"array length selects a form", `window.i18n.pluralidx(["one"])`, 0},
				{"catalog hash agrees with Go", `window.i18n.debug.fnv32aHashForPlural("café ☕", "cafés ☕")`, utf8Key},
				{"Unicode singular", `window.i18n.ngettext("café ☕", "cafés ☕", 1)`, "koffie ☕"},
				{"Unicode plural", `window.i18n.ngettext("café ☕", "cafés ☕", 3)`, "kopjes koffie ☕"},
				{"formatted singular", `window.i18n.ngettext("%d file", "%d files", 1, 1)`, "1 bestand"},
				{"formatted plural", `window.i18n.ngettext("%d file", "%d files", 2, 2)`, "2 bestanden"},
				{"available form of incomplete entry", `window.i18n.ngettext("cup", "cups", 1)`, "kopje"},
				{"missing form of incomplete entry", `window.i18n.ngettext("cup", "cups", 2)`, "cups"},
				{"blank plural form uses source", `window.i18n.ngettext("seat", "seats", 2)`, "seats"},
				{"untranslated singular", `window.i18n.ngettext("child", "children", 1)`, "child"},
				{"untranslated plural", `window.i18n.ngettext("child", "children", 2)`, "children"},
			},
		},
		{
			name:   "Arabic plural boundaries",
			locale: "ar",
			cases: []browserCatalogCase{
				{"zero", `window.i18n.ngettext("form", "forms", 0)`, "zero"},
				{"one", `window.i18n.ngettext("form", "forms", 1)`, "one"},
				{"two", `window.i18n.ngettext("form", "forms", 2)`, "two"},
				{"first few", `window.i18n.ngettext("form", "forms", 3)`, "few"},
				{"last few", `window.i18n.ngettext("form", "forms", 10)`, "few"},
				{"first many", `window.i18n.ngettext("form", "forms", 11)`, "many"},
				{"last many", `window.i18n.ngettext("form", "forms", 99)`, "many"},
				{"other", `window.i18n.ngettext("form", "forms", 100)`, "other"},
				{"few after wraparound", `window.i18n.ngettext("form", "forms", 103)`, "few"},
				{"many after wraparound", `window.i18n.ngettext("form", "forms", 111)`, "many"},
			},
		},
		{
			name:   "locale without translations",
			locale: "fr",
			cases: []browserCatalogCase{
				{"empty translation map", `Object.keys(window.i18n.translations).length`, 0},
				{"plain source fallback", `window.i18n.gettext("Missing")`, "Missing"},
				{"formatted source fallback", `window.i18n.gettext("Hi %s", "Ada")`, "Hi Ada"},
				{"French zero is singular", `window.i18n.ngettext("item", "items", 0)`, "item"},
				{"French two is plural", `window.i18n.ngettext("item", "items", 2)`, "items"},
			},
		},
		{
			name:   "locale metadata with blank rule",
			locale: "de",
			cases: []browserCatalogCase{
				{"default rule in header", `window.i18n.header.rule`, "n != 1"},
				{"configured time format", `window.i18n.header.short_time_format`, "%d.%m.%Y"},
				{"singular translation", `window.i18n.ngettext("book", "books", 1)`, "Buch"},
				{"plural translation", `window.i18n.ngettext("book", "books", 0)`, "Bücher"},
			},
		},
		{
			name:   "default catalog",
			locale: "en",
			cases: []browserCatalogCase{
				{"default rule in header", `window.i18n.header.rule`, "n != 1"},
				{"default singular fallback", `window.i18n.ngettext("item", "items", 1)`, "item"},
				{"default zero is plural", `window.i18n.ngettext("item", "items", 0)`, "items"},
				{"missing prototype key falls back", `window.i18n.gettext("__proto__")`, "__proto__"},
			},
		},
		{
			name:              "page already has a formatter",
			locale:            "nl",
			existingFormatter: true,
			cases: []browserCatalogCase{
				{"preserved formatter", `window.sprintf === originalSprintf`, true},
				{"gettext uses page formatter", `window.i18n.gettext("Hello %s", "Ada")`, "Hoi %s|Ada"},
				{"ngettext uses page formatter", `window.i18n.ngettext("%d file", "%d files", 4, 4)`, "%d bestanden|4"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/i18n/translations.js", nil)
			req.Header.Set("Accept-Language", tc.locale)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("catalog response status = %d, want 200", rec.Code)
			}
			runBrowserCatalogCases(t, node, rec.Body.String(), tc.existingFormatter, tc.cases)
		})
	}
}

func runBrowserCatalogCases(t *testing.T, node, source string, existingFormatter bool, cases []browserCatalogCase) {
	t.Helper()
	var expressions []string
	for _, tc := range cases {
		expressions = append(expressions, tc.expression)
	}
	input, err := json.Marshal(map[string]any{
		"source": source, "expressions": expressions, "existingFormatter": existingFormatter,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", browserCatalogRunner)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Node.js could not execute the catalog: %v\n%s", err, output)
	}
	var results []struct {
		Value json.RawMessage `json:"value"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(output, &results); err != nil {
		t.Fatalf("decode Node.js output: %v\n%s", err, output)
	}
	if len(results) != len(cases) {
		t.Fatalf("Node.js returned %d results for %d expressions", len(results), len(cases))
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := results[i]
			if result.Error != "" {
				t.Fatalf("%s threw %s", tc.expression, result.Error)
			}
			var got any
			if err := json.Unmarshal(result.Value, &got); err != nil {
				t.Fatalf("%s returned no JSON value: %v", tc.expression, err)
			}
			want := tc.want
			if number, ok := want.(int); ok {
				want = float64(number)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s = %#v, want %#v", tc.expression, got, want)
			}
		})
	}
}
