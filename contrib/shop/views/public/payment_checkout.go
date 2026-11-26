package public

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/broker"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"golang.org/x/text/currency"
)

// CheckoutAPI converts the caller's cart into an order and opens provider checkout.
// Origin is the public shop origin used for verified callback destinations.
// ReturnPath is a path on Origin and defaults to "/".
type CheckoutAPI struct {
	Currency   string
	Origin     string
	ReturnPath string
}

type PaymentCheckoutRequest struct {
	AccountID string `json:"account_id"`
}

func (c CheckoutAPI) Checkout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		api.RespondError(w, http.StatusMethodNotAllowed, "checkout requires POST")
		return
	}

	user, ok := authentication.Retrieve(r).(users.User)
	if !ok || attrs.IsZero(user) || !user.IsAuthenticated() {
		api.RespondError(w, http.StatusUnauthorized, "sign in to check out")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var request PaymentCheckoutRequest
	var contentType = r.Header.Get("Content-Type")
	var mediaType, _, err = mime.ParseMediaType(contentType)
	switch {
	case err != nil && contentType != "":
		api.RespondError(w, http.StatusUnsupportedMediaType, "submit a valid payment account ID")
		return

	case mediaType == "application/x-www-form-urlencoded":
		err = r.ParseForm()
		request.AccountID = r.PostForm.Get("account_id")

	case mediaType == "application/json",
		mediaType == "":
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&request)
		if err == nil {
			var extra json.RawMessage
			switch err = decoder.Decode(&extra); err {
			case io.EOF:
				err = nil
			case nil:
				err = errors.New("checkout accepts a single JSON object")
			}
		}

	default:
		api.RespondError(w, http.StatusUnsupportedMediaType, "submit a valid payment account ID")
		return
	}

	var tooLarge *http.MaxBytesError
	var accountID drivers.ULID
	switch {
	case errors.As(err, &tooLarge):
		api.RespondError(w, http.StatusRequestEntityTooLarge, "submit a valid payment account ID")
		return

	case err != nil,
		accountID.UnmarshalText([]byte(request.AccountID)) != nil,
		accountID.IsZero():
		api.RespondError(w, http.StatusBadRequest, "submit a valid payment account ID")
		return
	}

	unit, currencyErr := currency.ParseISO(c.Currency)
	origin, originErr := url.Parse(c.Origin)
	returnPath := c.ReturnPath
	if returnPath == "" {
		returnPath = "/"
	}

	returnRef, pathErr := url.Parse(returnPath)
	var configError string
	switch {
	case currencyErr != nil,
		unit.String() != c.Currency:
		configError = "invalid shop currency"

	case originErr != nil,
		origin.Scheme != "https" && origin.Scheme != "http",
		origin.Hostname() == "",
		origin.User != nil,
		origin.RawQuery != "",
		origin.Fragment != "",
		origin.Path != "" && origin.Path != "/":
		configError = "invalid checkout origin"

	case pathErr != nil,
		returnRef.IsAbs(),
		returnRef.Host != "",
		returnRef.User != nil,
		!strings.HasPrefix(returnRef.Path, "/"),
		strings.HasPrefix(returnRef.Path, "//"):
		configError = "invalid checkout return path"
	}
	if configError != "" {
		logger.Warnf("shop checkout configuration: %s", configError)
		api.RespondError(w, http.StatusServiceUnavailable, "shop checkout is unavailable")
		return
	}

	returnURL := origin.ResolveReference(returnRef)
	webhookURL := origin.ResolveReference(&url.URL{
		Path: django.Reverse("api:payments:webhook", accountID),
	})

	switch {
	case queries.IsTransactionContext(r.Context()),
		!queries.IsCommitContext(r.Context()):
		logger.Warnf("checkout preparation failed: the checkout endpoint must commit its own transaction")
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	defer tx.Rollback(ctx)

	account, err := queries.
		GetQuerySet(&payments.ProviderAccount{}).
		WithContext(ctx).
		Filter(
			expr.Q("ID", accountID),
			expr.Q("Enabled", true),
			expr.Q("Currency", c.Currency),
		).
		Get()

	switch {
	case errors.Is(err, dberrors.NoRows):
		api.RespondError(w, http.StatusNotFound, "payment account is unavailable")
		return

	case err != nil:
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return

	case !payments.ProviderRegistered(account.Object.Driver):
		api.RespondError(w, http.StatusNotFound, "payment account is unavailable")
		return
	}

	customersQuery := queries.
		GetQuerySet(&customers.Customer{}).
		WithContext(ctx).
		Filter(
			expr.Q("User", user),
		)

	existing, err := customersQuery.
		OrderBy("FirstSeen", "ID").
		First()

	var customer *customers.Customer
	switch {
	case err == nil:
		customer = existing.Object
		switch {
		case customer.Email == nil,
			customer.Email.Address == "":
			api.RespondError(w, http.StatusUnprocessableEntity, "a customer email address is required")
			return
		}

	case errors.Is(err, dberrors.NoRows):
		defs := attrs.Define(ctx, user)
		values := make(map[string]string, 3)
		names := []string{
			"Email",
			"FirstName",
			"LastName",
		}

		for _, name := range names {
			field, ok := defs.Field(name)
			if !ok {
				continue
			}
			data := field.GetValue()
			if !attrs.IsZero(data) {
				values[name] = attrs.ToString(data)
			}
		}

		address, err := mail.ParseAddress(values["Email"])
		if err != nil {
			api.RespondError(w, http.StatusUnprocessableEntity, "a customer email address is required")
			return
		}

		customer, err = customersQuery.
			Create(&customers.Customer{
				User:      user,
				Email:     (*drivers.Email)(address),
				FirstName: values["FirstName"],
				LastName:  values["LastName"],
			})

		if err != nil {
			logger.Warnf("checkout preparation failed: %v", err)
			api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
			return
		}

	default:
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	checkoutRequest := r.WithContext(ctx)
	cartQuery := carts.
		Query().
		WithContext(ctx).
		CartsForRequest(checkoutRequest)
	active, err := cartQuery.
		Filter(
			expr.Q("Active", true),
			expr.Or(
				expr.Q("User", user),
				expr.Q("User__isnull", true),
			),
		).
		Get()

	var cart *carts.Cart
	switch {
	case err == nil:
		hasItems, err := queries.
			GetQuerySet(&carts.CartItem{}).
			WithContext(ctx).
			Filter(
				expr.Q("Cart", active.Object.ID),
			).
			Exists()

		if err != nil {
			logger.Warnf("checkout preparation failed: %v", err)
			api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
			return
		}

		if hasItems {
			cart = active.Object
		}

	case errors.Is(err, dberrors.MultipleRows):
		api.RespondError(w, http.StatusConflict, "cart changed or already has a checkout")
		return

	case !errors.Is(err, dberrors.NoRows):
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	if cart == nil {
		attempts := queries.
			GetQuerySet(&payments.Payment{}).
			WithContext(ctx).
			Filter(
				expr.Q("Order.Customer", customer.ID),
			).
			Select("ID")
		converted, err := cartQuery.
			Filter(
				expr.Q("Active", false),
				expr.Q("ID__in", queries.Subquery(attempts)),
				expr.Or(
					expr.Q("User", user),
					expr.Q("User__isnull", true),
				),
			).
			OrderBy("-CreatedAt", "-ID").
			First()

		switch {
		case errors.Is(err, dberrors.NoRows) && active != nil:
			api.RespondError(w, http.StatusConflict, "cart is empty")
			return

		case errors.Is(err, dberrors.NoRows):
			api.RespondError(w, http.StatusNotFound, "cart not found")
			return

		case err != nil:
			logger.Warnf("checkout preparation failed: %v", err)
			api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
			return
		}
		cart = converted.Object
	}

	flow := broker.Broker{
		Currency: c.Currency,
	}
	// Use the cart ID as the durable ID of its first payment attempt.
	prepared, err := flow.PrepareCheckout(checkoutRequest, broker.CheckoutRequest{
		Customer:  customer,
		CartID:    cart.ID,
		AccountID: accountID,
		AttemptID: cart.ID,
	})
	switch {
	case errors.Is(err, broker.ErrCheckoutEmpty):
		api.RespondError(w, http.StatusConflict, "cart is empty")
		return

	case errors.Is(err, broker.ErrCheckoutConflict),
		errors.Is(err, dberrors.MultipleRows):
		api.RespondError(w, http.StatusConflict, "cart changed or already has a checkout")
		return

	case errors.Is(err, payments.ErrNonPositiveTotal):
		api.RespondError(w, http.StatusUnprocessableEntity, "payment total must be positive")
		return

	case err != nil:
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	if prepared.Cart != nil {
		cartID := prepared.Cart.Object.ID
		count, err := queries.
			GetQuerySet(&carts.Cart{}).
			WithContext(ctx).
			Filter(
				expr.Q("ID", cartID),
				expr.Q("Active", true),
				expr.Or(
					expr.Q("User", user),
					expr.Q("User__isnull", true),
				),
			).
			Select("Active", "User").
			Update(&carts.Cart{
				User:   user,
				Active: false,
			})

		switch {
		case err != nil:
			logger.Warnf("checkout preparation failed: %v", err)
			api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
			return

		case count != 1:
			api.RespondError(w, http.StatusConflict, "cart changed or already has a checkout")
			return
		}

		redemptions := queries.
			GetQuerySet(&coupons.Redemption{}).
			WithContext(ctx)

		if len(prepared.Cart.Coupons) != 0 {
			ids := make([]uint64, len(prepared.Cart.Coupons))
			for i, redemption := range prepared.Cart.Coupons {
				ids[i] = redemption.ID
			}

			count, err := redemptions.
				Filter(
					expr.Q("Cart", cartID),
					expr.Q("Invalidated", false),
					expr.Q("Order__isnull", true),
					expr.Q("ID__in", ids),
				).
				Select("Order", "Customer").
				Update(&coupons.Redemption{
					Order:    prepared.Order,
					Customer: prepared.Order.Customer,
				})

			switch {
			case err != nil:
				logger.Warnf("checkout preparation failed: %v", err)
				api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
				return

			case count != int64(len(ids)):
				api.RespondError(w, http.StatusConflict, "cart changed or already has a checkout")
				return
			}
		}

		// Release reservations for coupons that did not apply to the priced cart.
		_, err = redemptions.
			Filter(
				expr.Q("Cart", cartID),
				expr.Q("Invalidated", false),
				expr.Q("Order__isnull", true),
			).
			Select("Invalidated").
			Update(&coupons.Redemption{
				Invalidated: true,
			})
		if err != nil && !errors.Is(err, dberrors.NoRows) {
			logger.Warnf("checkout preparation failed: %v", err)
			api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Warnf("checkout preparation failed: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	query := returnURL.Query()
	query.Set("payment_id", prepared.Payment.ID.String())
	returnURL.RawQuery = query.Encode()
	checkout, err := flow.StartCheckout(
		r.Context(),
		prepared.Order.Customer.ID,
		prepared.Payment.ID,
		returnURL.String(),
		webhookURL.String(),
	)
	if err != nil {
		logger.Warnf("provider checkout failed for payment %s", prepared.Payment.ID)
		api.RespondError(w, http.StatusBadGateway, "could not open provider checkout")
		return
	}

	switch checkout.State {
	case paystate.STATUS_PAID,
		paystate.STATUS_CANCEL,
		paystate.STATUS_EXPIRED,
		paystate.STATUS_DENIED,
		paystate.STATUS_FAILURE,
		paystate.STATUS_REFUND,
		paystate.STATUS_PARTIAL_REFUND,
		paystate.STATUS_CHARGEBACK:
		http.Redirect(w, r, returnURL.String(), http.StatusSeeOther)
		return
	}

	redirect, err := url.Parse(checkout.URL)
	switch {
	case err != nil,
		redirect.Scheme != "https",
		redirect.Hostname() == "",
		redirect.User != nil:
		api.RespondError(w, http.StatusBadGateway, "provider did not return a valid checkout URL")
		return
	}

	http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
}
