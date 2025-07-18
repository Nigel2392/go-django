package todos

import (
	"embed"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var todosFS embed.FS

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewDBAppConfig(
		"todos",
	)

	// Set up routing for this app
	cfg.Routing = func(m django.Mux) {
		var todosGroup = m.Any("/todos", nil, "todos")
		todosGroup.Get("/list", mux.NewHandler(ListTodos), "list")
		todosGroup.Post("/<<id>>/done", mux.NewHandler(MarkTodoDone), "done")
	}

	// Models which are a part of this app
	// All models should be registered by an AppConfig.
	cfg.ModelObjects = []attrs.Definer{
		&Todo{},
	}

	// An initializer function that will be called when the app is initialized
	//
	// This is where you can set up templates, static files, and other app-specific configurations.
	cfg.Init = func(settings django.Settings, db drivers.Database) error {
		var (
			tplFS    = filesystem.Sub(todosFS, "assets/templates")
			staticFS = filesystem.Sub(todosFS, "assets/static")
		)

		// Set up the static files for this app
		// They are stored in the "assets/static" directory
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

		// Set up the templates for this app
		// They are stored in the "assets/templates" directory
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

		// Register the app to the admin interface
		var _ = admin.RegisterApp(
			"todos",

			// Register the Todo app to the admin interface
			admin.AppOptions{
				RegisterToAdminMenu: true,
				AppLabel:            trans.S("Todo App"),
				AppDescription:      trans.S("Manage the todos for your todo app."),
				MenuLabel:           trans.S("Todos"),
			},

			// Register the Todo model to the admin interface
			admin.ModelOptions{
				Model:               &Todo{},
				RegisterToAdminMenu: true,

				// Customize the labels for the fields in the admin interface
				Labels: map[string]func() string{
					"ID":          trans.S("ID"),
					"Title":       trans.S("Todo Title"),
					"Description": trans.S("Todo Description"),
					"Done":        trans.S("Is Done"),
				},

				// Custom fields for the admin interface
				// when creating a new todo
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

				// Custom fields for the admin interface
				// when editing an existing todo
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

				// Custom fields for the admin interface
				// when listing todos
				ListView: admin.ListViewOptions{

					// These fields will be displayed in the list view
					ViewOptions: admin.ViewOptions{
						Fields: []string{
							"ID",
							"Title",
							"Description",
							"Done",
						},
					},

					// Define custom columns for the list view
					//
					// The "Title" column will be a link to the edit view of the todo
					Columns: map[string]list.ListColumn[attrs.Definer]{
						"Title": list.LinkColumn(
							trans.S("Title"),
							"Title", func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								return django.Reverse("admin:apps:model:edit", "todos", "Todo", defs.Get("ID"))
							},
						),
					},

					// The amount of items to display per page
					PerPage: 25,
				},
			},
		)

		return nil
	}

	return cfg
}
