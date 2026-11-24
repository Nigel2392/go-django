package translations

import (
	"math"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

func TestTranslationYAMLScalarValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{"null spelling", "null", "null"},
		{"leading zeros", "0003", "0003"},
		{"boolean spelling", "false", "false"},
		{"date spelling", "2026-11-24", "2026-11-24"},
		{"literal line breaks", "|\n    first\n    second", "first\nsecond\n"},
		{"folded text", ">-\n    first\n    second", "first second"},
		{"quoted empty string", "''", ""},
		{"embedded punctuation", "'a: #b [c]'", "a: #b [c]"},
		{"unicode text", "مرحبا 👋", "مرحبا 👋"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var entry Translation
			input := "text: source\nlocales:\n  nl: " + tc.value + "\n"
			if err := yaml.Unmarshal([]byte(input), &entry); err != nil {
				t.Fatal(err)
			}
			assertTranslationForms(t, entry, "nl", []string{tc.want})
			content, err := yaml.Marshal(entry)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Translation
			if err := yaml.Unmarshal(content, &decoded); err != nil {
				t.Fatal(err)
			}
			assertTranslationForms(t, decoded, "nl", []string{tc.want})
		})
	}
}

func TestTranslationYAMLRejectsInvalidLocaleNode(t *testing.T) {
	for _, tc := range []struct {
		name, input, wantErr string
	}{
		{"sequence instead of locales", "locales: [nl, en]", "expected mapping node for locales"},
		{"scalar instead of locales", "locales: nl", "expected mapping node for locales"},
		{"sequence instead of plural mapping", "locales: {nl: [een, twee]}", "unexpected node kind"},
		{"alias instead of locale value", "value: &word hallo\nlocales: {nl: *word}", "unexpected node kind"},
		{"duplicate locales field", "locales: {}\nlocales: {}", "already defined"},
		{"duplicate plural field", "plural: files\nplural: items", "already defined"},
		{"invalid preference", "preference: higher", "cannot unmarshal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var entry Translation
			err := yaml.Unmarshal([]byte("text: source\n"+tc.input+"\n"), &entry)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Unmarshal() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestSortMatchesPreservesExtremePriorities(t *testing.T) {
	matches := []Translation{
		{Preference: math.MinInt, Text: "lowest", Path: "a.go"},
		{Preference: math.MaxInt, Text: "highest", Path: "z.go"},
		{Text: "normal", Path: "b.go"},
	}
	sortMatches(matches)
	got := []string{matches[0].Text, matches[1].Text, matches[2].Text}
	if want := []string{"highest", "normal", "lowest"}; !slices.Equal(got, want) {
		t.Errorf("sorted texts = %v, want %v", got, want)
	}
}

func FuzzTranslationYAMLRoundTrip(f *testing.F) {
	f.Add("hello", "", "hallo", "", 0)
	f.Add("file", "files", "bestand", "bestanden", 4)
	f.Add("file", "files", "", "ملفات", -5)
	f.Add("null", "true", "0", "false", math.MaxInt)
	f.Add("first\nsecond\n", "many\n\n", "  text", "# title: true", math.MinInt)
	f.Add("2026-11-24", "😃", "é\t🦊", "file\x00name", 1)
	f.Add("\n", "\n\n", "\n", "0", 0)
	f.Add("0", "0", "0", "\n", -81)
	f.Add("\nheader", "\nplural", "\ntranslation", "a\r\nb\u0085c\u2028", 6)
	f.Add("", "", "", "", 0)
	f.Fuzz(func(t *testing.T, text, plural, first, second string, preference int) {
		if !utf8.ValidString(text) || !utf8.ValidString(plural) || !utf8.ValidString(first) || !utf8.ValidString(second) {
			t.Skip("catalog text must be valid UTF-8")
		}
		entry := Translation{
			Text: text, Plural: plural, Preference: preference,
			Locales: orderedmap.NewOrderedMap[string, []string](),
		}
		entry.Locales.Set("nl", []string{first, second})
		entry.Locales.Set("ja", []string{first})
		entry.Locales.Set("de", []string{})

		content, err := yaml.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		var decoded Translation
		if err := yaml.Unmarshal(content, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Text != text || decoded.Plural != plural || decoded.Preference != preference {
			t.Errorf("round trip metadata = (%q, %q, %d), want (%q, %q, %d)", decoded.Text, decoded.Plural, decoded.Preference, text, plural, preference)
		}
		for _, locale := range entry.Locales.Keys() {
			want, _ := entry.Locales.Get(locale)
			assertTranslationForms(t, decoded, locale, want)
		}
		if got, want := decoded.Locales.Keys(), entry.Locales.Keys(); !slices.Equal(got, want) {
			t.Errorf("locale order = %v, want %v", got, want)
		}
	})
}

func assertTranslationForms(t *testing.T, entry Translation, locale string, want []string) {
	t.Helper()
	if entry.Locales == nil {
		t.Fatalf("decoded entry %q has no locales", entry.Text)
	}
	if got, ok := entry.Locales.Get(locale); !ok || !slices.Equal(got, want) {
		t.Errorf("locale %q = (%q, %t), want %q", locale, got, ok, want)
	}
}
