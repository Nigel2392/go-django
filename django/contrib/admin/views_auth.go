package admin

import (
	"net/http"
	"net/url"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type LoginForm interface {
	forms.Form
	SetRequest(req *http.Request)
	Login() error
}

var LoginHandler = &views.FormView[*AdminForm[LoginForm]]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "admin",
		TemplateName:    "admin/views/auth/login.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = ctx.RequestContext(req)
			// context.Set("next", req.URL.Query().Get("next"))
			var loginURL = django.Reverse("admin:login")
			var nextURL = req.URL.Query().Get("next")
			if nextURL != "" {
				var u, err = url.Parse(loginURL)
				if err != nil {
					goto returnContext
				}

				q := u.Query()
				q.Set("next", nextURL)
				u.RawQuery = q.Encode()
				loginURL = u.String()
			}

		returnContext:
			context.Set("LoginURL", loginURL)
			return context, nil
		},
	},
	GetFormFn: func(req *http.Request) *AdminForm[LoginForm] {
		return &AdminForm[LoginForm]{
			Form: AdminSite.LoginForm(req),
		}
	},
	ValidFn: func(req *http.Request, form *AdminForm[LoginForm]) error {
		form.Form.SetRequest(req)
		return form.Form.Login()
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *AdminForm[LoginForm]) {
		var nextURL = req.URL.Query().Get("next")
		if nextURL == "" {
			nextURL = django.Reverse("admin:home")
		}

		http.Redirect(w, req, nextURL, http.StatusSeeOther)
	},
	CheckPermissions: func(w http.ResponseWriter, req *http.Request) error {
		var user = authentication.Retrieve(req)
		if user != nil && user.IsAuthenticated() {
			return errs.Error("Already authenticated")
		}
		return nil
	},
	FailsPermissions: func(w http.ResponseWriter, req *http.Request, err error) {
		var redirectURL = django.Reverse("admin:home")
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
	},
}

func LogoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		except.Fail(http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := AdminSite.Logout(req); err != nil {
		except.Fail(http.StatusInternalServerError, "Failed to log out")
		return
	}

	http.Redirect(w, req, django.Reverse("admin:login"), http.StatusSeeOther)
}
