//go:build test
// +build test

package translations

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/urfave/cli/v2"
)

func TestMakeTranslationsDirectoryScopeAndCatalogReuse(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, dir := range []string{"shop", "outside"} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range map[string]string{
		"go.mod": "module example.com/project\n",
		"shop/checkout.go": fmt.Sprintf(`package shop
import (
	%q
	_ "example.com/project/outside"
)
func Checkout() {
	trans.S("Checkout fixture")
	trans.S("C Type")
}`, trans.PACKAGE_PATH),
		"outside/help.go": fmt.Sprintf(`package outside
import %q
var Help = trans.S("Imported source only")
`, trans.PACKAGE_PATH),
		translationsFile: `generator: project
---
text: Checkout fixture
locales: {nl: Afrekenen}
---
text: Old project entry only
locales: {nl: Oude tekst}
`,
	} {
		if err := os.WriteFile(name, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	hookCatalog := fstest.MapFS{
		"extension.html": {Data: []byte(`{{ T "Registered asset" }}`)},
		translationsFile: {Data: []byte(`generator: extension
---
text: Registered asset
locales: {nl: Extensie-element}
---
text: Hook catalog entry only
locales: {nl: Alleen in catalogus}
`)},
	}
	app := initTestCatalogApp(t, hookCatalog)
	outside := apps.NewAppConfig("outside")
	outside.ModelObjects = []attrs.Definer{&finderModel{}}
	djangoapps.ResetRegistry()
	djangoapps.RegisterApp("outside", outside)
	if err := app.Initialize(django.Global.Settings); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Old project entry only", "Hook catalog entry only"} {
		if len(app.translations["nl"][name]) == 0 {
			t.Errorf("%q was unavailable in the loaded Dutch catalog", name)
		}
	}

	runTranslationsCommand(t, "-dir", "./shop", "-locales", "nl", "-locales", "en", "-locales", "de", "-locales", "fr")
	data, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := readTranslationsYAML(strings.NewReader(string(data)), &FileTranslationsHeader{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	byText := make(map[string]Translation, len(entries))
	for _, entry := range entries {
		if _, exists := byText[entry.Text]; exists {
			t.Errorf("generated catalog contains duplicate entry %q", entry.Text)
		}
		byText[entry.Text] = entry
	}
	for _, name := range []string{"Imported source only", "Old project entry only", "Hook catalog entry only"} {
		if _, exists := byText[name]; exists {
			t.Errorf("generated catalog unexpectedly contains %q", name)
		}
	}
	for _, tc := range []struct{ text, dutch string }{
		{"Checkout fixture", "Afrekenen"},
		{"C Type", "Inhoudstype"},
		{"Registered asset", "Extensie-element"},
		{"Display title", ""},
	} {
		entry, exists := byText[tc.text]
		if !exists || entry.Locales == nil {
			t.Errorf("generated catalog missing translations for %q", tc.text)
			continue
		}
		for _, locale := range []string{"nl", "en", "de", "fr"} {
			want := ""
			if locale == "nl" {
				want = tc.dutch
			}
			if got, _ := entry.Locales.Get(locale); !slices.Equal(got, []string{want}) {
				t.Errorf("%q in %s = %q, want %q", tc.text, locale, got, want)
			}
		}
	}
	for _, source := range []string{"example.com/project/shop/checkout.go:", "extension.html:", ".finderModel.Title:"} {
		if !strings.Contains(string(data), source) {
			t.Errorf("generated catalog missing source %q", source)
		}
	}
}

func TestMakeTranslationsCommandPreservesLayeredCatalog(t *testing.T) {
	t.Chdir(t.TempDir())
	const old = `generator: project
locales:
  fr: {nplural: 2, rule: "n > 1"}
  nl: {nplural: 2, rule: "n != 1"}
---
text: file
plural: files
locales:
  nl: {0: oud bestand, 1: oude bestanden}
  fr: {0: fichier, 1: fichiers}
---
text: file
plural: files
locales:
  nl: {0: bestand, 1: bestanden}
  fr: ""
---
text: file
plural: files
locales:
  de: {0: Datei, 1: Dateien}
---
text: file
locales:
  nl: los bestand
`
	var header FileTranslationsHeader
	loaded, err := readTranslationsYAML(strings.NewReader(old), &header, nil)
	if err != nil {
		t.Fatal(err)
	}

	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders:            []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
		translationHeader:  newTranslationHeader(&header),
		translationMatches: loaded,
		filesystems: []fs.FS{
			fstest.MapFS{
				"hook.html": {Data: []byte(`{{ P "file" "files" .Count }}
{{ T "asset only" }}`)},
				"skip.html": {Data: []byte(`{{ T "ignored hook asset" }}`)},
			},
		},
	})
	if err := os.WriteFile("project.html", []byte(`{{ T "file" }}
{{ P "file" "files" .Count }}`), 0600); err != nil {
		t.Fatal(err)
	}

	runTranslationsCommand(t, "-locales=es", "-e=skip.html")
	generated, err := os.Open(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	defer generated.Close()
	entries, err := readTranslationsYAML(generated, &FileTranslationsHeader{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("generated %d entries, want 3: %+v", len(entries), entries)
	}

	for _, tc := range []struct {
		text, plural string
		want         map[string][]string
	}{
		{"file", "files", map[string][]string{
			"de": {"Datei", "Dateien"},
			"es": {"", ""},
			"fr": {"fichier", "fichiers"},
			"nl": {"bestand", "bestanden"},
		}},
		{"file", "", map[string][]string{"es": {""}, "nl": {"los bestand"}}},
		{"asset only", "", map[string][]string{"es": {""}}},
	} {
		t.Run(tc.text+"/"+tc.plural, func(t *testing.T) {
			idx := slices.IndexFunc(entries, func(m Translation) bool {
				return m.Text == tc.text && m.Plural == tc.plural
			})
			if idx < 0 {
				t.Fatalf("missing entry %q/%q", tc.text, tc.plural)
			}
			entry := entries[idx]
			if entry.Locales == nil || entry.Locales.Len() != len(tc.want) {
				t.Fatalf("unexpected locales for %q/%q: %v", entry.Text, entry.Plural, entry.Locales)
			}
			for locale, want := range tc.want {
				if got, ok := entry.Locales.Get(locale); !ok || !slices.Equal(got, want) {
					t.Errorf("locale %q = %q, want %q", locale, got, want)
				}
			}
		})
	}
	for _, entry := range loaded {
		if _, ok := entry.Locales.Get("es"); ok {
			t.Errorf("generating a placeholder mutated the loaded entry %q/%q", entry.Text, entry.Plural)
		}
	}
}

func TestMakeTranslationsCommandReportsInvalidOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders: []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
	})
	if err := os.WriteFile("home.html", []byte(`{{ T "Home" }}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(translationsFile, 0700); err != nil {
		t.Fatal(err)
	}
	err := executeTranslationsCommand("-locales=nl")
	var exitErr cli.ExitCoder
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(err.Error(), "Error creating output file") {
		t.Errorf("command error = %v, want output file creation failure with exit code 1", err)
	}
	if info, err := os.Stat(translationsFile); err != nil || !info.IsDir() {
		t.Errorf("existing output directory changed: (%v, %v)", info, err)
	}
}
