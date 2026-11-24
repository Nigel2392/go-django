package translations

import (
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
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
		{"empty configured locales", django.Config(map[string]any{APPVAR_TRANSLATIONS_LOCALES: []string{}}), nil, false},
		{"configured locale requires database", django.Config(map[string]any{APPVAR_TRANSLATIONS_LOCALES: []string{"nl"}}), nil, true},
		{"additional locale requires database", nil, []string{"-locales=nl"}, true},
		{"incorrect database type", django.Config(map[string]any{django.APPVAR_DATABASE: "sqlite", APPVAR_TRANSLATIONS_LOCALES: []string{"nl"}}), nil, true},
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
		name    string
		setting any
		want    []string
	}{
		{"duplicate strings", []string{"nl", "fr", "nl"}, []string{"nl", "fr"}},
		{"raw locale identifier", []string{"pt_PT", "fr"}, []string{"pt_PT", "fr"}},
		{"language tags", []language.Tag{language.Dutch, language.MustParse("fr-CA"), language.Dutch}, []string{"nl", "fr-CA"}},
		{"unset", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestDjango(t, django.Config(map[string]any{APPVAR_TRANSLATIONS_LOCALES: tc.setting}))
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

func TestMakeTranslationsRejectsWrongLocaleSettingType(t *testing.T) {
	setTestDjango(t, django.Config(map[string]any{APPVAR_TRANSLATIONS_LOCALES: "en-US"}))
	if err := executeTranslationsCommand(); err == nil || !strings.Contains(err.Error(), "must be []string or []language.Tag") {
		t.Errorf("make-translations error = %v, want invalid locale setting type", err)
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
