package translations

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/language"
)

func TestTranslationsAppInitialize(t *testing.T) {
	for _, tc := range []struct {
		name, hookCatalog string
		wantErr           string
	}{
		{"loads embedded local and hook catalogs", "generator: hook\n---\ntext: Hello\nlocales: {nl: Welkom}\n", ""},
		{"reports malformed hook catalog", "created: yesterday", "failed to read translations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			const catalog = "generator: local\n---\ntext: Hello\nlocales: {nl: Hallo}\n---\ntext: Local only\nlocales: {nl: Lokaal}\n"
			if err := os.WriteFile(translationsFile, []byte(catalog), 0600); err != nil {
				t.Fatal(err)
			}

			settings := django.Config(map[string]any{
				APPVAR_TRANSLATIONS_EXTENSIONS: []string{"custom"},
				APPVAR_TRANSLATIONS_PACKAGES:   []string{"i18n"},
			})
			setTestDjango(t, settings)
			setTestTranslationsApp(t, nil)
			previousHooks := goldcrest.DefaultRegistry
			t.Cleanup(func() { goldcrest.DefaultRegistry = previousHooks })
			goldcrest.DefaultRegistry = make(goldcrest.HookRegistry)
			extraFinder := &templateTranslationsFinder{}
			goldcrest.Register(TranslationFinderHookName, 0, TranslationFinderHook(func(got django.Settings) []Finder {
				if got != settings {
					t.Errorf("finder hook settings = %v, want configured settings", got)
				}
				return []Finder{extraFinder}
			}))
			goldcrest.Register(TranslationFilesystemHookName, 0, TranslationFilesystemHook(func(django.Settings) []fs.FS {
				return []fs.FS{
					fstest.MapFS{},
					fstest.MapFS{translationsFile: {Data: []byte(tc.hookCatalog)}},
				}
			}))

			cfg := NewAppConfig().(*TranslationsAppConfig)
			err := cfg.Initialize(settings)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initialize error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.finders) != 4 || cfg.finders[3] != extraFinder {
				t.Fatalf("registered finders = %v", cfg.finders)
			}
			if got := cfg.finders[0].(*templateTranslationsFinder).extensions; len(got) != 1 || got[0] != "custom" {
				t.Errorf("template extensions = %v, want [custom]", got)
			}
			if got := cfg.finders[1].(*goTranslationsFinder).packageAliases; len(got) != 1 || got[0] != "i18n" {
				t.Errorf("Go package aliases = %v, want [i18n]", got)
			}
			for key, want := range map[string]string{"Time": "Tijd", "Local only": "Lokaal", "Hello": "Welkom"} {
				if got := cfg.translations["nl"][key]; len(got) != 1 || got[0] != want {
					t.Errorf("catalog entry %q = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func setTestDjango(t *testing.T, settings django.Settings) {
	t.Helper()
	previous := django.Global
	t.Cleanup(func() { django.Global = previous })
	django.Global = &django.Application{Settings: settings}
	djangoapps.CONFIG.ApplicationSettings = settings
}

func setTestTranslationsApp(t *testing.T, app *TranslationsAppConfig) {
	t.Helper()
	previous := translatorApp
	t.Cleanup(func() { translatorApp = previous })
	translatorApp = app
}

func TestTranslatorMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name, header, cookie string
		configured           []language.Tag
		static               bool
		want                 language.Tag
	}{
		{name: "default language", want: language.English},
		{name: "from header", header: "nl", want: language.Dutch},
		{name: "weighted header", header: "en;q=0.2,nl;q=0.9", want: language.Dutch},
		{name: "cookie takes precedence", header: "nl", cookie: "en", want: language.English},
		{name: "Dutch cookie takes precedence", header: "en", cookie: "nl", want: language.Dutch},
		{name: "invalid cookie falls back to header", header: "nl", cookie: "not a tag", want: language.Dutch},
		{name: "unsupported cookie falls back to header", header: "nl", cookie: "ja", want: language.Dutch},
		{name: "unsupported header", header: "ja", want: language.English},
		{name: "configured languages", header: "fr", configured: []language.Tag{language.German, language.French}, want: language.French},
		{name: "configured default", configured: []language.Tag{language.German, language.French}, want: language.German},
		{name: "static request retains context", header: "nl", static: true, want: language.Japanese},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := django.Config(nil)
			if tc.configured != nil {
				settings.Set(APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE, tc.configured)
			}
			setTestDjango(t, settings)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(trans.ContextWithLocale(req.Context(), language.Japanese))
			req.Header.Set("Accept-Language", tc.header)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "lang", Value: tc.cookie})
			}
			if tc.static {
				req = django.RequestWithStaticMark(req, true)
			}

			rec := httptest.NewRecorder()
			TranslatorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := trans.LocaleFromContext(r.Context()); got != tc.want {
					t.Errorf("request locale = %v, want %v", got, tc.want)
				}
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Errorf("response code = %d, want %d", rec.Code, http.StatusNoContent)
			}
		})
	}
}

