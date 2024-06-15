package admin

import (
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/views"
)

var LoginHandler = &views.FormView[*AdminForm[*auth.BaseUserLoginForm]]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "admin",
		TemplateName:    "admin/views/auth/login.tmpl",
	},
	GetFormFn: func(req *http.Request) *AdminForm[*auth.BaseUserLoginForm] {
		return &AdminForm[*auth.BaseUserLoginForm]{
			Form: auth.UserLoginForm(req),
		}
	},
	ValidFn: func(req *http.Request, form *AdminForm[*auth.BaseUserLoginForm]) error {
		form.Form.Request = req
		return form.Form.Login()
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *AdminForm[*auth.BaseUserLoginForm]) {
		var adminIndex = django.Reverse("admin:home")
		http.Redirect(w, req, adminIndex, http.StatusSeeOther)
	},
}
