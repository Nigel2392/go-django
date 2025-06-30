package todos

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/http"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views/list"
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

	cfg.ModelObjects = []attrs.Definer{
		&Todo{},
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

	var _ = contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &Todo{},
		GetInstance: func(identifier any) (interface{}, error) {
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
		GetInstances: func(amount, offset uint) ([]interface{}, error) {
			var todos, err = ListAllTodos(
				context.Background(), int(amount), int(offset),
			)
			if err != nil {
				return nil, err
			}
			var items = make([]interface{}, 0)
			for _, u := range todos {
				var cpy = u
				items = append(items, &cpy)
			}
			return items, nil
		},
	})

	var _ = admin.RegisterApp(
		"todos",
		admin.AppOptions{
			RegisterToAdminMenu: true,
			AppLabel:            trans.S("Todo App"),
			AppDescription:      trans.S("Manage the todos for your todo app."),
			MenuLabel:           trans.S("Todos"),
		},
		admin.ModelOptions{
			Model:               &Todo{},
			RegisterToAdminMenu: true,
			Labels: map[string]func() string{
				"ID":          trans.S("ID"),
				"Title":       trans.S("Todo Title"),
				"Description": trans.S("Todo Description"),
				"Done":        trans.S("Is Done"),
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
						trans.S("Title"),
						"Title", func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
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
