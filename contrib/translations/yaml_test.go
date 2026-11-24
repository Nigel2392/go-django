package translations

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

func TestTranslationsHeaderYAMLRoundTrip(t *testing.T) {
	const input = `comment: Catalog metadata
generator: fixture
created: 2025-04-03T12:34:56Z
translators: [Ada, Lin]
license: MIT
version: 2
locales:
  nl:
    nplural: 2
    rule: n != 1
    short_time_format: "%d-%m-%Y"
    long_time_format: "%A %B"
  ja:
    nplural: 1
    rule: "0"
`
	var header FileTranslationsHeader
	if err := yaml.Unmarshal([]byte(input), &header); err != nil {
		t.Fatal(err)
	}
	if header.Comment != "Catalog metadata" || header.Generator != "fixture" || !header.Created.Equal(time.Date(2025, 4, 3, 12, 34, 56, 0, time.UTC)) {
		t.Errorf("wrong decoded metadata: %+v", header)
	}
	if !slices.Equal(header.Translators, []string{"Ada", "Lin"}) {
		t.Errorf("translators = %v", header.Translators)
	}
	if !reflect.DeepEqual(header.Data, map[string]any{"license": "MIT", "version": 2}) {
		t.Errorf("data = %v", header.Data)
	}
	wantDutch := TranslationHeaderLocale{2, "n != 1", "%d-%m-%Y", "%A %B"}
	if got, ok := header.Locales.Get("nl"); !ok || got != wantDutch {
		t.Errorf("Dutch header = (%+v, %t), want %+v", got, ok, wantDutch)
	}
	encoded, err := yaml.Marshal(&header)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "# Catalog metadata") {
		t.Errorf("encoded YAML dropped the header comment:\n%s", encoded)
	}

	var decoded FileTranslationsHeader
	if err := yaml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Generator != header.Generator || !decoded.Created.Equal(header.Created) ||
		!slices.Equal(decoded.Translators, header.Translators) || !reflect.DeepEqual(decoded.Data, header.Data) {
		t.Errorf("metadata changed after round trip: %+v", decoded)
	}
	if got := decoded.Locales.Keys(); !slices.Equal(got, []string{"nl", "ja"}) {
		t.Fatalf("locale order = %v, want [nl ja]", got)
	}
	for _, key := range header.Locales.Keys() {
		got, _ := decoded.Locales.Get(key)
		want, _ := header.Locales.Get(key)
		if got != want {
			t.Errorf("locale %s changed after round trip: %+v, want %+v", key, got, want)
		}
	}
}

func TestTranslationYAMLRoundTrip(t *testing.T) {
	entry := Translation{
		Text: "file", Plural: "files",
		Comment: "Shown in search", Path: "search.html", Line: 14, Col: 5,
		Paths:   []string{"results.html:8:2"},
		Locales: orderedmap.NewOrderedMap[string, []string](),
	}
	entry.Locales.Set("ja", []string{"ファイル"})
	entry.Locales.Set("nl", []string{"bestand", "bestanden"})
	entry.Locales.Set("fr", []string{""})

	encoded, err := yaml.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Shown in search", "search.html:14:5", "results.html:8:2"} {
		if !strings.Contains(string(encoded), value) {
			t.Errorf("encoded translation lacks %q:\n%s", value, encoded)
		}
	}
	var decoded Translation
	if err := yaml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Text != entry.Text || decoded.Plural != entry.Plural {
		t.Errorf("source text = (%q, %q), want (%q, %q)", decoded.Text, decoded.Plural, entry.Text, entry.Plural)
	}
	if got := decoded.Locales.Keys(); !slices.Equal(got, []string{"ja", "nl", "fr"}) {
		t.Errorf("locale order = %v, want [ja nl fr]", got)
	}
	for _, locale := range entry.Locales.Keys() {
		got, ok := decoded.Locales.Get(locale)
		want, _ := entry.Locales.Get(locale)
		if !ok || !slices.Equal(got, want) {
			t.Errorf("locale %s = (%q, %t), want %q", locale, got, ok, want)
		}
	}
}

