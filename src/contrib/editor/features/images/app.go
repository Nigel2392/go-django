package images

import (
	"github.com/Nigel2392/go-django/src/apps"

	_ "embed"
)

const defaultMaxBytes uint = 1024 * 1024 * 32 // 32MB

type (
	AppConfig struct {
		*apps.AppConfig
	}
)

var app *AppConfig

func NewAppConfig() *AppConfig {
	if app == nil {
		app = &AppConfig{
			AppConfig: apps.NewAppConfig(
				"editorjs_images",
			),
		}
	}

	app.Deps = []string{
		"editor",
		"images",
	}

	return app
}
