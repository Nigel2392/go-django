# Context Package

**Note**: This documentation page was (in part) generated by ChatGPT and has been fully reviewed by [Nigel2392](github.com/Nigel2392).

## Overview

The `ctx` package provides a flexible and simple way to manage context data in your Go applications. It allows you to store and retrieve key-value pairs, making it easier to pass data around in your application, particularly when rendering templates. The package supports both generic context storage and structured context with fields corresponding to struct members.

## Example Usage

Here’s how you might use the `context` package in a typical scenario where you're rendering templates in a web application:

```go
package main

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/core"
    "github.com/Nigel2392/go-django/src/tpl"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
    // Create a request context
    var ctx = ctx.Context(r)
    ctx.Set("Title", "Welcome to My Custom App")

    // Render the index template
    if err := tpl.FRender(w, ctx, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
    // Create a generic context
    var ctx = ctx.NewContext(nil)
    ctx.Set("Title", "About My Custom App")

    // Render the about template
    if html, err := tpl.Render(ctx, "mycustomapp/about.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    } else {
        w.Write(html)
    }
}

func main() {
    http.HandleFunc("/", IndexHandler)
    http.HandleFunc("/about", AboutHandler)
    http.ListenAndServe(":8080", nil)
}
```

## Key Components

### `Context` Interface

The `Context` interface is at the core of the package. It allows setting and getting values in a key-value store.

```go
package ctx

type Context interface {
    Setter
    Getter
}

type Setter interface {
    Set(key string, value any)
}

type Getter interface {
    Get(key string) any
}
```

- **Set**: Stores a value with a specified key.
- **Get**: Retrieves the value associated with a key.

### `ContextWithRequest` Interface

The `ContextWithRequest` interface extends the `Context` interface and adds a method to retrieve the original HTTP request.

```go
package ctx

type ContextWithRequest interface {
    Context
    Request() *http.Request
}
```

- **Request**: Returns the original HTTP request associated with the context.

### `NewContext` Function

`NewContext` creates a new context that stores data in a map. It returns an implementation of the `Context` interface.

```go
func NewContext(m map[string]any) Context {
    if m == nil {
        m = make(map[string]any)
    }
    return &context{data: m}
}
```

### `StructContext`

`StructContext` is a more advanced context that wraps a struct, allowing you to set and get values that correspond to the fields of the struct.

```go
type StructContext struct {
    obj           interface{}
    data          map[string]any
    fields        map[string]reflect.Value
    DeniesContext bool
}

func NewStructContext(obj interface{}) Context {
    // Implementation details...
}
```

- **Set**: Assigns a value to a field in the struct or stores it in a map if the field doesn't exist.
- **Get**: Retrieves a value from a struct field or the map.

### `EditContext` Interface

The `EditContext` interface is designed to allow values to modify the context itself during the `Set` operation. If a value implements this interface, its `EditContext` method is called instead of directly storing the value in the context.

```go
package ctx

type Editor interface {
    EditContext(key string, context Context)
}
```

- **EditContext**: This method is called when a value is set in the context. The method can modify the context based on the key and the value being set.

**Usage Example:**

Suppose you have a struct that needs to perform additional operations when added to the context. You can implement the `EditContext` interface to achieve this:

```go
type User struct {
    Name string
}

func (u *User) EditContext(key string, context ctx.Context) {
    context.Set("UserName", u.Name)
}

// In your handler
func IndexHandler(w http.ResponseWriter, r *http.Request) {
    var ctx = ctx.Context(r)
    var user = &User{Name: "John Doe"}
    
    // This will trigger the EditContext method on the User struct
    ctx.Set("User", user)

    // Now, ctx will have an additional "UserName" key set by EditContext
    if err := tpl.FRender(w, ctx, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

In this example, when you set the `User` object into the context, the `EditContext` method is called, which then sets an additional `UserName` key in the context.

The `User` context variable is not set if the `EditContext` method exists on the value being set.

This has to be done explicitly by the `EditContext` method itself.

Example:

```go
    type User struct {
        Name string
    }

    func (u *User) EditContext(key string, context ctx.Context) {
        context.Set(key, u)
        context.Set("UserName", u.Name)
    }
```

### `RequestContext`

`RequestContext` is a specific implementation used for HTTP requests. It embeds a `StructContext` and adds specific methods for handling requests.

```go
package ctx

type RequestContext struct {
    *ctx.StructContext
    HttpRequest *http.Request
    CsrfToken   string
}

func Context(r *http.Request) *RequestContext {
    // Implementation details...
}

func (c *RequestContext) Request() *http.Request {
    return c.HttpRequest
}

func (c *RequestContext) CSRFToken() string {
    return c.CsrfToken
}
```

### Rendering Templates

Rendering templates in a Go application involves setting up a context (which can include request data, user info, etc.), and then passing that context to the template engine to produce the final HTML.

#### Example Template

Let's say you have the following HTML templates:

```html
<!-- templates/mycustomapp/base.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Get "Title"}}</title>
</head>
<body>
    {{template "content" .}}
</body>
</html>
```

In this template, `{{.Get "Title"}}` retrieves the value associated with the key "Title" from the context.
