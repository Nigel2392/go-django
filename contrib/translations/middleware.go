package translations

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"golang.org/x/text/language"
)

func TranslatorMiddleware() func(next http.Handler) http.Handler {
	var matcher = language.NewMatcher(djangoapps.CONFIG.Get(
		trans.APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE,
		[]language.Tag{
			language.English,
			language.Dutch,
		},
	))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			if static.Request(req) {
				next.ServeHTTP(w, req)
				return
			}

			var preferred string
			if lang, err := req.Cookie("lang"); err == nil {
				preferred = lang.Value
			}

			var header = req.Header.Get("Accept-Language")
			var tag, _ = language.MatchStrings(
				matcher,
				preferred,
				header,
			)

			req = req.WithContext(
				trans.ContextWithLocale(req.Context(), tag),
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
