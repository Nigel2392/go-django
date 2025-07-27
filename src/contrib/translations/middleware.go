package translations

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
	"golang.org/x/text/language"
)

func TranslatorMiddleware() func(next mux.Handler) mux.Handler {
	var matcher = language.NewMatcher(django.ConfigGet(
		django.Global.Settings,
		APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE,
		[]language.Tag{
			language.English,
			language.Dutch,
		},
	))

	return func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

			if django.IsStaticRouteRequest(req) {
				next.ServeHTTP(w, req)
				return
			}

			var lang, _ = req.Cookie("lang")
			var header = req.Header.Get("Accept-Language")
			var tag, _ = language.MatchStrings(
				matcher,
				lang.String(),
				header,
			)

			req = req.WithContext(
				ContextWithLocale(req.Context(), tag),
			)

			logger.Debugf(
				"Using locale '%s' for header %s",
				tag.String(),
				header,
			)

			next.ServeHTTP(w, req)
		})
	}
}
