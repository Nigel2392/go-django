# DeleteView Example

A `DeleteView` typically presents a confirmation page on `GET` and deletes the object on `POST`. By embedding `views.DetailView`, you seamlessly inherit its context generation.

## Context Parameters

Since this embeds `DetailView`, the following variables are provided:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `<ContextName>`: The object to be deleted (default is `"object"`, configured as `"todo"` here).

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/examples/todoapp/todos"
    "github.com/Nigel2392/go-django/src/core/except"
)

type MyDeleteView struct {
    views.DetailView[todos.Todo]
}

func (v *MyDeleteView) ServePOST(w http.ResponseWriter, req *http.Request) {
    urlArg, err := v.GetURLArg(req)
    if err != nil {
        except.Fail(http.StatusBadRequest, err)
        return
    }

    _, err = v.GetObject(req, urlArg)
    if err != nil {
        except.Fail(http.StatusNotFound, "Object not found")
        return
    }

    // Pseudo-code to delete the object from db
    // db.Delete(&obj)

    http.Redirect(w, req, "/todos", http.StatusSeeOther)
}

var TodoDeleteView = &MyDeleteView{
    DetailView: views.DetailView[todos.Todo]{
        BaseView: views.BaseView{
            AllowedMethods:  []string{http.MethodGet, http.MethodPost},
            TemplateName:    "todos/delete_confirm.html",
            BaseTemplateKey: "base",
        },
        ContextName: "todo",
        URLArgName:  "id",
        GetObjectFn: func(req *http.Request, urlArg string) (todos.Todo, error) {
            return todos.Todo{}, nil
        },
    },
}
```

## Minimal Vanilla Template (`todos/delete_confirm.html`)

```html
<div>
    {{ $todo := .Get "todo" }}
    
    <h1>Confirm Deletion</h1>
    <p>Are you sure you want to delete the task "<strong>{{ $todo.Title }}</strong>"?</p>
    
    <form method="POST">
        <!-- Add CSRF token here if middleware is enabled -->
        <button type="submit" style="color: red;">Yes, delete it</button>
        <a href="/todos">Cancel</a>
    </form>
</div>
```
