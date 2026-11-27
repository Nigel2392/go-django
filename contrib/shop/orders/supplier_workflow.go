package orders

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	queries "github.com/Nigel2392/go-django/queries/src"
	qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

var (
	// ErrOrderRequired is returned when an order completion signal has no order or zero ID.
	ErrOrderRequired = stderrors.New("order is required")

	// ErrOrderRejected is returned when attempting to fulfill an already rejected order.
	ErrOrderRejected = stderrors.New("order is rejected")

	// ErrSupplierNotFound is returned when an order item references an empty or unregistered supplier.
	ErrSupplierNotFound = stderrors.New("order supplier not found")

	// ErrSupplierDisabled is returned when a supplier row exists in the database and is disabled.
	ErrSupplierDisabled = stderrors.New("order supplier is disabled")

	// ErrInvalidSupplierData is returned when supplier order data fails OrderBlock binding or validation.
	ErrInvalidSupplierData = stderrors.New("invalid supplier order data")
)

func init() {
	if _, err := Signals.Completed.Listen(context.Background(), handleOrderCompleted); err != nil {
		panic(fmt.Errorf("orders: register Completed signal listener: %w", err))
	}
}

func handleOrderCompleted(ctx context.Context, _ signals.Signal[OrderCompletedSignal], sig OrderCompletedSignal) error {
	if sig.Order == nil || sig.Order.ID.IsZero() {
		return ErrOrderRequired
	}
	if !queries.IsCommitContext(ctx) {
		return stderrors.New("order completion workflow requires a writable transaction")
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	orderQuery := Query().
		WithContext(ctx).
		Filter(expr.Q("ID", sig.Order.ID))

	// Lock the order row before inspecting state or invoking suppliers.
	count, err := orderQuery.BulkUpdate(expr.As("Completed", expr.Field("Completed")))
	if err != nil {
		return err
	}
	if count == 0 {
		return qerrors.NoRows.Wrapf("order %s not found", sig.Order.ID)
	}

	orderRow, err := orderQuery.
		Select("*", "Customer.*").
		Get()
	if err != nil {
		return err
	}
	order := orderRow.Object

	if order.State == OrderStateRejected {
		return fmt.Errorf("%w: %s", ErrOrderRejected, order.ID)
	}
	if order.Processed {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		syncOrderSignalState(sig.Order, order)
		return nil
	}

	itemRows, err := queries.
		GetQuerySet(&OrderItem{}).
		WithContext(ctx).
		Filter(expr.Q("Order", order.ID)).
		OrderBy("ID").
		All()
	if err != nil {
		return err
	}

	order.State = OrderStateWorkflow
	order.Completed = true
	if _, err := queries.GetQuerySet(&Order{}).
		WithContext(ctx).
		Select("State", "Completed").
		Filter(expr.Q("ID", order.ID)).
		Update(order); err != nil {
		return err
	}

	items := make([]*OrderItem, len(itemRows))
	for i, row := range itemRows {
		item := row.Object
		item.Order = order
		items[i] = item
		if err := fulfillOrderItem(ctx, order, item); err != nil {
			return err
		}
	}

	order.State = OrderStateAccepted
	order.Completed = true
	order.Processed = true
	if _, err := queries.GetQuerySet(&Order{}).
		WithContext(ctx).
		Select("State", "Completed", "Processed").
		Filter(expr.Q("ID", order.ID)).
		Update(order); err != nil {
		return err
	}

	if err := ReconcileCouponRedemptions(ctx, order.ID); err != nil {
		return err
	}

	base := sig.BaseSignal
	if base.Request != nil {
		base.Request = base.Request.WithContext(ctx)
	}

	if err := Signals.Accepted.Send(ctx, OrderAcceptedSignal{
		BaseSignal: base,
		Order:      order,
		Lines:      items,
	}); err != nil {
		return err
	}

	if err := Signals.Processed.Send(ctx, OrderProcessedSignal{
		BaseSignal: base,
		Order:      order,
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	syncOrderSignalState(sig.Order, order)
	return nil
}

func syncOrderSignalState(dst, src *Order) {
	if dst == nil || src == nil {
		return
	}
	dst.State = src.State
	dst.Completed = src.Completed
	dst.Processed = src.Processed
	if dst.Customer == nil {
		dst.Customer = src.Customer
	}
}

func resolveOrderItemSupplier(item *OrderItem) products.SupplierValue {
	if item.Supplier != "" {
		return item.Supplier
	}
	if item.Product.Data.Product != nil {
		return item.Product.Data.Product.Supplier
	}
	return ""
}

func ensureSupplierEnabled(ctx context.Context, supplierName products.SupplierValue) error {
	row, err := SuppliersQuery().
		WithContext(ctx).
		Filter(expr.Q("InternalName", string(supplierName))).
		Get()
	if err != nil {
		if stderrors.Is(err, qerrors.NoRows) {
			return nil
		}
		return err
	}
	if !row.Object.Enabled {
		return fmt.Errorf("%w: %q", ErrSupplierDisabled, supplierName)
	}
	return nil
}

func fulfillOrderItem(ctx context.Context, order *Order, item *OrderItem) error {
	supplierName := resolveOrderItemSupplier(item)
	if supplierName == "" {
		return fmt.Errorf("%w: order item %s has no supplier", ErrSupplierNotFound, item.ID)
	}
	item.Supplier = supplierName

	backend, ok := RetrieveSupplier(supplierName)
	if !ok {
		return fmt.Errorf("%w: %q", ErrSupplierNotFound, supplierName)
	}

	if err := ensureSupplierEnabled(ctx, supplierName); err != nil {
		return err
	}

	if item.SupplierData.HasData() {
		return nil
	}

	data, err := backend.OrderAccepted(ctx, order, item)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	block, err := backend.OrderBlock(ctx, order, item)
	if err != nil {
		return err
	}
	if block != nil {
		if data.Block() == nil {
			if err := data.BindToBlock(block); err != nil {
				return fmt.Errorf("%w: %w", ErrInvalidSupplierData, err)
			}
		}
		if errs := block.Validate(ctx, unwrapStructBlockValues(data)); len(errs) > 0 {
			return fmt.Errorf("%w: %w", ErrInvalidSupplierData, stderrors.Join(errs...))
		}
	}

	item.SupplierData = data
	_, err = queries.GetQuerySet(&OrderItem{}).
		WithContext(ctx).
		Select("SupplierData").
		Filter(expr.Q("ID", item.ID)).
		Update(item)
	return err
}

func unwrapStructBlockValues(data *blocks.StructBlockValue) *blocks.StructBlockValue {
	if data == nil || data.V == nil {
		return data
	}
	unwrapped := make(map[string]any, len(data.V))
	for k, v := range data.V {
		if bound, ok := v.(blocks.BoundBlockValue); ok {
			if nested, ok := bound.(*blocks.StructBlockValue); ok {
				unwrapped[k] = unwrapStructBlockValues(nested)
			} else {
				unwrapped[k] = bound.Data()
			}
		} else {
			unwrapped[k] = v
		}
	}
	return blocks.NewBlockValue(data.Block(), unwrapped)
}
