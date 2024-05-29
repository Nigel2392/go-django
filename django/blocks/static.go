package blocks

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
)

//go:embed assets/static/**
var _staticFS embed.FS

//go:embed assets/templates/**
var _templateFS embed.FS

var (
	staticFS   fs.FS
	templateFS fs.FS
)

func NewAppConfig() *apps.AppConfig {
	var cfg = apps.NewAppConfig(
		"blocks",
	)

	cfg.Init = func(settings django.Settings) error {
		staticfiles.AddFS(
			staticFS,
			tpl.MatchAnd(
				tpl.MatchPrefix("blocks"),
				tpl.MatchOr(
					tpl.MatchSuffix(".css"),
					tpl.MatchSuffix(".js"),
					tpl.MatchSuffix(".png"),
					tpl.MatchSuffix(".jpg"),
					tpl.MatchSuffix(".jpeg"),
					tpl.MatchSuffix(".svg"),
				),
			),
		)

		tpl.AddFS(
			templateFS,
			tpl.MatchAnd(
				tpl.MatchPrefix("blocks"),
				tpl.MatchOr(
					tpl.MatchSuffix(".html"),
					tpl.MatchSuffix(".tmpl"),
				),
			),
		)

		return tpl.Bases("blocks",
			"blocks/base.tmpl",
		)
	}

	return cfg
}

func init() {
	var err error
	staticFS, err = fs.Sub(_staticFS, "assets/static")
	if err != nil {
		panic(err)
	}

	templateFS, err = fs.Sub(_templateFS, "assets/templates")
	if err != nil {
		panic(err)
	}

}
