# Apps

## Introduction

Apps are at the core of the go-django framework.

They are the building blocks of your application, and can be used to create reusable and modular app-like components.

## Creating a new app

To create a new app let's first create a new sub-package.

This isn't necessarily required, but it is a good practice to keep your app's code separate from the rest of your codebase.

```bash
mkdir myapp
echo "package myapp" > myapp/myapp.go
```

Now let's create a new app struct.

```go
package myapp

import (
    "github.com/Nigel2392/django/apps"
)

type CustomApp struct {
	// *apps.AppConfig
    *apps.DBRequiredAppConfig
}
```

In [Configuring](./configuring.md#creating-the-app) we saw a function called `NewCustomAppConfig` being passed to the Go-Django initializer.

This function can be used to create a new instance of your app; and it should return an instance of your app struct.

This is a perfect time to set up your [templates](./rendering.md#initial-setup) and [static files](./staticfiles.md#initial-setup).

```go
var globalInstance *CustomApp

func NewCustomAppConfig() *CustomApp {
    var myCustomApp = &CustomApp{
        // AppConfig: apps.NewAppConfig("myapp"),
        DBRequiredAppConfig: apps.NewDBAppConfig("myapp"),
    }

    // Dependencies for this app
    // 
    // This is a list of app names that this app depends on.
    // 
    // I.E. if this app depends on the `session` app, you would add `session` to this list.
	myCustomApp.Deps = []string{

	}

    // Will be called for the app's initialization
	myCustomApp.Init = func(settings django.Settings, db *sql.DB) error {
	
    }

    // Will be called after all apps have been initialized
	myCustomApp.Ready = func() error {
        globalInstance = myCustomApp
	}

    // Do any other possible setup, like registering routes, templates or static files
    // ...

    return myCustomApp
}
```

## AppConfig useful methods & attributes

### Methods

The `AppConfig` struct has a few useful methods that can be used to configure your app.

This includes but is not limited to:

 * Registering routes
 * Registering templates
 * Registering commands
 * Registering middleware
 * Registering context processors

Every method below should be called inside the `NewCustomAppConfig` function, unless otherwise specified.

```go
func NewCustomAppConfig() *CustomApp {
    // ...

    // Register routes

    // Register templates

    // Register commands

    // Register middleware

    // Register context processors
}
```

#### `AddCommand(c ...command.Command)`

Register one or more commands for this AppConfig.

Please see the [Commands](./commands.md#registering-a-command) documentation for more information.

### Attributes

#### `Routing`

URLs can be registered with the app by setting the `Routing` attribute on your AppConfig inside the `Init` function, or `NewAppConfig` function.

Let's expand on the `NewCustomAppConfig` function from above.

The `AppConfig.Routing` function is used to register routes for the app.

This function is called with a `django.Mux` object, which is used to register routes.

Routes and middleware further explained in the [routing](./routing.md#URLs) documentation.

```go
// Register routes
myCustomApp.Routing = func(m django.Mux) {
    m.Handle(mux.GET, "/", Index, "index"),
    m.Handle(mux.GET, "/about", About, "about"),
}
```

#### `TemplateConfig`

A `*tpl.Config` object that can will be used to register templates for this app.

This skips the need to call `tpl.Add` manually.

#### `CtxProcessors`

A list of context processors that will be run before rendering a template.

This can be used to add context extra context to the template, if [Context Processors](./rendering.md#context-processors) are used while rendering.

#### `Deps`

A list of app names that this app depends on.

This is used to ensure that the dependencies are initialized before this app.

When these dependencies are not found, the app will not be initialized.