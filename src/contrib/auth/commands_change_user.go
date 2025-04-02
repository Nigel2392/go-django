package auth

import (
	"context"
	"flag"
	"net/mail"

	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type changeUserStorage struct {
	super    bool
	inactive bool
}

var command_change_user = &command.Cmd[changeUserStorage]{
	ID:   "changeuser",
	Desc: "Change user's administrator and active status by email or username",
	FlagFunc: func(m command.Manager, stored *changeUserStorage, f *flag.FlagSet) error {
		f.BoolVar(&stored.super, "s", false, "Change to superuser")
		f.BoolVar(&stored.inactive, "i", false, "Change to inactive user")
		return nil
	},
	Execute: func(m command.Manager, stored changeUserStorage, args []string) error {
		var (
			u          = &models.User{}
			retrieve   func(context.Context, string) (*models.User, error)
			identifier string
			err        error
		)

		if Auth.LoginWithEmail {
			retrieve = Auth.Queries.RetrieveByEmail
		} else {
			retrieve = Auth.Queries.RetrieveByUsername
		}

		for {
			if Auth.LoginWithEmail {
				if identifier, err = m.Input("Enter email: "); err != nil {
					continue
				}

				_, err = mail.ParseAddress(identifier)
				if err != nil {
					logger.Warn("invalid email address")
					identifier = ""
					continue
				}
			} else {
				if identifier, err = m.Input("Enter username: "); err != nil {
					continue
				}
			}

			u, err = retrieve(
				context.Background(),
				identifier,
			)

			if err != nil {
				logger.Warn("Error retrieving user: %v", err)
				identifier = ""
				continue
			}

			break
		}

		u.IsAdministrator = stored.super
		u.IsActive = !stored.inactive

		var ctx = context.Background()
		err = models.UpdateUser(ctx, u)
		return err
	},
}
