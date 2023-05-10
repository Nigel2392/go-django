package auth

import (
	"errors"
	"strings"

	"github.com/Nigel2392/forms"
	"github.com/Nigel2392/forms/validators"
	"github.com/Nigel2392/router/v3/request"
	"github.com/gosimple/slug"
)

func LoginForm(inputClass, labelClass string) *forms.Form {
	var titled = forms.DefaultTitleCaser(USER_MODEL_LOGIN_FIELD)
	var login_field *forms.Field = &forms.Field{
		LabelText:   titled,
		Placeholder: titled,
		Name:        strings.ToLower(USER_MODEL_LOGIN_FIELD),
		Required:    true,
		Class:       inputClass,
		LabelClass:  labelClass,
	}
	switch USER_MODEL_LOGIN_FIELD {
	case "email":
		login_field.Type = forms.TypeEmail
		login_field.Validators = validators.New(validators.Email)
	case "username":
		login_field.Type = forms.TypeText
		login_field.Validators = validators.New(validators.Length(3, 75))
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
				Validators:  DEFAULT_PASSWORD_VALIDATORS,
			},
		},
		AfterValid: func(r *request.Request, f *forms.Form) error {
			var login_field = f.Get(strings.ToLower(USER_MODEL_LOGIN_FIELD))
			var password_field = f.Get("password")
			if login_field == nil || password_field == nil {
				return errors.New("Please fill out all fields")
			}

			var login = login_field.Value()
			var password = password_field.Value()
			if len(login) == 0 || len(password) == 0 {
				return errors.New("Please fill out all fields")
			}
			var _, err = Login(r, login[0], password[0])
			return err
		},
	}
	return &loginForm
}

func RegisterForm(autologin, requireNames bool, inputClass, labelClass string) *forms.Form {
	var registerForm = forms.Form{
		Fields: []forms.FormElement{
			authFormField(forms.TypeEmail, "Email", inputClass, labelClass, true, validators.New(validators.Email)),
			authFormField(forms.TypeText, "Username", inputClass, labelClass, true, validators.New(validators.Length(3, 75))),
			authFormField(forms.TypeText, "First Name", inputClass, labelClass, false || requireNames, validators.New(validators.Length(1, 75))),
			authFormField(forms.TypeText, "Last Name", inputClass, labelClass, false || requireNames, validators.New(validators.Length(1, 75))),
			authFormField(forms.TypePassword, "Password", inputClass, labelClass, true, DEFAULT_PASSWORD_VALIDATORS),
			authFormField(forms.TypePassword, "Password Confirm", inputClass, labelClass, true, DEFAULT_PASSWORD_VALIDATORS),
		},
		BeforeValid: func(r *request.Request, f *forms.Form) error {
			var password, err = getFormValueStr(f, makeSlug("Password"))
			if err != nil {
				return err
			}
			password_confirm, err := getFormValueStr(f, makeSlug("Password Confirm"))
			if err != nil {
				return err
			}

			if password != password_confirm {
				return errors.New("Passwords do not match")
			}
			return nil
		},
		AfterValid: func(r *request.Request, f *forms.Form) error {
			email, err := getFormValueStr(f, makeSlug("Email"))
			if err != nil {
				return err
			}
			username, err := getFormValueStr(f, makeSlug("Username"))
			if err != nil {
				return err
			}
			first_name, err := getFormValueStr(f, makeSlug("First Name"))
			if err != nil {
				return err
			}
			last_name, err := getFormValueStr(f, makeSlug("Last Name"))
			if err != nil {
				return err
			}
			password, err := getFormValueStr(f, makeSlug("Password"))
			if err != nil {
				return err
			}
			var u *User
			u, err = Register(email, username, first_name, last_name, password)
			if err != nil {
				return err
			}
			if u.ID == 0 {
				return errors.New("Invalid user")
			}
			if autologin {
				LoginUnsafe(r, u)
			}
			return nil
		},
	}
	return &registerForm
}

func authFormField(typ, textLower, inputClass, labelClass string, required bool, v []validators.Validator) *forms.Field {
	return &forms.Field{
		Type:        typ,
		ID:          makeSlug(textLower),
		Name:        makeSlug(textLower),
		LabelText:   forms.DefaultTitleCaser(textLower),
		Placeholder: forms.DefaultTitleCaser(textLower),
		Required:    required,
		Class:       inputClass,
		LabelClass:  labelClass,
		Validators:  v,
	}
}

func makeSlug(textLower string) string {
	return slug.Make(strings.ToLower(textLower))
}

func getFormValueStr(f *forms.Form, name string) (string, error) {
	var field = f.Get(name)
	if field == nil {
		return "", errors.New("Invalid form")
	}
	var value = field.Value()
	if len(value) == 0 {
		return "", errors.New("Invalid form")
	}

	return value[0], nil
}