func TestReadTranslationsYAML(t *testing.T) {
	const header = "generator: fixture\nlocales: {}\n"
	const valid = "---\ntext: hello\npreference: 4\nlocales: {nl: hallo}\n"
	for _, tc := range []struct {
		name, input string
		want        []string
		wantErr     string
	}{
		{"empty stream", "", []string{"existing"}, ""},
		{"header only", header, []string{"existing"}, ""},
		{"multiple documents", header + valid + "---\ntext: world\n", []string{"existing", "hello", "world"}, ""},
		{"invalid created timestamp", "created: yesterday", []string{"existing"}, "failed to parse created time"},
		{"invalid translators", "translators: Ada", []string{"existing"}, "failed to decode translators"},
		{"invalid header locale collection", "locales: [nl]", []string{"existing"}, "expected mapping node"},
		{"invalid header locale value", "locales: {nl: {nplural: two}}", []string{"existing"}, "failed to decode locale nl"},
		{"invalid translation field", header + "---\ntext: [hello]", []string{"existing"}, "failed to decode Match"},
		{"invalid translation locale collection", header + valid + "---\ntext: invalid\nlocales: [nl]", []string{"existing", "hello"}, "expected mapping node for locales"},
		{"invalid translation forms", header + "---\ntext: invalid\nlocales: {nl: [een, twee]}", []string{"existing"}, "unexpected node kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var metadata FileTranslationsHeader
			got, err := readTranslationsYAML(strings.NewReader(tc.input), &metadata, []Translation{{Text: "existing"}})
			if tc.wantErr == "" && err != nil || tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("readTranslationsYAML error = %v, want %q", err, tc.wantErr)
			}
			var texts []string
			for _, entry := range got {
				texts = append(texts, entry.Text)
			}
			if !slices.Equal(texts, tc.want) {
				t.Fatalf("decoded texts = %v, want %v", texts, tc.want)
			}
			if len(got) > 1 && got[1].Preference != 4 {
				t.Errorf("decoded entry metadata = %+v", got[1])
			}
		})
	}
}

func TestMapFromTranslations(t *testing.T) {
	first := orderedmap.NewOrderedMap[string, []string]()
	first.Set("nl", []string{"oud"})
	first.Set("fr", []string{"bonjour"})
	first.Set("es", []string{})
	first.Set("de", []string{""})

	replacement := orderedmap.NewOrderedMap[string, []string]()
	replacement.Set("nl", []string{"nieuw"})
	replacement.Set("fr", []string{""})

	plural := orderedmap.NewOrderedMap[string, []string]()
	plural.Set("nl", []string{"bestand", "bestanden"})
	got := mapFromTranslations([]Translation{
		{Text: "untranslated"},
		{Text: "hello", Locales: first},
		{Text: "hello", Locales: replacement},
		{Text: "file", Plural: "files", Locales: plural},
	})
	want := map[string]map[string][]string{
		"fr": {"hello": {"bonjour"}},
		"nl": {"hello": {"nieuw"}, getHashForPluralTexts("file", "files"): {"bestand", "bestanden"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("catalog = %#v, want %#v", got, want)
	}
}

func TestSortMatches(t *testing.T) {
	entries := []Translation{
		{Text: "last path", Path: "z.go"},
		{Text: "later column", Path: "a.go", Line: 2, Col: 4},
		{Text: "later line", Path: "a.go", Line: 3},
		{Text: "earlier column", Path: "a.go", Line: 2, Col: 1},
		{Text: "", Path: "a.go", Line: 1},
		{Text: "preferred", Path: "z.go", Preference: 2},
		{Text: "same position", Path: "z.go"},
		{Text: "", Path: "b.go", Line: 1},
	}
	sortMatches(entries)
	want := []string{"preferred", "earlier column", "later column", "later line", "last path", "same position", "", ""}
	for i, entry := range entries {
		if entry.Text != want[i] {
			t.Errorf("entry %d = %q, want %q", i, entry.Text, want[i])
		}
	}
	if entries[6].Path != "a.go" || entries[7].Path != "b.go" {
		t.Errorf("empty entries are out of path order: %+v", entries[6:])
	}
}
