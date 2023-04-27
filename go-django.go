package godjango

import (
	"net/http"
	"text/template"

	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/netcache/src/client"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates"
	"gorm.io/gorm"
)

// A package intended to be used as a framework for web applications.
// This is still in very early development, and is not ready for production use.
// If you would like to contribute, please do so on the github page.
//
// https://github.com/Nigel2392/go-django

// Everything defined in here is to keep things centralized.
var defaultApp *App

type App app.Application

type AppConfig app.Config

// Create a new application.
func New(c AppConfig) *App {
	var dj = (*App)(app.New(app.Config(c)))
	defaultApp = dj
	return dj
}

func panicIfNoDefaultApp() {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
}

func Run() error {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Run()
}

func Flags() *flag.Flags {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Flags()
}

func DefaultDb() db.PoolItem[*gorm.DB] {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).DefaultDB()
}

func Mailer() *email.Manager {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Mailer()
}

func FS() *fs.Manager {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).FS()
}

func Templates() *templates.Manager {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Templates()
}

func Admin() *admin.AdminSite {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Admin()
}

func Cache() client.Cache {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Cache()
}

func TemplateFuncs(t template.FuncMap) {
	panicIfNoDefaultApp()
	(*app.Application)(defaultApp).TemplateFuncs(t)
}

func Middlewares(m ...router.Middleware) {
	panicIfNoDefaultApp()
	(*app.Application)(defaultApp).Middlewares(m...)
}

func Serve() (http.Handler, error) {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Serve()
}
func ServeRedirect() error {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Redirect()
}

func Register(toAdmin bool, key db.DATABASE_KEY, models ...any) {
	panicIfNoDefaultApp()
	(*app.Application)(defaultApp).Register(toAdmin, key, models...)
}

func Logger() request.Logger {
	panicIfNoDefaultApp()
	return (*app.Application)(defaultApp).Logger
}
