package messages

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/ctx"
)

var (
	app *MessageAppConfig = &MessageAppConfig{
		AppConfig: apps.NewAppConfig("messages"),
	}
)

type MessageAppConfig struct {
	*apps.AppConfig

	initBackend func(r *http.Request) (MessageBackend, error)
	Tags        *MessageTags
}

func NewAppConfig() django.AppConfig {
	app.Ready = func() error {
		app.checkTags()

		if app.initBackend == nil {
			if django.AppInstalled("session") {
				app.initBackend = NewSessionBackend
			} else {
				app.initBackend = NewCookieBackend
			}
		}

		return nil
	}

	app.CtxProcessors = []func(ctx.ContextWithRequest){
		RequestMessageContext,
	}

	app.Routing = func(m django.Mux) {
		m.Use(MessagesMiddleware)
	}

	return app
}

func (m *MessageAppConfig) ConfigureTags(tags MessageTags) {
	m.Tags = &tags
}

func (m *MessageAppConfig) ConfigureBackend(initBackend func(r *http.Request) (MessageBackend, error)) {
	m.initBackend = initBackend
}

func (m *MessageAppConfig) checkTags() {
	if app.Tags == nil {
		app.Tags = &DefaultTags
	}

	if app.Tags.Debug == "" {
		app.Tags.Debug = DEBUG
	}

	if app.Tags.Info == "" {
		app.Tags.Info = INFO
	}

	if app.Tags.Success == "" {
		app.Tags.Success = SUCCESS
	}

	if app.Tags.Warning == "" {
		app.Tags.Warning = WARNING
	}

	if app.Tags.Error == "" {
		app.Tags.Error = ERROR
	}
}
