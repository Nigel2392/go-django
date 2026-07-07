# DetailView Example

A `DetailView` is designed to display the details of a single object. It abstracts away the common pattern of extracting a URL parameter, fetching the object from the database, and passing it to a template.

## Context Parameters

This view automatically provides the following variables to your template:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `<ContextName>`: The fetched object (default is `"object"`, but customized to `"todo"` in this example).

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/examples/todoapp/todos"
)

var MyDetailView = &views.DetailView[todos.Todo]{
    BaseView: views.BaseView{
        AllowedMethods:  []string{http.MethodGet},
        TemplateName:    "todos/detail.html",
        BaseTemplateKey: "base",
    },
    ContextName: "todo",
    URLArgName:  "id",
    GetObjectFn: func(req *http.Request, urlArg string) (todos.Todo, error) {
        var t todos.Todo
        return t, nil
    },
}
```

## Minimal Vanilla Template (`todos/detail.html`)

```html
<div>
    {{ $todo := .Get "todo" }}
    
    <h1>{{ $todo.Title }}</h1>
    <p>Status: {{ if $todo.Done }}Completed{{ else }}Pending{{ end }}</p>
    <p>{{ $todo.Description }}</p>
    <a href="/todos">Back to List</a>
</div>
```
