# TemplateView Example

A `TemplateView` serves static templates, allowing you to inject custom context variables. Instantiating `views.BaseView` grants template rendering out-of-the-box.

## Context Parameters

By default, the base view context automatically provides:

- `Request`: The `*http.Request` object.
- `View`: The view instance itself.

Any custom variables injected via `GetContextFn` (like `Title` or `Version` below) become directly available in the template.

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views"
    "github.com/Nigel2392/go-django/src/core/ctx"
)

var AboutTemplateView = &views.BaseView{
    AllowedMethods:  []string{http.MethodGet},
    TemplateName:    "pages/about.html",
    BaseTemplateKey: "base",
    GetContextFn: func(req *http.Request) (ctx.Context, error) {
        c := ctx.RequestContext(req)
        
        c.Set("Title", "About Us")
        c.Set("Version", "1.0.0")
        
        return c, nil
    },
}
```

## Minimal Vanilla Template (`pages/about.html`)

```html
<div>
    {{ $title := .Get "Title" }}
    {{ $version := .Get "Version" }}
    {{ $request := .Get "Request" }}
    
    <h1>{{ $title }}</h1>
    <p>Welcome to our application.</p>
    
    <div class="meta-info">
        <p>System Version: <strong>{{ $version }}</strong></p>
        <p>Your User Agent: <code>{{ $request.UserAgent }}</code></p>
    </div>
</div>
```
