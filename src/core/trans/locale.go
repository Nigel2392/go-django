package trans

import (
	"context"
	"net/http"

	"github.com/Nigel2392/mux/v2/middleware"
	"golang.org/x/text/language"
)

type localeContextKey struct{}

// ContextWithLocale returns a new context with the given locale set.
func ContextWithLocale(ctx context.Context, locale language.Tag) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// LocaleFromContext retrieves the locale from the current context.
// If the default backend is not set, it returns an empty string.
func LocaleFromContext(ctx context.Context) language.Tag {
	if locale, ok := ctx.Value(localeContextKey{}).(language.Tag); ok {
		return locale
	}

	return TRANSLATIONS_DEFAULT_LOCALE
}

func DefaultLocale() language.Tag {
	return TRANSLATIONS_DEFAULT_LOCALE
}

func LocaleMiddleware(localeFromRequest func(r *http.Request) (language.Tag, bool)) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tag, ok := localeFromRequest(r)
			if !ok {
				tag = DefaultLocale()
			}

			next.ServeHTTP(w, r.WithContext(
				ContextWithLocale(r.Context(), tag),
			))
		})
	}
}
