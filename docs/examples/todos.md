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
        // ...<a href="#defining-your-templates">Setting up templates</a>
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
    m.Get("/todos", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
        // ...<a href="#setting-up-the-list-view">Setting up views</a>
    }), "list")

    m.Post("/todos/&lt;&lt;id&gt;&gt;/done", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
        // ...<a href="#finishing-a-todo">Finishing a todo</a>
    }), "done")
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
    countTodos = `SELECT COUNT(id) FROM todos`
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

We will also define a function to count the number of todos in the database.

This is mainly used for pagination.

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

func CountTodos(ctx context.Context) (int, error) {
    var count int
    if err := db.QueryRowContext(ctx, countTodos).Scan(&count); err != nil {
        return 0, err
    }
    return count, nil
}
```

## Setting up views

In `views.go`, we will define the views for the todo app.

The views will be responsible for rendering the list of todos, and for marking todos as done.

View functions in Go-Django are equivalent to a `http.HandlerFunc`.

### Setting up the list view

In the `GET` route for `/todos`, we will define the view that will display the list of todos.

We will also [paginate](../pagination.md#example-usage) the list of todos.

Let's define the `ListTodos` function.

```go
func ListTodos(w http.ResponseWriter, r *http.Request) {
    // Create a new paginator for the Todo model
    var paginator = pagination.Paginator[Todo]{
        // Define a function to retrieve a list of objects based on the amount and offset
        GetObjects: func(amount, offset int) ([]Todo, error) {
            return ListAllTodos(
                r.Context(), amount, offset,
            )
        },
        GetCount: func() (int, error) {
            return CountTodos(r.Context())
        },
    }

    // Get the page number from the request's query string
    // We provide a utility function to get the page number from a string, int(8/16/32/64) and uint(8/16/32/64/ptr).
    var pageNum = pagination.GetPageNum(
        r.URL.Query().Get("page"),
    )

    // Get the page from the paginator
    // 
    // This will return a PageObject[Todo] which contains the list of todos for the current page.
    var page, err = paginator.Page(pageNum)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Create a new RequestContext
    // Add the page object to the context
    var context = ctx.RequestContext(r)
    context.Set("Page", page)

    // Render the template
    err = tpl.FRender(
        w, context,
        "todos/list.html",
    )
    if err != nil {
        http.Error(w, err.Error(), 500)
    }
}
```

For the `ListTodos` function to work, we need to define the template for the list of todos.

This is done in [the next section](#setting-up-the-list-template).

### Finishing a todo

For marking todos as done, we will define the `MarkTodoDone` function.

This function will be called when a `POST` request is made to `/todos/<<id>>/done`.

If the todo is already marked as done; we will unmark it, and vice versa.

Then we will send a JSON response back to the client, indicating the status of the todo.

```go
func MarkTodoDone(w http.ResponseWriter, r *http.Request) {
    // Get the todo ID from the URL
    var vars = mux.Vars(r)
    var id = vars.GetInt("id")
    if id == 0 {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "error",
            "error":  "Invalid todo ID",
        })
        return
    }

    // Get the todo from the database
    var todo, err = GetTodoByID(r.Context(), id)
    if err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "error",
            "error":  err.Error(),
        })
        return
    }

    // Mark the todo as done
    todo.Done = !todo.Done

    // Save the todo
    err = todo.Save(r.Context())
    if err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "error",
            "error":  err.Error(),
        })
        return
    }

    // Send a JSON response
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "success",
        "done":   todo.Done,
    })
}
```

## Defining your templates

We can now define our `template/html` templates.

We will create the following filetree structure inside of our assets folder.

```filesystem
assets/
    templates/
        todos/
            base.html
            list.html
