package auth

import (
	"context"
	"net/http"
	"net/mail"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/mux"
	"github.com/pkg/errors"
)

type BaseUserForm struct {
	*forms.BaseForm
	Request  *http.Request
	Instance *models.User
	canSave  bool
	config   *RegisterFormConfig
}

const postMethod = mux.POST

func UserLoginForm(r *http.Request, formOpts ...func(forms.Form)) *BaseUserForm {
	var f = &BaseUserForm{
		Request: r,
		BaseForm: forms.NewBaseForm(
			append([]func(forms.Form){forms.WithRequestData(postMethod, r)}, formOpts...)...,
		),
	}

	f.OnInvalid(func(_ forms.Form) {
		SIGNAL_LOGIN_FAILED.Send(f.Raw)
	})

	if Auth.LoginWithEmail {
		f.AddField(
			"email",
			fields.Protect(fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email address"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(254),
			), func(err error) error { return ErrInvalidLogin }),
		)
	} else {
		f.AddField(
			"username",
			fields.Protect(fields.CharField(
				fields.Label("Username"),
				fields.HelpText("Enter your username"),
				fields.Required(true),
				fields.MinLength(3),
				fields.MaxLength(75),
				fields.Regex(`^[a-zA-Z][a-zA-Z0-9_]+$`),
			), func(err error) error { return ErrInvalidLogin }),
		)
	}

	f.AddField(
		"password",
		fields.Protect(NewPasswordField(
			ChrFlagDEFAULT,
			false,
			fields.Label("Password"),
			fields.HelpText("Enter your password"),
			fields.Required(true),
		), func(err error) error { return ErrInvalidLogin }),
	)

	return f
}

type RegisterFormConfig struct {

	// Include both email and username fields in the registration form.
	//
	// If this is false - only the field specified by `LoginWithEmail` will be
	// included in the form.
	AlwaysAllLoginFields bool

	// Automatically login the user after registration.
	//
	// This requires a non-nil http request to be passed to the form.
	AutoLogin bool

	// Ask for the user's first and last name.
	AskForNames bool

	// Create an inactive user account.
	//
	// This is useful for when the user needs to verify their email address
	// before they can login.
	IsInactive bool
}

func UserRegisterForm(r *http.Request, registerConfig RegisterFormConfig, formOpts ...func(forms.Form)) *BaseUserForm {
	var f = &BaseUserForm{
		Request: r,
		BaseForm: forms.NewBaseForm(
			append([]func(forms.Form){forms.WithRequestData(postMethod, r)}, formOpts...)...,
		),
		canSave: true,
		config:  &registerConfig,
	}

	if registerConfig.AlwaysAllLoginFields || Auth.LoginWithEmail {
		f.AddField(
			"email",
			fields.EmailField(
				fields.Label("Email"),
				fields.HelpText("Enter your email address"),
				fields.Required(true),
				fields.MinLength(5),
				fields.MaxLength(254),
			),
		)
	}

	if registerConfig.AlwaysAllLoginFields || !Auth.LoginWithEmail {
		f.AddField(
			"username",
			fields.CharField(
				fields.Label("Username"),
				fields.HelpText("Enter your username"),
				fields.Required(true),
				fields.MinLength(3),
				fields.MaxLength(75),
				fields.Regex(`^[a-zA-Z][a-zA-Z0-9_]+$`),
			),
		)
	}

	if registerConfig.AskForNames {
		f.AddField(
			"firstName",
			fields.CharField(
				fields.Label("First Name"),
				fields.HelpText("Enter your first name"),
				fields.Required(true),
				fields.MinLength(2),
				fields.MaxLength(75),
			),
		)

		f.AddField(
			"lastName",
			fields.CharField(
				fields.Label("Last Name"),
				fields.HelpText("Enter your last name"),
				fields.Required(true),
				fields.MinLength(2),
				fields.MaxLength(75),
			),
		)
	}

	f.AddField(
		"password",
		NewPasswordField(
			ChrFlagDEFAULT,
			true,
			fields.Label("Password"),
			fields.HelpText("Enter your password"),
			fields.Required(true),
		),
	)

	f.AddField(
		"passwordConfirm",
		NewPasswordField(
			ChrFlagDEFAULT,
			true,
			fields.Label("Password Confirm"),
			fields.HelpText("Enter the password again to confirm"),
			fields.Required(true),
		),
	)

	f.SetValidators(func(f forms.Form) []error {
		var cleaned = f.CleanedData()
		if cleaned["password"] != cleaned["passwordConfirm"] {
			return []error{ErrPwdNoMatch}
		}
		return nil
	})
	f.SetValidators(func(f forms.Form) []error {
		var (
			ctx      = context.Background()
			cleaned  = f.CleanedData()
			err      error
			email    = cleaned["email"]
			username = cleaned["username"]
		)

		if registerConfig.AlwaysAllLoginFields || Auth.LoginWithEmail {
			if email == nil {
				return []error{errors.Wrap(
					errs.ErrFieldRequired, "Email is required",
				)}
			}
			_, err = Auth.Queries.RetrieveByEmail(ctx, email.(*mail.Address).Address)
			if err == nil {
				return []error{errors.Wrap(
					ErrUserExists, "Email exists",
				)}
			}

			email, err := mail.ParseAddress(email.(*mail.Address).Address)
			if err != nil {
				return []error{ErrInvalidEmail}
			}

			cleaned["email"] = (*models.Email)(email)
		}

		if registerConfig.AlwaysAllLoginFields || !Auth.LoginWithEmail {
			if username == nil {
				return []error{errors.Wrap(
					errs.ErrFieldRequired, "Username is required",
				)}
			}
			_, err = Auth.Queries.RetrieveByUsername(ctx, username.(string))
			if err == nil {
				return []error{errors.Wrap(
					ErrUserExists, "Username exists",
				)}
			}
		}
		return nil
	})

	return f
}

