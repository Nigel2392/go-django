package trans

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/go-django/src/core/logger"
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

		matcher, err := Matcher()
		if err != nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			if static.Request(req) {
				next.ServeHTTP(w, req)
				return
			}

			var (
				err            error
				tag            language.Tag
				match          language.Tag
				conf           language.Confidence
				vars           = mux.Vars(req)
				languageId, ok = vars.GetOK(localeParam)
			)

			// if not ok, get from header and `GOTO addTagRequest` label (skip parsing languageId)
			if !ok {
				var preferred string
				if lang, err := req.Cookie("lang"); err == nil {
					preferred = lang.Value
				}

				var header = req.Header.Get("Accept-Language")
				tag, _ = language.MatchStrings(
					matcher,
					preferred,
					header,
				)

				goto addTagRequest
			}

			tag, err = language.Parse(languageId)
			if err != nil {
				tag = DefaultLocale()
				logger.Warnf("error parsing languageId %q: %v", languageId, err)
				goto addTagRequest
			}

			match, _, conf = matcher.Match(tag)
			if conf != language.No {
				tag = match
			}

		addTagRequest:
			req = req.WithContext(
				ContextWithLocale(req.Context(), tag),
			)

			next.ServeHTTP(w, req)
		})
	}

	//	return func(next http.Handler) http.Handler {
	//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//			var (
	//				localeId        language.Tag
	//				vars            = mux.Vars(r)
	//				localeIdStr, ok = vars.GetOK(localeParam)
	//			)
	//
	//			if ok {
	//				var err error
	//				localeId, err = language.Parse(localeIdStr)
	//				if err != nil {
	//					localeId = TRANSLATIONS_DEFAULT_LOCALE
	//				}
	//
	//				if !IsAcceptedLocale(r.Context(), localeId) {
	//					localeId = TRANSLATIONS_DEFAULT_LOCALE
	//				}
	//
	//			} else {
	//				localeId = TRANSLATIONS_DEFAULT_LOCALE
	//			}
	//
	//			next.ServeHTTP(w, r.WithContext(
	//				ContextWithLocale(r.Context(), localeId)))
	//		})
	//	}
}
