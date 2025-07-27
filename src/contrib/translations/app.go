package translations

import (
	"context"
	"io/fs"
	"os"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
)

type Finder interface {
	Find(fSys fs.FS) ([]Match, error)
}

type (
	Locale       = string
	Translation  = string
	Untranslated = string
)

type TranslationsAppConfig struct {
	*apps.AppConfig
	finders            []Finder
	filesystems        []fs.FS
	translations       map[Locale]map[Untranslated]Translation
	translationMatches []Match
}

var translatorApp *TranslationsAppConfig

func NewAppConfig() django.AppConfig {

	if translatorApp != nil {
		return translatorApp
	}

	var cfg = &TranslationsAppConfig{
		AppConfig:    apps.NewAppConfig("translations"),
		translations: make(map[Locale]map[Untranslated]Translation),
	}

	cfg.Cmd = []command.Command{
		makeTranslationsCommand,
	}

	cfg.Routing = func(m django.Mux) {
		m.Use(TranslatorMiddleware())
	}

	cfg.Init = func(settings django.Settings) error {
		trans.DefaultBackend = &Translator{
			translations: cfg.translations,
		}

		cfg.finders = []Finder{
			&templateTranslationsFinder{
				extensions: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_EXTENSIONS, []string{
						"tmpl",
						"html",
						"txt",
					},
				),
			},
			&goTranslationsFinder{
				packageAliases: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_PACKAGES, []string{
						"trans",
					},
				),
			},
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

		var readTranslationsFrom, ok = django.ConfigGetOK(
			settings, APPVAR_TRANSLATIONS_FILE, translationsFile,
		)
		if !ok {
			readTranslationsFrom = translationsFile
		}

		var file, err = os.Open(readTranslationsFrom)
		if err != nil {
			return err
		}
		defer file.Close()

		cfg.translationMatches, err = readTranslationsYAML(file, make([]Match, 0))
		if err != nil {
			return err
		}

		for _, m := range cfg.translationMatches {
			if m.Locales == nil || m.Locales.Len() == 0 {
				continue
			}

			for head := m.Locales.Front(); head != nil; head = head.Next() {
				if head.Value == "" {
					continue
				}

				var locale = Locale(head.Key)
				if _, ok := cfg.translations[locale]; !ok {
					cfg.translations[locale] = make(map[Untranslated]Translation)
				}

				cfg.translations[locale][m.Text] = Translation(head.Value)
			}
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
