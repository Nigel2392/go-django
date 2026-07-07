# Views

Views are the core of handling HTTP requests in `go-django`. A view takes a Web request and returns a Web response. In this framework, views are primarily struct-based and use composition and interfaces to define their behavior.

## The View Interface

At its core, a view must implement the `View` interface:

```go
type View interface {
    // ServeXXX is a method that will never get called directly by the interface.
    // It is a placeholder for the actual method that will be called based on the HTTP method.
    // For example, if the request is a GET request, then ServeGET will be called.
    ServeXXX(w http.ResponseWriter, req *http.Request)
}
```

When you define a view struct, you implement methods like `ServeGET`, `ServePOST`, etc. The framework uses reflection during initialization to map these methods to the appropriate HTTP verbs.

```go
type IndexView struct{}

func (v *IndexView) ServeGET(w http.ResponseWriter, req *http.Request) {
    w.Write([]byte("Hello, World!"))
}
```

## Context Handling

If your view needs access to a context, it can accept a `ctx.Context` in the serve method. The signature then becomes `ContextHandlerFunc`:

```go
func (v *IndexView) ServeGET(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
    context.Set("Message", "Hello with context!")
    return nil
}
```

To initialize the context, your view can implement the `ContextGetter` interface:

```go
type ContextGetter interface {
    View
    GetContext(req *http.Request) (ctx.Context, error)
}
```

If not implemented, a default context (`ctx.RequestContext(req)`) is created for you.

## Template Views

Views can automatically render templates by implementing specific template interfaces:

- **`TemplateGetter`**: Returns the template path to render.
- **`TemplateKeyer`**: Returns the base key (base template) for rendering with inheritance.
- **`Renderer`**: Allows rendering directly to the response writer with a context.

```go
type MyTemplateView struct {}

func (v *MyTemplateView) GetTemplate(req *http.Request) string {
    return "mycustomapp/index.tmpl"
}

func (v *MyTemplateView) ServeGET(w http.ResponseWriter, req *http.Request, ctx ctx.Context) error {
    ctx.Set("Title", "Home Page")
    // The framework will automatically render the template returned by GetTemplate
    // after ServeGET returns successfully.
    return nil
}
```

## View Validation and Checking

If you need to validate a request (e.g. checking permissions) before the main handler runs, implement the `Checker` interface:

```go
type Checker interface {
    View
    Check(w http.ResponseWriter, req *http.Request) error
    Fail(w http.ResponseWriter, req *http.Request, err error)
}
```

`Check` is called before the main view logic. If it returns an error, `Fail` is called to handle the rejection (like redirecting or showing a 403 Forbidden page).

## Error Handling

A view can provide its own error handling logic by implementing `ErrorHandler`:

```go
type ErrorHandler interface {
    HandleError(w http.ResponseWriter, req *http.Request, err error, code int)
}
```

If your view returns an error (from `ServeGET` or a template rendering error), this method will be invoked to handle the HTTP response.

## Serving a View

To use a view in your routing configuration, wrap it with `views.Serve`:

```go
import "github.com/Nigel2392/go-django/src/views"

myCustomApp.Routing = func(m mux.Multiplexer) {
    m.Handle(mux.GET, "/", views.Serve(&IndexView{}), "index")
}
```

The `Serve` function wraps your struct into an `http.Handler`, inspecting all implemented interfaces and methods to set up the correct execution pipeline.

## Advanced View Examples

In many apps, you will use pre-built generic views from `go-django` instead of building everything from scratch. A great example of this is `list.View`, which handles paginating and rendering a list of model instances.

### Creating a Generic List View

Here is a practical example of a generic `list.View` that handles fetching a queryset, parsing pagination parameters, defining table columns, and adding context variables, derived from [docker-mailserver-mailman](https://github.com/Nigel2392/docker-mailserver-mailman):

```go
import (
    "net/http"
    "github.com/Nigel2392/go-django/queries/src"
    "github.com/Nigel2392/go-django/src/core/ctx"
    "github.com/Nigel2392/go-django/src/views/list"
)

var ViewEmails = &list.View[*UserMailProfileProxy]{
    AllowedMethods:  []string{http.MethodGet},
    BaseTemplateKey: "main",
    TemplateName:    "mailmgmt/emails/emails.tmpl",
    PageParam:       "page",
    AmountParam:     "limit",
    MaxAmount:       50,
    DefaultAmount:   10,

    // Define the queryset used to fetch items
    QuerySet: func(r *http.Request) *queries.QuerySet[*UserMailProfileProxy] {
        return queries.
            GetQuerySetWithContext(r.Context(), &UserMailProfileProxy{}).
            OrderBy("User.Email")
    },

    // Inject extra data into the template context
    GetContextFn: func(r *http.Request, qs *queries.QuerySet[*UserMailProfileProxy]) (ctx.Context, error) {
        c := ctx.RequestContext(r)
        c.Set("view.query", r.URL.Query().Get("search"))
        return c, nil
    },

    // Map your struct fields to table columns
    ListColumns: []list.ListColumn[*UserMailProfileProxy]{
        list.Column[*UserMailProfileProxy]("Email", "Email"),
        list.BooleanFieldColumn[*UserMailProfileProxy]("IsActive", "IsActive"),
    },
}
```

### Routing with Generic Views

When you route an advanced view like `ViewEmails`, you still pass it through `views.Serve()`. In this example, we create a URL group `/emails` and assign our view to it:

```go
import (
    "github.com/Nigel2392/mux"
    "github.com/Nigel2392/go-django/src/views"
)

func (c *MailManagementConfig) Routing(m mux.Multiplexer) {
    // Group routes under /mailmgmt
    var group = m.Any("", mux.NewHandler(c.ViewIndex), "mailmgmt")

    // Define the /emails endpoint and map it to ViewEmails
    emails := group.Get("/emails", views.Serve(ViewEmails), "emails")

    // You can also chain additional routes off the group
    emails.Get("/delete", views.Serve(ViewDeleteEmail), "delete")
    emails.Post("/delete", views.Serve(ViewDeleteEmail))
}
```
