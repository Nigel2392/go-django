package apps

import (
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

type AppConfig struct {
	AppName        string
	Deps           []string
	Routing        func(django.Mux)
	Cmd            []command.Command
	Init           func(settings django.Settings) error
	Ready          func() error
	CtxProcessors  []func(tpl.RequestContext)
	TemplateConfig *tpl.Config
	ready          bool
}

type DBRequiredAppConfig struct {
	*AppConfig
	DB   *sql.DB
	Init func(settings django.Settings, db *sql.DB) error
}

func NewDBAppConfig(name string) *DBRequiredAppConfig {
	var app = &DBRequiredAppConfig{
		AppConfig: NewAppConfig(name),
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

func NewAppConfig(name string) *AppConfig {
	var app = &AppConfig{
		AppName:       name,
		CtxProcessors: make([]func(tpl.RequestContext), 0),
		Cmd:           make([]command.Command, 0),
	}

	return app
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

func (a *AppConfig) Name() string {
	return a.AppName
}

func (a *AppConfig) IsReady() bool {
	return a.ready
}

func (a *AppConfig) Templates() *tpl.Config {
	return a.TemplateConfig
}

func (a *AppConfig) Processors() []func(tpl.RequestContext) {
	return a.CtxProcessors
}

func (a *AppConfig) BuildRouting(m django.Mux) {
	if a.Routing != nil {
		a.Routing(m)
	}
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
