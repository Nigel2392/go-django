package settings

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/command"
)

var SyncSettingsCmd = command.Cmd[any]{
	ID: "sync-settings",
	Execute: func(m command.Manager, stored any, args []string) error {
		ctx, tx, err := queries.StartTransaction(context.Background())
		if err != nil {
			return err
		}

		defer tx.Rollback(ctx)

		groups, err := groupedSettings(ctx)
		if err != nil {
			return err
		}

		perms := make([]*users.Permission, 0)
		seenPerms := make(map[string]struct{})
		notExists := make([]*RegisteredSetting, 0)
		for group := range groups {
			for _, setting := range group.Settings {
				if setting.instance.ID == 0 {
					notExists = append(notExists, setting.instance)
				}

				for _, p := range setting.Permissions() {
					if _, ok := seenPerms[p]; ok {
						continue
					}

					seenPerms[p] = struct{}{}
					perms = append(perms, &users.Permission{
						Name:        p,
						Description: fmt.Sprintf("Permission for setting %q", setting.Name()),
					})
				}
			}
		}

		perms = append(perms,
			&users.Permission{
				Name:        "settings:view",
				Description: "Allow viewing of settings module",
			},
			&users.Permission{
				Name:        "settings:edit",
				Description: "Allow making changes within settings module",
			},
		)

		createdAny, err := users.CreatePermissions(ctx, perms...)
		if err != nil {
			return err
		}

		if len(notExists) == 0 {
			if createdAny {
				return tx.Commit(ctx)
			}
			return nil
		}

		_, err = queries.
			GetQuerySet(&RegisteredSetting{}).
			WithContext(ctx).
			BulkCreate(notExists)
		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	},
}
