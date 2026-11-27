package translations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"reflect"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/v2"
)

//go:embed translations.yml
var translationsFileFS embed.FS

//go:embed migrations/translations
var translationsMigrationFS embed.FS

type Finder interface {
	Find(fSys fs.FS, isExcluded func(filename string) bool) ([]Translation, error)
}

type TranslationsAppConfig struct {
	*apps.AppConfig
	finders         []Finder
	filesystems     []fs.FS
	translations    map[trans.Locale]map[trans.Untranslated][]trans.Translation
	appTranslations map[string]map[trans.Locale]map[trans.Untranslated][]trans.Translation

	translationHeader  *translationHeader
	translationMatches []Translation
}

var translatorApp *TranslationsAppConfig

func (c *TranslationsAppConfig) GetMigrationFS() fs.FS {
	return filesystem.Sub(translationsMigrationFS, "migrations/translations")
}

func NewAppConfig() django.AppConfig {

	if translatorApp != nil {
		return translatorApp
	}

	var cfg = &TranslationsAppConfig{
		AppConfig:    apps.NewAppConfig("translations"),
		translations: make(map[trans.Locale]map[trans.Untranslated][]trans.Translation),
	}

	cfg.Cmd = []command.Command{
		makeTranslationsCommand,
		syncLocalesCommand,
	}

	cfg.Routing = func(m mux.Multiplexer) {
		m.Use(TranslatorMiddleware())

		m.Get(
			"/i18n/translations.js",
			http.HandlerFunc(translationCatalog),
			"translations.js",
		)
	}

	cfg.ModelObjects = []attrs.Definer{
		&Locale{},
	}

	cfg.Init = func(settings django.Settings) error {

		admin.RegisterGlobalMedia(admin.RegisterMediaHookFunc(func(adminSite *admin.AdminApplication) media.Media {
			var m = media.NewMedia()
			m.AddJS(media.JS(
				django.Reverse("translations.js"),
			))
			return m
		}))

		cfg.finders = []Finder{
			&templateTranslationsFinder{
				extensions: djangoapps.CONFIG.Get(
					APPVAR_TRANSLATIONS_EXTENSIONS, []string{
						"tmpl",
						"html",
						"txt",
					},
				),
				matches: templateTranslationMatchers,
			},
			&goTranslationsFinder{
				packageAliases: djangoapps.CONFIG.Get(
					APPVAR_TRANSLATIONS_PACKAGES, []string{},
				),
				functions: goFileTranslationMatchers,
			},
			&godjangoModelsFinder{},
		}

		for _, hook := range goldcrest.Get[TranslationFinderHook](TranslationFinderHookName) {
			var f = hook(settings)
			if len(f) > 0 {
				cfg.finders = append(cfg.finders, f...)
			}
		}

		for _, hook := range goldcrest.Get[TranslationFilesystemHook](TranslationFilesystemHookName) {
			var fsys = hook(settings)
			if len(fsys) > 0 {
				cfg.filesystems = append(cfg.filesystems, fsys...)
			}
		}

		for _, app := range djangoapps.Apps {
			translatorFs, ok := app.(AppTranslationFilesystemDefiner)
			if !ok {
				continue
			}

			var fsys = translatorFs.TranslationsFS(settings)
			if len(fsys) > 0 {
				cfg.filesystems = append(cfg.filesystems, fsys...)
			}
		}

		var hdr = &FileTranslationsHeader{}
		var translationsList []Translation

		var dirs = make([]fs.FS, 0, len(cfg.filesystems)+2)
		dirs = append(dirs, translationsFileFS)
		wd, err := os.Getwd()
		if err == nil {
			var dirFS = os.DirFS(wd)
			var f, err = fs.Stat(dirFS, translationsFile)
			if err == nil && !f.IsDir() {
				dirs = append(dirs, dirFS)
			}
		}

		for _, fsys := range append(dirs, cfg.filesystems...) {
			var f, err = fs.Stat(fsys, translationsFile)
			if err != nil || f.IsDir() {
				continue
			}

			file, err := fsys.Open(translationsFile)
			if err != nil {
				return errors.Wrapf(
					err, "failed to open %s in filesystem", translationsFile,
				)
			}

			previousLocales := hdr.Locales
			translationsList, err = readTranslationsYAML(
				file, hdr, translationsList,
			)
			if err != nil {
				file.Close()
				return errors.Wrapf(
					err, "failed to read translations from %s in filesystem",
					translationsFile,
				)
			}

			file.Close()

			if previousLocales != nil && hdr.Locales != previousLocales {
				for locale := hdr.Locales.Front(); locale != nil; locale = locale.Next() {
					previousLocales.Set(locale.Key, locale.Value)
				}
				hdr.Locales = previousLocales
			}
		}

		cfg.translationMatches = translationsList
		cfg.translationHeader = newTranslationHeader(hdr)
		cfg.translations = mapFromTranslations(cfg.translationMatches)

		return nil
	}

	cfg.Ready = func() error {
		trans.DefaultBackend = &Translator{
			hdr:             cfg.translationHeader,
			translations:    cfg.translations,
			appTranslations: cfg.appTranslations,
		}
		return nil
	}

	translatorApp = cfg
	return cfg
}

func (c *TranslationsAppConfig) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = c.AppConfig.Check(ctx, settings)

	if len(c.translations) == 0 {
		messages = append(messages, checks.Warning(
			"translations.no_translations",
			"Translations are not set up. Please run the make-translations command to find and add translations.",
			nil,
		))
	}

	return messages
}

func errIfNotOk(ok bool, err any) error {
	if ok {
		return nil
	}

	switch e := err.(type) {
	case error:
		return e
	case string:
		return errors.ValueError.Wrap(e)
	}

	panic(fmt.Sprintf(
		"unexpected error type: %s", reflect.TypeOf(err),
	))
}
