package trans

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"golang.org/x/text/language"
)

var matcherCache struct {
	sync.Mutex
	locales []language.Tag
	matcher language.Matcher
}

func Matcher() (language.Matcher, error) {
	configured, err := ConfiguredLocales()
	if err != nil {
		return nil, err
	}

	switch allowed := djangoapps.CONFIG.Get[any](APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE).(type) {
	case nil:
	case []language.Tag:
		if len(allowed) > 0 {
			configured = allowed
		}
	default:
		return nil, fmt.Errorf("%s must be []language.Tag, got %T", APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE, allowed)
	}
	if len(configured) == 0 {
		configured = []language.Tag{DefaultLocale()}
	}

	matcherCache.Lock()
	defer matcherCache.Unlock()
	if matcherCache.matcher == nil || !slices.Equal(matcherCache.locales, configured) {
		matcherCache.locales = slices.Clone(configured)
		matcherCache.matcher = language.NewMatcher(matcherCache.locales)
	}
	return matcherCache.matcher, nil
}

func ConfiguredLocales() ([]language.Tag, error) {
	var localeList []language.Tag
	switch locales := djangoapps.CONFIG.Get[any](APPVAR_TRANSLATIONS_LOCALES).(type) {
	case nil:
		return nil, nil
	case []string:
		localeList = make([]language.Tag, 0, len(locales))
		for _, locale := range locales {
			tag, err := language.Parse(strings.TrimSpace(locale))
			if err != nil {
				return nil, fmt.Errorf("invalid locale %q: %w", locale, err)
			}
			localeList = append(localeList, tag)
		}
	case []language.Tag:
		localeList = locales
	default:
		return nil, fmt.Errorf("%s must be []string or []language.Tag, got %T", APPVAR_TRANSLATIONS_LOCALES, locales)
	}

	seen := make(map[language.Tag]struct{}, len(localeList))
	unique := make([]language.Tag, 0, len(localeList))
	for _, locale := range localeList {
		if _, exists := seen[locale]; !exists {
			unique = append(unique, locale)
			seen[locale] = struct{}{}
		}
	}
	return unique, nil
}
