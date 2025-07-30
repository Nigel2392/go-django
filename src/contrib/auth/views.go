package auth

import (
	"errors"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux/middleware/authentication"
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

func callOrString(fnOrStr interface{}, req *http.Request) string {
	if fn, ok := fnOrStr.(func(*http.Request) string); ok {
		return fn(req)
	}
	return fnOrStr.(string)
}

func (v *AuthView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	except.Assert(v.TemplateName != "", http.StatusInternalServerError, "TemplateName is required")
	except.Assert(v.getForm != nil, http.StatusInternalServerError, "GetForm is required")
	except.Assert(v.onValid != nil, http.StatusInternalServerError, "OnValid is required")

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
		var nextURLSetting = django.ConfigGet[interface{}](
			django.Global.Settings,
			APPVAR_LOGIN_REDIRECT_URL,
			DEFAULT_LOGIN_REDIRECT_URL,
		)
		context.Set("NextURL", callOrString(
			nextURLSetting, req,
		))
	}

	return context, nil
}

func (v *AuthView[T]) Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) (err error) {

	var form = v.getForm(req)
	if req.Method == http.MethodPost {
		if form.IsValid() {
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
		any(form).(forms.ErrorAdder).AddFormError(err)
	}
	context.Set("Form", form)
	return v.BaseView.Render(w, req, v.TemplateName, context)

formSuccess:
	var nextURL = req.URL.Query().Get("next")
	if nextURL == "" {
		var nextURLSetting = django.ConfigGet[interface{}](
			django.Global.Settings,
			APPVAR_LOGIN_REDIRECT_URL,
			DEFAULT_LOGIN_REDIRECT_URL,
		)
		nextURL = callOrString(nextURLSetting, req)
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

func viewUserRegister(w http.ResponseWriter, r *http.Request) {
	var v = RegisterView(&views.BaseView{
		BaseTemplateKey: "auth",
		TemplateName:    "auth/register.tmpl",
	}, RegisterFormConfig{
		AlwaysAllLoginFields: true,
	})

	views.Invoke(v, w, r)
}

func LogoutView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		except.Fail(
			http.StatusMethodNotAllowed,
			"Method not allowed",
		)
		return
	}

	var redirectURL = r.URL.Query().Get(
		"next",
	)
	if redirectURL == "" {
		redirectURL = django.Reverse(
			"auth:login",
		)
	}
	var u = authentication.Retrieve(
		r,
	)
	if u == nil || !u.IsAuthenticated() {
		http.Redirect(
			w, r,
			redirectURL,
			http.StatusSeeOther,
		)
		return
	}

	if err := Logout(r); err != nil && !errors.Is(err, autherrors.ErrNoSession) {
		logger.Errorf(
			"Failed to log user out: %v", err,
		)
		except.Fail(
			500, trans.T(r.Context(),
				"Failed to logout due to unexpected error",
			),
		)
		return
	}

	http.Redirect(
		w, r,
		redirectURL,
		http.StatusSeeOther,
	)
}
