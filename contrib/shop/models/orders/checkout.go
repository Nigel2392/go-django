package orders

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var (
	ErrCheckoutEmpty              = errors.New("checkout cart is empty")
	ErrCustomerRequired           = errors.New("a customer email address is required")
	ErrCheckoutAccountRequired    = errors.New("payment account is required")
	ErrCheckoutAccountUnavailable = errors.New("payment account is unavailable")
	ErrCheckoutConflict           = errors.New("cart changed or already has a checkout")
)

// CheckoutPricer calculates a custom server-side subtotal before the payment
// provider account adds its fee.
type CheckoutPricer func(ctx context.Context, cart *carts.CartRow) (price.Money, error)

func (fn CheckoutPricer) CheckoutSubtotal(ctx context.Context, cart *carts.CartRow) (price.Money, error) {
	return fn(ctx, cart)
}

type CheckoutOptions struct {
	Cart        *carts.Cart
	AccountID   drivers.ULID
	Currency    currency.Unit
	GetCurrency func(context.Context) currency.Unit
	ReturnURL   func(paymentID, orderID drivers.ULID) string
	WebhookURL  func(account *payments.ProviderAccount) string
	Pricer      CheckoutPricer
}

func (opts CheckoutOptions) resolveCurrency(ctx context.Context) currency.Unit {
	if opts.GetCurrency != nil {
		if unit := opts.GetCurrency(ctx); unit != (currency.Unit{}) {
			return unit
		}
	}
	return opts.Currency
}

type CheckoutResult struct {
	Order    *Order
	Items    []*OrderItem
	Payment  *payments.Payment
	Checkout payments.Checkout
	Free     bool
}

// CheckoutCart prices an active cart, opens provider checkout when payment is due,
// and converts the cart into an order and payment record only after the PSP accepts
// the checkout request.
func CheckoutCart(r *http.Request, opts CheckoutOptions) (*CheckoutResult, error) {
	if r == nil {
		return nil, errors.New("checkout requires a request")
	}
	ctx := r.Context()
	if !queries.IsCommitContext(ctx) {
		return nil, errors.New("checkout requires a writable commit context")
	}

	var cartQuery *carts.CartQuery
	if opts.Cart != nil {
		if opts.Cart.ID.IsZero() {
			return nil, dberrors.NoRows.Wrap("cart not found")
		}
		cartQuery = carts.Query().WithContext(ctx).Filter("ID", opts.Cart.ID)
	} else {
		cartQuery = carts.Query().WithContext(ctx).CartForRequest(r)
	}

	pricedCart, err := cartQuery.CalcOneWithProducts()
	if err != nil {
		return nil, err
	}
	if pricedCart == nil || pricedCart.Object == nil || !pricedCart.Object.Active {
		return nil, ErrCheckoutConflict
	}
	if len(pricedCart.Items) == 0 {
		return nil, ErrCheckoutEmpty
	}

	customer, pendingCustomer, authUser, err := resolveCheckoutCustomer(ctx, r, pricedCart.Object)
	if err != nil {
		return nil, err
	}

	unit := opts.resolveCurrency(ctx)
	subtotal := price.Money{
		Amount:   pricedCart.Totals.Payable(),
		Currency: unit,
	}
	if opts.Pricer != nil {
		subtotal, err = opts.Pricer(ctx, pricedCart)
		if err != nil {
			return nil, fmt.Errorf("price checkout: %w", err)
		}
		if subtotal.Currency == (currency.Unit{}) {
			subtotal.Currency = unit
		}
	}
	if subtotal.Amount.IsNegative() {
		return nil, payments.ErrNonPositiveTotal
	}

	zeroQuote, err := price.QuoteFee(subtotal, decimal.Zero, decimal.Zero)
	if err != nil {
		return nil, fmt.Errorf("price checkout: %w", err)
	}

	paymentID := drivers.NewULID()
	if zeroQuote.Total.IsZero() {
		return finalizeCheckout(
			r,
			pricedCart,
			customer,
			pendingCustomer,
			authUser,
			func(txCtx context.Context, order *Order) (*payments.Payment, error) {
				return payments.CreateFreePayment(txCtx, order, subtotal.Currency, paymentID)
			},
			payments.Checkout{},
			true,
		)
	}

	if queries.IsTransactionContext(ctx) {
		return nil, errors.New("provider checkout requires a context outside an existing transaction")
	}

	account, err := loadCheckoutAccount(ctx, opts.AccountID, subtotal.Currency)
	if err != nil {
		return nil, err
	}

	var returnURL string
	if opts.ReturnURL != nil {
		returnURL = opts.ReturnURL(paymentID, drivers.ULID{})
	}
	var webhookURL string
	if opts.WebhookURL != nil {
		webhookURL = opts.WebhookURL(account)
	}

	quote, checkout, err := payments.OpenCheckout(ctx, account, paymentID, subtotal, returnURL, webhookURL)
	if err != nil {
		return nil, err
	}

	return finalizeCheckout(
		r,
		pricedCart,
		customer,
		pendingCustomer,
		authUser,
		func(txCtx context.Context, order *Order) (*payments.Payment, error) {
			return payments.RecordPayment(txCtx, order, account, paymentID, quote, checkout)
		},
		checkout,
		false,
	)
}

