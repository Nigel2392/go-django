package admin

import (
	"net/http"

	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/views"
)

var LoginHandler = &views.FormView[*auth.BaseUserLoginForm]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "admin",
		TemplateName:    "admin/views/auth/login.tmpl",
	},
	GetFormFn: func(req *http.Request) *auth.BaseUserLoginForm {
		return auth.UserLoginForm(req)
	},
	ValidFn: func(req *http.Request, form *auth.BaseUserLoginForm) error {
		return form.Login()
	},
}
