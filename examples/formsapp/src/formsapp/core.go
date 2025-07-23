package formsapp

import (
	"embed"
	"io/fs"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var assetFilesystem embed.FS

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewAppConfig(
		"core",
	)

	cfg.Routing = func(m django.Mux) {
		m.Handle(mux.GET, "/", mux.NewHandler(Index), "index")
		m.Handle(mux.GET, "/contact", mux.NewHandler(Contact), "contact")
		m.Handle(mux.POST, "/contact", mux.NewHandler(Contact), "contact")
	}

	cfg.Init = func(settings django.Settings) error {
		var tplFS, err = fs.Sub(assetFilesystem, "assets/templates")
		if err != nil {
			return err
		}

		staticFS, err := fs.Sub(assetFilesystem, "assets/static")
		if err != nil {
			return err
		}

		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("formsapp/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".png"),
				filesystem.MatchExt(".jpg"),
				filesystem.MatchExt(".jpeg"),
				filesystem.MatchExt(".svg"),
				filesystem.MatchExt(".gif"),
				filesystem.MatchExt(".ico"),
			),
		))

		tpl.Add(tpl.Config{
			AppName: "core",
			FS:      tplFS,
			Bases: []string{
				"formsapp/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("formsapp/"),
				filesystem.MatchExt(".tmpl"),
			),
		})

		return nil
	}

	return cfg
}
