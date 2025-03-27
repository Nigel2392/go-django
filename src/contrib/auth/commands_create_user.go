package auth

import (
	"context"
	"flag"
	"net/mail"

	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
	django_models "github.com/Nigel2392/go-django/src/models"
)

type createUserStorage struct {
	super    bool
	inactive bool
}

var command_create_user = &command.Cmd[createUserStorage]{
	ID:   "createuser",
	Desc: "Create a user with the given email, username, and password",
	FlagFunc: func(m command.Manager, stored *createUserStorage, f *flag.FlagSet) error {
		f.BoolVar(&stored.super, "s", false, "Create a superuser")
		f.BoolVar(&stored.inactive, "i", false, "Create an inactive user")
		return nil
	},
	Execute: func(m command.Manager, stored createUserStorage, args []string) error {
		var (
			u                   = &models.User{}
			isValid             = false
			email, username     string
			password, password2 string
			err                 error
		)

		for !isValid {
			if email, err = m.Input("Enter email: "); err != nil {
				continue
			}
			if username, err = m.Input("Enter username: "); err != nil {
				continue
			}
			if password, err = m.ProtectedInput("Enter password: "); err != nil {
				continue
			}
			if password2, err = m.ProtectedInput("Re-enter password: "); err != nil {
				continue
			}

			var e, err = mail.ParseAddress(email)
			assert.Err(err)

			u.Email = (*django_models.Email)(e)
			u.Username = username

			if password != password2 {
				logger.Warn("passwords do not match")
				continue
			}

			var validator = &PasswordCharValidator{
				Flags: ChrFlagAll,
			}

			if err = validator.Validate(password); err != nil {
				logger.Warn(err)
				continue
			}

			if err = SetPassword(u, password); err != nil {
				logger.Warn(err)
				continue
			}
			isValid = true
		}

		u.IsAdministrator = stored.super
		u.IsActive = !stored.inactive

		var ctx = context.Background()
		_, err = models.CreateUser(ctx, u)
		return err
	},
}
