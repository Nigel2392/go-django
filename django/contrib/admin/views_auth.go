package admin

import (
	"net/http"

	"github.com/Nigel2392/django"
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
		form.Request = req
		return form.Login()
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *auth.BaseUserLoginForm) {
		var adminIndex = django.Reverse("admin:home")
		http.Redirect(w, req, adminIndex, http.StatusSeeOther)
	},
}
