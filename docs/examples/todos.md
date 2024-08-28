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

### Setting up static files

### Creating the app structure

### Setting up the directory structure

## Defining the model

## Setting up views

## Defining your templates
