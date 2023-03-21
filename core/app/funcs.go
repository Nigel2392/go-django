package app

import (
	"html/template"

	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/templates"
	"gorm.io/gorm"
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

func (a *Application) DefaultDB() db.PoolItem[*gorm.DB] {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.DefaultDB()")
	}
	if a.defaultDatabase == nil {
		panic("You must initialize the app with a database before calling app.DefaultDB()")
	}
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

func (a *Application) FS() *fs.Manager {
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

func (a *Application) Admin() *admin.AdminSite {
	if !a.initted {
		panic("You must initialize the app (call app.New(...)) before calling app.Admin()")
	}
	if a.adminSite == nil {
		panic("You must initialize the app with an admin before calling app.Admin()")
	}
	return a.adminSite
}