package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// ReconcileCouponRedemptions accounts for the coupons on a saved order.
// Completed orders consume their reservations. Rejected orders release pending
// reservations. Calling it again is safe. Call it in the same transaction as
// the order update to commit both changes together.
func ReconcileCouponRedemptions(ctx context.Context, orderID drivers.ULID) error {
	if !queries.IsCommitContext(ctx) {
		return errors.New("coupon redemption reconciliation requires a writable transaction")
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	orderQuery := Query().
		WithContext(ctx).
		Filter(
			expr.Q("ID", orderID),
		)
	// Lock the order before reading its state so a concurrent state change
	// cannot race with consumption or release of the same reservations.
	_, err = orderQuery.
		BulkUpdate(expr.As("Completed", expr.Field("Completed")))
	if err != nil {
		return err
	}

	order, err := orderQuery.
		Select("ID", "State", "Completed").
		Get()
	if err != nil {
		return err
	}

	pending := queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(ctx).
		Filter(
			expr.Q("Order", orderID),
			expr.Q("Invalidated", false),
			expr.Q("Counted", false),
		)
	switch {
	case order.Object.State == OrderStateRejected:
		_, err = pending.
			BulkUpdate(expr.As("Invalidated", expr.Value(true)))
		if err != nil {
			return err
		}

	case order.Object.Completed:
		rows, err := pending.
			Select("ID", "Coupon").
			OrderBy("Coupon", "ID").
			All()
		if err != nil {
			return err
		}
		for _, row := range rows {
			couponQuery := queries.
				GetQuerySet(&coupons.Coupon{}).
				WithContext(ctx).
				Filter(
					expr.Q("ID", row.Object.Coupon.ID),
				)
			// Reserve and count operations acquire the coupon row before
			// changing any redemption for that coupon.
			_, err := couponQuery.
				BulkUpdate(expr.As("UsageCount", expr.Field("UsageCount")))
			if err != nil {
				return err
			}

			count, err := pending.
				Filter(
					expr.Q("ID", row.Object.ID),
				).
				BulkUpdate(expr.As("Counted", expr.Value(true)))
			if err != nil {
				return err
			}
			if count == 0 {
				continue
			}

			count, err = couponQuery.
				BulkUpdate(expr.As("UsageCount", expr.Logical("UsageCount").ADD(1)))
			if err != nil {
				return err
			}
			if count != 1 {
				return fmt.Errorf("coupon %d was not found while counting redemption %d", row.Object.Coupon.ID, row.Object.ID)
			}
		}
	}

	return tx.Commit(ctx)
}
