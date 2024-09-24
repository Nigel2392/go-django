package todos

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/filesystem"
	"github.com/Nigel2392/django/core/filesystem/staticfiles"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/views/list"
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
		var todosGroup = m.Any("/todos", nil, "todos")
		todosGroup.Get("/list", mux.NewHandler(ListTodos), "list")
		todosGroup.Post("/<<id>>/done", mux.NewHandler(MarkTodoDone), "done")
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

	var _ = admin.RegisterApp(
		"todos",
		admin.AppOptions{
			RegisterToAdminMenu: true,
			AppLabel:            fields.S("Todo App"),
			AppDescription:      fields.S("Manage the todos for your todo app."),
			MenuLabel:           fields.S("Todos"),
		},
		admin.ModelOptions{
			Model:               &Todo{},
			RegisterToAdminMenu: true,
			Labels: map[string]func() string{
				"ID":          fields.S("ID"),
				"Title":       fields.S("Todo Title"),
				"Description": fields.S("Todo Description"),
				"Done":        fields.S("Is Done"),
			},
			GetForID: func(identifier any) (attrs.Definer, error) {
				var id, ok = identifier.(int)
				if !ok {
					var u, err = strconv.Atoi(fmt.Sprint(identifier))
					if err != nil {
						return nil, err
					}
					id = u
				}
				return GetTodoByID(
					context.Background(),
					id,
				)
			},
			GetList: func(amount, offset uint, fields []string) ([]attrs.Definer, error) {
				var todos, err = ListAllTodos(
					context.Background(), int(amount), int(offset),
				)
				if err != nil {
					return nil, err
				}
				var items = make([]attrs.Definer, 0)
				for _, u := range todos {
					var cpy = u
					items = append(items, &cpy)
				}
				return items, nil
			},
			AddView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Exclude: []string{"ID"},
				},
				Panels: []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Title"),
					),
					admin.FieldPanel("Description"),
					admin.FieldPanel("Done"),
				},
			},
			EditView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Exclude: []string{"ID"},
				},
				Panels: []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Title"),
					),
					admin.FieldPanel("Description"),
					admin.FieldPanel("Done"),
				},
			},
			ListView: admin.ListViewOptions{
				ViewOptions: admin.ViewOptions{
					Fields: []string{
						"ID",
						"Title",
						"Description",
						"Done",
					},
				},
				Columns: map[string]list.ListColumn[attrs.Definer]{
					"Title": list.LinkColumn(
						fields.S("Title"),
						"Title", func(defs attrs.Definitions, row attrs.Definer) string {
							return django.Reverse("admin:apps:model:edit", "todos", "Todo", defs.Get("ID"))
						},
					),
				},
				PerPage: 25,
			},
		},
	)

	return cfg
}
