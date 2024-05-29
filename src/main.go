package main

import (
	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/blocks"
	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/mux/middleware"
	"github.com/Nigel2392/src/core"
)

func main() {
	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         true,
			"HOST":          "127.0.0.1",
			"PORT":          "8080",
		}),
		django.AppMiddleware(
			http_.NewMiddleware(
				middleware.DefaultLogger.Intercept,
			),
		),
		django.Apps(
			core.NewAppConfig,
			blocks.NewAppConfig,
		),
	)

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
