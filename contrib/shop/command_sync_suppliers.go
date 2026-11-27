package shop

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/orders"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/command"
)

var syncSuppliersCmd = &command.Cmd[any]{
	ID: "sync-suppliers",
	Execute: func(m command.Manager, stored any, args []string) error {
		ctx, tx, err := queries.StartTransaction(context.Background())
		if err != nil {
			return err
		}

		defer tx.Rollback(ctx)

		if err := orders.SuppliersQuery().WithContext(ctx).Sync(); err != nil {
			return errors.Wrap(err, "failed to sync suppliers")
		}

		return tx.Commit(ctx)
	},
}
