# Rendering

## Introduction

To render templates there needs to be some setup done.

First, you will have to create your GO-HTML (`html/template`) templates and store them in a directory. 

This directory has to be "registered" with our template storage engine.

There is also a way provided to easily match template paths, and overrides.

## Initial Setup

Let's now create a directory for the templates.
The directory can be named anything you want, but for this example, we will name it `templates`.

Our app name will be `mycustomapp`, just like the regular Django we like to namespace our apps and templates.

Initial setup is best done in your `AppConfig.Init()` function.

### Creating templates

```bash
mkdir templates
mkdir templates/mycustomapp
```

Let's create a simple base template that will be used to render other templates.

```html
<!-- templates/mycustomapp/base.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{index . "Title"}}</title>
</head>
<body>
    {{template "content" .}}
</body>
</html>
```

Let's also create an index and about page.

```html
<!-- templates/mycustomapp/index.html -->
{{define "content"}}
    <h1>Welcome to the Index Page</h1>
{{end}}
```

```html
<!-- templates/mycustomapp/about.html -->
{{define "content"}}
    <h1>Welcome to the About Page</h1>
{{end}}
```

### Registering the templates

Now that we have created the templates, we need to register them with the template storage engine.

This can be done by passing a configuration object to the `Register` function.

```go
//go:embed templates/*
var tplFSFull embed.FS
var tplFS, _ = fs.Sub(tplFSFull, "templates")

tpl.Add(tpl.Config{
	AppName: "mycustomapp",
	FS:      tplFS,
	Bases: []string{
		"mycustomapp/base.tmpl",
	},
	Matches: filesystem.MatchAnd(
		filesystem.MatchPrefix("mycustomapp/"),
		filesystem.MatchExt(".tmpl"),
	),
    Funcs: template.FuncMap{
        "title": strings.Title,
    },
})
```

In the previous example, we have registered the `mycustomapp` templates with the `tplFS` filesystem.

Only files that match the `mycustomapp/` prefix and have the `.tmpl` extension will be available for rendering (see [MatchFS](./filesystem.md#MatchFS)).

We have also added a custom function `title` that will be available in the templates.

### Context Processors

Context processors are functions that are called before rendering a template.

These will only run (but run every time) if the context inherits from the `ctx.Context` interface and has a method called "`Request()`" which returns a `*http.Request`.

```go
tpl.Processors(
    func(ctx tpl.RequestContext) {
        ctx.Set("Title", "My Custom App")
    },
    func(ctx tpl.RequestContext) {
        var user = ctx.Get("User")
        if user == nil {
            ctx.Set("User", &UnauthenticatedUser{})
        }
    },
)
```

### Template Functions

Template functions are functions that can be called from within the templates.

These functions are registered globally, and are available in all templates.

```go
tpl.Funcs(template.FuncMap{
    "lower": strings.ToLower,
})
```

## Rendering Templates

Rendering templates is as simple as calling the `Render`, or `FRender` function.

Much like standart `fmt.Print` and `fmt.Fprintf` functions, `FRender` will be able to render to an `io.Writer` - returning an error if anything goes wrong, while `Render` will return the `template.HTML` string and an error.

Let's say we're rendering the index page from an HTTP handler.

```go
func IndexHandler(w http.ResponseWriter, r *http.Request) {
    // Create a request context:
    var ctx = core.Context(r)
    ctx.Set("Title", "My Custom App")

    // Render the template:
    // Also adressing the app name will allow for simpler overrides, but is not required.
    if err := tpl.FRender(w, ctx, "mycustomapp", "mycustomapp/index.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

    // Rendering the template with the full path is also possible.
    // if err := tpl.FRender(w, ctx, "mycustomapp/index.tmpl"); err != nil {
    //     http.Error(w, err.Error(), http.StatusInternalServerError)
    // }
}
```

For the about page, we will use the Render function, and a regular ctx.Context object.

```go
func AboutHandler(w http.ResponseWriter, r *http.Request) {
    // Create a context:
    var m = make(map[string]interface{})
    var ctx = ctx.NewContext(m)
    ctx.Set("Title", "My Custom App")

    // Render the template:
    if html, err := tpl.Render(ctx, "mycustomapp/about.tmpl"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    } else {
        w.Write(html)
    }
}
```