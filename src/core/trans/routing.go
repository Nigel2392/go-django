package trans

import (
	"net/http"

	"golang.org/x/text/language"
)

type TranslationsHandler struct {
	AllowedLocales []language.Tag
	Next           http.Handler
}

func (h *TranslationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//	var preferred string
	//	if lang, err := r.Cookie("lang"); err == nil {
	//		preferred = lang.Value
	//	}
	//
	//	var header = r.Header.Get("Accept-Language")
	//	var tag, _ = language.MatchStrings(
	//		matcher,
	//		preferred,
	//		header,
	//	)
}
