package settings

import (
	"context"

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

		notExists := make([]*RegisteredSetting, 0)
		for group := range groups {
			for _, setting := range group.Settings {
				if setting.instance.ID == 0 {
					notExists = append(notExists, setting.instance)
				}
			}
		}

		if len(notExists) == 0 {
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
