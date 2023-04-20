package django

import (
	"net/http"
	"text/template"

	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/cache"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
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
	return (*App)(app.New(app.Config(c)))
}

func Flags() *flag.Flags {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Flags()
}

func DefaultDb() db.PoolItem[*gorm.DB] {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).DefaultDB()
}

func Mailer() *email.Manager {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Mailer()
}

func FS() *fs.Manager {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).FS()
}

func Templates() *templates.Manager {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Templates()
}

func Admin() *admin.AdminSite {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Admin()
}

func Cache() cache.Cache {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Cache()
}

func TemplateFuncs(t template.FuncMap) {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	(*app.Application)(defaultApp).TemplateFuncs(t)
}

func Middlewares(m ...router.Middleware) {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	(*app.Application)(defaultApp).Middlewares(m...)
}

func Serve() http.Handler {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Serve()
}
func Run() error {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Run()
}
func ServeRedirect() error {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Redirect()
}

func Register(toAdmin bool, key db.DATABASE_KEY, models ...any) {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	(*app.Application)(defaultApp).Register(toAdmin, key, models...)
}

func Logger() request.Logger {
	if defaultApp == nil {
		panic("Call godjango.New to initialize the application first!")
	}
	return (*app.Application)(defaultApp).Logger
}
