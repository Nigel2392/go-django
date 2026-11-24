package main

import (
	"embed"
	"net/http"

	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2"
)

//go:embed assets/**
var coreFS embed.FS

type CoreAppConfig struct {
	*apps.AppConfig
}

func NewAppConfig() *CoreAppConfig {
	var CONFIG = &CoreAppConfig{
		AppConfig: apps.NewAppConfig("core"),
	}

	CONFIG.ModelObjects = []attrs.Definer{}

	staticfiles.AddFS(filesystem.Sub(coreFS, "assets/static"), filesystem.MatchAnd(
		filesystem.MatchPrefix("commerce/"),
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

	CONFIG.TemplateConfig = &tpl.Config{
		AppName: "commerce",
		Bases: []string{
			"commerce/base.tmpl",
		},
		FS: filesystem.Sub(coreFS, "assets/templates"),
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("commerce/"),
			filesystem.MatchExt(".tmpl"),
		),
	}
	CONFIG.Routing = func(m mux.Multiplexer) {

		m.Handle(mux.GET, "/", http.HandlerFunc(IndexView))
	}

	return CONFIG
}
