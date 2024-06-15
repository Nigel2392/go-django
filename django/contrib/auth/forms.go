package auth

import (
	"net/http"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/mux"
	"github.com/pkg/errors"
)

type BaseUserLoginForm struct {
	*forms.BaseForm
	Request  *http.Request
	Instance *models.User
}

const postMethod = mux.POST

func UserLoginForm(r *http.Request, formOpts ...func(forms.Form)) *BaseUserLoginForm {
	var f = &BaseUserLoginForm{
		Request: r,
		BaseForm: forms.NewBaseForm(
			append([]func(forms.Form){forms.WithRequestData(postMethod, r)}, formOpts...)...,
		),
	}

	if Auth.LoginWithEmail {
		f.AddField(
			"email",
			fields.Protect(fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email address"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(254),
			), func(err error) error { return errs.Error("Invalid value provided") }),
		)
	} else {
		f.AddField(
			"username",
			fields.Protect(fields.CharField(
				fields.Label("Username"),
				fields.HelpText("Enter your username"),
				fields.Required(true),
				fields.MinLength(3),
				fields.MaxLength(32),
				fields.Regex(`^[a-zA-Z][a-zA-Z0-9_]+$`),
			), func(err error) error { return errs.Error("Invalid value provided") }),
		)
	}

	f.AddField(
		"password",
		fields.Protect(NewPasswordField(
			fields.Label("Password"),
			fields.HelpText("Enter your password"),
			fields.Required(true),
			fields.MinLength(8),
			fields.MaxLength(64),
			ValidateCharacters(false, ChrFlagDigit|ChrFlagLower|ChrFlagUpper|ChrFlagSpecial),
		), func(err error) error { return errs.Error("Invalid value provided") }),
	)

	return f
}

func (f *BaseUserLoginForm) Login() error {
	if f.Errors != nil && f.Errors.Len() > 0 {
		for head := f.Errors.Front(); head != nil; head = head.Next() {
			return errs.NewMultiError(head.Value...)
		}
	}

	if f.Cleaned == nil {
		return errs.Error("Form not cleaned")
	}

	var (
		ctx     = f.Request.Context()
		cleaned = f.CleanedData()
		user    models.User
		err     error
	)
	if Auth.LoginWithEmail {
		user, err = Auth.Queries.UserByEmail(ctx, cleaned["email"].(string))
	} else {
		user, err = Auth.Queries.UserByUsername(ctx, cleaned["username"].(string))
	}
	if err != nil {
		return errors.Wrap(
			err, "Error retrieving user",
		)
	}

	var u = &user
	if err := models.CheckPassword(u, string(cleaned["password"].(PasswordString))); err != nil {
		return errs.Error("Invalid password")
	}

	if !u.IsActive {
		return errs.Error("User account is not active")
	}

	Login(f.Request, u)
	f.Instance = u
	return nil
}
