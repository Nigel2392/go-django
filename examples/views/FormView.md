# FormView Example

A `FormView` strictly handles form display, validation, and processing. It is useful for generic non-model forms like Contact, Feedback, or Search forms.

## Context Parameters

The `FormView` automatically provides:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `form`: The form instance mapped to template generation and validation (e.g., `*ContactForm`).

## Implementation

```go
package myviews

import (
    "fmt"
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/src/forms"
)

type ContactForm struct {
    forms.BaseForm
    Email   string `form:"email" validate:"required,email"`
    Message string `form:"message" validate:"required"`
}

var ContactFormView = &views.FormView[*ContactForm]{
    BaseView: views.BaseView{
        AllowedMethods:  []string{http.MethodGet, http.MethodPost},
        TemplateName:    "contact/form.html",
        BaseTemplateKey: "base",
    },
    GetFormFn: func(req *http.Request) *ContactForm {
        var form = &ContactForm{}
        form.WithContext(req.Context())
        return form
    },
    ValidFn: func(req *http.Request, form *ContactForm) error {
        fmt.Printf("Message from %s: %s\n", form.Email, form.Message)
        return nil
    },
    SuccessFn: func(w http.ResponseWriter, req *http.Request, form *ContactForm) {
        http.Redirect(w, req, "/contact/success", http.StatusSeeOther)
    },
}
```

## Minimal Vanilla Template (`contact/form.html`)

```html
<div>
    {{ $form := .Get "form" }}
    
    <h1>Contact Us</h1>
    
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
        
        <button type="submit">Send Message</button>
    </form>
</div>
```
