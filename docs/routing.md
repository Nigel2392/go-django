# Routing

Routing is the process of mapping URLs to views.
This is done by defining URL patterns in the `Routing` attribute of your `AppConfig`.

The `Routing` attribute is a function that takes a `django.Mux` object as an argument.

The `django.Mux` object is used to register routes, the interface is defined as follows:

```go
type Mux interface {
	Use(middleware ...mux.Middleware)
	Handle(method string, path string, handler mux.Handler, name ...string) *mux.Route
	AddRoute(route *mux.Route)

	Any(path string, handler mux.Handler, name ...string) *mux.Route
	Get(path string, handler mux.Handler, name ...string) *mux.Route
	Post(path string, handler mux.Handler, name ...string) *mux.Route
	Put(path string, handler mux.Handler, name ...string) *mux.Route
	Patch(path string, handler mux.Handler, name ...string) *mux.Route
	Delete(path string, handler mux.Handler, name ...string) *mux.Route
}
```

Any `*mux.Route` returned from this interface will also adhere to the `django.Mux` interface.

## URLs

Routes are registered using the `Handle` method on the `django.Mux` object.

There are also convenience methods for the most common HTTP methods:

- `Any`  
    A route that matches any HTTP method.
- `Get`  
    Only matches `GET` requests.
- `Post`  
    Only matches `POST` requests.
- `Put`  
    Only matches `PUT` requests.
- `Patch`  
    Only matches `PATCH` requests.
- `Delete`  
    Only matches `DELETE` requests.

### Example

```go
myCustomApp.Routing = func(m django.Mux) {
    m.Handle(mux.GET, "/", mux.NewHandler(Index), "index"),
    m.Handle(mux.GET, "/about", mux.NewHandler(About), "about"),
}
```

### Variables

Variables can be used in the URL pattern to capture parts of the URL.

```go
m.Handle(mux.GET, "/user/<<id>>", mux.NewHandler(User), "user"),
```

In the above example, the `User` handler will be called for any URL that matches `/user/<<id>>`.

The `<<id>>` part of the URL will be available as a named parameter in the request's path variable context.

#### The Variables type

This can be retrieved using the `mux.Vars` function on the request.

The `mux.Vars` function takes the request and returns the following type to map the named parameters to their values:

```go
type Variables map[string][]string
func (v Variables) Get(key string) string {...}
func (v Variables) GetAll(key string) []string {...}
func (v Variables) GetInt(key string) int {...}
```

#### Example usage

This can then be used to get the value of the named parameter.

```go
func User(r *http.Request, w http.ResponseWriter) {
    var pathVars = mux.Vars(r)
    var id = pathVars.Get("id")
}
```

### Wildcards

Wildcards can be used in the URL pattern to capture parts of the URL.

```go
m.Handle(mux.GET, "/static/*", mux.NewHandler(Static), "static"),
```

In the above example, the `Static` handler will be called for any URL that starts with `/static/`.

Each wildcard part of the URL will be available as a named parameter in the request's path variable context.

This can be retrieved using the `mux.Vars` function on the request.

```go
func Static(r *http.Request, w http.ResponseWriter) {
    var pathVars = mux.Vars(r)
    var pathParts = pathVars.GetAll("*")
}
```

## Middleware

Middleware can be registered using the `Use` method on the `django.Mux` object.

It works the same way as the `http.Handler` middleware, but with a few differences.

### Example

```go
myCustomApp.Routing = func(m django.Mux) {
	m.Use(func(next mux.Handler) mux.Handler {
		return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

			// Do stuff before the handler

			next.ServeHTTP(w, req)

            // Do stuff after the handler
		})
	})
}
```

The `Use` method takes a list of middleware functions that return a `mux.Handler`.

The `mux.Handler` interface is defined as follows:

```go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

The `ServeHTTP` method is called with the `http.ResponseWriter` and `*http.Request` objects.

The `ServeHTTP` method should call the next middleware in the chain, or the final handler if there are no more middleware functions.

### Disabling Middleware

Middleware can also be disabled for individual routes.

```go
route := m.Handle(mux.GET, "/", mux.NewHandler(Index), "index")
route.RunsMiddleware(false)
```
