package auth

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
)

var command_set_password = &command.Cmd[interface{}]{
	ID:   "set_password",
	Desc: "Set password for a user by email or username",
	Execute: func(m command.Manager, stored interface{}, args []string) error {
		var (
			ctx                 = context.Background()
			user                *User
			identifier          string
			password, password2 string
			err                 error
		)

		for {
			if Auth.LoginWithEmail {
				if identifier, err = m.Input("Enter email: "); err != nil {
					continue
				}
			} else {
				if identifier, err = m.Input("Enter username: "); err != nil {
					continue
				}
			}

			if password, err = m.ProtectedInput("Enter password: \n"); err != nil {
				continue
			}
			if password2, err = m.ProtectedInput("Re-enter password: \n"); err != nil {
				continue
			}

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

			var userRow *queries.Row[*User]
			if Auth.LoginWithEmail {
				userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
					Filter("Email", identifier).
					Get()
			} else {
				userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
					Filter("Username", identifier).
					Get()
			}
			if err != nil {
				logger.Fatal(1, err)
			}

			user = userRow.Object
			user.SetPassword(password)
			break
		}

		return user.Save(ctx)
	},
}
