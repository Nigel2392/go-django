package links

import (
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/pages"
)

func NewAppConfig() django.AppConfig {

	var app = apps.NewAppConfig(
		"links",
	)

	pages.SetUseRedirectHandler(true)

	return app
}
