package openauth2

import (
	"context"
	"flag"

	queries "github.com/Nigel2392/go-django/queries/src"
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
		identifier, err := m.Input("Enter user's primary identifier: ")
		if err != nil {
			return err
		}

		provider, err := m.Input("Enter provider (e.g. 'google', 'github'): ")
		if err != nil {
			return err
		}

		var ctx = context.Background()
		return queries.RunInTransaction(ctx, func(ctx context.Context, NewQuerySet queries.ObjectsFunc[*User]) (commit bool, err error) {
			var qs = NewQuerySet(&User{})

			// Retrieve the user by identifier and provider
			u, err := qs.Filter(map[string]interface{}{
				"UniqueIdentifier": identifier,
				"ProviderName":     provider,
			}).Get()
			if err != nil {
				return false, err
			}

			// Update the user's administrator and active status
			u.Object.IsAdministrator = stored.super
			u.Object.IsActive = !stored.inactive

			// Save the changes to the user
			_, err = qs.Select("IsAdministrator", "IsActive").Filter("ID", u.Object.ID).ExplicitSave().Update(&User{
				IsAdministrator: u.Object.IsAdministrator,
				IsActive:        u.Object.IsActive,
			})

			return err == nil, err
		})
	},
}