func TestTranslationsAppCatalogRoute(t *testing.T) {
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, nil)
	cfg := NewAppConfig().(*TranslationsAppConfig)
	if NewAppConfig() != cfg || cfg.Name() != "translations" {
		t.Fatalf("unexpected app config: %+v", cfg)
	}
	var commands []string
	for _, command := range cfg.Commands() {
		commands = append(commands, command.Name())
	}
	if !slices.Equal(commands, []string{"make-translations", "sync-locales"}) {
		t.Errorf("commands = %v, want [make-translations sync-locales]", commands)
	}
	isCatalogWarning := func(message checks.Message) bool {
		return message.ID == "translations.no_translations" && message.Type == checks.WARNING
	}
	if messages := cfg.Check(t.Context(), django.Global.Settings); !slices.ContainsFunc(messages, isCatalogWarning) {
		t.Errorf("empty catalog checks = %+v", messages)
	}

	dutch := TranslationHeaderLocale{2, "n != 1", "%d-%m-%Y", "%A %B"}
	cfg.translationHeader = newTranslationHeader(testHeader(map[string]TranslationHeaderLocale{"nl": dutch}))
	cfg.translations = map[string]map[string][]string{"nl": {"Hello": {"Hallo"}, "<tag>": {"\"tekst\""}}}
	if messages := cfg.Check(t.Context(), django.Global.Settings); slices.ContainsFunc(messages, isCatalogWarning) {
		t.Errorf("populated catalog checks = %+v", messages)
	}

	previous := trans.DefaultBackend
	t.Cleanup(func() { trans.DefaultBackend = previous })
	if err := cfg.OnReady(); err != nil {
		t.Fatal(err)
	}
	ctx := trans.ContextWithLocale(t.Context(), language.Dutch)
	if got := trans.T(ctx, "Hello"); got != "Hallo" {
		t.Errorf("framework translation after OnReady = %q, want Hallo", got)
	}

	router := mux.New()
	cfg.Routing(router)
	for _, tc := range []struct {
		name, language string
		wantHello      string
		wantHeader     TranslationHeaderLocale
	}{
		{"matched regional language", "nl-BE", "Hallo", dutch},
		{"missing locale uses default header", "en", "", TranslationHeaderLocale{
			NumPluralForms:  2,
			PluralRule:      "n != 1",
			ShortTimeFormat: trans.SHORT_TIME_FORMAT,
			LongTimeFormat:  trans.LONG_TIME_FORMAT,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/i18n/translations.js", nil)
			req.Header.Set("Accept-Language", tc.language)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/javascript; charset=utf-8" {
				t.Fatalf("catalog response = %d %q", rec.Code, rec.Header().Get("Content-Type"))
			}
			body := rec.Body.String()
			var catalogJSON string
			readCatalogJSON(t, body, "let _translations = JSON.parse(", ") ?? {};", &catalogJSON)
			var translations map[string][]string
			if err := json.Unmarshal([]byte(catalogJSON), &translations); err != nil {
				t.Fatalf("decode JavaScript catalog: %v", err)
			}
			if tc.wantHello == "" {
				if len(translations) != 0 {
					t.Errorf("default catalog = %v, want empty", translations)
				}
			} else {
				if got := translations["Hello"]; len(got) != 1 || got[0] != tc.wantHello {
					t.Errorf("Hello = %q, want %q", got, tc.wantHello)
				}
				if got := translations["<tag>"]; len(got) != 1 || got[0] != "\"tekst\"" {
					t.Errorf("escaped translation = %q", got)
				}
			}

			var header TranslationHeaderLocale
			readCatalogJSON(t, body, "    header: ", ",\n", &header)
			if header != tc.wantHeader {
				t.Errorf("JavaScript header = %+v, want %+v", header, tc.wantHeader)
			}
		})
	}
}

func readCatalogJSON(t *testing.T, body, prefix, suffix string, target any) {
	t.Helper()
	_, rest, ok := strings.Cut(strings.ReplaceAll(body, "\r\n", "\n"), prefix)
	value, _, ended := strings.Cut(rest, suffix)
	if !ok || !ended {
		t.Fatalf("missing JavaScript value after %q", prefix)
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		t.Fatalf("decode JavaScript value after %q: %v", prefix, err)
	}
}
