package public

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"golang.org/x/text/currency"
)

// CheckoutAPI converts the caller's cart into an order and opens provider checkout when payment is due.
// Origin is the public shop origin used for verified callback destinations.
// ReturnPath is a path on Origin and defaults to "/".
type CheckoutAPI struct {
	Currency    currency.Unit
	GetCurrency func(context.Context) currency.Unit
	Origin      string
	ReturnPath  string
}

func (c CheckoutAPI) resolveCurrency(ctx context.Context) currency.Unit {
	if c.GetCurrency != nil {
		if unit := c.GetCurrency(ctx); unit != (currency.Unit{}) {
			return unit
		}
	}
	return c.Currency
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
		request.AccountID != "" && accountID.UnmarshalText([]byte(request.AccountID)) != nil:
		api.RespondError(w, http.StatusBadRequest, "submit a valid payment account ID")
		return
	}

	unit := c.resolveCurrency(r.Context())
	origin, originErr := url.Parse(c.Origin)
	returnPath := c.ReturnPath
	if returnPath == "" {
		returnPath = "/"
	}

	returnRef, pathErr := url.Parse(returnPath)
	var configError string
	switch {
	case unit == (currency.Unit{}):
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

	switch {
	case queries.IsTransactionContext(r.Context()),
		!queries.IsCommitContext(r.Context()):
		logger.Warnf("checkout preparation failed: the checkout endpoint must commit its own transaction")
		api.RespondError(w, http.StatusInternalServerError, "could not prepare checkout")
		return
	}

	buildReturnURL := func(paymentID, orderID drivers.ULID) string {
		u := origin.ResolveReference(returnRef)
		query := u.Query()
		if !paymentID.IsZero() {
			query.Set("payment_id", paymentID.String())
		}
		if !orderID.IsZero() {
			query.Set("order_id", orderID.String())
		}
		u.RawQuery = query.Encode()
		return u.String()
	}

	result, err := orders.CheckoutCart(r, orders.CheckoutOptions{
		AccountID:   accountID,
		Currency:    c.Currency,
		GetCurrency: c.GetCurrency,
		ReturnURL:   buildReturnURL,
		WebhookURL: func(account *payments.ProviderAccount) string {
			return origin.ResolveReference(&url.URL{
				Path: django.Reverse("api:payments:webhook", account.ID),
			}).String()
		},
	})

	switch {
	case errors.Is(err, orders.ErrCheckoutAccountRequired):
		api.RespondError(w, http.StatusBadRequest, "select a payment account")
		return

	case errors.Is(err, orders.ErrCheckoutAccountUnavailable):
		api.RespondError(w, http.StatusNotFound, "payment account is unavailable")
		return

	case errors.Is(err, dberrors.NoRows):
		api.RespondError(w, http.StatusNotFound, "cart not found")
		return

	case errors.Is(err, orders.ErrCheckoutEmpty):
		api.RespondError(w, http.StatusConflict, "cart is empty")
		return

	case errors.Is(err, orders.ErrCheckoutConflict),
		errors.Is(err, dberrors.MultipleRows):
		api.RespondError(w, http.StatusConflict, "cart changed or already has a checkout")
		return

	case errors.Is(err, orders.ErrCustomerRequired):
		api.RespondError(w, http.StatusUnprocessableEntity, "a customer email address is required")
		return

	case errors.Is(err, payments.ErrNonPositiveTotal):
		api.RespondError(w, http.StatusUnprocessableEntity, "payment total must be positive")
		return

	case err != nil:
		logger.Warnf("provider checkout failed: %v", err)
		api.RespondError(w, http.StatusBadGateway, "could not open provider checkout")
		return
	}

	if result.Free {
		http.Redirect(w, r, buildReturnURL(result.Payment.ID, result.Order.ID), http.StatusSeeOther)
		return
	}

	returnURL := buildReturnURL(result.Payment.ID, drivers.ULID{})
	switch result.Checkout.State {
	case paystate.STATUS_PAID,
		paystate.STATUS_CANCEL,
		paystate.STATUS_EXPIRED,
		paystate.STATUS_DENIED,
		paystate.STATUS_FAILURE,
		paystate.STATUS_REFUND,
		paystate.STATUS_PARTIAL_REFUND,
		paystate.STATUS_CHARGEBACK:
		http.Redirect(w, r, returnURL, http.StatusSeeOther)
		return
	}

	redirect, err := url.Parse(result.Checkout.URL)
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
