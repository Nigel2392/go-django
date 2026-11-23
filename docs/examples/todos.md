# Todo Application Example

The `todoapp` example (`examples/todoapp`) demonstrates how to build a database-backed Go-Django application with custom models, migrations, paginated views, HTML templates, embedded static assets, CSRF-protected fetch requests, and admin integration.

## Table of Contents

- [Project Structure](#project-structure)
- [Running the Example](#running-the-example)
- [Application Entrypoint (`main.go`)](#application-entrypoint-maingo)
- [Defining the Model (`todos/model.go`)](#defining-the-model-todosmodelgo)
- [App Configuration, Routing, and Admin (`todos/todo.go`)](#app-configuration-routing-and-admin-todostodogo)
- [Views (`todos/views.go`)](#views-todosviewsgo)
- [Templates and Static Assets](#templates-and-static-assets)

---

## Project Structure

Inside `examples/todoapp`, the project is organized as:

```text
examples/todoapp/
├── go.mod
├── main.go
└── todos/
    ├── model.go
    ├── todo.go
    ├── views.go
    └── assets/
        ├── static/
        │   └── todos/
        │       ├── css/
        │       │   └── todos.css
        │       └── js/
        │           └── todos.js
        └── templates/
            └── todos/
                ├── base.tmpl
                └── list.tmpl
```

---

## Running the Example

Because `examples/todoapp` uses `migrator` to manage database tables, generate and apply migrations before starting the server. From the repository root:

```bash
go run ./examples/todoapp makemigrations
go run ./examples/todoapp migrate
go run ./examples/todoapp
```

Alternatively, from `examples/todoapp`:

```bash
go run . makemigrations
go run . migrate
go run .
```

On startup, `main.go` ensures a demo administrator account exists (`admin@localhost` / `Administrator123!`).

- **Admin interface**: Open `http://127.0.0.1:8080/admin/login/` and sign in with `admin@localhost` / `Administrator123!` to add or edit todos.
- **Public todo list**: Open `http://127.0.0.1:8080/todos/list` to browse paginated todos and toggle completion state.

See [CLI](../cli.md) for more management commands.

---

## Application Entrypoint (`main.go`)

`examples/todoapp/main.go` creates `./.private/todoapp`, opens a SQLite database with `drivers.Open`, configures settings with `django.APPVAR_*` constants, installs the contrib apps alongside `todos.NewAppConfig`, initializes the application, ensures the demo admin user exists, and calls `app.Serve()`:

```go
package main

import (
	"context"
	"fmt"
	"net/mail"
	"os"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func main() {
	os.MkdirAll("./.private/todoapp", 0755)

	db, err := drivers.Open(context.Background(), "sqlite3", "./.private/todoapp/db.sqlite3")
	if err != nil {
		panic(err)
	}

	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         true,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE:      db,

			auth.APPVAR_AUTH_EMAIL_LOGIN:  true,
			migrator.APPVAR_MIGRATION_DIR: "./.private/todoapp/migrations",
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			messages.NewAppConfig,
			revisions.NewAppConfig(),
			auditlogs.NewAppConfig,
			reports.NewAppConfig(),
			migrator.NewAppConfig,
			todos.NewAppConfig,
		),
	)

	checks.Shutup("model.cant_check", true)
	checks.Shutup("auth.login_redirect_url_not_set", true)
	checks.Shutup("admin.model_not_fully_implemented", true)
	checks.Shutup("field.invalid_db_type", func(m checks.Message) bool {
		return m.Object.(attrs.Field).Name() == "GroupPermissions"
	})

	if err := app.Initialize(); err != nil {
		panic(err)
	}

	user := &auth.User{}
	e, _ := mail.ParseAddress("admin@localhost")
	user.Email = (*drivers.Email)(e)
	user.Username = "admin"
	user.IsAdministrator = true
	user.IsActive = true
	user.SetPassword("Administrator123!")

	user, _, err = queries.
		GetQuerySet(&auth.User{}).
		Preload("EntrySet").
		Filter("Email", e.Address).
		GetOrCreate(user)
	if err != nil {
		panic(fmt.Errorf("failed to create admin user: %w", err))
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
```

See [Configuring](../configuring.md) and [Apps](../apps.md) for all `APPVAR_*` settings and app registration options.

---

## Defining the Model (`todos/model.go`)

`Todo` embeds `models.Model` (which binds the `"todos"` database table via `table:"todos"` and provides `Save(ctx)` and ORM integration) and implements `FieldDefs(ctx context.Context)`:

```go
package todos

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type Todo struct {
	models.Model `table:"todos"`
	ID           int
	Title        string
	Description  string
	Done         bool
}

func (m *Todo) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m, func(attrs.Definer) []attrs.Field {
		return []attrs.Field{
			attrs.NewField(m, "ID", &attrs.Config{
				Primary:  true,
				ReadOnly: true,
				Label:    "ID",
				HelpText: trans.S("The unique identifier of the model"),
			}),
			attrs.NewField(m, "Title", &attrs.Config{
				Label:    "Title",
				HelpText: trans.S("The title of the todo"),
			}),
			attrs.NewField(m, "Description", &attrs.Config{
				Label:    "Description",
				HelpText: trans.S("A description of the todo"),
				FormWidget: func(cfg attrs.Config) widgets.Widget {
					return widgets.NewTextarea(nil)
				},
			}),
			attrs.NewField(m, "Done", &attrs.Config{
				Label:    "Done",
				HelpText: trans.S("Indicates whether the todo is done or not"),
				Blank:    true,
			}),
		}
	})
}
```

Notice `attrs.Config` for field definitions, `trans.S(...)` for translatable labels and help text, and `m.Model.Define(ctx, m, ...)` inside `FieldDefs(ctx)`. See [Attrs](../attrs/attrs.md) and [Queries](../../queries/docs/README.md).

---

## App Configuration, Routing, and Admin (`todos/todo.go`)

`todos/todo.go` embeds `assets/**`, defines `NewAppConfig()`, mounts `/todos/list` and `/todos/<<id>>/done`, registers static files and templates, and registers `Todo` with the admin interface:

```go
package todos

import (
	"context"
	"embed"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
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
	cfg := apps.NewDBAppConfig("todos")

	cfg.Routing = func(m mux.Multiplexer) {
		todosGroup := m.Any("/todos", nil, "todos")
		todosGroup.Get("/list", mux.NewHandler(ListTodos), "list")
		todosGroup.Post("/<<id>>/done", mux.NewHandler(MarkTodoDone), "done")
	}

	cfg.ModelObjects = []attrs.Definer{
		&Todo{},
	}

	cfg.Init = func(settings django.Settings, db drivers.Database) error {
		tplFS := filesystem.Sub(todosFS, "assets/templates")
		staticFS := filesystem.Sub(todosFS, "assets/static")

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
			),
		})

		admin.RegisterApp(
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
				Labels: map[string]func(context.Context) string{
					"ID":          trans.S("ID"),
					"Title":       trans.S("Todo Title"),
					"Description": trans.S("Todo Description"),
					"Done":        trans.S("Is Done"),
				},
				AddView: admin.FormViewOptions{
					ViewOptions: admin.ViewOptions{Exclude: []string{"ID"}},
					Panels: []admin.Panel{
						admin.TitlePanel(admin.FieldPanel("Title")),
						admin.FieldPanel("Description"),
						admin.FieldPanel("Done"),
					},
				},
				EditView: admin.FormViewOptions{
					ViewOptions: admin.ViewOptions{Exclude: []string{"ID"}},
					Panels: []admin.Panel{
						admin.TitlePanel(admin.FieldPanel("Title")),
						admin.FieldPanel("Description"),
						admin.FieldPanel("Done"),
					},
				},
				ListView: admin.ListViewOptions{
					ViewOptions: admin.ViewOptions{
						Fields: []string{"ID", "Title", "Description", "Done"},
					},
					Columns: map[string]list.ListColumn[attrs.Definer]{
						"Title": list.LinkColumn(
							trans.S("Title"),
							"Title",
							func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								return django.Reverse("admin:apps:model:edit", "todos", "Todo", defs.Get("ID"))
							},
						),
					},
					PerPage: 25,
				},
			},
		)

		return nil
	}

	return cfg
}
```

See [Admin App](../apps/admin/app.md), [Routing](../routing.md), and [Static Files](../staticfiles.md).

---

## Views (`todos/views.go`)

`ListTodos` paginates todos using `pagination.Paginator[[]*Todo, *Todo]`, `queries.ListObjects`, and `queries.CountObjects`, treating `drivers/errors.NoRows` as an empty list rather than an internal server error:

```go
package todos

import (
	"encoding/json"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/mux"
)

func ListTodos(w http.ResponseWriter, r *http.Request) {
	paginator := pagination.Paginator[[]*Todo, *Todo]{
		Context: r.Context(),
		Amount:  25,
		GetObjects: func(amount, offset int) ([]*Todo, error) {
			return queries.ListObjects(&Todo{}, uint64(offset), uint64(amount))
		},
		GetCount: func() (int, error) {
			ct, err := queries.CountObjects(&Todo{})
			return int(ct), err
		},
	}

	pageNum := pagination.GetPageNum(r.URL.Query().Get("page"))
	page, err := paginator.Page(pageNum)
	if err != nil && !errors.Is(err, errors.NoRows) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	context := ctx.RequestContext(r)
	context.Set("Page", page)

	if err := tpl.FRender(w, context, "todos", "todos/list.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func MarkTodoDone(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r).GetInt("id")
	if id == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "Invalid todo ID",
		})
		return
	}

	todo, err := queries.GetObject(&Todo{}, id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	todo.Done = !todo.Done
	if err := todo.Save(r.Context()); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"done":   todo.Done,
	})
}
```

See [Pagination](../pagination.md) and [Views](../views.md).

---

## Templates and Static Assets

Static assets live under `todos/assets/static/todos/` and templates live under `todos/assets/templates/todos/`.

`todos/assets/templates/todos/list.tmpl` inherits from `todos/base.tmpl`, loads `todos/css/todos.css` and `todos/js/todos.js` via `{{ static ... }}`, reverses `todos:done` for each item's form action, embeds `{{ csrf_token }}`, and renders pagination links with `$page.HTML`:

```html
{{ define "title" }}Todos{{ end }}

{{ define "extra_css" }}
    <link rel="stylesheet" href="{{ static "todos/css/todos.css" }}">
{{ end }}

{{ define "extra_js" }}
    <script src="{{ static "todos/js/todos.js" }}"></script>
{{ end }}

{{ define "content" }}
    <div class="todo-list-wrapper">
        {{ $page := (.Get "Page") }}
        {{ $csrfToken := (csrf_token) }}

        {{ range $todo := $page.Results }}
            <div class="todo-item">
                <h3>{{ $todo.Title }}</h3>
                <p>{{ $todo.Description }}</p>

                <form class="todo-form" action="{{ url "todos:done" $todo.ID }}" method="POST">
                    <input type="hidden" class="csrftoken-input" name="csrf_token" value="{{ $csrfToken }}">
                    <button type="submit" class="update">
                        {{ if $todo.Done }}Unmark{{ else }}Mark{{ end }} as done
                    </button>
                </form>
            </div>
        {{ else }}
            <p>No todos found</p>
        {{ end }}

        {{ $page.HTML "page" 5 .Request.URL.Query }}
    </div>
{{ end }}
```

`todos/assets/static/todos/js/todos.js` intercepts form submissions and sends an asynchronous `POST` request with the `X-CSRF-Token` header:

```javascript
async function markAsDone(url, csrftoken, update) {
    var response = await fetch(url, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrftoken,
        },
    });

    var data = await response.json();
    if (data.status === "success") {
        if (data.done) {
            update("Task marked as done!");
        } else {
            update("Task marked as not done!");
        }
    } else {
        alert("An error occurred");
    }
}
```

See [Rendering](../rendering.md) and [Auth App](../apps/auth/app.md) for template rendering and authentication details.
