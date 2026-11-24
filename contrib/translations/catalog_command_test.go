package translations

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	django "github.com/Nigel2392/go-django/src"
	"github.com/urfave/cli/v2"
)

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
