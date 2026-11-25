package trans

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"golang.org/x/text/language"
)

type localeTestSettings map[string]any

func (s localeTestSettings) Get(key string) (any, bool) {
	val, ok := s[key]
	return val, ok
}

func (s localeTestSettings) Set(key string, value any) {
	s[key] = value
}

func setLocaleTestSettings(t *testing.T, settings localeTestSettings) {
	t.Helper()
	previous := djangoapps.CONFIG.ApplicationSettings
	t.Cleanup(func() { djangoapps.CONFIG.ApplicationSettings = previous })
	if settings == nil {
		djangoapps.CONFIG.ApplicationSettings = nil
		return
	}
	djangoapps.CONFIG.ApplicationSettings = settings
}

func TestConfiguredLocales(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setting   any
		want      []string
		wantError string
	}{
		{name: "unset"},
		{name: "empty list", setting: []string{}},
		{name: "empty typed list", setting: []language.Tag(nil)},
		{name: "single string", setting: []string{"nl"}, want: []string{"nl"}},
		{name: "typed tags", setting: []language.Tag{language.Japanese, language.French, language.Und}, want: []string{"ja", "fr", "und"}},
		{name: "canonicalizes script and region", setting: []string{" zh-hant-tw ", "en_gb"}, want: []string{"zh-Hant-TW", "en-GB"}},
		{name: "duplicates keep first position", setting: []string{"fr", "EN-us", "fr", "en-US", "nl"}, want: []string{"fr", "en-US", "nl"}},
		{name: "typed duplicates", setting: []language.Tag{language.Dutch, language.English, language.Dutch}, want: []string{"nl", "en"}},
		{name: "private use and extension", setting: []string{"x-project", "de-u-co-phonebk"}, want: []string{"x-project", "de-u-co-phonebk"}},
		{name: "wrong setting type", setting: "en", wantError: "must be []string or []language.Tag"},
		{name: "mixed slice type", setting: []any{"en", language.French}, wantError: "must be []string or []language.Tag"},
		{name: "invalid tag after valid one", setting: []string{"en", "??"}, wantError: "invalid locale \"??\""},
		{name: "empty tag", setting: []string{""}, wantError: "invalid locale \"\""},
		{name: "blank tag retains original spelling in error", setting: []string{" "}, wantError: "invalid locale \" \""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setLocaleTestSettings(t, localeTestSettings{APPVAR_TRANSLATIONS_LOCALES: tc.setting})
			got, err := ConfiguredLocales()
			if tc.wantError != "" {
				if len(got) != 0 || err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("ConfiguredLocales() = %v, %v, want no tags and error %q", got, err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, tag := range got {
				names = append(names, tag.String())
			}
			if !slices.Equal(names, tc.want) {
				t.Errorf("ConfiguredLocales() = %v, want %v", names, tc.want)
			}
		})
	}
}

func TestConfiguredLocalesReadCurrentSettings(t *testing.T) {
	setLocaleTestSettings(t, nil)
	got, err := ConfiguredLocales()
	if err != nil || len(got) != 0 {
		t.Fatalf("before settings bind: got %v, %v, want empty locales", got, err)
	}

	settings := localeTestSettings{}
	setLocaleTestSettings(t, settings)
	input := []language.Tag{language.French, language.Japanese}
	settings.Set(APPVAR_TRANSLATIONS_LOCALES, input)
	got, err = ConfiguredLocales()
	if err != nil || !slices.Equal(got, input) {
		t.Fatalf("after binding: got %v, %v, want %v", got, err, input)
	}
	got[0] = language.German
	got, err = ConfiguredLocales()
	if err != nil || !slices.Equal(got, []language.Tag{language.French, language.Japanese}) {
		t.Fatalf("mutating result changed settings: got %v, %v", got, err)
	}
	settings.Set(APPVAR_TRANSLATIONS_LOCALES, []string{"nl"})
	got, err = ConfiguredLocales()
	if err != nil || !slices.Equal(got, []language.Tag{language.Dutch}) {
		t.Errorf("after settings change: got %v, %v, want Dutch", got, err)
	}
	delete(settings, APPVAR_TRANSLATIONS_LOCALES)
	got, err = ConfiguredLocales()
	if err != nil || len(got) != 0 {
		t.Errorf("after setting removal: got %v, %v, want empty locales", got, err)
	}
}

