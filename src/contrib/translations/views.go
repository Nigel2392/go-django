package translations

import (
	"encoding/json"
	"net/http"
	"text/template"

	_ "embed"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
)

//go:embed views_translations.js
var jsFile string

func translationCatalog(w http.ResponseWriter, req *http.Request) {

	var (
		locale      = trans.LocaleFromContext(req.Context())
		checks      = localeChecks(locale)
		foundChecks = make([]string, 0, len(checks))
	)

	// Loop over all locale checks to find the first matching translation
	//
	// If no translation is found, we will try to find the appropriate plural form
	// based on the plural rules defined in the translation header.
	//
	// If no translation header is found, we will use the default plural rule (n != 1).
	for _, check := range checks {
		if _, ok := translatorApp.translationHeader.hdr.Locales.Get(check); !ok {
			//	logger.Debugf(
			//		"Locale '%s' not found in translation header, using default plural rule (n != 1)",
			//		check,
			//	)
			continue
		}

		// add the check to foundChecks in case no translation is found in the first
		// loop, so we can check it again later
		foundChecks = append(foundChecks, check)
	}

	if len(foundChecks) == 0 {
		foundChecks = []string{trans.DefaultLocale().String()}
	}

	header, ok := translatorApp.translationHeader.hdr.Locales.Get(foundChecks[0])
	if !ok {
		// Set up default header when none was found
		header = TranslationHeaderLocale{
			NumPluralForms:  2,
			PluralRule:      "n != 1",
			ShortTimeFormat: trans.SHORT_TIME_FORMAT,
			LongTimeFormat:  trans.LONG_TIME_FORMAT,
		}
	}

	var translations = translatorApp.translations[foundChecks[0]]
	var ctx = ctx.RequestContext(req)

	ctx.Set("header", header)
	ctx.Set("translations", translations)

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	var funcMap = template.FuncMap{
		"json": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				logger.Errorf(
					"failed to marshal JSON: %v",
					err,
				)
				return "null"
			}
			return string(b)
		},
	}

	var err error
	var tpl = template.New("translations")
	tpl = tpl.Funcs(funcMap)
	tpl, err = tpl.Parse(jsFile)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError, err,
		)
	}

	err = tpl.Execute(w, ctx)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError, err,
		)
	}
}
