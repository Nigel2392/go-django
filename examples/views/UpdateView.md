# UpdateView Example

An `UpdateView` pre-populates a form with existing data to modify a database record. We combine `views.FormView` and `GetInitialFn` to load data.

## Context Parameters

The `FormView` automatically provides:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `form`: The populated form object (containing database data on `GET`, and request payload data on `POST` with potential validation errors).

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/src/forms"
    "github.com/Nigel2392/mux"
)

type TodoUpdateForm struct {
    forms.BaseForm
    Title string `form:"title" validate:"required"`
}

var TodoUpdateView = &views.FormView[*TodoUpdateForm]{
    BaseView: views.BaseView{
        AllowedMethods:  []string{http.MethodGet, http.MethodPost},
        TemplateName:    "todos/update.html",
        BaseTemplateKey: "base",
    },
    GetFormFn: func(req *http.Request) *TodoUpdateForm {
        var form = &TodoUpdateForm{}
        form.WithContext(req.Context())
        return form
    },
    GetInitialFn: func(req *http.Request) map[string]interface{} {
        // Fetch the existing object and populate the form
        return map[string]interface{}{
            "title": "Existing DB Title",
        }
    },
    ValidFn: func(req *http.Request, form *TodoUpdateForm) error {
        // Perform database UPDATE operation
        return nil
    },
    SuccessFn: func(w http.ResponseWriter, req *http.Request, form *TodoUpdateForm) {
        http.Redirect(w, req, "/todos", http.StatusSeeOther)
    },
}
```

## Minimal Vanilla Template (`todos/update.html`)

```html
<div>
    {{ $form := .Get "form" }}
    
    <h1>Update Task</h1>

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
        
        <button type="submit">Update Details</button>
        <a href="/todos">Cancel</a>
    </form>
</div>
```
