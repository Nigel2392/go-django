package auth

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/views"
)

type AuthView[T forms.Form] struct {
	*views.BaseView
	getForm   func(req *http.Request) T
	OnSuccess func(w http.ResponseWriter, req *http.Request, form T) error
	onValid   func(req *http.Request, form T) error
	onInvalid func(req *http.Request, form T) error
}

func newAuthView[T forms.Form](baseView *views.BaseView) *AuthView[T] {
	var v = &AuthView[T]{
		BaseView: baseView,
	}

	if v.AllowedMethods == nil {
		v.AllowedMethods = []string{http.MethodGet, http.MethodPost}
	}

	return v
}

func (v *AuthView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	fmt.Println("Getting context")
	except.Assert(v.TemplateName != "", 500, "TemplateName is required")
	except.Assert(v.getForm != nil, 500, "GetForm is required")
	except.Assert(v.onValid != nil, 500, "OnValid is required")

	var context, err = v.BaseView.GetContext(req)
	if err != nil {
		return nil, err
	}

	var loginURL = django.Reverse("auth:login")
	var nextURL = req.URL.Query().Get("next")

	context.Set("FormURL", loginURL)
	if nextURL != "" {
		context.Set("NextURL", nextURL)
	} else {
		context.Set("NextURL", django.ConfigGet(
			django.Global.Settings,
			"LOGIN_REDIRECT_URL",
			"/",
		))
	}

	return context, nil
}

func (v *AuthView[T]) Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) (err error) {

	var form = v.getForm(req)
	if req.Method == http.MethodPost {
		if form.IsValid() {
			fmt.Println("Form is valid")
			err = v.onValid(req, form)
			if err != nil {
				if v.onInvalid != nil {
					err = v.onInvalid(req, form)
				}
				goto checkFormErr
			}

			if v.OnSuccess != nil {
				err = v.OnSuccess(w, req, form)
				if err != nil {
					return err
				}
			}
			goto formSuccess
		} else {
			if v.onInvalid != nil {
				err = v.onInvalid(req, form)
			}
		}
	}

checkFormErr:
	if err != nil {
		fmt.Println("Error in form", err)
		any(form).(forms.ErrorAdder).AddFormError(err)
	}
	context.Set("Form", form)
	return v.BaseView.Render(w, req, v.TemplateName, context)

formSuccess:
	var nextURL = req.URL.Query().Get("next")
	if nextURL == "" {
		nextURL = django.ConfigGet(
			django.Global.Settings,
			"LOGIN_REDIRECT_URL",
			"/",
		)
	}
	http.Redirect(w, req, nextURL, http.StatusSeeOther)
	return nil
}

func LoginView(baseView *views.BaseView, opts ...func(forms.Form)) *AuthView[*BaseUserForm] {
	var v = newAuthView[*BaseUserForm](baseView)
	v.getForm = func(req *http.Request) *BaseUserForm {
		var f = UserLoginForm(req, opts...)
		f.Request = req
		return f
	}
	v.onValid = func(req *http.Request, form *BaseUserForm) error {
		form.Request = req
		return form.Login()
	}
	return v
}

func RegisterView(baseView *views.BaseView, cfg RegisterFormConfig, opts ...func(forms.Form)) *AuthView[*BaseUserForm] {
	var v = newAuthView[*BaseUserForm](baseView)
	v.getForm = func(req *http.Request) *BaseUserForm {
		var f = UserRegisterForm(req, cfg, opts...)
		f.Request = req
		return f
	}
	v.onValid = func(req *http.Request, form *BaseUserForm) error {
		var _, err = form.Save()
		return err
	}
	return v
}

func viewUserLogin(w http.ResponseWriter, r *http.Request) {
	var v = LoginView(&views.BaseView{
		BaseTemplateKey: "auth",
		TemplateName:    "auth/login.tmpl",
	})

	views.Invoke(v, w, r)
}

func viewUserLogout(w http.ResponseWriter, r *http.Request) {
	Logout(r)
	http.Redirect(w, r, django.Reverse("auth:login"), http.StatusSeeOther)
}

func viewUserRegister(w http.ResponseWriter, r *http.Request) {
	var v = RegisterView(&views.BaseView{
		BaseTemplateKey: "auth",
		TemplateName:    "auth/register.tmpl",
	}, RegisterFormConfig{
		AlwaysAllLoginFields: true,
	})

	views.Invoke(v, w, r)
}
