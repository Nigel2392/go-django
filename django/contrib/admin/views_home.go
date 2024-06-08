package admin

import (
	"html/template"
	"net/http"

	"github.com/Nigel2392/django/contrib/admin/menu"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/views"
)

var HomeHandler = &views.BaseView{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "admin",
	TemplateName:    "admin/views/home.tmpl",
	GetContextFn: func(req *http.Request) (ctx.Context, error) {
		var context = core.Context(req)

		var menu = &menu.Menu{
			Items: []menu.MenuItem{
				&menu.Item{
					Label: fields.S("Users"),
					Link:  fields.S("/admin/users/"),
				},
			},
		}

		context.Set("menu", template.HTML(menu.HTML()))

		return context, nil
	},
}
