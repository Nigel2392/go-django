package openauth2

import (
	"context"
	"flag"
	"time"

	"github.com/Nigel2392/go-django/src/core/command"
)

type createUserStorage struct {
	super    bool
	inactive bool
}

var command_create_user = &command.Cmd[createUserStorage]{
	ID:   "createuser",
	Desc: "Create a oauth2 user with the given identifier and provider",
	FlagFunc: func(m command.Manager, stored *createUserStorage, f *flag.FlagSet) error {
		f.BoolVar(&stored.super, "s", false, "Create a superuser")
		f.BoolVar(&stored.inactive, "i", false, "Create an inactive user")
		return nil
	},
	Execute: func(m command.Manager, stored createUserStorage, args []string) error {
		var (
			qs                   = App.Querier()
			ctx                  = context.Background()
			identifier, provider string
			err                  error
		)

		for {
			if identifier == "" {
				if identifier, err = m.Input("Enter user's primary identifier: "); err != nil {
					continue
				}
			}

			if provider, err = m.Input("Enter provider (e.g. 'google', 'github'): "); err != nil {
				continue
			}

			if _, ok := App._cnfs[provider]; !ok {
				m.Logf("Unknown provider '%s'", provider)
				continue
			}

			break
		}

		u, err := qs.CreateUser(
			ctx,
			identifier,
			provider,
			[]byte{},
			"",
			"",
			"",
			time.Time{},
			stored.super,
			!stored.inactive,
		)
		if err != nil {
			m.Logf("Error creating user: %s", err)
		}

		m.Logf("Successfully created user with ID: %d", u)

		return nil
	},
}
