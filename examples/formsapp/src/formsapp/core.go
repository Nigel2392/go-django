package formsapp

import (
	"embed"
	"io/fs"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2"
)

//go:embed assets/**
var assetFilesystem embed.FS

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewAppConfig(
		"core",
	)

	cfg.Routing = func(m mux.Multiplexer) {
		m.Handle(mux.GET, "/", http.HandlerFunc(Index), "index")
		m.Handle(mux.GET, "/contact", http.HandlerFunc(Contact), "contact")
		m.Handle(mux.POST, "/contact", http.HandlerFunc(Contact), "contact")
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

		static.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("formsapp/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
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
