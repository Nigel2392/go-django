# Sessions

The sessions app provides a single middleware, with the abstraction based on [alexedwards/scs](github.com/alexedwards/scs/v2).

## Session Interface

The main interface which is used to interact with the user's session is the `session.Session` interface.

This provides a simple, but powerful and customizable way to interact with the user's session, and thus allows you to implement your own session app if required.

```go
type Session interface {
    Set(key string, value interface{})
    Get(key string) interface{}
    Exists(key string) bool
    Delete(key string)
    Destroy() error
    RenewToken() error
}
```

To see what can be serialized (in the default implementation), please refer to the [alexedwards/scs](github.com/alexedwards/scs/v2) documentation.

## Requirements

The sessions app requires the [`DATABASE`/django.APPVAR_DATABASE](../configuring.md#pre-defined-settings) setting to be set, it allows for a SQLite3, MySQL, or PostgreSQL database.

## Retrieving the session

The sessions package provides a single exported function (and a middleware) to retrieve the session from the request.

```go
func Retrieve(r *http.Request) sessions.Session {
    ...
}
```

## Example usage

Let's expand on the examples from the [rendering](../rendering.md#rendering-templates) documentation.

```go
func Index(w http.ResponseWriter, r *http.Request) {
    var session = sessions.Retrieve(r)
    fmt.Println(session.Get("page_key"))
    session.Set("page_key", "Last visited the index page")
    
    if err := tpl.FRender(w, ctx, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

In the above example, we retrieve the session from the request, and then set a key-value pair in the session.

If the key already exists, it will be overwritten.

On the initial visit, the console will print `nil`, and on subsequent visits, it will print `Last visited the index page`.

```go
func About(w http.ResponseWriter, r *http.Request) {
    var session = sessions.Retrieve(r)
    
    fmt.Println(session.Get("page_key"))
    session.Set("page_key", "Last visited the about page")
    
    if html, err := tpl.Render(ctx, "mycustomapp/about.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    } else {
        w.Write(html)
    }
}
```

In the above example, we do the same as in the previous example, but we render the template directly to the response writer.

If the key already exists, it will be overwritten.

If we visit the about page after the index page, the console will print `Last visited the index page`, and on subsequent visits, it will print `Last visited the about page`.
