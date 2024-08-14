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

In [Configuring](./configuring.md) we saw a function called `NewCustomAppConfig` being passed to the Go-Django initializer.

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

#### `Register(p ...core.URL)`

URLs can be registered with the app by calling the `Register` method inside the `Init` function, or NewAppConfig function.

Let's expand on the `NewCustomAppConfig` function from above.

The `AppConfig.Register` method takes a `core.URL` object, which can be created using the `urls.Func`, `urls.Group` or `urls.Pattern` functions.

These are further explained in the [URLs](./routing.md#URLs) documentation.

```go
// Register routes
myCustomApp.Register(
    urls.Func(func(m core.Mux) {
        // Full access to the multiplexer.
        // This is likely either a *mux.Mux or *mux.Route
        // Custom logic can be implemented here (but is not recommended).
    }),
    urls.Pattern(
        // Allows for both POST and GET requests
        // The path for this route is an empty string, thus it will be used as the root route.
        // Multiple root routes can be registered, but only one will be used, depending on the order of registration.
        urls.M("GET", "POST"),
		mux.NewHandler(Index),
	),
	urls.Pattern(
        // This route will only accept GET and POST requests
        // It will be available at the `/about` path.
		urls.P("/about", mux.GET, mux.POST),
		mux.NewHandler(About),
	),
)
```

#### `AddCommand(c ...command.Command)`

Register one or more commands for this AppConfig.

Please see the [Commands](./commands.md) documentation for more information.

#### `AddMiddleware(m ...mux.Middleware)`

Register one or more middleware for this AppConfig.

Please see the [Middleware](./routing.md#Middleware) documentation for more information.

#### `Use(m ...core.Middleware)`

Register one or more middleware for this AppConfig.

Please see the [Middleware](./routing.md#Middleware) documentation for more information.

### Attributes

#### `TemplateConfig`

A `*tpl.Config` object that can will be used to register templates for this app.

This skips the need to call `tpl.Add` manually.

#### `CtxProcessors`

A list of context processors that will be run before rendering a template.

This can be used to add context extra context to the template, if [Context Processors](./rendering.md#context-processors) are used while rendering.

#### `Path`

A string that represents the base URL path for this app.

This will auto-register a route for the app at the given path.

This means that instead of a `*mux.Mux` object, a `*mux.Route` object will be passed to the `urls.Func` (and other) functions.

```go
myCustomApp.Path = "/myapp"
```

#### `Deps`

A list of app names that this app depends on.

This is used to ensure that the dependencies are initialized before this app.

When these dependencies are not found, the app will not be initialized.