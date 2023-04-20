package auth

import (
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/forms"
	"github.com/Nigel2392/go-django/forms/validators"
	"github.com/Nigel2392/router/v3/request"
)

// Predefined login form.
//
// Will use the USER_MODEL_LOGIN_FIELD to determine what type of field to use.
//
//	func login(r *request.Request) {
//		var form = auth.LoginForm("my-input-class", "my-label-class")
//		if r.Method() == http.MethodPost {
//			if form.Fill(r) {
//				// form is valid used is logged in.
//			} else {
//				// form not valid ...
//				// errors can be accessed via form.Errors
//			}
//		}
//
//		r.Data.Set("form", form)
//		var err = r.Render("auth/login.tmpl")
//		if err != nil {
//			...
//		}
//	}
func LoginForm(inputClass, labelClass string) *forms.Form {
	var titled = httputils.TitleCaser.String(USER_MODEL_LOGIN_FIELD)
	var login_field *forms.Field = &forms.Field{
		LabelText:   titled,
		Placeholder: titled,
		Name:        USER_MODEL_LOGIN_FIELD,
		Required:    true,
		Class:       inputClass,
		LabelClass:  labelClass,
	}
	switch strings.ToLower(USER_MODEL_LOGIN_FIELD) {
	case "email":
		login_field.Type = forms.TypeEmail
		login_field.Validators = append(login_field.Validators, validators.Regex(validators.REGEX_EMAIL))
	default:
		login_field.Type = forms.TypeText
	}

	var loginForm = forms.Form{
		Fields: []forms.FormElement{
			login_field,
			&forms.Field{
				LabelText:   "Password",
				Type:        forms.TypePassword,
				Name:        "password",
				Placeholder: "Password",
				Required:    true,
				Class:       inputClass,
				LabelClass:  labelClass,
			},
		},
		AfterValid: func(r *request.Request, f *forms.Form) error {
			var _, err = Login(r, f.Field(USER_MODEL_LOGIN_FIELD).Value().Value(), f.Field("password").Value().Value())
			return err
		},
	}
	return &loginForm
}
