package trans

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/mux/v2/middleware"
	"golang.org/x/text/language"
)

type localeContextKey struct{}

// ContextWithLocale returns a new context with the given locale set.
func ContextWithLocale(ctx context.Context, locale language.Tag) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// LocaleFromContext retrieves the locale from the current context.
// It uses DefaultLocale when no locale is stored in the context.
func LocaleFromContext(ctx context.Context) language.Tag {
	if locale, ok := ctx.Value(localeContextKey{}).(language.Tag); ok {
		return locale
	}

	return DefaultLocale()
}

func DefaultLocale() language.Tag {
	if locale, ok := djangoapps.CONFIG.GetOK[language.Tag](APPVAR_TRANSLATIONS_DEFAULT_LOCALE); ok {
		return locale
	}
	return language.English
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
