package apps

import (
	"database/sql"

	"github.com/Nigel2392/django"
	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/command"
	"github.com/Nigel2392/django/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

type AppConfig struct {
	AppName        string
	Path           string
	Deps           []string
	Cmd            []command.Command
	Init           func(settings django.Settings) error
	Ready          func() error
	URLPatterns    []core.URL
	Middlewares    []core.Middleware
	CtxProcessors  []func(tpl.RequestContext)
	TemplateConfig *tpl.Config
	ready          bool
}

type DBRequiredAppConfig struct {
	*AppConfig
	DB   *sql.DB
	Init func(settings django.Settings, db *sql.DB) error
}

func NewDBAppConfig(name string, patterns ...core.URL) *DBRequiredAppConfig {
	var app = &DBRequiredAppConfig{
		AppConfig: NewAppConfig(name, patterns...),
	}
	return app
}

func (a *DBRequiredAppConfig) Initialize(settings django.Settings) error {
	var dbInt, ok = settings.Get(django.APPVAR_DATABASE)
	assert.True(ok, "DATABASE setting is required for '%s' app", a.AppName)

	db, ok := dbInt.(*sql.DB)
	assert.True(ok, "DATABASE setting must be of type *sql.DB")

	a.DB = db

	if a.Init != nil {
		return a.Init(settings, db)
	}
	return nil
}

func NewAppConfig(name string, patterns ...core.URL) *AppConfig {
	var app = &AppConfig{
		AppName:       name,
		URLPatterns:   make([]core.URL, 0),
		Middlewares:   make([]core.Middleware, 0),
		CtxProcessors: make([]func(tpl.RequestContext), 0),
		Cmd:           make([]command.Command, 0),
	}

	app.URLPatterns = append(app.URLPatterns, patterns...)

	return app
}

func (a *AppConfig) Register(p ...core.URL) {
	if a.URLPatterns == nil {
		a.URLPatterns = make([]core.URL, 0)
	}
	a.URLPatterns = append(a.URLPatterns, p...)
}

func (a *AppConfig) AddCommand(c ...command.Command) {
	if a.Cmd == nil {
		a.Cmd = make([]command.Command, 0)
	}
	a.Cmd = append(a.Cmd, c...)
}

func (a *AppConfig) Commands() []command.Command {
	return a.Cmd
}

func (a *AppConfig) Dependencies() []string {
	return a.Deps
}

func (a *AppConfig) AddMiddleware(m ...mux.Middleware) {
	if a.Middlewares == nil {
		a.Middlewares = make([]core.Middleware, 0)
	}
	var mws = make([]core.Middleware, len(m))
	for i, mw := range m {
		mws[i] = core.NewMiddleware(mw)
	}
	a.Middlewares = append(a.Middlewares, mws...)
}

func (a *AppConfig) Use(m ...core.Middleware) {
	if a.Middlewares == nil {
		a.Middlewares = make([]core.Middleware, 0)
	}
	a.Middlewares = append(a.Middlewares, m...)
}

func (a *AppConfig) Name() string {
	return a.AppName
}

func (a *AppConfig) IsReady() bool {
	return a.ready
}

func (a *AppConfig) URLPath() string {
	return a.Path
}

func (a *AppConfig) URLs() []core.URL {
	return a.URLPatterns
}

func (a *AppConfig) Middleware() []core.Middleware {
	return a.Middlewares
}

func (a *AppConfig) Templates() *tpl.Config {
	return a.TemplateConfig
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
	var err error
	if a.Ready != nil {
		err = a.Ready()
	}
	a.ready = true
	return err
}
