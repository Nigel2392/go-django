package translations

import (
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"golang.org/x/text/language"
)

var (
	translationsFile = "translations.yml"
)

const (
	APPVAR_TRANSLATIONS_DIR             = "APPVAR_TRANSLATIONS_DIR"            // string
	APPVAR_TRANSLATIONS_LOCALES         = "APPVAR_TRANSLATIONS_LOCALES"        // []string or []language.Tag
	APPVAR_TRANSLATIONS_PACKAGES        = "APPVAR_TRANSLATIONS_PACKAGES"       // []string
	APPVAR_TRANSLATIONS_EXTENSIONS      = "APPVAR_TRANSLATIONS_EXTENSIONS"     // []string
	APPVAR_TRANSLATIONS_ACCEPT_LANGUAGE = "APPVAR_TRANSLATION_ACCEPT_LANGUAGE" // language.Tag slice
)

func configuredLocaleNames(settings django.Settings) ([]string, error) {
	if settings == nil {
		return nil, nil
	}
	value, _ := settings.Get(APPVAR_TRANSLATIONS_LOCALES)
	var names []string
	switch locales := value.(type) {
	case nil:
		return nil, nil
	case []string:
		names = locales
	case []language.Tag:
		names = make([]string, 0, len(locales))
		for _, tag := range locales {
			names = append(names, tag.String())
		}
	default:
		return nil, fmt.Errorf("%s must be []string or []language.Tag, got %T", APPVAR_TRANSLATIONS_LOCALES, value)
	}
	seen := make(map[string]struct{}, len(names))
	unique := make([]string, 0, len(names))
	for _, name := range names {
		if _, exists := seen[name]; !exists {
			unique = append(unique, name)
			seen[name] = struct{}{}
		}
	}
	return unique, nil
}