```

### Setting up the base template

The base template will contain the basic structure of our HTML page.

This will include the `<!DOCTYPE html>`, `<html>`, `<head>`, and `<body>` tags, as well as some `template/html` tags for overrides in the child templates.

First, we will make sure that the base template is defined in the `base.html` file.

This can be done by defining the `base` block, and later inheriting from this block in the child templates.

Following that we will define the following blocks:

* `title` - This block will be used to set the title of the page.
* `content` - This block will be used to include the content of the page.
* `extra_css` - This block will be used to include any extra CSS files.
* `extra_js` - This block will be used to include any extra JS files.

```html
{{ define "base" }}
<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        
        {{ block "extra_css" . }}{{ end }}

        <title>{{ block "title" . }}{{ end }}</title>
    </head>
    <body>

        {{ block "content" . }}{{ end }}

        {{ block "extra_js" . }}{{ end }}
    </body>
</html>
{{ end }}
```

### Setting up the list template

The list template will contain the list of todos.

It will also populate the previously defined blocks in the base template.

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

        <!-- Range over the paginator results -->
        {{ range $todo := $page.Results }}

            <div class="todo-item">

                <h3>{{ $todo.Title }}</h3>
                <p>{{ $todo.Description }}</p>
                
                <!-- Submit to the todos app URL, use the template function to generate the URL based on what was previously defined. -->
                <form class="todo-form" action="{{ url "todos:done" $todo.ID }}" method="POST">
                    <input type="hidden" class="csrftoken-input" name="csrf_token" value="{{ .CsrfToken }}">
                    <button type="submit">
                        {{ if $todo.Done }}Unmark{{ else }}Mark{{ end }} as done
                    </button>
                </form>
            </div>

        {{ else }}
            <p>No todos found</p>
        {{ end }}

        <!-- 
         Paginator buttons - takes in:
            1. Page query parameter name.
            2. max amount page numbers shown.
            3. included and the query parameters. 

         Under the hood this uses a templ.Component.
         -->
        {{ $page.HTML "page" 5 .Request.URL.Query }}
    </div>

{{ end }}
```

### Adding CSS for styling the todos

Now that we have defined the templates it is time to add some CSS to style the todos.

We will create a new file called `todos.css` in the `assets/static/css` directory.

```filesystem
assets/
    static/
        css/
            todos.css
```

The CSS file will contain the following styles:

```css
.todo-list-wrapper {
    display: flex;
    flex-direction: column;
    align-items: center;
}

.todo-item {
    margin: 10px;
    padding: 10px;
    border: 1px solid #ccc;
    border-radius: 5px;
    width: 50%;
}

.todo-item h3 {
    margin: 0;
}

.todo-item p {
    margin: 0;
}

.todo-item form {
    margin-top: 10px;
}

.todo-item button {
    padding: 5px 10px;
    border: 1px solid #ccc;
    border-radius: 5px;
    background-color: #f0f0f0;
    cursor: pointer;
}
```

### Adding javascript for marking todos as done

We will also need to add some JavaScript to handle marking todos as done.

This will make a `POST` request to the `/todos/<<id>>/done` URL - this will be retrieved from the form's action attribute.

We will create a new file called `todos.js` in the `assets/static/js` directory.

```filesystem
assets/
    static/
        js/
            todos.js
```

We will define a simple function that will make a request to the todo app URL.

```javascript
async function markAsDone(url, csrftoken) {
    var response = await fetch(url, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-CSRFToken": csrftoken,
        },
    });

    var data = await response.json();
    if (data.status === "success") {
        alert("Todo marked as done");
    } else {
        alert("An error occurred");
    }
}

function initForm(form) {
    const formUrl = form.getAttribute("action");
    const csrfTokenInput = form.querySelector(".csrftoken-input");

    form.addEventListener("submit", function(e) {
        e.preventDefault();
        markAsDone(formUrl, csrfTokenInput.value);
    });
}

document.addEventListener("DOMContentLoaded", function() {
    const forms = document.querySelectorAll(".todo-form");
    forms.forEach(initForm);
});
```