func (f *BaseUserForm) basicChecks() error {
	if f.Errors != nil && f.Errors.Len() > 0 {
		for head := f.Errors.Front(); head != nil; head = head.Next() {
			return errs.NewMultiError(head.Value...)
		}
	}

	if len(f.ErrorList_) > 0 {
		return errs.NewMultiError(f.ErrorList_...)
	}

	if f.Cleaned == nil {
		return errs.Error("Form not cleaned")
	}

	return nil
}

func (f *BaseUserForm) Save() (*models.User, error) {
	if err := f.basicChecks(); err != nil {
		return nil, err
	}

	if !f.canSave {
		return nil, errs.Error("Form cannot be saved")
	}

	var (
		cleaned = f.CleanedData()
		err     error
	)
	if f.Instance == nil {
		f.Instance = &models.User{}
	}

	if f.config.AlwaysAllLoginFields || Auth.LoginWithEmail {
		f.Instance.Email = cleaned["email"].(*models.Email)
	}

	if f.config.AlwaysAllLoginFields || !Auth.LoginWithEmail {
		f.Instance.Username = cleaned["username"].(string)
	}

	if f.config.AskForNames {
		f.Instance.FirstName = cleaned["firstName"].(string)
		f.Instance.LastName = cleaned["lastName"].(string)
	}

	f.Instance.IsActive = !f.config.IsInactive
	f.Instance.IsAdministrator = false

	err = f.Instance.Save(f.Request.Context())
	if err != nil {
		return nil, errors.Wrap(err, "Error saving user")
	}

	if f.config.AutoLogin {
		*f.Instance = *Login(f.Request, f.Instance)
	}

	return f.Instance, nil
}

func (f *BaseUserForm) Login() error {
	if err := f.basicChecks(); err != nil {
		return err
	}

	var (
		ctx     = f.Request.Context()
		cleaned = f.CleanedData()
		user    *models.User
		err     error
	)
	if Auth.LoginWithEmail {
		user, err = Auth.Queries.RetrieveByEmail(ctx, cleaned["email"].(string))
	} else {
		user, err = Auth.Queries.RetrieveByUsername(ctx, cleaned["username"].(string))
	}
	if err != nil {
		return errors.Wrap(
			err, "Error retrieving user",
		)
	}

	if err := CheckPassword(user, string(cleaned["password"].(PasswordString))); err != nil {
		return ErrPasswordInvalid
	}

	if !user.IsActive {
		return ErrIsActive
	}

	Login(f.Request, user)
	f.Instance = user
	return nil
}
