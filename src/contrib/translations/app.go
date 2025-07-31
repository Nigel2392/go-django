package translations

import (
	"context"
	"fmt"
	"go/ast"
	"io/fs"
	"os"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/dlclark/regexp2"
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
	}

	cfg.Init = func(settings django.Settings) error {
		cfg.finders = []Finder{
			&templateTranslationsFinder{
				extensions: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_EXTENSIONS, []string{
						"tmpl",
						"html",
						"txt",
					},
				),
				matches: []templateTranslationMatcher{
					{
						regex: translationTemplateRegex,
						exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
							var capture = match.Groups()[1].Captures[0].String()
							var col = match.Index + 1 // column is 1-based
							return capture, "", col, nil
						},
					},
					{
						regex: translationTemplateRegexPlural,
						exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
							var capture = match.Groups()[1].Captures[0].String()
							var plural = match.Groups()[2].Captures[0].String()
							var col = match.Index + 1 // column is 1-based
							return capture, plural, col, nil
						},
					},
					{
						regex: translationTemplatePipeRegex,
						exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
							var capture = match.Groups()[1].Captures[0].String()
							var col = match.Index + 1 // column is 1-based
							return capture, "", col, nil
						},
					},
				},
			},
			&goTranslationsFinder{
				packageAliases: django.ConfigGet(
					django.Global.Settings, APPVAR_TRANSLATIONS_PACKAGES, []string{
						"trans",
					},
				),
				functions: map[string]func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error){
					"S": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
						if len(call.Args) == 0 {
							return nil, nil, errors.TypeMismatch.Wrapf(
								"expected at least 1 argument for S, got %d", len(call.Args),
							)
						}
						singular, ok := call.Args[0].(*ast.BasicLit)
						return singular, nil, errIfNotOk(ok, "expected a string literal for S")
					},
					"T": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
						if len(call.Args) < 2 {
							return nil, nil, errors.TypeMismatch.Wrapf(
								"expected at least 2 arguments for T, got %d", len(call.Args),
							)
						}
						singular, ok := call.Args[1].(*ast.BasicLit)
						return singular, nil, errIfNotOk(ok, "expected a string literal for T")
					},
					"P": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
						if len(call.Args) < 3 {
							return nil, nil, errors.TypeMismatch.Wrapf(
								"expected at least 3 arguments for P, got %d", len(call.Args),
							)
						}
						singular, ok := call.Args[1].(*ast.BasicLit)
						if !ok {
							return nil, nil, errors.TypeMismatch.Wrapf(
								"expected a string literal for P, got %s", reflect.TypeOf(call.Args[1]),
							)
						}

						plural, ok = call.Args[2].(*ast.BasicLit)
						return singular, plural, errIfNotOk(ok, "expected a string literal for P")
					},
				},
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
