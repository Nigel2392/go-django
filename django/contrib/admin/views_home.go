package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/views"
)

var HomeHandler = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "admin",
	TemplateName:    "admin/views/home.tmpl",
	GetContextFn: func(req *http.Request) (ctx.Context, error) {
		var context = core.Context(req)

		return context, nil
	},
}
