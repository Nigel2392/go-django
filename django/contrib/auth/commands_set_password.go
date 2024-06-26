package auth

import (
	"context"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/command"
	"github.com/Nigel2392/django/core/logger"
)

var command_set_password = &command.Cmd[interface{}]{
	ID:   "set_password",
	Desc: "Set password for a user by email or username",
	Execute: func(m command.Manager, stored interface{}, args []string) error {
		var (
			ctx                 = context.Background()
			isValid             = false
			uRow                *models.User
			identifier          string
			password, password2 string
			err                 error
		)

		for !isValid {
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

			var err error
			if Auth.LoginWithEmail {
				uRow, err = Auth.Queries.RetrieveByEmail(ctx, identifier)
			} else {
				uRow, err = Auth.Queries.RetrieveByUsername(ctx, identifier)
			}
			if err != nil {
				logger.Fatal(1, err)
			}

			if err = models.SetPassword(uRow, password); err != nil {
				logger.Warn(err)
				continue
			}
			isValid = true
		}

		return Auth.Queries.UpdateUser(
			ctx, uRow.Email.Address, uRow.Username, string(uRow.Password),
			uRow.FirstName, uRow.LastName, uRow.IsAdministrator, uRow.IsActive, uRow.ID,
		)

	},
}
