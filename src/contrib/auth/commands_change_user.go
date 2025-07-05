package auth

import (
	"context"
	"flag"
	"net/mail"

	queries "github.com/Nigel2392/go-django/queries/src"
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
			u          = &User{}
			ctx        = context.Background()
			identifier string
			err        error
		)

		for {
			var userRow *queries.Row[*User]
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

				userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
					Filter("Email", identifier).
					Get()

			} else {
				if identifier, err = m.Input("Enter username: "); err != nil {
					continue
				}

				userRow, err = queries.GetQuerySetWithContext(ctx, &User{}).
					Filter("Username", identifier).
					Get()
			}

			if err != nil {
				logger.Warn("Error retrieving user: %v", err)
				identifier = ""
				continue
			}
			u = userRow.Object
			break
		}

		u.IsAdministrator = stored.super
		u.IsActive = !stored.inactive

		return u.Save(ctx)
	},
}
