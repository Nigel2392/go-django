# Forms Application Example

The `formsapp` example (`examples/formsapp`) demonstrates how to define, bind, validate, and render HTML forms in Go-Django using `src/forms`, `src/forms/fields`, `src/forms/widgets`, and `html/template` with CSRF protection.

## Table of Contents

- [Project Structure](#project-structure)
- [Running the Example](#running-the-example)
- [App Configuration and Routes (`src/formsapp/core.go`)](#app-configuration-and-routes-srcformsappcorego)
- [Form Definition (`src/formsapp/forms.go`)](#form-definition-srcformsappformsgo)
- [Views (`src/formsapp/views.go`)](#views-srcformsappviewsgo)
- [Contact Template (`contact.tmpl`)](#contact-template-contacttmpl)
- [Notes and Caveats](#notes-and-caveats)

---

## Project Structure

```text
examples/formsapp/
├── go.mod
└── src/
    ├── main.go
    └── formsapp/
        ├── core.go
        ├── forms.go
        ├── views.go
        └── assets/
            ├── static/
            │   └── formsapp/
            │       └── css/
            │           └── formsapp.css
            └── templates/
                └── formsapp/
                    ├── base.tmpl
                    ├── contact.tmpl
                    └── index.tmpl
```

---

## Running the Example

The entrypoint lives at `src/main.go`. Run it from inside `examples/formsapp`:

```bash
cd examples/formsapp
go run ./src
```

Or run it from the repository root:

```bash
go run ./examples/formsapp/src
```

Then visit:

- `http://127.0.0.1:8080/` — home page linking to the contact page.
- `http://127.0.0.1:8080/contact` — contact form (`GET` displays the form; `POST` validates input and redirects to `/` on success).

---

## App Configuration and Routes (`src/formsapp/core.go`)

`NewAppConfig()` registers an app named `"core"` with routes `/` (`"index"`) and `/contact` (`"contact"` for both `GET` and `POST`), plus embedded static CSS (`formsapp/`) and templates (`AppName: "core"`, base `"formsapp/base.tmpl"`):

```go
package formsapp

import (
	"embed"
	"io/fs"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2"
)

//go:embed assets/**
var assetFilesystem embed.FS

func NewAppConfig() django.AppConfig {
	cfg := apps.NewAppConfig("core")

	cfg.Routing = func(m mux.Multiplexer) {
		m.Handle(mux.GET, "/", http.HandlerFunc(Index), "index")
		m.Handle(mux.GET, "/contact", http.HandlerFunc(Contact), "contact")
		m.Handle(mux.POST, "/contact", http.HandlerFunc(Contact), "contact")
	}

	cfg.Init = func(settings django.Settings) error {
		tplFS, err := fs.Sub(assetFilesystem, "assets/templates")
		if err != nil {
			return err
		}

		staticFS, err := fs.Sub(assetFilesystem, "assets/static")
		if err != nil {
			return err
		}

		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("formsapp/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
			),
		))

		tpl.Add(tpl.Config{
			AppName: "core",
			FS:      tplFS,
			Bases: []string{
				"formsapp/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("formsapp/"),
				filesystem.MatchExt(".tmpl"),
			),
		})

		return nil
	}

	return cfg
}
```

See [Routing](../routing.md), [Static Files](../staticfiles.md), and [Rendering](../rendering.md).

---

## Form Definition (`src/formsapp/forms.go`)

`NewContactForm` builds four fields (`name`, `email`, `subject`, `message`), creates a `*forms.BaseForm` bound to `POST` request data via `forms.WithRequestData`, and attaches the fields using `forms.Initialize` and `forms.WithFields`:

```go
package formsapp

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func NewContactForm(request *http.Request) forms.Form {
	formFields := []fields.Field{
		fields.CharField(
			fields.Name("name"),
			fields.Label("Name"),
			fields.Required(true),
			fields.MinLength(2),
			fields.MaxLength(100),
			fields.Placeholder("Enter your full name"),
			fields.HelpText("Please enter your full name."),
			fields.Default("John Doe"),
		),
		fields.EmailField(
			fields.Name("email"),
			fields.Label("Email"),
			fields.Required(true),
			fields.Placeholder("Enter your email address"),
			fields.HelpText("Please enter a valid email address."),
			fields.MinLength(5),
			fields.MaxLength(255),
			fields.Default("example@example.com"),
		),
		fields.CharField(
			fields.Name("subject"),
			fields.Label("Subject"),
			fields.Required(true),
			fields.MinLength(2),
			fields.MaxLength(100),
			fields.Placeholder("Enter the subject of your message"),
			fields.HelpText("Please enter the subject of your message."),
			fields.Attributes(map[string]string{"autocomplete": "off"}),
			fields.Default("Hello"),
		),
		fields.CharField(
			fields.Name("message"),
			fields.Label("Message"),
			fields.Required(true),
			fields.MinLength(10),
			fields.MaxLength(5000),
			fields.Placeholder("Enter your message"),
			fields.HelpText("Please enter your message."),
			fields.Attributes(map[string]string{"autocomplete": "off"}),
			fields.Default("I would like to get in touch with you."),
			fields.Widget(widgets.NewTextarea(nil)),
		),
	}

	form := forms.NewBaseForm(
		request.Context(),
		forms.WithRequestData(http.MethodPost, request),
	)

	return forms.Initialize(
		form,
		forms.WithFields(formFields...),
	)
}
```

See [Forms](../forms/readme.md), [Fields](../forms/fields.md), and [Widgets](../forms/widgets.md).

---

## Views (`src/formsapp/views.go`)

`Index` renders `formsapp/index.tmpl` with the reversed `contact` route URL. `Contact` validates submitted `POST` requests with `forms.IsValid(r.Context(), form)`, prints `form.CleanedData()` and redirects to `django.Reverse("index")` on success, or logs `form.ErrorList()` / `form.BoundErrors()` and re-renders `formsapp/contact.tmpl` on validation failure:

```go
package formsapp

import (
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms"
)

func Index(w http.ResponseWriter, r *http.Request) {
	context := ctx.RequestContext(r)
	context.Set("Title", "Home")
	context.Set("ProjectURL", "https://github.com/nigel2392/go-django")
	context.Set("ContactURL", django.Reverse("contact"))
	context.Set("ProjectName", "GO-Django")

	if err := tpl.FRender(w, context, "core", "formsapp/index.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Contact(w http.ResponseWriter, r *http.Request) {
	context := ctx.RequestContext(r)
	context.Set("Title", "Contact")
	context.Set("ProjectURL", "https://github.com/nigel2392/go-django")
	context.Set("ContactURL", django.Reverse("contact"))
	context.Set("ProjectName", "GO-Django")

	form := NewContactForm(r)
	if r.Method == http.MethodPost {
		if forms.IsValid(r.Context(), form) {
			data := form.CleanedData()
			fmt.Println("Form is valid!")
			fmt.Printf("Name (%T): %s\n", data["name"], data["name"])
			fmt.Printf("Email (%T): %s\n", data["email"], data["email"])
			fmt.Printf("Subject (%T): %s\n", data["subject"], data["subject"])
			fmt.Printf("Message (%T): %s\n", data["message"], data["message"])

			http.Redirect(w, r, django.Reverse("index"), http.StatusSeeOther)
			return
		}

		for _, err := range form.ErrorList() {
			fmt.Printf("Error: %s\n", err)
		}
		if errs := form.BoundErrors(); errs != nil {
			for head := errs.Front(); head != nil; head = head.Next() {
				fmt.Printf("Error [%s]: %s\n", head.Key, head.Value)
			}
		}
	}

	context.Set("Form", form)
	if err := tpl.FRender(w, context, "core", "formsapp/contact.tmpl"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

---

## Contact Template (`contact.tmpl`)

`formsapp/assets/templates/formsapp/contact.tmpl` renders the hidden CSRF token via `{{ csrf_field }}`, form-level errors from `ErrorList`, and each bound field (`$Field.Label`, `$Field.Errors`, `$Field.HelpText`, `$Field.Field`):

```html
<form method="POST" action="{{ .Get "ContactURL" }}">
    {{ csrf_field }}

    {{ $errorList := (.Get "Form").ErrorList }}
    {{ if (gt (len $errorList) 0) }}
        <ul class="error-list">
            {{ range $Error := $errorList }}
                <li class="error"><p>{{ $Error }}</p></li>
            {{ end }}
        </ul>
    {{ end }}

    {{ range $Field := (.Get "Form").Fields }}
        <div class="form-field">
            {{ $Field.Label }}

            {{ if (gt (len $Field.Errors) 0) }}
                <ul class="error-list">
                    {{ range $Error := $Field.Errors }}
                        <li class="error"><p>{{ $Field.FormField.Label }}: {{ $Error }}</p></li>
                    {{ end }}
                </ul>
            {{ end }}

            {{ if (ne $Field.HelpText "") }}
                <p class="help-text">{{ $Field.HelpText }}</p>
            {{ end }}

            {{ $Field.Field }}
        </div>
    {{ end }}

    <input type="submit" value="{{ T "Submit" }}">
</form>
```

---

## Notes and Caveats

- **Template key case in `contact.tmpl`**: `views.go` sets `context.Set("Form", form)` (capital `F`), while `extra_css` and `extra_js` in `contact.tmpl` call `(.Get "form").Media` (lowercase `f`). If you adapt this template for widgets with custom CSS/JS media, align both to `"Form"` (or `"form"`).
- **Run from `./src`**: Because `main.go` sits inside `examples/formsapp/src/`, run `go run ./src` (from `examples/formsapp`) rather than `go run main.go`.
