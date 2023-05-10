package app

import (
	"database/sql"
	"html/template"

	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/netcache/src/client"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/templates"
)

// Application flags.
//
// These are flags which can be added to the application.
//
// They exist to centralize the flags used by the application,
// and possibly other packages.
func (a *Application) Flags() *flag.Flags {
	return a.flags
}

// Add functions to the template.
func (a *Application) TemplateFuncs(t template.FuncMap) {
	if a.config.DefaultTemplateFuncs == nil {
		a.config.DefaultTemplateFuncs = make(template.FuncMap)
	}
	for k, v := range t {
		a.config.DefaultTemplateFuncs[k] = v
	}
}

// Add a middleware to the router.
func (a *Application) Middlewares(m ...router.Middleware) {
	a.Router.Use(m...)
}

func (a *Application) Database() *sql.DB {
	return a.defaultDatabase
}

func (a *Application) Mailer() *email.Manager {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.Mailer()")
	}
	if a.config.Mail == nil {
		panic("You must initialize the app with a mailer before calling app.Mailer()")
	}
	return a.config.Mail
}

func (a *Application) Filer() fs.Filer {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.FS()")
	}
	if a.config.File == nil {
		panic("You must initialize the app with a file manager before calling app.FS()")
	}
	return a.config.File
}

func (a *Application) Templates() *templates.Manager {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.Templates()")
	}
	if a.config.Templates == nil {
		panic("You must initialize the app with a template manager before calling app.Templates()")
	}
	return a.config.Templates
}

func (a *Application) Cache() client.Cache {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.Cache()")
	}
	if a.cache == nil {
		panic("You must initialize the app with a cache before calling app.Cache()")
	}
	return a.cache
}

// Becaus there can only be one __app, we can use some shorthands.

func Flags() *flag.Flags {
	return App().Flags()
}

func Database() *sql.DB {
	return App().Database()
}

func Mailer() *email.Manager {
	return App().Mailer()
}

func Filer() fs.Filer {
	return App().Filer()
}

func Templates() *templates.Manager {
	return App().Templates()
}

func Cache() client.Cache {
	return App().Cache()
}
