package openauth2

import (
	"context"
	"flag"

	"github.com/Nigel2392/go-django/src/core/command"
)

type changeUserStorage struct {
	super    bool
	inactive bool
}

var command_change_user = &command.Cmd[changeUserStorage]{
	ID:   "changeuser",
	Desc: "Change oauth2 user's administrator and active status by identifier and provider",
	FlagFunc: func(m command.Manager, stored *changeUserStorage, f *flag.FlagSet) error {
		f.BoolVar(&stored.super, "s", false, "Change to superuser")
		f.BoolVar(&stored.inactive, "i", false, "Change to inactive user")
		return nil
	},
	Execute: func(m command.Manager, stored changeUserStorage, args []string) error {
		var (
			qs  = App.Querier()
			ctx = context.Background()
		)

		identifier, err := m.Input("Enter user's primary identifier: ")
		if err != nil {
			return err
		}

		provider, err := m.Input("Enter provider (e.g. 'google', 'github'): ")
		if err != nil {
			return err
		}

		u, err := qs.RetrieveUserByIdentifier(
			ctx, identifier, provider,
		)
		if err != nil {
			return err
		}

		u.IsAdministrator = stored.super
		u.IsActive = !stored.inactive

		return qs.UpdateUser(
			ctx,
			u.ProviderName,
			u.Data,
			u.AccessToken,
			u.RefreshToken,
			u.TokenType,
			u.ExpiresAt,
			u.IsAdministrator,
			u.IsActive,
			u.ID,
		)
	},
}
