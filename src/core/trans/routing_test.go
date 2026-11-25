package trans

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/language"
)

func TestLocalizedRouteNestedDispatch(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{
		APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE: []language.Tag{language.French, language.Dutch, language.MustParse("pt-PT")},
		APPVAR_TRANSLATIONS_DEFAULT_LOCALE:  language.German,
	})

	router := mux.New()
	portal := mux.NewRoute(mux.ANY, "/portal", nil, "portal")
	localized := NewLocalizedRoute()
	portal.AddRoute(localized)
	router.AddRoute(portal)

	article := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%s:%s:%s", req.Method,
			LocaleFromContext(req.Context()), mux.Vars(req).Get("slug"))
	})
	localized.Get("/articles/<<slug>>", article, "article")
	localized.Post("/articles/<<slug>>/publish", article, "publish")
	localized.Get("/assets", http.HandlerFunc(writeRequestLocale), "assets").
		Preprocess(static.MarkerMiddleware)
	portal.Get("/health", http.HandlerFunc(writeRequestLocale), "health")

	for _, tc := range []struct {
		name, method, path, cookie, header string
		status                             int
		body                               string
	}{
		{name: "captures locale and article slug", method: http.MethodGet, path: "/portal/fr/articles/intro", status: http.StatusOK, body: "GET:fr:intro"},
		{name: "path locale wins over header and cookie", method: http.MethodGet, path: "/portal/pt-PT/articles/intro", cookie: "nl", header: "fr", status: http.StatusOK, body: "GET:pt-PT:intro"},
		{name: "POST child inherits locale", method: http.MethodPost, path: "/portal/nl/articles/intro/publish", status: http.StatusOK, body: "POST:nl:intro"},
		{name: "wrong method cannot invoke publish", method: http.MethodGet, path: "/portal/nl/articles/intro/publish", status: http.StatusNotFound},
		{name: "route needs the locale segment", method: http.MethodGet, path: "/portal/articles/intro", status: http.StatusNotFound},
		{name: "route needs the mount path", method: http.MethodGet, path: "/fr/articles/intro", status: http.StatusNotFound},
		{name: "locale middleware stays in its subtree", method: http.MethodGet, path: "/portal/health", cookie: "nl", header: "fr", status: http.StatusOK, body: "ko"},
		{name: "static preprocessing bypasses localization", method: http.MethodGet, path: "/portal/fr/assets", status: http.StatusOK, body: "ko"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ContextWithLocale(t.Context(), language.Korean)
			req := httptest.NewRequestWithContext(ctx, tc.method, tc.path, nil)
			req.Header.Set("Accept-Language", tc.header)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "lang", Value: tc.cookie})
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.status || (tc.body != "" && rec.Body.String() != tc.body) {
				t.Errorf("%s %s = (%d, %q), want (%d, %q)", tc.method, tc.path, rec.Code, rec.Body.String(), tc.status, tc.body)
			}
		})
	}
}

func TestLocalizedRouteReverseRoundTrip(t *testing.T) {
	setLocaleTestSettings(t, localeTestSettings{
		APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE: []language.Tag{language.French, language.MustParse("pt-PT")},
	})
	router := mux.New()
	localized := NewLocalizedRoute()
	localized.Get("/articles/<<slug>>", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%s:%s", LocaleFromContext(req.Context()), mux.Vars(req).Get("slug"))
	}), "localized-article")
	router.AddRoute(localized)

	for _, tc := range []struct {
		locale, slug string
	}{
		{"fr", "getting-started"},
		{"pt-PT", "installation"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			path, err := router.Reverse("localized-article", tc.locale, tc.slug)
			if want := "/" + tc.locale + "/articles/" + tc.slug + "/"; err != nil || path != want {
				t.Fatalf("Reverse = (%q, %v), want (%q, nil)", path, err, want)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if want := tc.locale + ":" + tc.slug; rec.Code != http.StatusOK || rec.Body.String() != want {
				t.Errorf("GET %s = (%d, %q), want (200, %q)", path, rec.Code, rec.Body.String(), want)
			}
		})
	}

	if _, err := router.Reverse("localized-article", "fr"); err == nil {
		t.Error("Reverse accepted a missing article slug")
	}
}
