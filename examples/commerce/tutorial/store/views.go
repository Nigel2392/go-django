package store

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/core/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/examples/commerce/tutorial/paymentproviders"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/v2"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
)

func checkoutPage(w http.ResponseWriter, r *http.Request) {
	notebooks, err := queries.GetQuerySet(&products.Product{}).WithContext(r.Context()).
		Filter("Supplier", notebookSupplier).OrderBy("ID").All()
	if err != nil {
		serverError(w, err)
		return
	}
	page := ctx.RequestContext(r)
	page.Set("Notebooks", notebooks)
	if err := tpl.FRender(w, page, "store", "store/checkout.tmpl"); err != nil {
		serverError(w, err)
	}
}

func checkout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid checkout form", http.StatusBadRequest)
		return
	}
	notebook, err := queries.GetQuerySet(&products.Product{}).WithContext(r.Context()).
		Filter("Supplier", notebookSupplier).Filter("Slug", r.PostForm.Get("notebook")).Get()
	if errors.Is(err, qerrors.NoRows) {
		http.Error(w, "Choose a notebook from the catalog", http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	pen, err := queries.GetQuerySet(&products.Product{}).WithContext(r.Context()).Filter("Slug", "pen").Get()
	if err != nil {
		serverError(w, err)
		return
	}
	session := sessions.Retrieve(r)
	if session.Token() == "" {
		if err := session.RenewToken(); err != nil {
			serverError(w, err)
			return
		}
	}
	cart, err := createCart(r.Context(), session.Token(), notebook.Object, pen.Object, r.PostForm.Get("email"))
	if errors.Is(err, carts.ErrInvalidCustomerEmail) {
		http.Error(w, "Enter a valid email address", http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	account, err := queries.GetQuerySet(&payments.ProviderAccount{}).WithContext(r.Context()).
		Filter("InternalName", paymentproviders.DemoDriver).Filter("Enabled", true).Get()
	if err != nil {
		serverError(w, err)
		return
	}
	// CheckoutCart opens the provider session before its order transaction.
	// The Order.Created listener applies the decision when the order is saved.
	result, err := orders.CheckoutCart(r, orders.CheckoutOptions{
		Cart: cart, AccountID: account.Object.ID, Currency: shopCurrency,
	})
	if err != nil {
		serverError(w, err)
		return
	}
	if result.Free {
		// The internal zero-total payment record emits no payment signals.
		if err := scheduleDeliveries(r.Context(), result.Payment); err != nil {
			serverError(w, err)
			return
		}
	}
	session.Set("store-order:"+result.Order.ID.String(), true)
	// The local demo payment action lives on the order page.
	http.Redirect(w, r, django.Reverse("store:order", result.Order.ID), http.StatusSeeOther)
}

// Save the cart and its buyer together before opening the payment session.
func createCart(ctx context.Context, session string, notebook, pen *products.Product, email string) (*carts.Cart, error) {
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	cart, err := carts.Query().WithContext(ctx).Create(&carts.Cart{Session: session, Active: true})
	if err != nil {
		return nil, err
	}
	if _, err := queries.GetQuerySet(&carts.CartItem{}).WithContext(ctx).BulkCreate([]*carts.CartItem{
		{Cart: cart, Product: notebook, Amount: 2},
		{Cart: cart, Product: pen, Amount: 1},
	}); err != nil {
		return nil, err
	}
	if _, err := carts.SetCustomer(ctx, cart, carts.CustomerDetails{Email: email}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return cart, nil
}

func orderPage(w http.ResponseWriter, r *http.Request) {
	order := orderForRequest(w, r)
	if order == nil {
		return
	}
	payment, err := queries.GetQuerySet(&payments.Payment{}).WithContext(r.Context()).Filter("Order", order.ID).Get()
	if err != nil {
		serverError(w, err)
		return
	}
	lines, err := queries.GetQuerySet(&orders.OrderItem{}).WithContext(r.Context()).
		Filter("Order", order.ID).OrderBy("ID").All()
	if err != nil {
		serverError(w, err)
		return
	}
	deliveryRows, err := queries.GetQuerySet(&Delivery{}).WithContext(r.Context()).Filter("Line.Order", order.ID).All()
	if err != nil {
		serverError(w, err)
		return
	}
	deliveries := make(map[drivers.ULID]*Delivery, len(deliveryRows))
	for _, row := range deliveryRows {
		deliveries[row.Object.Line.ID] = row.Object
	}
	stateLabel := order.State.Label(r.Context())
	switch order.State {
	case orders.OrderStateAccepted:
		stateLabel = "Accepted"
	case orders.OrderStatePending:
		stateLabel = "Pending"
	case orders.OrderStateRejected:
		stateLabel = "Denied"
	}

	page := ctx.RequestContext(r)
	page.Set("Order", order)
	page.Set("StateLabel", stateLabel)
	page.Set("Payment", payment.Object)
	page.Set("CanPay", order.State != orders.OrderStateRejected && payment.Object.State == paystate.STATUS_PENDING)
	page.Set("Lines", lines)
	page.Set("Deliveries", deliveries)
	if err := tpl.FRender(w, page, "store", "store/order.tmpl"); err != nil {
		serverError(w, err)
	}
}

func pay(w http.ResponseWriter, r *http.Request, provider *paymentproviders.Demo) {
	order := orderForRequest(w, r)
	if order == nil {
		return
	}
	if order.State == orders.OrderStateRejected {
		http.Error(w, "This order was denied", http.StatusConflict)
		return
	}
	payment, err := queries.GetQuerySet(&payments.Payment{}).WithContext(r.Context()).Filter("Order", order.ID).Get()
	if err != nil {
		serverError(w, err)
		return
	}
	attempt := payment.Object
	if attempt.State != paystate.STATUS_PAID && !attempt.IsFreeCheckout() {
		if attempt.Provider != paymentproviders.DemoDriver || attempt.State != paystate.STATUS_PENDING || !attempt.TransactionID.Valid {
			http.Error(w, "This payment cannot be simulated", http.StatusConflict)
			return
		}
		// Change the demo provider's state, then verify it through the normal PSP API.
		if err := provider.MarkPaid(attempt.TransactionID.String); err != nil {
			serverError(w, err)
			return
		}
		if _, err := payments.Refresh(r.Context(), attempt.ID); err != nil {
			serverError(w, err)
			return
		}
	}
	http.Redirect(w, r, django.Reverse("store:order", order.ID), http.StatusSeeOther)
}

func confirmDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(mux.Vars(r).Get("delivery"), 10, 64)
	if err != nil || id == 0 {
		http.NotFound(w, r)
		return
	}

	// Save the supplier receipt and run its signal in the same transaction.
	deliveryCtx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(deliveryCtx)
	order := orderForRequest(w, r.WithContext(deliveryCtx))
	if order == nil {
		return
	}
	if order.State == orders.OrderStateRejected {
		http.Error(w, "This order was denied", http.StatusConflict)
		return
	}
	job := queries.GetQuerySet(&Delivery{}).WithContext(deliveryCtx).Filter("ID", id)
	delivery, err := job.Filter("Line.Order", order.ID).Get()
	if errors.Is(err, qerrors.NoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if delivery.Object.Receipt == "" {
		// In this demo the form simulates a supplier callback with a stable receipt.
		delivery.Object.Receipt = "DELIVERED-" + delivery.Object.Line.ID.String()
		if _, err := job.Select("Receipt").Update(delivery.Object); err != nil {
			serverError(w, err)
			return
		}
	}
	if !order.Processed {
		if err := DeliveryConfirmed.Send(deliveryCtx, DeliveryConfirmedSignal{
			BaseSignal: signals.BaseSignal{Request: r.WithContext(deliveryCtx)},
			Order:      order,
		}); err != nil {
			serverError(w, err)
			return
		}
	}
	if err := tx.Commit(deliveryCtx); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, django.Reverse("store:order", order.ID), http.StatusSeeOther)
}

// Use the same session ownership check for the order page and both actions.
func orderForRequest(w http.ResponseWriter, r *http.Request) *orders.Order {
	var id drivers.ULID
	if err := id.UnmarshalText([]byte(mux.Vars(r).Get("id"))); err != nil || id.IsZero() ||
		!sessions.Retrieve(r).Exists("store-order:"+id.String()) {
		http.NotFound(w, r)
		return nil
	}
	row, err := orders.Query().WithContext(r.Context()).Filter("ID", id).Get()
	if errors.Is(err, qerrors.NoRows) {
		http.NotFound(w, r)
		return nil
	}
	if err != nil {
		serverError(w, err)
		return nil
	}
	w.Header().Set("Cache-Control", "no-store")
	return row.Object
}

func serverError(w http.ResponseWriter, err error) {
	logger.Error(err)
	http.Error(w, "Could not update or load checkout", http.StatusInternalServerError)
}
