# Messages

The messages app provides functionality akin to django.contrib.messages, allowing you to store messages inside of the session, or cookies, and retrieve them later.  
This is particularly useful for displaying flash messages to users after a form submission or other actions.

## Installing the messages app with go-django

The messages app has to be included in your `django.Apps(...)` function.

The app will then setup any required middleware, and add the messages context processor.

Messages will only be available in templates when rendered with `tpl.Render()` or `tpl.FRender()`, and when the context's type adheres to `tpl.RequestContext`.

Example setup of this can be done accordingly:

```go
package main

import (
    "github.com/Nigel2392/mux"
    "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/apps"
    "github.com/Nigel2392/go-django/src/contrib/messages"
    "github.com/Nigel2392/go-django/src/contrib/session"
)

func main() {
    var settings = make(map[string]any)
    settings[django.APPVAR_ALLOWED_HOSTS] = []string{"*"}
    settings[django.APPVAR_DEBUG] = true
   
    // This does not nescessarily have to be setup explicity.
    // If the session app is installed and no backend was explicitly configured; the session backend will be used automatically.
    // If the session app is not installed and no backend was explicitly configured; the cookie backend will be used automatically.
    messages.ConfigureBackend(
        // Explicitly set the messages backend to use cookies
        // messages.NewCookieBackend,
   
        // Explicitly set the messages backend to use sessions
        messages.NewSessionBackend,
    )

    app = django.App(
        django.Configure(settings),
        django.Apps(
            session.NewAppConfig,
            messages.NewAppConfig,
            apps.NewAppConfigForHandler(
                "myapp", 
                mux.GET, "/myapp", 
                mux.NewHandler(myView),
            ),
        ),
    )

    if err := app.Serve(); err != nil {
        panic(err)
    }
}
```

In the above example, the session app is included, and the messages app is configured to use the session backend.

If you want to use the cookie backend, it is not required to also include the session's app.

After setting up the app, you can use the messages app in your views and templates.

### Using the messages app in views

The messages app provides a `messages` package that you can use to add messages to the session or cookies.

You can use the following functions to add messages:

- `messages.AddMessage(request, level, message)` - Adds a message to the session or cookies.
- `messages.Debug(request, message)` - Adds a debug message to the session or cookies.
- `messages.Success(request, message)` - Adds a success message to the session or cookies.
- `messages.Info(request, message)` - Adds an info message to the session or cookies.
- `messages.Warning(request, message)` - Adds a warning message to the session or cookies.
- `messages.Error(request, message)` - Adds an error message to the session or cookies.

Example usage:

```go
package main

import (
    "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/contrib/messages"
    "net/http"
)

func myView(w http.ResponseWriter, r *http.Request) {
    messages.Debug(r, "This is a debug message")
    messages.Info(r, "This is an info message")
    messages.Warning(r, "This is a warning message")
    messages.Error(r, "This is an error message")
    messages.Success(r, "This is a success message")

    var context = ctx.RequestContext(r)
    if err := tpl.FRender(w, context, "my/template.tmpl"); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
}
```

Do note that no templates were setup; you will have to do this yourself according to the [rendering docs](../rendering.md).

### Showing messages in templates

As mentioned before; when using context which adheres to `tpl.RequestContext`, the messages will automatically be available in the template context due to the messages context processor.

After the context processor has been executed, the messages will be available in the template as a slice of `messages.Message` structs and **all messages will be removed from the session or cookies**.

These can then be rendered in the template using the `messages` variable.

```html
<section class="messages">
    {{ $ctx := . }}
    {{range $msg := (.Get "messages")}}
        {{ if eq $msg.Tag "success" }}
            <div class="message bg-success">
                <div class="message-text">{{$msg.Message}}</div>
            </div>
        {{else if eq $msg.Tag "error" }}
            <div class="message bg-danger">
                <div class="message-text">{{$msg.Message}}</div>
            </div>
        {{else if eq $msg.Tag "warning" }}
            <div class="message bg-warning">
                <div class="message-text">{{$msg.Message}}</div>
            </div>
        {{else if eq $msg.Tag "info" }}
            <div class="message bg-info">
                <div class="message-text">{{$msg.Message}}</div>
            </div>
        {{else if eq $msg.Tag "debug" }}
            <div class="message bg-debug">
                <div class="message-text">{{$msg.Message}}</div>
            </div>
        {{end}}
    {{end}}
</section>
```

## The cookie backend

The cookie backend stores the messages in a cookie in the user's browser.

This is useful for small messages, but has a limit of 4kb.

If you are storing large messages, it is recommended to use the [session backend](#the-session-backend) instead.

The cookie backend does allow for defining a custom cookie name, along with other settings like:

- `MaxAge` - The maximum age of the cookie in seconds. Defaults to 7 days.
- `Path` - The path of the cookie. Not set by default.
- `Domain` - The domain of the cookie. Not set by default.
- `Expires` - The expiration date of the cookie. Not set by default.
- `HttpOnly` - Whether the cookie is HTTP only. Not set by default.
- `SameSite` - The SameSite attribute of the cookie. Not set by default.

This can be configured by setting the appropriate setting in the go-django settings.

```go
var settings = make(map[string]any)
settings[django.APPVAR_ALLOWED_HOSTS] = []string{"*"}
settings[django.APPVAR_DEBUG] = true
settings[messages.APPVAR_COOKIE_KEY] = &http.Cookie{
    Name:    "my_cookie_name", // Define a custom cookie name for your messages cookie backend.
    MaxAge:  60 * 60 * 24 * 30, // Set the maximum age of the cookie to 30 days.
}
```

## The session backend

The session backend stores the messages in the session.

This is useful for larger messages, and allows for more flexibility in terms of storage.

It also provides a more secure way of storing messages, as they are not exposed to the user.

The session backend does not have any additional settings.
