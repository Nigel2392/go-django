# Todo Application Example

This example demonstrates how to create a simple todo application using Go-Django.

Not only will you learn how to create a simple todo application, but you will also learn how to structure your application, define models, views, and templates.

To get started, you need to have Go-Django installed on your machine. If you haven't installed Go-Django yet, you can follow the installation instructions [here](../installation.md).

## Setting up `main.go`

To get started, create a new directory for your project and create a new file named `main.go`.

```bash
go mod init todos
mkdir todos
touch main.go
```

In `main.go`, you will need to import the `django` package and create a new instance of the `django.Application` struct.

We will set this up with local SQLite storage for the database, as well as define the web server configuration.

The configuration for our todos app will also be referenced, but we will [define this later](#creating-the-app).

```go
package main

import (
    "embed"
    "net/http"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"

    "github.com/Nigel2392/go-django"
    "github.com/Nigel2392/go-django/contrib/admin"
    "github.com/Nigel2392/go-django/contrib/auth"
    "github.com/Nigel2392/go-django/contrib/session"
)

//go:embed assets/*
var assetsFS embed.FS

func main() {
    var app = django.App(
        django.Configure(map[string]interface{}{
            "ALLOWED_HOSTS": []string{"*"},
            "DEBUG":         false,
            "HOST":          "127.0.0.1",
            "PORT":          "8080",
            "DATABASE": func() *sql.DB {
                // var db, err = sql.Open("mysql", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true")
                var db, err = sql.Open("sqlite3", "./.private/db.sqlite3")
                if err != nil {
                    panic(err)
                }
                return db
            }(),
        }),
        django.Apps(
            session.NewAppConfig,
            auth.NewAppConfig,
            admin.NewAppConfig,
            todos.NewAppConfig,
        ),
    )
}
```

We have defined the following in the above code snippet:

* The [configuration](../configuring.md) for our app. We have set the `ALLOWED_HOSTS` to `*`, which means that all hosts are allowed to access the app.  
  We have also set the `DEBUG` flag to `true`, which means that the app will run in debug mode.

* A filesystem for our assets (staticfiles and templates).

* The app will be running on [`http://127.0.0.1:8080`](http://127.0.0.1:8080).

* The database configuration. We are using SQLite as the database for this app, then we created a new SQLite database named `db.sqlite3` in the `.private` directory.

* The apps that we want to include in our app: `session`, `auth`, `admin`, and `todos`.

* The `todos` app is not yet defined.

## Creating the app

We have previously created the directory called `todos`.  

For this app to work properly we will create 3 files to maintain separation of concerns:

* `app.go` - This file will contain the app struct and its configuration.
* `models.go` - This file will contain the model definition for the todo app.
* `views.go` - This file will contain the views for the todo app.

```bash
touch todos/app.go
touch todos/models.go
touch todos/views.go
```

In `app.go`, we will define the app struct and its configuration.

It is possible to retrieve apps later on with [`django.GetApp[TodosAppConfig]("todos")`](../apps#retrieving-your-app-for-later-use).

```go
type TodosAppConfig struct {
    *apps.DBRequiredAppConfig
}
```

Let's now create the `NewAppConfig` function that will return an instance of the `TodosAppConfig` struct.

<pre>
func NewAppConfig() *TodosAppConfig {
    var todosApp = &TodosAppConfig{
        DBRequiredAppConfig: apps.NewDBAppConfig(
            "todos",
        ),
    }

    // Dependencies for this app
    // This is a list of app names that this app depends on.
    // This app depends on the `auth` app, and cannot properly function without them.
    todosApp.Deps = []string{
        "auth",
    }

    // Will be called for the app's initialization (before any `OnReady` functions are called)
    todosApp.Init = func(settings django.Settings, db *sql.DB) error {
        // ...<a href="#setting-up-templates">Setting up templates</a>
        // ...<a href="#setting-up-static-files">Setting up static files</a>
    }

    // Will be called after all apps have been initialized
    todosApp.Ready = func() error {
        // Do any extra setup which depends on other apps
    }

    todosApp.Routing = func(m django.Mux) {
        // ...<a href="#setting-up-routes">Setting up routes</a>
    }

    return todosApp
}
</pre>

### Setting up routes

In the previously defined `Routing` function, we will define the routes for our app.

Creation, editing and deletion of todos will be done via the [admin interface](../apps/admin/app.md).

This app will only need a route for displaying the list of todos, and for marking todos as done.

<pre>
todosApp.Routing = func(m django.Mux) {
    m.Get("/todos", func(w http.ResponseWriter, r *http.Request) {
        // ...<a href="#setting-up-the-list-view">Setting up views</a>
    })

    m.Post("/todos/&lt;id&gt;/done", func(w http.ResponseWriter, r *http.Request) {
        // ...<a href="#finishing-a-todo">Finishing a todo</a>
    })
}
</pre>

We defined the route for marking todos on the `POST` method, a variable `id` is passed in the URL.

### Setting up static files

Serving static files is important for most web applications.

We will use the [staticfiles](../staticfiles.md#configuration) package to setup serving static files.

Our static directory is located in `assets/static`.

We will use [`filesystem.Sub`](../filesystem.md#sub) to make sure that only files in the `assets/static` directory are served.

The only files we will be serving as staticfiles are CSS, JS and PNG- image files.

```go
var staticFS = filesystem.Sub(
    assetsFS, "assets/static",
)

staticfiles.AddFS(staticFS, filesystem.MatchAnd(
    filesystem.MatchPrefix("core/"),
    filesystem.MatchOr(
        filesystem.MatchExt(".css"),
        filesystem.MatchExt(".js"),
        filesystem.MatchExt(".png"),
    ),
))
```

## Defining the model

In `models.go`, we will define the model for the todo app.

The model for todos is relatively simple, and contains 4 fields:

* `ID` - The unique identifier for the todo.
* `Title` - The title of the todo.
* `Description` - A description of the todo.
* `Done` - A boolean field that indicates whether the todo is done or not.

```go
type Todo struct {
    ID          int
    Title       string
    Description string
    Done        bool
}
```

As seen in [attrs](../attrs.md#defining-model-attributes), we will define the model attributes for the `Todo` model.

This should be a method called `FieldDefs`.

```go
func (m *Todo) FieldDefs() attrs.Definitions {
  return attrs.Define(m,
    attrs.NewField(m, "ID", &attrs.FieldConfig{
        Primary:  true,
        ReadOnly: true,
        Label:    "ID",
        HelpText: "The unique identifier of the model",
    }),
    attrs.NewField(m, "Title", &attrs.FieldConfig{
        Label:    "Title",
        HelpText: "The title of the todo",
    }),
    attrs.NewField(m, "Description", &attrs.FieldConfig{
        Label:    "Description",
        HelpText: "A description of the todo",
        FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
            return widgets.NewTextarea(nil)
        },
    }),
    attrs.NewField(m, "Done", &attrs.FieldConfig{
        Label:    "Done",
        HelpText: "Indicates whether the todo is done or not",
    }),
  )
}
```

### Creating the model's queries

Our model will need a few queries to interact with the database.

We will need to define all database logic pertaining to the `Todo` model - Go-Django does not do this and only looks for interface methods.

The following columns should be present in the `todos` table:

* `id` - The unique identifier for the todo.
* `title` - The title of the todo.
* `description` - A description of the todo.
* `done` - A boolean field that indicates whether the todo is done or not.

```go
const (
    createTable = `CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT,
        description TEXT,
        done BOOLEAN
    )`
    listTodos = `SELECT id, title, description, done FROM todos ORDER BY id DESC LIMIT ? OFFSET ?`
    insertTodo = `INSERT INTO todos (title, description, done) VALUES (?, ?, ?)`
    updateTodo = `UPDATE todos SET title = ?, description = ?, done = ? WHERE id = ?`
    selectTodo = `SELECT id, title, description, done FROM todos WHERE id = ?`
)
```

Let's set up the methods for our todo model.

```go
// Save is a method that will either insert or update the todo in the database.
// 
// If the todo has an ID of 0, it will be inserted into the database; otherwise, it will be updated.
// 
// This method should exist on all models that need to be saved to the database.
func (t *Todo) Save(ctx context.Context) error {
    if t.ID == 0 {
        return t.Insert(ctx)
    }
    return t.Update(ctx)
}

// Not Required
func (t *Todo) Insert(ctx context.Context) error {
    var res, err = db.ExecContext(ctx, insertTodo, t.Title, t.Description, t.Done)
    if err != nil {
        return err
    }
    var id, err = res.LastInsertId()
    if err != nil {
        return err
    }
    t.ID = int(id)
    return nil
}

// Not Required
func (t *Todo) Update(ctx context.Context) error {
    _, err := db.ExecContext(ctx, updateTodo, t.Title, t.Description, t.Done, t.ID)
    return err
}
```

Let's also define a function to list all todos, or retrieve a single one by it's ID.

```go
func ListAllTodos(ctx context.Context, limit, offset int) ([]Todo, error) {
    var rows, err = db.QueryContext(ctx, listTodos, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var todos []Todo
    for rows.Next() {
        var todo Todo
        if err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Done); err != nil {
            return nil, err
        }
        todos = append(todos, todo)
    }
    return todos, nil
}

func GetTodoByID(ctx context.Context, id int) (*Todo, error) {
    var todo Todo
    if err := db.QueryRowContext(ctx, selectTodo, id).Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Done); err != nil {
        return nil, err
    }
    return &todo, nil
}
```

## Setting up views

In `views.go`, we will define the views for the todo app.

The views will be responsible for rendering the list of todos, and for marking todos as done.

View functions in Go-Django are equivalent to a `http.HandlerFunc`.

```go
func ListTodos(w http.ResponseWriter, r *http.Request) {}

func FinishTodo(w http.ResponseWriter, r *http.Request) {}
```

### Setting up the list view

In the `GET` route for `/todos`, we will define the view that will display the list of todos.

We will also [paginate](../pagination.md#example-usage) the list of todos.



### Finishing a todo

## Defining your templates

### Adding simple CSS
