# ErrorView Example

`go-django` allows views to act as custom error handlers by implementing the `views.ErrorHandler` interface.

## Context Parameters

Since this view handles errors dynamically, it injects error-specific data into the base context:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.
- `ErrorCode`: The HTTP status code of the error (e.g. `404`, `500`).
- `ErrorMessage`: The string interpretation of the generic `error` interface.

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/src/core/ctx"
)

type CustomErrorView struct {
    views.BaseView
}

func (v *CustomErrorView) HandleError(w http.ResponseWriter, req *http.Request, err error, code int) {
    c, _ := v.GetContext(req)
    
    c.Set("ErrorCode", code)
    c.Set("ErrorMessage", err.Error())
    
    w.WriteHeader(code)
    _ = v.Render(w, req, v.TemplateName, c)
}

var GlobalErrorView = &CustomErrorView{
    BaseView: views.BaseView{
        TemplateName:    "errors/custom_error.html",
        BaseTemplateKey: "base",
    },
}
```

## Minimal Vanilla Template (`errors/custom_error.html`)

```html
<div class="error-container" style="text-align: center; padding: 50px;">
    {{ $errorCode := .Get "ErrorCode" }}
    {{ $errorMessage := .Get "ErrorMessage" }}
    
    <h1 style="color: red; font-size: 3em;">{{ $errorCode }}</h1>
    <h2>Oops! Something went wrong.</h2>
    <p>{{ $errorMessage }}</p>
    <a href="/">Return to Home</a>
</div>
```
