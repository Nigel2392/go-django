package trans

import (
	"net/http"

	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/language"
)

//	type TranslationsHandler struct {
//		Next              http.Handler
//		AllowedLocales    []language.Tag
//		LocaleFromRequest func(*http.Request) language.Tag
//	}
//
//	func (h *TranslationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//
//		//	var preferred string
//		//	if lang, err := r.Cookie("lang"); err == nil {
//		//		preferred = lang.Value
//		//	}
//		//
//		//	var header = r.Header.Get("Accept-Language")
//		//	var tag, _ = language.MatchStrings(
//		//		matcher,
//		//		preferred,
//		//		header,
//		//	)
//		r = r.WithContext(ContextWithLocale(
//			r.Context(), h.LocaleFromRequest(r),
//		))
//
//
//	}

func NewLocalizedRoute() *mux.Route {
	const param = "language_tag"
	var route = mux.NewRoute(mux.ANY, "/<<"+param+">>", nil)
	route.Use(localizationMiddleware(param))
	return route
}

func localizationMiddleware(localeParam string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				localeId        language.Tag
				vars            = mux.Vars(r)
				localeIdStr, ok = vars.GetOK(localeParam)
			)

			if ok {
				var err error
				localeId, err = language.Parse(localeIdStr)
				if err != nil {
					localeId = TRANSLATIONS_DEFAULT_LOCALE
				}

				if !IsAcceptedLocale(r.Context(), localeId) {
					localeId = TRANSLATIONS_DEFAULT_LOCALE
				}

			} else {
				localeId = TRANSLATIONS_DEFAULT_LOCALE
			}

			next.ServeHTTP(w, r.WithContext(
				ContextWithLocale(r.Context(), localeId)))
		})
	}
}
