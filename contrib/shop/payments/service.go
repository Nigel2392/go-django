package payments

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var ErrInvalidWebhook = errors.New("invalid payment webhook")

// CreateAttempt snapshots the server-calculated amount and the account's fee.
func CreateAttempt(ctx context.Context, order attrs.Definer, accountID drivers.ULID, subtotal Money) (*Payment, error) {
	if order == nil {
		return nil, errors.New("payment requires an order")
	}
	row, err := queries.GetQuerySet(&ProviderAccount{}).WithContext(ctx).
		Filter("ID", accountID).Filter("Enabled", true).Get()
	if err != nil {
		return nil, err
	}
	quote, err := row.Object.Quote(subtotal)
	if err != nil {
		return nil, err
	}
	if !quote.Total.IsPositive() {
		return nil, errors.New("payment total must be positive")
	}

	return queries.GetQuerySet(&Payment{}).WithContext(ctx).Create(&Payment{
		Order: order, Account: row.Object, Provider: row.Object.Driver,
		Amount: quote.Total, Fee: quote.Fee, Currency: quote.Currency, State: paystate.STATUS_INIT,
	})
}

// Start opens checkout. Retry with the same attempt ID and URLs if the request fails.
func Start(ctx context.Context, id drivers.ULID, returnURL, webhookURL string) (Checkout, error) {
	payment, err := getAttempt(ctx, id)
	if err != nil {
		return Checkout{}, err
	}
	if payment.TransactionID.Valid {
		return Checkout{PaymentStatus: payment.status(), URL: payment.CheckoutURL}, nil
	}
	if !payment.Account.Enabled {
		return Checkout{}, errors.New("payment account is disabled")
	}
	provider, err := payment.Account.provider(ctx)
	if err != nil {
		return Checkout{}, err
	}
	checkout, err := provider.CreatePayment(ctx, CreatePaymentRequest{
		AttemptID: id, Amount: Money{Amount: payment.Amount, Currency: payment.Currency},
		ReturnURL: returnURL, WebhookURL: webhookURL,
	})
	if err != nil {
		return Checkout{}, err
	}
	if err := payment.validateStatus(checkout.PaymentStatus); err != nil {
		return Checkout{}, err
	}
	payment.TransactionID = sql.NullString{String: checkout.ProviderRef, Valid: true}
	payment.CheckoutURL, payment.State = checkout.URL, checkout.State
	n, err := queries.GetQuerySet(&Payment{}).WithContext(ctx).
		Filter("ID", id).Filter("TransactionID__isnull", true).
		Select("TransactionID", "CheckoutURL", "State", "UpdatedAt").Update(payment)
	if err == nil && n == 0 {
		payment, err = getAttempt(ctx, id)
		if err == nil {
			return Checkout{PaymentStatus: payment.status(), URL: payment.CheckoutURL}, nil
		}
	}
	return checkout, err
}

// Refresh reconciles an attempt with the provider. It also works on disabled accounts.
func Refresh(ctx context.Context, id drivers.ULID) (*Payment, error) {
	payment, err := getAttempt(ctx, id)
	if err != nil {
		return nil, err
	}
	if !payment.TransactionID.Valid {
		return payment, nil
	}
	provider, err := payment.Account.provider(ctx)
	if err != nil {
		return nil, err
	}
	status, err := provider.GetPayment(ctx, payment.TransactionID.String)
	if err != nil {
		return nil, err
	}
	return applyStatus(ctx, payment, status)
}

func getAttempt(ctx context.Context, id drivers.ULID) (*Payment, error) {
	row, err := queries.GetQuerySet(&Payment{}).WithContext(ctx).
		Select("*", "Account.*").Filter("ID", id).Get()
	if err != nil {
		return nil, err
	}
	if row.Object.Account == nil || row.Object.Account.Driver != row.Object.Provider {
		return nil, errors.New("payment account is missing or its driver has changed")
	}
	return row.Object, nil
}

