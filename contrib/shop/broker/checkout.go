// Package broker coordinates order and payment workflows for the shop.
package broker

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

var (
	ErrCheckoutConflict = errors.New("checkout attempt ID is already in use")
	ErrCheckoutEmpty    = errors.New("checkout cart is empty")
)

// CheckoutPricer calculates the server-side amount before the payment account
// adds its fee. The cart contains the calculated totals and complete products.
type CheckoutPricer interface {
	CheckoutSubtotal(context.Context, *carts.CartRow) (payments.Money, error)
}

// Broker uses the cart's payable amount in Currency by default. Applications
// can provide a Pricer for rules such as shipping or custom discounts.
type Broker struct {
	Currency string
	Pricer   CheckoutPricer
}

// CheckoutRequest contains inputs selected and authorized by the application.
// AttemptID is issued by the application once per checkout and reused on retry.
type CheckoutRequest struct {
	Customer  *customers.Customer
	CartID    drivers.ULID
	AccountID drivers.ULID
	AttemptID drivers.ULID
}

type PreparedCheckout struct {
	Order   *orders.Order
	Payment *payments.Payment
}

// PrepareCheckout snapshots an active cart and saves the order and its first
// payment attempt together. The same attempt ID resumes a saved checkout for
// the same customer and account, even when the cart has since changed.
//
// If the request already has a transaction, the caller owns its commit or
// rollback. A concurrent preparation error can be retried with the same ID
// after rolling back.
func (b Broker) PrepareCheckout(r *http.Request, request CheckoutRequest) (*PreparedCheckout, error) {
	if r == nil || request.Customer == nil || request.Customer.ID.IsZero() ||
		request.CartID.IsZero() || request.AccountID.IsZero() || request.AttemptID.IsZero() {
		return nil, errors.New("checkout requires a request, customer, cart, payment account, and attempt ID")
	}
	if !queries.IsCommitContext(r.Context()) {
		return nil, errors.New("checkout preparation requires a writable transaction")
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	prepared, err := loadCheckout(ctx, request.Customer.ID, request.AttemptID)
	if err == nil {
		if prepared.Payment.Account == nil || prepared.Payment.Account.ID != request.AccountID {
			return nil, ErrCheckoutConflict
		}
		return prepared, tx.Commit(ctx)
	}
	if !errors.Is(err, dberrors.NoRows) {
		return nil, err
	}

	cart, err := carts.Query().WithContext(ctx).
		Filter("ID", request.CartID).
		Filter("Active", true).
		CalcOneWithProducts()
	if err != nil {
		return nil, fmt.Errorf("calculate checkout cart: %w", err)
	}
	if len(cart.Items) == 0 {
		return nil, ErrCheckoutEmpty
	}

	subtotal := payments.Money{Amount: cart.Totals.Payable(), Currency: b.Currency}
	if b.Pricer != nil {
		subtotal, err = b.Pricer.CheckoutSubtotal(ctx, cart)
		if err != nil {
			return nil, fmt.Errorf("price checkout: %w", err)
		}
	}

	items := make([]*carts.CartItem, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = item.CartItem
	}
	order, _, err := orders.CreateOrder(r.WithContext(ctx), request.Customer, cart.Object, items...)
	if err != nil {
		return nil, err
	}

	attempt, err := payments.CreateAttemptWithID(ctx, order, request.AccountID, subtotal, request.AttemptID)
	if err != nil {
		return nil, fmt.Errorf("create checkout payment attempt: %w", err)
	}
	if !payments.ProviderRegistered(attempt.Provider) {
		return nil, fmt.Errorf("payment provider %q is unavailable", attempt.Provider)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &PreparedCheckout{Order: order, Payment: attempt}, nil
}

// StartCheckout opens provider checkout for a committed attempt owned by the
// customer. Applications must supply the same trusted URLs when retrying.
func (b Broker) StartCheckout(ctx context.Context, customerID, attemptID drivers.ULID, returnURL, webhookURL string) (payments.Checkout, error) {
	if queries.IsTransactionContext(ctx) || !queries.IsCommitContext(ctx) {
		return payments.Checkout{}, errors.New("provider checkout requires a committed payment attempt")
	}
	prepared, err := loadCheckout(ctx, customerID, attemptID)
	if err != nil {
		if errors.Is(err, ErrCheckoutConflict) {
			err = dberrors.NoRows.Wrap("checkout attempt not found")
		}
		return payments.Checkout{}, err
	}
	return payments.Start(ctx, prepared.Payment.ID, returnURL, webhookURL)
}

func loadCheckout(ctx context.Context, customerID, attemptID drivers.ULID) (*PreparedCheckout, error) {
	row, err := queries.GetQuerySet(&payments.Payment{}).WithContext(ctx).
		Filter("ID", attemptID).
		Select("*", "Order.*").
		Get()
	if err != nil {
		return nil, err
	}
	payment := row.Object
	order, ok := payment.Order.(*orders.Order)
	if !ok || order == nil || order.Customer == nil || order.Customer.ID != customerID {
		return nil, ErrCheckoutConflict
	}
	return &PreparedCheckout{Order: order, Payment: payment}, nil
}
