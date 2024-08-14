# Configure your web server

The application state / configuration is stored in a global struct, this is to allow for a singleton pattern where the application cannot be initialized more than once.

Initializing multiple application structs is out of scope.

## Configuration

Configuration of the server can be done in multiple ways.

### The `Settings` interface

There is an interface defined for configuring the `*django.Application`.

```go
type Settings interface {
	Set(key string, value interface{})
	Get(key string) (any, bool)
	Bind(app *Application) error
	App() *Application
}
```

We have provided a simple implementation of this interface to be used with a `map[string]interface{}`, instantiated with `django.Configure(map[string]interface{})` when initializing the application.

Exact usage will be shown after we have explained the default settings and initialization pattern.

### Pre-defined settings

All settings' names used by go-django are made available through variables.

This makes it easier to avoid typos and to use the correct setting names.

All settings are prefixed with `APPVAR_`.

More settings may be added by pre-built apps, these **should** be documented in the package's documentation.

```go
// The debug mode of the application
// If true, the application will log debug information with each request
django.APPVAR_DEBUG

// Allowed hosts for the application
// Any request with a host not in this list will be rejected
django.APPVAR_ALLOWED_HOSTS

// Whether to use the recoverer middleware
// If false, the application will panic and not recover.
// This might be useful for debugging.
django.APPVAR_RECOVERER

// The host to bind the server to
// This is the address the server will listen on
django.APPVAR_HOST

// The port to bind the server to
// This is the port the server will listen on
django.APPVAR_PORT

// The path to the TLS certificate file
// This is the certificate file used for TLS
django.APPVAR_TLS_CERT

// The path to the TLS key file
// This is the key file used for TLS
django.APPVAR_TLS_KEY

// The TLS configuration
// This is a tls.Config struct
// Allows for more advanced TLS configuration
django.APPVAR_TLS_CONFIG

// The database object.
// This is a *sql.DB, and allows for easily sharing a database connection
// between multiple packages
django.APPVAR_DATABASE
```

### Retrieving a value from settings

Retrieving a value from within an app from the Django settings is done by adressing the global singleton struct, and can only be done after creating your Django program.

```
var ALLOWED_HOSTS = django.ConfigGet[[]string](
	django.Global.Settings,
	django.APPVAR_ALLOWED_HOSTS,
	[]string{"*"},
)
```

This will retrieve a value from the settings, and if it is not found, it will return the default value provided as the third argument.

If no default value is provided, a type hint of the correct type is required.

When the value does not meet this type, the program will panic.

## Go-Django initialization

Configuration of the app instance can only be done once by calling `django.App` with different option- funcs.

These option funcs must have the following type signature:

```go
func MyCustomOption(a *django.Application) error {
	// ... do something
	return nil
}
```

### Creating the app

Subsequent calls to `django.App` will ignore all options and return the global singleton instance.

To see how to initialize custom apps have a look at the [apps documentation](./apps.md).

Example of a simple app's creation:

```go
// Open a new database connection
var db, err = sql.Open("sqlite3", "./.private/db.sqlite3")
if err != nil {
	panic(err)
}

// Register the database connection with the auditlogs package
auditlogs.RegisterBackend(
	auditlogs_sqlite.NewSQLiteStorageBackend(db),
)

// Create the application struct with all the necessary settings/options
var app = django.App(
	django.Configure(map[string]interface{}{
		django.APPVAR_ALLOWED_HOSTS: []string{"*"},
		django.APPVAR_HOST: "127.0.0.1",
		django.APPVAR_PORT: "8080",
		django.APPVAR_DEBUG: false,
		django.APPVAR_DATABASE: db,
	}),
	django.Apps(
		// Initialize sessions...
		session.NewAppConfig,

		// Initialize custom app...
		NewCustomAppConfig,
	),
)
```

### Initializing the app

The app struct is now created but left uninitialized.

It should be initialized by calling `err := app.Initialize()`.

This will set up a few things, (in order):

 * The logger (By default setup with level `INFO`)

 * Default middleware

 * Static routes

 * Django app middleware

 * Django app routes

 * Custom registered apps initialization (Loop 1)

   * Check installed (required) dependencies for this application

   * Initialized in the order they were registered (Call each apps' `Initialize` function)

 * Setup the command registry

 * Setup custom apps (Loop 2)
   
   * Register the URL's of the application

   * Register the middleware of the application

   * Register the context processors of the application

   * Initialize the apps' template configurations

   * Register the apps' commands

 * Call the `OnReady` function of each registered app (Loop 3)

   * This is the place to do any final setup of the application

 * Setup recoverer middleware if enabled

 * Check if any command was passed to the application and execute accordingly

### Serving the app

When all configuration is done it is time to serve the application.

This is done by calling `err := app.Serve()`.

This will start the server and listen for incoming requests.

We look for the following settings when initializing to serve:

 * "HOST" - The host to bind the server to
 * "PORT" - The port to bind the server to
   * If "PORT" is explicitly set to "" or "0", the server will not listen on the HTTP port.
 * "TLS_PORT" - The port to bind the server to when using TLS
   * This allows for serving both HTTP and HTTPS.
   * It will only be used if "TLS_CERT" and "TLS_KEY" are set.
   * If "TLS_PORT" is not set, is set to "" or "0", the server will not listen with TLS, and only serves HTTP.
 * "TLS_CERT" - The path to the TLS certificate file
 * "TLS_KEY" - The path to the TLS key file
 * "TLS_CONFIG" - The TLS configuration

### Adding routes or middleware to Django without an application

If you want to add routes to the Django struct without having to create a custom app & appconfig, it is possible to directly interface with the multiplexer.

This can be done by adressing the `app.Mux` field.

It will likely be better to create your own app for code organization and maintainability, but it remains a possibility.