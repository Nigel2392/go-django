package auth

import (
	"context"
	"net/http"
	"net/mail"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/mux"
)

const postMethod = mux.POST

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

func UserLoginForm(r *http.Request, formOpts ...func(forms.Form)) *BaseUserForm {
	var f = &BaseUserForm{
		Request: r,
		BaseForm: forms.NewBaseForm(
			r.Context(),
			forms.WithRequestData(postMethod, r),
		),
	}

	f.OnInvalid(func(_ forms.Form) {
		core.SIGNAL_LOGIN_FAILED.Send(f.Raw)
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
			), func(err error) error { return autherrors.ErrInvalidLogin }),
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
			), func(err error) error { return autherrors.ErrInvalidLogin }),
		)
	}

	f.AddField(
		"password",
		fields.Protect(NewPasswordField(
			PasswordFieldOptions{
				Flags:         ChrFlagDEFAULT,
				IsRegistering: false,
			},
			fields.Label("Password"),
			fields.HelpText("Enter your password"),
			fields.Required(true),
		), func(err error) error { return autherrors.ErrInvalidLogin }),
	)

	return forms.Initialize(f, formOpts...)
}

func UserRegisterForm(r *http.Request, registerConfig RegisterFormConfig, formOpts ...func(forms.Form)) *BaseUserForm {
	var f = &BaseUserForm{
		Request: r,
		BaseForm: forms.NewBaseForm(
			r.Context(),
			forms.WithRequestData(postMethod, r),
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
			PasswordFieldOptions{
				Flags:         ChrFlagDEFAULT,
				IsRegistering: true,
			},
			fields.Label("Password"),
			fields.HelpText("Enter your password"),
			fields.Required(true),
		),
	)

	f.AddField(
		"passwordConfirm",
		NewPasswordField(
			PasswordFieldOptions{
				Flags:         ChrFlagDEFAULT,
				IsRegistering: true,
			},
			fields.Label("Password Confirm"),
			fields.HelpText("Enter the password again to confirm"),
			fields.Required(true),
		),
	)

	f.SetValidators(func(f forms.Form, cleaned map[string]any) []error {
		if cleaned["password"].(*Password).Raw != cleaned["passwordConfirm"].(*Password).Raw {
			return []error{autherrors.ErrPwdNoMatch}
		}
		return nil
	})

	f.SetValidators(func(f forms.Form, cleaned map[string]any) []error {
		var (
			ctx      = context.Background()
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
			_, err = queries.GetQuerySetWithContext(ctx, &User{}).
				Filter("Email", email.(*mail.Address).Address).
				Get()
			if err == nil {
				return []error{errors.Wrap(
					autherrors.ErrUserExists, "Email exists",
				)}
			}

			email, err := mail.ParseAddress(email.(*mail.Address).Address)
			if err != nil {
				return []error{autherrors.ErrInvalidEmail}
			}

			cleaned["email"] = (*drivers.Email)(email)
		}

		if registerConfig.AlwaysAllLoginFields || !Auth.LoginWithEmail {
			if username == nil {
				return []error{errors.Wrap(
					errs.ErrFieldRequired, "Username is required",
				)}
			}
			_, err = queries.GetQuerySetWithContext(ctx, &User{}).
				Filter("Username", username.(string)).
				Get()
			if err == nil {
				return []error{errors.Wrap(
					autherrors.ErrUserExists, "Username exists",
				)}
			}
		}
		return nil
	})

	f.BaseForm.OnInvalidFuncs = append(f.BaseForm.OnInvalidFuncs, func(_ forms.Form) {
		delete(f.BaseForm.InvalidDefaults, "password")
		delete(f.BaseForm.InvalidDefaults, "passwordConfirm")
	})

	return forms.Initialize(f, formOpts...)
}

type BaseUserForm struct {
	*forms.BaseForm
	Request  *http.Request
	Instance *User
	canSave  bool
	config   *RegisterFormConfig
}

func (f *BaseUserForm) SetRequest(r *http.Request) {
	f.Request = r
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

func (f *BaseUserForm) Save() (*User, error) {
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
		f.Instance = attrs.NewObject[*User](&User{})
	}

	f.Instance = models.Setup(f.Instance)

	if f.config == nil {
		f.config = &RegisterFormConfig{}
	}

	if f.config.AlwaysAllLoginFields || Auth.LoginWithEmail {
		f.Instance.Email = cleaned["email"].(*drivers.Email)
	}

	if f.config.AlwaysAllLoginFields || !Auth.LoginWithEmail {
		f.Instance.Username = cleaned["username"].(string)
	}

	if f.config.AskForNames {
		f.Instance.FirstName = cleaned["firstName"].(string)
		f.Instance.LastName = cleaned["lastName"].(string)
	}

	var pw = cleaned["password"].(*Password)
	f.Instance.Password = pw

	f.Instance.IsActive = !f.config.IsInactive
	f.Instance.IsAdministrator = false

	err = f.Instance.Save(f.Request.Context())
	if err != nil {
		return nil, errors.Wrap(err, "Error saving user")
	}

	if f.config.AutoLogin {
		var user, err = Login(f.Request, f.Instance)
		if err != nil {
			return nil, errors.Wrap(err, "Error logging in user")
		}
		f.Instance = user
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
		userRow *queries.Row[*User]
		err     error
	)
	if Auth.LoginWithEmail {
		var m = cleaned["email"].(*mail.Address)
		userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
			Filter("Email", m.Address).
			Get()
	} else {
		userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
			Filter("Username", cleaned["username"]).
			Get()
	}
	if err != nil {
		if !errors.Is(err, errors.NoRows) {
			logger.Errorf("Error getting user: %v", err)
		}
		return autherrors.ErrGenericAuthFail
	}

	var user = userRow.Object
	if err := CheckPassword(user, string(cleaned["password"].(*Password).Raw)); err != nil {
		return autherrors.ErrGenericAuthFail
	}

	if !user.IsActive {
		return autherrors.ErrIsActive
	}

	if user, err = Login(f.Request, user); err != nil {
		return err
	}
	f.Instance = user
	return nil
}
