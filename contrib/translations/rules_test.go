package translations

import (
	"bytes"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

func TestBuiltInPluralRules(t *testing.T) {
	for _, locale := range slices.Sorted(maps.Keys(pluralRules)) {
		t.Run(locale, func(t *testing.T) {
			rule := pluralRules[locale]
			if rule.l < 1 || rule.r == "" {
				t.Fatalf("built-in rule has %d plural forms and expression %q", rule.l, rule.r)
			}
			if strings.TrimSpace(rule.stf) == "" || strings.TrimSpace(rule.ltf) == "" {
				t.Errorf("built-in time formats are short=%q, long=%q", rule.stf, rule.ltf)
			}

			header := testPluralRuleHeader(locale, rule)
			seen := make([]bool, rule.l)
			check := func(count int) {
				t.Helper()
				index, err := header.pluralIndex(locale, count)
				if err != nil {
					t.Fatalf("pluralIndex(%d): %v", count, err)
				}
				if index < 0 || index >= rule.l {
					t.Fatalf("pluralIndex(%d) = %d, want an index in [0, %d)", count, index, rule.l)
				}
				seen[index] = true
			}
			for count := 0; count < 1_000; count++ {
				check(count)
			}
			for _, count := range []int{-21, -1, 10_000, 1_000_000, 2_000_000, math.MaxInt32} {
				check(count)
			}
			for index, reachable := range seen {
				if !reachable {
					t.Errorf("plural form %d was never selected by %q", index, rule.r)
				}
			}
		})
	}
}

func TestBuiltInPluralRuleBoundaries(t *testing.T) {
	type sample struct{ count, want int }
	for _, tc := range []struct {
		locale string
		cases  []sample
	}{
		{"ja", []sample{{0, 0}, {1, 0}, {2, 0}, {100, 0}}},
		{"nl", []sample{{0, 1}, {1, 0}, {2, 1}, {101, 1}}},
		{"fr", []sample{{0, 0}, {1, 0}, {2, 1}, {100, 1}}},
		{"ar", []sample{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {10, 3}, {11, 4}, {99, 4}, {100, 5}, {101, 5}, {102, 5}, {103, 3}, {111, 4}}},
		{"bs", []sample{{0, 2}, {1, 0}, {2, 1}, {4, 1}, {5, 2}, {11, 2}, {12, 2}, {14, 2}, {21, 0}, {22, 1}}},
		{"cs", []sample{{0, 2}, {1, 0}, {2, 1}, {4, 1}, {5, 2}, {102, 2}}},
		{"hr", []sample{{0, 2}, {1, 0}, {2, 1}, {11, 2}, {12, 2}, {21, 0}, {22, 1}}},
		{"is", []sample{{0, 1}, {1, 0}, {2, 1}, {11, 1}, {21, 0}, {111, 1}}},
		{"lt", []sample{{0, 2}, {1, 0}, {11, 2}, {21, 1}, {101, 1}, {111, 2}}},
		{"lv", []sample{{0, 2}, {1, 0}, {2, 1}, {9, 1}, {11, 2}, {19, 2}, {22, 1}}},
		{"mo", []sample{{0, 2}, {1, 0}, {2, 1}, {11, 2}, {12, 2}, {21, 0}, {22, 1}}},
		{"mt", []sample{{0, 1}, {1, 0}, {2, 4}, {3, 1}, {10, 1}, {11, 2}, {19, 2}, {20, 3}, {102, 3}, {103, 1}, {111, 2}}},
		{"pt_PT", []sample{{0, 0}, {1, 1}, {2, 2}, {101, 2}}},
		{"sk", []sample{{0, 2}, {1, 0}, {2, 1}, {4, 1}, {5, 2}, {102, 2}}},
		{"sl", []sample{{0, 3}, {1, 0}, {2, 1}, {3, 2}, {4, 2}, {5, 3}, {100, 3}, {101, 0}, {102, 1}, {103, 2}, {104, 2}}},
		{"sr", []sample{{0, 2}, {1, 0}, {2, 1}, {11, 2}, {12, 2}, {21, 0}, {22, 1}}},
		{"uk", []sample{{0, 2}, {1, 0}, {2, 1}, {4, 1}, {5, 2}, {11, 2}, {12, 2}, {14, 2}, {21, 0}, {22, 1}}},
		{"cy", []sample{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 5}, {6, 4}, {7, 5}, {8, 5}, {11, 5}, {100, 5}}},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			rule, ok := pluralRules[tc.locale]
			if !ok {
				t.Fatalf("built-in rule for %s is missing", tc.locale)
			}
			header := testPluralRuleHeader(tc.locale, rule)
			for _, sample := range tc.cases {
				t.Run(strconv.Itoa(sample.count), func(t *testing.T) {
					got, err := header.pluralIndex(tc.locale, sample.count)
					if err != nil || got != sample.want {
						t.Errorf("pluralIndex(%d) = (%d, %v), want (%d, nil)", sample.count, got, err, sample.want)
					}
				})
			}
		})
	}
}

