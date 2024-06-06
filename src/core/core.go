package core

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/core/urls"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var coreFS embed.FS

var _ = admin.RegisterApp(
	"core",
	admin.ModelOptions{
		Fields: []string{"Email", "Name", "Password", "Age", "Data", "Block"},
		Model:  &MainStruct{},
	},
)

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewAppConfig(
		"core", urls.Group("", "core").Add(
			urls.Pattern(
				urls.M("GET", "POST"),
				mux.NewHandler(Index),
			),
			urls.Pattern(
				urls.P("/about", mux.ANY),
				mux.NewHandler(About),
			),
		),
	)
	cfg.Init = func(settings django.Settings) error {
		var tplFS, err = fs.Sub(coreFS, "assets/templates")
		if err != nil {
			return err
		}

		staticFS, err := fs.Sub(coreFS, "assets/static")
		if err != nil {
			return err
		}

		tpl.AddFS(tplFS, tpl.MatchAnd(
			tpl.MatchPrefix("core/"),
			tpl.MatchOr(
				tpl.MatchExt(".tmpl"),
			),
		))

		staticfiles.AddFS(staticFS, tpl.MatchAnd(
			tpl.MatchPrefix("core/"),
			tpl.MatchOr(
				tpl.MatchExt(".css"),
				tpl.MatchExt(".js"),
				tpl.MatchExt(".png"),
				tpl.MatchExt(".jpg"),
				tpl.MatchExt(".jpeg"),
				tpl.MatchExt(".svg"),
				tpl.MatchExt(".gif"),
				tpl.MatchExt(".ico"),
			),
		))

		return tpl.Bases("core",
			"core/base.tmpl",
		)
	}

	return cfg
}
