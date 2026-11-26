package translations

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/urfave/cli/v2"
)

func TestMakeTranslationsCommand(t *testing.T) {
	dir := t.TempDir()
	for name, contents := range map[string]string{
		"index.html": `{{ T "Hello" }}
{{ P "file" "files" .Count }}
{{ T "Brand new" }}`,
		"repeat.html": `{{ T "Hello" }}`,
		"skip.html":   `{{ T "Ignored" }}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	const old = `generator: fixture
locales:
  nl: {short_time_format: "custom format"}
---
text: Hello
locales: {nl: Hallo}
---
text: Removed
locales: {nl: Verwijderd}
`
	var header FileTranslationsHeader
	entries, err := readTranslationsYAML(strings.NewReader(old), &header, nil)
	if err != nil {
		t.Fatal(err)
	}
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders:            []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
		translationHeader:  newTranslationHeader(&header),
		translationMatches: entries,
	})
	settings := django.Config(map[string]any{
		APPVAR_TRANSLATIONS_DIR:           []string{dir},
		trans.APPVAR_TRANSLATIONS_LOCALES: []string{"nl"},
	})
	setTestDjango(t, settings)
	t.Chdir(dir)

	runTranslationsCommand(t, "-locales=xx", "-e=skip*")

	contents, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	var generatedHeader FileTranslationsHeader
	generated, err := readTranslationsYAML(strings.NewReader(string(contents)), &generatedHeader, nil)
	if err != nil {
		t.Fatal(err)
	}
	if generatedHeader.Generator != "fixture" {
		t.Errorf("generator = %q, want fixture", generatedHeader.Generator)
	}
	for _, locale := range []string{"nl", "xx"} {
		metadata, ok := generatedHeader.Locales.Get(locale)
		if !ok || metadata.NumPluralForms != 2 || metadata.PluralRule == "" ||
			metadata.LongTimeFormat == "" || metadata.ShortTimeFormat == "" {
			t.Errorf("generated header for %s = (%+v, %t)", locale, metadata, ok)
		}
		if locale == "nl" && metadata.ShortTimeFormat != "custom format" {
			t.Errorf("existing short time format changed to %q", metadata.ShortTimeFormat)
		}
	}
	if len(generated) != 3 {
		t.Fatalf("generated %d entries, want 3: %+v", len(generated), generated)
	}
	for i, want := range []struct {
		text, plural string
		dutch, extra []string
	}{
		{"Hello", "", []string{"Hallo"}, []string{""}},
		{"file", "files", []string{"", ""}, []string{"", ""}},
		{"Brand new", "", []string{""}, []string{""}},
	} {
		entry := generated[i]
		if entry.Text != want.text || entry.Plural != want.plural {
			t.Errorf("entry %d = (%q, %q), want (%q, %q)", i, entry.Text, entry.Plural, want.text, want.plural)
		}
		if entry.Locales == nil || !slices.Equal(entry.Locales.Keys(), []string{"nl", "xx"}) {
			t.Fatalf("entry %q has unexpected locales: %+v", entry.Text, entry.Locales)
		}
		if got, _ := entry.Locales.Get("nl"); !slices.Equal(got, want.dutch) {
			t.Errorf("entry %q Dutch forms = %q, want %q", entry.Text, got, want.dutch)
		}
		if got, _ := entry.Locales.Get("xx"); !slices.Equal(got, want.extra) {
			t.Errorf("entry %q extra forms = %q, want %q", entry.Text, got, want.extra)
		}
	}
	for _, path := range []string{"index.html:1:", "repeat.html:1:"} {
		if !strings.Contains(string(contents), path) {
			t.Errorf("deduplicated message dropped source %s", path)
		}
	}
}

func TestMakeTranslationsCommandWithNoMatches(t *testing.T) {
	t.Chdir(t.TempDir())
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders: []Finder{&goTranslationsFinder{}},
	})
	const existing = "generator: fixture\n"
	if err := os.WriteFile(translationsFile, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	runTranslationsCommand(t)
	got, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("catalog contents = %q, want %q", got, existing)
	}
}

func TestMakeTranslationsCommandGoPackagePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{finders: []Finder{&goTranslationsFinder{}}})

	source := fmt.Sprintf("package catalogue\nimport %q\nfunc Display() { trans.S(\"Products\") }\n", trans.PACKAGE_PATH)
	for name, contents := range map[string]string{
		"go.mod":                        "module example.com/store\n",
		"shop/products.go":              source,
		"shop/admin/views.go":           source,
		"shop/addon/go.mod":             "module \"example.com/addon/v2\" // nested module\n",
		"shop/addon/catalog/prices.go":  source,
		"shop/addon/catalog/ignored.go": source,
	} {
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}

	runTranslationsCommand(t, "-dir=shop", "-e=addon/catalog/ignored.go")

	data, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"example.com/store/shop/products.go:3:26",
		"example.com/store/shop/admin/views.go:3:26",
		"example.com/addon/v2/catalog/prices.go:3:26",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("catalog missing source %q:\n%s", want, data)
		}
	}
	if strings.Contains(string(data), "ignored.go") {
		t.Errorf("catalog includes an excluded Go file:\n%s", data)
	}
}

func TestMakeTranslationsCommandCreatesCatalog(t *testing.T) {
	t.Chdir(t.TempDir())
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{
		finders: []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
	})
	if err := os.WriteFile("index.html", []byte("{{ T \"Hello\" }}"), 0600); err != nil {
		t.Fatal(err)
	}
	runTranslationsCommand(t, "-locales=nl")
	contents, err := os.ReadFile(translationsFile)
	if err != nil {
		t.Fatal(err)
	}
	var header FileTranslationsHeader
	entries, err := readTranslationsYAML(strings.NewReader(string(contents)), &header, nil)
	if err != nil {
		t.Fatal(err)
	}
	if header.Generator != "godjango" || header.Created.IsZero() {
		t.Errorf("new catalog header = %+v", header)
	}
	if len(entries) != 1 || entries[0].Text != "Hello" {
		t.Fatalf("new catalog entries = %+v, want Hello", entries)
	}
	if got, ok := entries[0].Locales.Get("nl"); !ok || !slices.Equal(got, []string{""}) {
		t.Errorf("placeholder = (%q, %t), want one empty form", got, ok)
	}
}

func TestMakeTranslationsCommandReportsFinderFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, &TranslationsAppConfig{finders: []Finder{&goTranslationsFinder{}}})
	err := executeTranslationsCommand("-dir=" + filepath.Join(t.TempDir(), "missing"))
	var exitErr cli.ExitCoder
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(err.Error(), "Error finding translations") {
		t.Errorf("command error = %v, want finder failure with exit code 1", err)
	}
	if _, err := os.Stat(translationsFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("catalog stat after finder failure = %v, want file absent", err)
	}
}

func runTranslationsCommand(t *testing.T, args ...string) {
	t.Helper()
	if err := executeTranslationsCommand(args...); err != nil {
		t.Fatal(err)
	}
}

func executeTranslationsCommand(args ...string) error {
	command := *makeTranslationsCommand
	flags := flag.NewFlagSet(command.Name(), flag.ContinueOnError)
	if err := command.AddFlags(nil, flags); err != nil {
		return err
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	return command.Exec(nil, flags.Args())
}
