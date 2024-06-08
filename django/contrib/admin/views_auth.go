package admin

import (
	"net/http"

	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/views"
)

var LoginHandler = &views.FormView[forms.Form]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "admin",
		TemplateName:    "admin/views/auth/login.tmpl",
	},
	GetFormFn: func(req *http.Request) forms.Form {
		return auth.UserLoginForm(req)
	},
}