func (p *Payment) status() PaymentStatus {
	return PaymentStatus{
		ProviderRef: p.TransactionID.String, State: p.State,
		Amount: Money{Amount: p.Amount, Currency: p.Currency},
	}
}

func (p *Payment) validateStatus(status PaymentStatus) error {
	if status.ProviderRef == "" || status.State == "" ||
		(p.TransactionID.Valid && status.ProviderRef != p.TransactionID.String) ||
		status.Amount.Currency != p.Currency || !status.Amount.Amount.Equal(p.Amount) {
		return errors.New("provider status does not match payment")
	}
	return nil
}

func applyStatus(ctx context.Context, p *Payment, status PaymentStatus) (*Payment, error) {
	if err := p.validateStatus(status); err != nil {
		return nil, err
	}
	if p.State == status.State || !canTransition(p.State, status.State) {
		return p, nil
	}
	prior := p.State
	p.State = status.State
	n, err := queries.GetQuerySet(&Payment{}).WithContext(ctx).
		Filter("ID", p.ID).Filter("State", prior).Select("State", "UpdatedAt").Update(p)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("payment status updated concurrently; retry reconciliation")
	}
	return p, nil
}

func canTransition(from, to paystate.PayState) bool {
	switch from {
	case paystate.STATUS_INIT:
		return true
	case paystate.STATUS_PENDING:
		return to != paystate.STATUS_INIT
	case paystate.STATUS_VERIFY, paystate.STATUS_PARTIAL_PAYMENT, paystate.STATUS_AUTHORIZE:
		return to != paystate.STATUS_INIT && to != paystate.STATUS_PENDING
	case paystate.STATUS_PARTLY_CAPTURED:
		return to == paystate.STATUS_PAID || to == paystate.STATUS_VERIFY
	case paystate.STATUS_PAID, paystate.STATUS_REFUNDING, paystate.STATUS_PARTIAL_REFUND:
		return to == paystate.STATUS_REFUNDING || to == paystate.STATUS_PARTIAL_REFUND ||
			to == paystate.STATUS_REFUND || to == paystate.STATUS_CHARGEBACK
	default:
		return false
	}
}

// ServeWebhook dispatches a provider callback using the account in the webhook URL.
func ServeWebhook(w http.ResponseWriter, r *http.Request, accountID string) {
	ctx := r.Context()
	var id drivers.ULID
	if err := id.UnmarshalText([]byte(accountID)); err != nil {
		http.NotFound(w, r)
		return
	}
	account, err := queries.GetQuerySet(&ProviderAccount{}).WithContext(ctx).Filter("ID", id).Get()
	if dberrors.Is(err, dberrors.NoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "payment account unavailable", http.StatusServiceUnavailable)
		return
	}
	provider, err := account.Object.provider(ctx)
	if err != nil {
		http.Error(w, "payment provider unavailable", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}
	result, err := provider.HandleWebhook(ctx, WebhookRequest{
		Method: r.Method, Header: r.Header, URL: r.URL, Body: body,
	})
	if errors.Is(err, ErrInvalidWebhook) {
		http.Error(w, "invalid webhook", http.StatusBadRequest)
		return
	}
	if err == nil {
		err = reconcileWebhook(ctx, account.Object, result.Payment)
	}
	if err != nil {
		http.Error(w, "payment reconciliation failed", http.StatusServiceUnavailable)
		return
	}
	if result.Acknowledge != nil {
		result.Acknowledge(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func reconcileWebhook(ctx context.Context, account *ProviderAccount, status PaymentStatus) error {
	if status.ProviderRef == "" {
		return ErrInvalidWebhook
	}
	row, err := queries.GetQuerySet(&Payment{}).WithContext(ctx).
		Filter("Account", account.ID).Filter("TransactionID", status.ProviderRef).Get()
	if err != nil {
		return err
	}
	if row.Object.Provider != account.Driver {
		return errors.New("payment account's driver has changed")
	}
	_, err = applyStatus(ctx, row.Object, status)
	return err
}
