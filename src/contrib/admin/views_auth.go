package admin

import (
	"net/http"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
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
		var loginForm = AdminSite.AuthLoginForm(
			req,
		)
		return &AdminForm[LoginForm]{
			Form: loginForm,
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
	//	CheckPermissions: func(w http.ResponseWriter, req *http.Request) error {
	//		var user = authentication.Retrieve(req)
	//		if user != nil && user.IsAuthenticated() {
	//			return errs.Error("Already authenticated")
	//		}
	//		return nil
	//	},
	//	FailsPermissions: func(w http.ResponseWriter, req *http.Request, err error) {
	//		var redirectURL = django.Reverse("admin:home")
	//		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
	//	},
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	// Check if the user is already authenticated
	var user = authentication.Retrieve(r)
	var next = r.URL.Query().Get("next")
	if user != nil && user.IsAuthenticated() {
		// See if the user already has permissions to access the admin interface
		if user.IsAdmin() || permissions.HasPermission(r, "admin:access_admin") {
			if next == "" {
				next = django.Reverse("admin:home")
			}
			http.Redirect(
				w, r, next,
				http.StatusSeeOther,
			)
			return
		}

		// If the user is not an admin, but is authenticated, redirect to the relogin page
		// and pass the next URL as a query parameter
		ReLogin(w, r, r.URL.Path)
		return
	}

	var handler = AdminSite.AuthLoginHandler()
	if handler == nil {
		views.Invoke(LoginHandler, w, r)
		return
	}

	handler(w, r)
}

func reloginHandler(w http.ResponseWriter, req *http.Request) {
	var user = authentication.Retrieve(req)
	var next = req.URL.Query().Get("next")

	if user == nil || !user.IsAuthenticated() {
		autherrors.Fail(
			http.StatusUnauthorized,
			"You need to login",
			next,
		)
		return
	}

	if user.IsAuthenticated() && user.IsAdmin() {
		if next == "" {
			next = django.Reverse("admin:home")
		}
		http.Redirect(w, req, next, http.StatusSeeOther)
		return
	}

	var context = ctx.RequestContext(req)
	if next != "" {
		context.Set("next", next)
	}

	tpl.FRender(
		w, context,
		"admin", "admin/views/auth/relogin.tmpl",
	)
}

func logoutHandler(w http.ResponseWriter, req *http.Request) {

	var redirectURL = req.URL.Query().Get("next")
	if redirectURL == "" {
		redirectURL = django.Reverse("admin:login")
	}

	var user = authentication.Retrieve(req)
	if user == nil || !user.IsAuthenticated() {
		autherrors.Fail(http.StatusUnauthorized, "You are already logged out", redirectURL)
	}

	if err := AdminSite.AuthLogout(req); err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to logout due to unexpected error",
		)
		return
	}

	http.Redirect(
		w, req,
		redirectURL,
		http.StatusSeeOther,
	)
}
