package translations

import (
	"io/fs"

	django "github.com/Nigel2392/go-django/src"
)

type (
	TranslationFinderHook     func(settings django.Settings) []Finder
	TranslationFilesystemHook func(settings django.Settings) []fs.FS
)

const (
	TranslationFinderHookName     = "translations.finder.register"
	TranslationFilesystemHookName = "translations.filesystem.register"
)
