package todos

import (
	"database/sql"
	"embed"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/filesystem"
	"github.com/Nigel2392/django/core/filesystem/staticfiles"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var todosFS embed.FS

var globalDB *sql.DB

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewDBAppConfig(
		"todos",
	)

	cfg.Routing = func(m django.Mux) {
		m.Get("/list", mux.NewHandler(ListTodos))
		m.Post("/done", mux.NewHandler(MarkTodoDone))
	}

	cfg.Init = func(settings django.Settings, db *sql.DB) error {
		var (
			tplFS    = filesystem.Sub(todosFS, "assets/templates")
			staticFS = filesystem.Sub(todosFS, "assets/static")
		)

		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("todos/"),
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
			AppName: "todos",
			FS:      tplFS,
			Bases: []string{
				"todos/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("todos/"),
				filesystem.MatchExt(".tmpl"),
				filesystem.MatchExt(".tmpl"),
			),
		})

		// Set the global database
		globalDB = db

		// Create the todos table
		_, err := db.Exec(createTable)
		return err
	}

	return cfg
}
