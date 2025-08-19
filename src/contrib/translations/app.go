package translations

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

type Finder interface {
	Find(fSys fs.FS) ([]Translation, error)
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
	}

	cfg.Routing = func(m mux.Multiplexer) {
		m.Use(TranslatorMiddleware())

		m.Get(
			"/i18n/translations.js",
			mux.NewHandler(translationCatalog),
			"translations.js",
		)
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
				extensions: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_EXTENSIONS, []string{
						"tmpl",
						"html",
						"txt",
					},
				),
				matches: templateTranslationMatchers,
			},
			&goTranslationsFinder{
				packageAliases: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_PACKAGES, []string{},
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

		var hdr = &FileTranslationsHeader{}
		cfg.translationMatches, err = readTranslationsYAML(file, hdr, make([]Translation, 0))
		if err != nil {
			return err
		}

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
