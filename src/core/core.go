package core

import (
	"database/sql"
	"embed"
	"io/fs"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/filesystem"
	"github.com/Nigel2392/django/core/filesystem/staticfiles"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var coreFS embed.FS

var globalDB *sql.DB

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewDBAppConfig(
		"core",
	)

	cfg.Routing = func(m django.Mux) {
		//urls.Group("", "core").Add(
		//	urls.Pattern(
		//		urls.M("GET", "POST"),
		//		mux.NewHandler(Index),
		//	),
		//	urls.Pattern(
		//		urls.P("/about", mux.ANY),
		//		mux.NewHandler(About),
		//	),
		//),
		m.Handle(mux.GET, "/", mux.NewHandler(Index))
		m.Handle(mux.POST, "/", mux.NewHandler(Index))

		m.Handle(mux.GET, "/about", mux.NewHandler(About))
		m.Handle(mux.POST, "/about", mux.NewHandler(About))

	}

	cfg.Init = func(settings django.Settings, db *sql.DB) error {
		var tplFS, err = fs.Sub(coreFS, "assets/templates")
		if err != nil {
			return err
		}

		staticFS, err := fs.Sub(coreFS, "assets/static")
		if err != nil {
			return err
		}

		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("core/"),
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
				"core/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchOr(
					filesystem.MatchPrefix("core/"),
					filesystem.MatchPrefix("auth/"),
				),
				filesystem.MatchOr(
					filesystem.MatchExt(".tmpl"),
				),
			),
		})

		return nil
	}

	return cfg
}
