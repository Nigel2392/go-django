package auth

import (
	"context"
	"flag"
	"net/mail"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
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
			u                   = &User{}
			isValid             = false
			email, username     string
			password, password2 string
			pwProvided          bool
			err                 error
		)

		switch {
		case len(args) == 1:
			email = args[0]
		case len(args) == 2:
			email = args[0]
			username = args[1]
		case len(args) == 3:
			email = args[0]
			username = args[1]
			password = args[2]
			password2 = args[2]
			pwProvided = true
		case len(args) == 0:
		default:
			return errors.ValueError.Wrap("too many arguments provided")
		}

		for !isValid {
			if email == "" {
				if email, err = m.Input("Enter email: "); err != nil {
					continue
				}
			}

			if username == "" {
				if username, err = m.Input("Enter username: "); err != nil {
					continue
				}
			}

			if !pwProvided {
				if password, err = m.ProtectedInput("Enter password: "); err != nil {
					continue
				}
				if password2, err = m.ProtectedInput("Re-enter password: "); err != nil {
					continue
				}
			}

			var e, err = mail.ParseAddress(email)
			if err != nil {
				logger.Warn("invalid email address")
				email = ""
			}

			u.Email = (*drivers.Email)(e)
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

			u.SetPassword(password)
			isValid = true
		}

		u.IsAdministrator = stored.super
		u.IsActive = !stored.inactive

		var ctx = context.Background()
		return models.Setup(u).Create(ctx)
	},
}
