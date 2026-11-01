package chooser

import (
	"context"
	"embed"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var choosersFS embed.FS

var _, _ = core.OnDjangoReady.Listen(context.Background(), func(ctx context.Context, s signals.Signal[any], a any) error {
	for head := choosers.Front(); head != nil; head = head.Next() {
		for valHead := head.Value.Front(); valHead != nil; valHead = valHead.Next() {
			if err := valHead.Value.Setup(valHead.Key); err != nil {
				return errors.Wrapf(err, "Error setting up chooser for model type %q", head.Key)
			}
		}
	}
	return nil
})

func NewAppConfig() django.AppConfig {
	var (
		templateFS = filesystem.Sub(choosersFS, "assets/templates")
		staticFS   = filesystem.Sub(choosersFS, "assets/static")
	)

	var app = apps.NewAppConfig("chooser")
	app.TemplateConfig = &tpl.Config{
		AppName: "chooser",
		FS:      templateFS,
		Bases: []string{
			"chooser/modal/skeleton.tmpl",
			"chooser/modal/controls.tmpl",
			"chooser/modal/modal.tmpl",
		},
	}

	app.Init = func(settings django.Settings) error {
		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("chooser/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".png"),
				filesystem.MatchExt(".jpg"),
			),
		))
		return nil
	}

	app.Routing = func(m mux.Multiplexer) {
		var chooserRoot = m.Any("chooser/<<model_key>>/<<chooser_key>>", nil, "chooser")
		chooserRoot.Handle(mux.ANY, "list/", mux.NewHandler(viewChooserList), "list")
	}

	return app
}

func viewChooserList(w http.ResponseWriter, r *http.Request) {
	var (
		vars       = mux.Vars(r)
		modelKey   = vars.Get("model_key")
		chooserKey = vars.Get("chooser_key")
	)

	chooserMap, ok := choosers.Get(modelKey)
	if !ok {
		logger.Error("No chooser registered for model type %s", modelKey)
		except.Fail(
			http.StatusNotFound,
			"Chooser not found for model type %q", modelKey,
		)
		return
	}

	chooser, ok := chooserMap.Get(chooserKey)
	if !ok {
		logger.Error("No chooser registered for key %s", chooserKey)
		except.Fail(
			http.StatusNotFound,
			"Chooser not found for key %s", chooserKey,
		)
		return
	}

	var view = chooser.ListView()
	if view == nil {
		logger.Error("No list view registered for model type %s", modelKey)
		except.Fail(
			http.StatusNotFound,
			"List view not found for model type %q", modelKey,
		)
		return
	}

	views.Invoke(view, w, r)
}
