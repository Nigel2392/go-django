package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/views"
)

var HomeHandler = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: BASE_KEY,
	TemplateName:    "admin/views/home.tmpl",
	GetContextFn: func(req *http.Request) (ctx.Context, error) {
		var context = NewContext(req, AdminSite, nil)
		context.SetPage(PageOptions{
			TitleFn:    fields.S("Home"),
			SubtitleFn: fields.S("Welcome to the Django Admin"),
		})
		return context, nil
	},
}
