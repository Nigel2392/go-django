package trans

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/language"
)

type localeTestContextKey struct{}

func TestLocaleFromContext(t *testing.T) {
	for _, tc := range []struct {
		name        string
		settings    localeTestSettings
		contextTag  *language.Tag
		wantDefault language.Tag
		want        language.Tag
	}{
		{name: "no bound settings", wantDefault: language.English, want: language.English},
		{name: "no configured default", settings: localeTestSettings{}, wantDefault: language.English, want: language.English},
		{name: "nil setting", settings: localeTestSettings{APPVAR_TRANSLATIONS_DEFAULT_LOCALE: nil}, wantDefault: language.English, want: language.English},
		{name: "configured default", settings: localeTestSettings{APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.Dutch}, wantDefault: language.Dutch, want: language.Dutch},
		{name: "explicit undetermined default", settings: localeTestSettings{APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.Und}, wantDefault: language.Und, want: language.Und},
		{name: "explicit locale", settings: localeTestSettings{APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.Dutch}, contextTag: &language.French, wantDefault: language.Dutch, want: language.French},
		{name: "undetermined context locale overrides default", contextTag: &language.Und, wantDefault: language.English, want: language.Und},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setLocaleTestSettings(t, tc.settings)
			parent := context.WithValue(t.Context(), localeTestContextKey{}, "keep")
			ctx := parent
			if tc.contextTag != nil {
				ctx = ContextWithLocale(parent, *tc.contextTag)
			}
			if got := DefaultLocale(); got != tc.wantDefault {
				t.Errorf("DefaultLocale() = %s, want %s", got, tc.wantDefault)
			}
			if got := LocaleFromContext(ctx); got != tc.want {
				t.Errorf("LocaleFromContext() = %s, want %s", got, tc.want)
			}
			if got := LocaleFromContext(parent); got != tc.wantDefault {
				t.Errorf("parent locale = %s, want %s", got, tc.wantDefault)
			}
			if got := ctx.Value(localeTestContextKey{}); got != "keep" {
				t.Errorf("parent context value = %v, want keep", got)
			}
		})
	}
}

func TestLocaleMiddlewareResolvesEachRequest(t *testing.T) {
	settings := localeTestSettings{APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.French}
	setLocaleTestSettings(t, settings)
	handler := LocaleMiddleware(func(req *http.Request) (language.Tag, bool) {
		switch req.URL.Path {
		case "/dutch":
			return language.Dutch, true
		case "/und":
			return language.Und, true
		default:
			return language.Japanese, false
		}
	})(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := req.Context().Value(localeTestContextKey{}); got != "keep" {
			t.Errorf("request context value = %v, want keep", got)
		}
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, LocaleFromContext(req.Context()).String())
	}))

	for _, tc := range []struct {
		name, path, want string
		defaultTag       language.Tag
	}{
		{"resolved locale", "/dutch", "nl", language.French},
		{"resolved undetermined locale", "/und", "und", language.French},
		{"unresolved locale uses configured default", "/other", "fr", language.French},
		{"existing handler follows updated default", "/other", "de", language.German},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings.Set(APPVAR_TRANSLATIONS_DEFAULT_LOCALE, tc.defaultTag)
			ctx := ContextWithLocale(context.WithValue(t.Context(), localeTestContextKey{}, "keep"), language.Korean)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusAccepted || rec.Body.String() != tc.want {
				t.Errorf("response = (%d, %q), want (202, %q)", rec.Code, rec.Body.String(), tc.want)
			}
			if got := LocaleFromContext(req.Context()); got != language.Korean {
				t.Errorf("middleware changed the input request locale to %s", got)
			}
		})
	}
}

func TestLocalizationMiddlewareNegotiation(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{
		APPVAR_TRANSLATIONS_LOCALES:        []string{"en", "fr", "nl"},
		APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.Dutch,
	})
	for _, tc := range []struct {
		name, param, cookie, header, want string
		static                            bool
	}{
		{name: "path locale beats cookie and header", param: "fr", cookie: "nl", header: "en", want: "fr"},
		{name: "well-formed unmatched path locale stays explicit", param: "ja", cookie: "nl", header: "en", want: "ja"},
		{name: "malformed path locale uses configured default", param: "??", header: "fr", want: "nl"},
		{name: "cookie beats header", cookie: "nl", header: "fr", want: "nl"},
		{name: "malformed cookie falls back to header", cookie: "not a tag", header: "fr", want: "fr"},
		{name: "unmatched cookie falls back to header", cookie: "ja", header: "nl", want: "nl"},
		{name: "weighted headers", header: "en;q=0.3, fr;q=0.9, nl;q=0.5", want: "fr"},
		{name: "no preference uses first supported locale", want: "en"},
		{name: "unsupported header uses first supported locale", header: "ja", want: "en"},
		{name: "static requests bypass localization", param: "nl", cookie: "nl", header: "fr", static: true, want: "ko"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(ContextWithLocale(t.Context(), language.Korean), http.MethodGet, "/resource", nil)
			if tc.param != "" {
				req = mux.SetContextVars(req, mux.Variables{"locale": {tc.param}})
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "lang", Value: tc.cookie})
			}
			req.Header.Set("Accept-Language", tc.header)
			if tc.static {
				req = static.MarkRequest(req, true)
			}
			rec := httptest.NewRecorder()
			localizationMiddleware("locale")(http.HandlerFunc(writeRequestLocale)).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
				t.Errorf("localized response = (%d, %q), want (200, %q)", rec.Code, rec.Body.String(), tc.want)
			}
		})
	}
}

func TestLocalizationMiddlewareHandlesInvalidSettings(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{APPVAR_TRANSLATIONS_LOCALES: []string{"??"}})
	req := httptest.NewRequestWithContext(ContextWithLocale(t.Context(), language.Japanese), http.MethodGet, "/resource", nil)
	req.Header.Set("Accept-Language", "fr")
	rec := httptest.NewRecorder()
	localizationMiddleware("locale")(http.HandlerFunc(writeRequestLocale)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ja" {
		t.Errorf("response = (%d, %q), want (200, ja)", rec.Code, rec.Body.String())
	}
}

func TestNewLocalizedRoute(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{
		APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE: []language.Tag{language.French, language.MustParse("pt-PT")},
		APPVAR_TRANSLATIONS_DEFAULT_LOCALE:  language.German,
	})
	router := mux.New()
	localized := NewLocalizedRoute()
	localized.Get("/docs", http.HandlerFunc(writeRequestLocale), "translated-docs")
	router.AddRoute(localized)
	router.Get("/plain", http.HandlerFunc(writeRequestLocale), "plain-docs")

	for _, tc := range []struct {
		path, want string
	}{
		{"/fr/docs", "fr"},
		{"/pt-PT/docs", "pt-PT"},
		{"/pt_pt/docs", "pt-PT"},
		{"/es/docs", "es"},
		{"/invalid!/docs", "de"},
		{"/plain", "de"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
				t.Errorf("GET %s = (%d, %q), want (200, %q)", tc.path, rec.Code, rec.Body.String(), tc.want)
			}
		})
	}
}

func writeRequestLocale(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, LocaleFromContext(req.Context()).String())
}
