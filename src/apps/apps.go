package apps

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

var (
	_ django.AppConfig = (*AppConfig)(nil)
	_ django.AppConfig = (*DBRequiredAppConfig)(nil)
)

type AppConfig struct {
	AppName        string
	Deps           []string
	Routing        func(django.Mux)
	Cmd            []command.Command
	Init           func(settings django.Settings) error
	Ready          func() error
	CtxProcessors  []func(ctx.ContextWithRequest)
	TemplateConfig *tpl.Config
	ModelObjects   []attrs.Definer
	ready          bool
}

type DBRequiredAppConfig struct {
	*AppConfig
	DB   drivers.Database
	Init func(settings django.Settings, db drivers.Database) error
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

	db, ok := dbInt.(drivers.Database)
	assert.True(ok, "DATABASE setting must be of type drivers.Database")

	a.DB = db

	if a.AppConfig.Init != nil {
		err := a.AppConfig.Init(settings)
		if err != nil {
			return err
		}
	}

	if a.Init != nil {
		return a.Init(settings, db)
	}
	return nil
}

func NewAppConfig(name string) *AppConfig {
	var app = &AppConfig{
		AppName:       name,
		CtxProcessors: make([]func(ctx.ContextWithRequest), 0),
		Cmd:           make([]command.Command, 0),
	}

	return app
}

func NewAppConfigForHandler(name string, method string, path string, handler mux.Handler) *AppConfig {
	var app = NewAppConfig(name)
	app.Routing = func(m django.Mux) {
		m.Handle(method, path, handler, name)
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
	if a.TemplateConfig != nil && a.TemplateConfig.AppName == "" {
		a.TemplateConfig.AppName = a.AppName
	}
	return a.TemplateConfig
}

func (a *AppConfig) Processors() []func(ctx.ContextWithRequest) {
	return a.CtxProcessors
}

func (a *AppConfig) BuildRouting(m django.Mux) {
	if a.Routing != nil {
		a.Routing(m)
	}
}

func (a *AppConfig) Models() []attrs.Definer {
	return a.ModelObjects
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

func (a *AppConfig) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = make([]checks.Message, 0)
	for _, model := range a.ModelObjects {
		if checker, ok := model.(checks.Checker); ok {
			var m = checker.Check(ctx)
			messages = append(messages, m...)
			continue
		}

		messages = append(messages, checks.Warning(
			"model.cant_check",
			"Model does not implement checks.Checker interface",
			model,
		))

		var primary attrs.Field
		var defs = model.FieldDefs()
		for _, field := range defs.Fields() {

			if field.IsPrimary() {
				if primary != nil {
					var messageText = fmt.Sprintf(
						"Model has multiple primary key fields: \"%s\" and \"%s\"",
						primary.Name(),
						field.Name(),
					)
					messages = append(messages, checks.Warning(
						"model.multiple_primary_keys",
						messageText,
						model,
					))
					continue
				}

				primary = field
			}

			var _, ok = drivers.DBType(field)
			if !ok {
				messages = append(messages, checks.Warning(
					"field.invalid_db_type",
					"Field does not have a valid database type",
					field,
				))
				continue
			}

			if fieldChecker, ok := field.(checks.Checker); ok {
				var m = fieldChecker.Check(ctx)
				messages = append(messages, m...)
				continue
			}
		}

		if primary == nil && !attrs.ThroughModelMeta(model).IsThroughModel {
			messages = append(messages, checks.Warning(
				"model.no_primary_key",
				"Model does not have a primary key field",
				model,
			))
		}
	}
	return messages
}
