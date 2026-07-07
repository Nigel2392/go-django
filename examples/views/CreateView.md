# CreateView Example

A `CreateView` simplifies the process of displaying a form, validating it, and saving a new object to the database. We can build this using `views.FormView`.

## Context Parameters

The `FormView` automatically injects these variables into your template:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `form`: The initialized or populated form object (in this case, your `*TodoForm` struct). Form validation errors are attached to it if the submission fails.

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/src/forms"
)

type TodoForm struct {
    forms.BaseForm
    Title       string `form:"title" validate:"required"`
    Description string `form:"description"`
}

var TodoCreateView = &views.FormView[*TodoForm]{
    BaseView: views.BaseView{
        AllowedMethods:  []string{http.MethodGet, http.MethodPost},
        TemplateName:    "todos/create.html",
        BaseTemplateKey: "base",
    },
    GetFormFn: func(req *http.Request) *TodoForm {
        var form = &TodoForm{}
        form.WithContext(req.Context())
        return form
    },
    ValidFn: func(req *http.Request, form *TodoForm) error {
        // e.g. queries.Insert(db, &todos.Todo{Title: form.Title})
        return nil
    },
    SuccessFn: func(w http.ResponseWriter, req *http.Request, form *TodoForm) {
        http.Redirect(w, req, "/todos", http.StatusSeeOther)
    },
}
```

## Minimal Vanilla Template (`todos/create.html`)

```html
<div>
    {{ $form := .Get "form" }}
    
    <h1>Create New Task</h1>
    
    {{ $formErrors := $form.FormErrors }}
    {{ if gt (len $formErrors) 0 }}
        <div class="error-summary">
            <ul>
                {{ range $err := $formErrors }}<li>{{ $err }}</li>{{ end }}
            </ul>
        </div>
    {{ end }}

    <form method="POST">
        {{ range $Field := $form.Fields }}
            <div class="form-group">
                <div class="form-label">{{ $Field.Label }}</div>
                
                {{ $errors := $Field.Errors }}
                {{ if gt (len $errors) 0 }}
                    <div class="field-errors">
                        <ul>
                            {{ range $err := $errors }}
                                <li class="error" style="color:red;">{{ $err }}</li>
                            {{ end }}
                        </ul>
                    </div>
                {{ end }}

                <div class="form-field">{{ $Field.Field }}</div>
                
                {{ $help := $Field.HelpText }}
                {{ if gt (len $help) 0 }}
                    <div class="form-help">{{ $help }}</div>
                {{ end }}
            </div>
        {{ end }}
        
        <button type="submit">Create</button>
    </form>
</div>
```