func TestMatcherLocalePreferences(t *testing.T) {
	for _, tc := range []struct {
		name                string
		configured, allowed any
		defaultLocale       any
		preferred           string
		want                string
		matches             bool
		wantError           string
	}{
		{name: "English fallback", preferred: "en", want: "en", matches: true},
		{name: "configured default fallback", defaultLocale: language.French, want: "fr"},
		{name: "configured order", configured: []string{"ja", "fr"}, want: "ja"},
		{name: "accepted order overrides configured", configured: []string{"ja"}, allowed: []language.Tag{language.Dutch, language.French}, want: "nl"},
		{name: "matching result", configured: []string{"ja", "fr"}, preferred: "fr", want: "fr", matches: true},
		{name: "first supported preference", configured: []string{"ja", "fr"}, preferred: "de,fr;q=0.9,ja;q=0.7", want: "fr", matches: true},
		{name: "unsupported preference has no confidence", configured: []string{"ja", "fr"}, preferred: "de"},
		{name: "explicit override excludes translation locales", configured: []string{"ja"}, allowed: []language.Tag{language.French}, preferred: "ja", want: "fr"},
		{name: "empty override uses configured", configured: []string{"nl"}, allowed: []language.Tag{}, preferred: "nl", want: "nl", matches: true},
		{name: "accepted regional locale", allowed: []language.Tag{language.MustParse("pt-PT")}, preferred: "pt-PT", want: "pt-PT", matches: true},
		{name: "malformed locale setting", configured: []string{"??"}, wantError: "invalid locale"},
		{name: "malformed accepted language setting", allowed: []string{"nl"}, wantError: "must be []language.Tag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setLocaleTestSettings(t, localeTestSettings{
				APPVAR_TRANSLATIONS_LOCALES:         tc.configured,
				APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE: tc.allowed,
				APPVAR_TRANSLATIONS_DEFAULT_LOCALE:  tc.defaultLocale,
			})
			matcher, err := Matcher()
			if tc.wantError != "" {
				if matcher != nil || err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Matcher() = %v, %v, want nil matcher and error %q", matcher, err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			preferred, _, err := language.ParseAcceptLanguage(tc.preferred)
			if err != nil {
				t.Fatal(err)
			}
			tag, _, confidence := matcher.Match(preferred...)
			if (tc.want != "" && tag.String() != tc.want) || (confidence != language.No) != tc.matches {
				t.Errorf("Match(%q) = %s (%s), want %s, matched=%t", tc.preferred, tag, confidence, tc.want, tc.matches)
			}
		})
	}
}

func TestMatcherFollowsSettingChanges(t *testing.T) {
	settings := localeTestSettings{}
	setLocaleTestSettings(t, settings)
	first := checkMatcherDefault(t, language.English)
	if same := checkMatcherDefault(t, language.English); same != first {
		t.Error("unchanged settings built another matcher")
	}
	settings.Set(APPVAR_TRANSLATIONS_LOCALES, []string{"fr", "nl"})
	french := checkMatcherDefault(t, language.French)
	if french == first {
		t.Error("updated settings reused the previous matcher")
	}
	settings.Set(APPVAR_TRANSLATIONS_LOCALES, []string{"invalid!"})
	if m, err := Matcher(); err == nil || m != nil {
		t.Fatalf("Matcher() = %v, %v, want a configuration error", m, err)
	}
	settings.Set(APPVAR_TRANSLATIONS_LOCALES, []string{"fr"})
	allowed := []language.Tag{language.Japanese}
	settings.Set(APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE, allowed)
	checkMatcherDefault(t, language.Japanese)
	allowed[0] = language.German
	checkMatcherDefault(t, language.German)
	delete(settings, APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE)
	checkMatcherDefault(t, language.French)
}

func TestMatcherConcurrentReuse(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{
		APPVAR_TRANSLATIONS_LOCALES: []string{"nl", "ja", "fr"},
	})
	start := make(chan struct{})
	var workers sync.WaitGroup
	results := make(chan language.Matcher, 20)
	for range cap(results) {
		workers.Go(func() {
			<-start
			results <- checkMatcherDefault(t, language.Dutch)
		})
	}
	close(start)
	workers.Wait()
	close(results)
	want := checkMatcherDefault(t, language.Dutch)
	for got := range results {
		if got != want {
			t.Error("concurrent callers received different matchers for the same configuration")
		}
	}
}

func checkMatcherDefault(t *testing.T, want language.Tag) language.Matcher {
	t.Helper()
	matcher, err := Matcher()
	if err != nil {
		t.Error(err)
		return nil
	}
	if tag, _, _ := matcher.Match(); tag != want {
		t.Errorf("matcher default = %s, want %s", tag, want)
	}
	return matcher
}
