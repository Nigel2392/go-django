package translations

import (
	"errors"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

func TestSyncLocalesWithoutDatabase(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings django.Settings
		args     []string
		wantErr  bool
	}{
		{"no settings", nil, nil, false},
		{"no configured locales", django.Config(nil), nil, false},
		{"empty configured locales", django.Config(map[string]any{trans.APPVAR_TRANSLATIONS_LOCALES: []string{}}), nil, false},
		{"configured locale requires database", django.Config(map[string]any{trans.APPVAR_TRANSLATIONS_LOCALES: []string{"nl"}}), nil, true},
		{"additional locale requires database", nil, []string{"-locales=nl"}, true},
		{"incorrect database type", django.Config(map[string]any{django.APPVAR_DATABASE: "sqlite", trans.APPVAR_TRANSLATIONS_LOCALES: []string{"nl"}}), nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setTestDjango(t, tc.settings)
			err := executeSyncLocalesCommand(tc.args...)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), django.APPVAR_DATABASE) {
					t.Errorf("sync-locales error = %v, want missing database", err)
				}
			} else if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSyncLocalesResetsFlagsBetweenInvocations(t *testing.T) {
	setTestDjango(t, django.Config(nil))
	cmd := *syncLocalesCommand
	for _, args := range [][]string{{"-locales=en"}, nil} {
		set := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
		if err := cmd.AddFlags(nil, set); err != nil {
			t.Fatal(err)
		}
		if err := set.Parse(args); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Exec(nil, set.Args()); (err != nil) != (len(args) != 0) {
			t.Errorf("sync-locales %v error = %v", args, err)
		}
	}
}

func TestMakeTranslationsLocaleSettings(t *testing.T) {
	for _, tc := range []struct {
		name          string
		setting       any
		want          []string
		wantFormCount map[string]int
	}{
		{name: "duplicate strings", setting: []string{"nl", "fr", "nl"}, want: []string{"nl", "fr"}},
		{
			name:          "canonicalizes locale identifier",
			setting:       []string{"pt_PT", "fr"},
			want:          []string{"pt-PT", "fr"},
			wantFormCount: map[string]int{"pt-PT": 3},
		},
		{name: "trims and deduplicates equivalent identifiers", setting: []string{" EN-us ", "en-US", "nl"}, want: []string{"en-US", "nl"}},
		{name: "language tags", setting: []language.Tag{language.Dutch, language.MustParse("fr-CA"), language.Dutch}, want: []string{"nl", "fr-CA"}},
		{name: "unset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestDjango(t, django.Config(map[string]any{trans.APPVAR_TRANSLATIONS_LOCALES: tc.setting}))
			setTestTranslationsApp(t, &TranslationsAppConfig{
				finders: []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
			})
			if err := os.WriteFile("index.html", []byte(`{{ T "Hello" }}`), 0600); err != nil {
				t.Fatal(err)
			}
			runTranslationsCommand(t)
			output, err := os.ReadFile(translationsFile)
			if err != nil {
				t.Fatal(err)
			}
			var header FileTranslationsHeader
			entries, err := readTranslationsYAML(strings.NewReader(string(output)), &header, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := header.Locales.Keys(); !slices.Equal(got, tc.want) {
				t.Errorf("generated locale keys = %v, want %v", got, tc.want)
			}
			for locale, want := range tc.wantFormCount {
				got, ok := header.Locales.Get(locale)
				if !ok || got.NumPluralForms != want {
					t.Errorf("%s has %d plural forms, want %d", locale, got.NumPluralForms, want)
				}
			}
			if len(entries) != 1 || entries[0].Text != "Hello" {
				t.Fatalf("generated entries = %v, want Hello", entries)
			}
			for _, locale := range tc.want {
				if got, ok := entries[0].Locales.Get(locale); !ok || !slices.Equal(got, []string{""}) {
					t.Errorf("%s placeholder = (%q, %t), want one empty form", locale, got, ok)
				}
			}
		})
	}
}

