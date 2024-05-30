package apps

import (
	"database/sql"
	"io/fs"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/mux"
)

type AppConfig struct {
	AppName       string
	Init          func(settings django.Settings) error
	Ready         func() error
	URLPatterns   []http_.URL
	Middlewares   []http_.Middleware
	CtxProcessors []func(tpl.RequestContext)
	TemplateFS    fs.FS
}

type DBRequiredAppConfig struct {
	*AppConfig
	DB   *sql.DB
	Init func(settings django.Settings, db *sql.DB) error
}

func NewDBAppConfig(name string, patterns ...http_.URL) *DBRequiredAppConfig {
	var app = &DBRequiredAppConfig{
		AppConfig: NewAppConfig(name, patterns...),
	}
	return app
}

func (a *DBRequiredAppConfig) Initialize(settings django.Settings) error {
	var dbInt, ok = settings.Get("DATABASE")
	assert.True(ok, "DATABASE setting is required for '%s' app", a.AppName)

	db, ok := dbInt.(*sql.DB)
	assert.True(ok, "DATABASE setting must be of type *sql.DB")

	a.DB = db

	if a.Init != nil {
		return a.Init(settings, db)
	}
	return nil
}

func NewAppConfig(name string, patterns ...http_.URL) *AppConfig {
	var app = &AppConfig{
		AppName:       name,
		URLPatterns:   make([]http_.URL, 0),
		Middlewares:   make([]http_.Middleware, 0),
		CtxProcessors: make([]func(tpl.RequestContext), 0),
	}

	app.URLPatterns = append(app.URLPatterns, patterns...)

	return app
}

func (a *AppConfig) Register(p ...http_.URL) {
	if a.URLPatterns == nil {
		a.URLPatterns = make([]http_.URL, 0)
	}
	a.URLPatterns = append(a.URLPatterns, p...)
}

func (a *AppConfig) AddMiddleware(m ...mux.Middleware) {
	if a.Middlewares == nil {
		a.Middlewares = make([]http_.Middleware, 0)
	}
	var mws = make([]http_.Middleware, len(m))
	for i, mw := range m {
		mws[i] = http_.NewMiddleware(mw)
	}
	a.Middlewares = append(a.Middlewares, mws...)
}

func (a *AppConfig) Use(m ...http_.Middleware) {
	if a.Middlewares == nil {
		a.Middlewares = make([]http_.Middleware, 0)
	}
	a.Middlewares = append(a.Middlewares, m...)
}

func (a *AppConfig) Name() string {
	return a.AppName
}

func (a *AppConfig) URLs() []http_.URL {
	return a.URLPatterns
}

func (a *AppConfig) Middleware() []http_.Middleware {
	return a.Middlewares
}

func (a *AppConfig) Templates() fs.FS {
	return a.TemplateFS
}

func (a *AppConfig) Processors() []func(tpl.RequestContext) {
	return a.CtxProcessors
}

func (a *AppConfig) Initialize(settings django.Settings) error {
	if a.Init != nil {
		return a.Init(settings)
	}
	return nil
}

func (a *AppConfig) OnReady() error {
	if a.Ready != nil {
		return a.Ready()
	}
	return nil
}