func TestMakeTranslationsGeneratesUsablePluralRules(t *testing.T) {
	locales := []struct {
		locale string
		counts []int // One count for each translated slot, in catalog order.
	}{
		{"ja", []int{2}},
		{"is", []int{21, 11}},
		{"uk", []int{21, 22, 0}},
		{"sl", []int{101, 102, 103, 100}},
		{"mt", []int{1, 0, 11, 20, 2}},
		{"ar", []int{0, 1, 2, 3, 11, 100}},
		{"cy", []int{0, 1, 2, 3, 6, 4}},
	}
	t.Chdir(t.TempDir())
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders: []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
	})
	if err := os.WriteFile("index.html", []byte("{{ P \"file\" \"files\" .Count }}"), 0600); err != nil {
		t.Fatal(err)
	}
	var args []string
	for _, tc := range locales {
		args = append(args, "-locales="+tc.locale)
	}
	runTranslationsCommand(t, args...)

	data, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	var header FileTranslationsHeader
	matches, err := readTranslationsYAML(bytes.NewReader(data), &header, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Text != "file" || matches[0].Plural != "files" {
		t.Fatalf("generated messages = %+v, want one message for file/files", matches)
	}
	for _, tc := range locales {
		meta, exists := header.Locales.Get(tc.locale)
		if !exists || meta.NumPluralForms != len(tc.counts) {
			t.Fatalf("%s header = %+v (present=%t), want %d plural forms", tc.locale, meta, exists, len(tc.counts))
		}
		forms, exists := matches[0].Locales.Get(tc.locale)
		if !exists || len(forms) != len(tc.counts) {
			t.Fatalf("%s generated placeholders = %q (present=%t), want %d", tc.locale, forms, exists, len(tc.counts))
		}
		for i := range forms {
			if forms[i] != "" {
				t.Fatalf("%s placeholder %d = %q, want empty", tc.locale, i, forms[i])
			}
			forms[i] = tc.locale + "-form-" + strconv.Itoa(i)
		}
	}

	backend := NewTranslator(&header, mapFromTranslations(matches), nil)
	for _, tc := range locales {
		t.Run(tc.locale, func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), language.MustParse(tc.locale))
			for i, count := range tc.counts {
				want := tc.locale + "-form-" + strconv.Itoa(i)
				if got := backend.Pluralize(ctx, "file", "files", count); got != want {
					t.Errorf("Pluralize(%d) = %q, want %q", count, got, want)
				}
			}
		})
	}
}

func testPluralRuleHeader(locale string, rule pluralRule) *translationHeader {
	return newTranslationHeader(testHeader(map[string]TranslationHeaderLocale{
		locale: {NumPluralForms: rule.l, PluralRule: rule.r, ShortTimeFormat: rule.stf, LongTimeFormat: rule.ltf},
	}))
}