func TestMakeTranslationsPreservesLegacyPortugueseCatalog(t *testing.T) {
	const legacyHeader = `generator: fixture
locales:
  pt_PT:
    nplural: 2
    rule: "n > 1"
    short_time_format: "legacy short"
    long_time_format: "legacy long"
`
	const canonicalHeader = `  pt-PT:
    nplural: 3
    rule: "n % 3"
    short_time_format: "canonical short"
    long_time_format: "canonical long"
`
	const legacyEntry = `---
text: archive file
plural: archive files
locales:
  pt_PT: {0: ficheiro, 1: ficheiros}
`
	const canonicalEntry = `  pt-PT: {0: arquivo, 1: arquivos, 2: muitos arquivos}
`
	legacyMetadata := TranslationHeaderLocale{
		NumPluralForms:  2,
		PluralRule:      "n > 1",
		ShortTimeFormat: "legacy short",
		LongTimeFormat:  "legacy long",
	}
	legacyForms := []string{"ficheiro", "ficheiros"}

	for _, tc := range []struct {
		name         string
		catalog      string
		wantMetadata TranslationHeaderLocale
		wantForms    []string
	}{
		{
			name:         "copies legacy values to canonical key",
			catalog:      legacyHeader + legacyEntry,
			wantMetadata: legacyMetadata,
			wantForms:    legacyForms,
		},
		{
			name:    "keeps existing canonical values",
			catalog: legacyHeader + canonicalHeader + legacyEntry + canonicalEntry,
			wantMetadata: TranslationHeaderLocale{
				NumPluralForms:  3,
				PluralRule:      "n % 3",
				ShortTimeFormat: "canonical short",
				LongTimeFormat:  "canonical long",
			},
			wantForms: []string{"arquivo", "arquivos", "muitos arquivos"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.WriteFile(translationsFile, []byte(tc.catalog), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("index.html", []byte(`{{ P "archive file" "archive files" .Count }}`), 0600); err != nil {
				t.Fatal(err)
			}

			app := initTestCatalogApp(t)
			django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"pt_PT"})
			if err := app.Initialize(django.Global.Settings); err != nil {
				t.Fatal(err)
			}
			runTranslationsCommand(t)

			output, err := os.ReadFile(translationsFile)
			if err != nil {
				t.Fatal(err)
			}
			var header FileTranslationsHeader
			entries, err := readTranslationsYAML(strings.NewReader(string(output)), &header, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Text != "archive file" || entries[0].Plural != "archive files" {
				t.Fatalf("generated entries = %+v, want the original singular and plural entry", entries)
			}
			for locale, want := range map[string]TranslationHeaderLocale{"pt_PT": legacyMetadata, "pt-PT": tc.wantMetadata} {
				if got, ok := header.Locales.Get(locale); !ok || got != want {
					t.Errorf("header %s = (%+v, %t), want %+v", locale, got, ok, want)
				}
			}
			assertTranslationForms(t, entries[0], "pt_PT", legacyForms)
			assertTranslationForms(t, entries[0], "pt-PT", tc.wantForms)
		})
	}
}

func TestMakeTranslationsRejectsInvalidLocaleSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setting any
		wantErr string
	}{
		{"wrong setting type", "en-US", "must be []string or []language.Tag"},
		{"malformed tag", []string{"nl", "nl!!"}, "invalid locale \"nl!!\""},
		{"empty tag", []string{"nl", ""}, "invalid locale \"\""},
		{"whitespace-only tag", []string{"nl", " "}, "invalid locale \" \""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestDjango(t, django.Config(map[string]any{trans.APPVAR_TRANSLATIONS_LOCALES: tc.setting}))
			setTestTranslationsApp(t, &TranslationsAppConfig{
				finders: []Finder{&templateTranslationsFinder{extensions: []string{"html"}}},
			})
			if err := os.WriteFile("index.html", []byte("{{ T \"Hello\" }}"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := executeTranslationsCommand(); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("make-translations error = %v, want %q", err, tc.wantErr)
			}
			if _, err := os.Stat(translationsFile); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("catalog stat after invalid settings = %v, want file absent", err)
			}
		})
	}
}

func executeSyncLocalesCommand(args ...string) error {
	cmd := *syncLocalesCommand
	flags := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
	if err := cmd.AddFlags(nil, flags); err != nil {
		return err
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	return cmd.Exec(nil, flags.Args())
}