func resolveCheckoutCustomer(ctx context.Context, r *http.Request, cart *carts.Cart) (*customers.Customer, *customers.Customer, users.User, error) {
	var authUser users.User
	if u, ok := authentication.Retrieve(r).(users.User); ok && u != nil && !attrs.IsZero(u) && u.IsAuthenticated() {
		authUser = u
	}

	if cart != nil && cart.Customer != nil && !cart.Customer.ID.IsZero() {
		customer := cart.Customer
		if customer.Email == nil || strings.TrimSpace(customer.Email.Address) == "" {
			row, err := queries.GetQuerySet(&customers.Customer{}).
				WithContext(ctx).
				Filter("ID", customer.ID).
				Get()
			if err != nil {
				if errors.Is(err, dberrors.NoRows) {
					return nil, nil, authUser, ErrCustomerRequired
				}
				return nil, nil, authUser, err
			}
			customer = row.Object
		}
		if customer.Email == nil || strings.TrimSpace(customer.Email.Address) == "" {
			return nil, nil, authUser, ErrCustomerRequired
		}
		return customer, nil, authUser, nil
	}

	if authUser == nil {
		return nil, nil, nil, ErrCustomerRequired
	}

	existing, err := queries.
		GetQuerySet(&customers.Customer{}).
		WithContext(ctx).
		Filter(expr.Q("User", authUser)).
		OrderBy("FirstSeen", "ID").
		First()
	switch {
	case err == nil:
		if existing.Object.Email == nil || strings.TrimSpace(existing.Object.Email.Address) == "" {
			return nil, nil, authUser, ErrCustomerRequired
		}
		return existing.Object, nil, authUser, nil

	case errors.Is(err, dberrors.NoRows):
		defs := attrs.Define(ctx, authUser)
		values := make(map[string]string, 3)
		for _, name := range []string{"Email", "FirstName", "LastName"} {
			field, ok := defs.Field(name)
			if !ok {
				continue
			}
			data := field.GetValue()
			if !attrs.IsZero(data) {
				values[name] = strings.TrimSpace(attrs.ToString(data))
			}
		}
		address, err := mail.ParseAddress(values["Email"])
		if err != nil || address == nil || strings.TrimSpace(address.Address) == "" {
			return nil, nil, authUser, ErrCustomerRequired
		}
		pending := &customers.Customer{
			User:      authUser,
			Email:     (*drivers.Email)(address),
			FirstName: values["FirstName"],
			LastName:  values["LastName"],
		}
		return nil, pending, authUser, nil

	default:
		return nil, nil, authUser, err
	}
}

func loadCheckoutAccount(ctx context.Context, accountID drivers.ULID, unit currency.Unit) (*payments.ProviderAccount, error) {
	if accountID.IsZero() {
		return nil, ErrCheckoutAccountRequired
	}
	row, err := queries.
		GetQuerySet(&payments.ProviderAccount{}).
		WithContext(ctx).
		Filter(
			expr.Q("ID", accountID),
			expr.Q("Enabled", true),
		).
		Get()
	switch {
	case errors.Is(err, dberrors.NoRows):
		return nil, errors.Join(ErrCheckoutAccountUnavailable, err)
	case err != nil:
		return nil, err
	}

	account := row.Object
	backend, ok := payments.RetrieveProvider(account.InternalName)
	if !ok || backend == nil {
		return nil, ErrCheckoutAccountUnavailable
	}
	if backendCurrency := backend.Currency(ctx); backendCurrency == (currency.Unit{}) || backendCurrency != unit {
		return nil, ErrCheckoutAccountUnavailable
	}
	return account, nil
}

func finalizeCheckout(
	r *http.Request,
	pricedCart *carts.CartRow,
	customer *customers.Customer,
	pendingCustomer *customers.Customer,
	authUser users.User,
	createPayment func(txCtx context.Context, order *Order) (*payments.Payment, error),
	checkout payments.Checkout,
	free bool,
) (*CheckoutResult, error) {
	txCtx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(txCtx)

	if customer == nil {
		if pendingCustomer == nil {
			return nil, ErrCustomerRequired
		}
		customer, err = queries.
			GetQuerySet(&customers.Customer{}).
			WithContext(txCtx).
			Create(pendingCustomer)
		if err != nil {
			return nil, err
		}
	}

	cart := pricedCart.Object
	cart.Active = false
	selectFields := []any{"Active"}
	if cart.Customer == nil && customer != nil {
		cart.Customer = customer
		selectFields = append(selectFields, "Customer")
	}
	if authUser != nil && attrs.IsZero(cart.User) {
		cart.User = authUser
		selectFields = append(selectFields, "User")
	}

	count, err := queries.
		GetQuerySet(&carts.Cart{}).
		WithContext(txCtx).
		Filter(
			expr.Q("ID", cart.ID),
			expr.Q("Active", true),
		).
		Select(selectFields...).
		Update(cart)
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, ErrCheckoutConflict
	}

	order, items, err := CreateOrderFromCart(r.WithContext(txCtx), customer, pricedCart)
	if errors.Is(err, ErrCartRedemptionsChanged) {
		return nil, ErrCheckoutConflict
	}
	if err != nil {
		return nil, err
	}

	_, err = queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(txCtx).
		Filter(
			expr.Q("Cart", cart.ID),
			expr.Q("Invalidated", false),
			expr.Q("Order__isnull", true),
		).
		Select("Invalidated").
		Update(&coupons.Redemption{
			Invalidated: true,
		})
	if err != nil && !errors.Is(err, dberrors.NoRows) {
		return nil, err
	}

	payment, err := createPayment(txCtx, order)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Order:    order,
		Items:    items,
		Payment:  payment,
		Checkout: checkout,
		Free:     free,
	}, nil
}
