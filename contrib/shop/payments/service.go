package payments

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var (
	ErrInvalidWebhook       = errors.New("invalid payment webhook")
	ErrUnsupportedOperation = errors.New("payment provider does not support operation")
	ErrNonPositiveTotal     = errors.New("payment total must be positive")
)

func (p *Payment) appendEvent(event PaymentEvent) {
	event.At = time.Now().UTC()
	event.State = p.State
	p.History.Null = false
	p.History.Data = append(p.History.Data, event)
}

// OpenCheckout quotes subtotal and opens checkout with the PSP without writing to the database.
func OpenCheckout(ctx context.Context, account *ProviderAccount, attemptID drivers.ULID, subtotal price.Money, returnURL, webhookURL string) (price.Quote, Checkout, error) {
	if attemptID.IsZero() {
		return price.Quote{}, Checkout{}, errors.New("payment attempt ID is required")
	}
	if account == nil || account.ID.IsZero() {
		return price.Quote{}, Checkout{}, errors.New("payment account is required")
	}
	if !account.Enabled {
		return price.Quote{}, Checkout{}, errors.New("payment account is disabled")
	}

	quote, err := account.Quote(ctx, subtotal)
	if err != nil {
		return price.Quote{}, Checkout{}, err
	}
	if !quote.Total.IsPositive() {
		return price.Quote{}, Checkout{}, ErrNonPositiveTotal
	}

	provider, err := account.Provider(ctx)
	if err != nil {
		return price.Quote{}, Checkout{}, err
	}

	checkout, err := provider.CreatePayment(ctx, CreatePaymentRequest{
		AttemptID:  attemptID,
		Amount:     price.Money{Amount: quote.Total, Currency: quote.Currency},
		ReturnURL:  returnURL,
		WebhookURL: webhookURL,
	})
	if err != nil {
		return price.Quote{}, Checkout{}, err
	}
	if err := validateStatus(quote.Total, quote.Currency, "", checkout.PaymentStatus); err != nil {
		return price.Quote{}, Checkout{}, err
	}

	checkout.ProviderFee = copyProviderFee(checkout.ProviderFee)
	return quote, checkout, nil
}

// RecordPayment persists a successfully opened checkout attempt and emits Signals.Start.
func RecordPayment(ctx context.Context, order attrs.Definer, account *ProviderAccount, id drivers.ULID, quote price.Quote, checkout Checkout) (*Payment, error) {
	if order == nil || attrs.IsZero(order) {
		return nil, errors.New("payment requires an order")
	}
	if account == nil || account.ID.IsZero() || account.InternalName == "" {
		return nil, errors.New("payment requires an account")
	}
	if id.IsZero() {
		return nil, errors.New("payment attempt ID is required")
	}
	if quote.Currency == (currency.Unit{}) || quote.Fee.IsNegative() {
		return nil, errors.New("invalid payment quote")
	}
	if !quote.Total.IsPositive() {
		return nil, ErrNonPositiveTotal
	}
	if err := validateStatus(quote.Total, quote.Currency, "", checkout.PaymentStatus); err != nil {
		return nil, err
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var providerFee drivers.JSON[*price.Money]
	if checkout.ProviderFee != nil {
		providerFee = drivers.JSON[*price.Money]{Data: copyProviderFee(checkout.ProviderFee)}
	}

	payment, err := queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Create(&Payment{
			ID:             id,
			Order:          order,
			Account:        account,
			Provider:       account.InternalName,
			TransactionID:  sql.NullString{String: checkout.ProviderRef, Valid: true},
			CheckoutURL:    checkout.URL,
			State:          checkout.State,
			Amount:         quote.Total,
			Fee:            quote.Fee,
			RefundedAmount: decimal.Zero,
			ProviderFee:    providerFee,
			Currency:       quote.Currency.String(),
			History: drivers.JSON[[]PaymentEvent]{
				Data: []PaymentEvent{
					{
						At:          time.Now().UTC(),
						Kind:        EventCheckoutOpened,
						State:       checkout.State,
						ProviderRef: checkout.ProviderRef,
						ProviderFee: copyProviderFee(checkout.ProviderFee),
					},
				},
			},
		})
	if err != nil {
		return nil, err
	}

	if err := Signals.Start.Send(ctx, PaymentStartedSignal{Payment: payment}); err != nil {
		return nil, err
	}

	return payment, tx.Commit(ctx)
}

// CreateFreePayment records a zero-total order checkout without contacting a payment provider.
func CreateFreePayment(ctx context.Context, order attrs.Definer, unit currency.Unit, id drivers.ULID) (*Payment, error) {
	if order == nil || attrs.IsZero(order) || id.IsZero() {
		return nil, errors.New("free checkout requires an order and attempt ID")
	}
	if unit == (currency.Unit{}) {
		return nil, errors.New("free checkout requires a currency unit")
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	payment, err := queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Create(&Payment{
			ID:             id,
			Order:          order,
			Provider:       freeCheckoutProvider,
			State:          paystate.STATUS_NO_PAYMENT_REQUIRED,
			Amount:         decimal.Zero,
			Fee:            decimal.Zero,
			RefundedAmount: decimal.Zero,
			Currency:       unit.String(),
			History: drivers.JSON[[]PaymentEvent]{
				Data: []PaymentEvent{
					{
						At:    time.Now().UTC(),
						Kind:  EventNoPaymentRequired,
						State: paystate.STATUS_NO_PAYMENT_REQUIRED,
					},
				},
			},
		})
	if err != nil {
		return nil, err
	}

	return payment, tx.Commit(ctx)
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

	provider, err := payment.Account.Provider(ctx)
	if err != nil {
		return nil, err
	}

	status, err := provider.GetPayment(ctx, payment.TransactionID.String)
	if err != nil {
		return nil, err
	}

	return applyStatus(ctx, payment, status)
}

// ServeWebhook dispatches a provider callback using the account in the webhook URL.
func ServeWebhook(w http.ResponseWriter, request *http.Request, accountID string) {
	ctx := request.Context()
	var id drivers.ULID
	if err := id.UnmarshalText([]byte(accountID)); err != nil {
		http.NotFound(w, request)
		return
	}

	account, err := queries.
		GetQuerySet(&ProviderAccount{}).
		WithContext(ctx).
		Filter("ID", id).
		Get()
	if dberrors.Is(err, dberrors.NoRows) {
		http.NotFound(w, request)
		return
	}
	if err != nil {
		http.Error(w, "payment account unavailable", http.StatusServiceUnavailable)
		return
	}

	provider, err := account.Object.Provider(ctx)
	if err != nil {
		http.Error(w, "payment provider unavailable", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, request.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}

	result, err := provider.HandleWebhook(ctx, WebhookRequest{
		Method: request.Method,
		Header: request.Header,
		URL:    request.URL,
		Body:   body,
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

// Refund issues a partial or full refund through the PSP's native Refunder capability.
func Refund(ctx context.Context, id drivers.ULID, requestID string, amount decimal.Decimal) (RefundResult, error) {
	payment, provider, err := operationPayment(ctx, id, requestID)
	if err != nil {
		return RefundResult{}, err
	}

	refunder, ok := provider.(Refunder)
	if !ok {
		return RefundResult{}, ErrUnsupportedOperation
	}

	for _, event := range payment.History.Data {
		if event.RequestID == requestID {
			if event.Kind == EventRefunded && event.Amount != nil && event.Amount.Amount.Equal(amount) {
				return RefundResult{ProviderRef: event.ProviderRef}, nil
			}
			return RefundResult{}, errors.New("operation request ID already used with a different payload")
		}
	}

	if payment.State != paystate.STATUS_PAID && payment.State != paystate.STATUS_PARTIAL_REFUND {
		return RefundResult{}, errors.New("payment is not refundable")
	}

	unit := payment.CurrencyUnit()
	if !amount.IsPositive() {
		return RefundResult{}, errors.New("refund amount must be positive")
	}
	scale, _ := currency.Standard.Rounding(unit)
	if !amount.Equal(amount.Round(int32(scale))) {
		return RefundResult{}, errors.New("refund amount exceeds currency precision")
	}

	newRefunded := payment.RefundedAmount.Add(amount)
	if newRefunded.GreaterThan(payment.Amount) {
		return RefundResult{}, errors.New("refund exceeds remaining payment balance")
	}

	refundMoney := price.Money{Amount: amount, Currency: unit}
	result, err := refunder.RefundPayment(ctx, RefundRequest{
		ProviderRef: payment.TransactionID.String,
		RequestID:   requestID,
		Amount:      refundMoney,
	})
	if err != nil {
		return RefundResult{}, err
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return RefundResult{}, err
	}
	defer tx.Rollback(ctx)

	priorState := payment.State
	priorRefunded := payment.RefundedAmount
	payment.RefundedAmount = newRefunded
	payment.State = paystate.STATUS_PARTIAL_REFUND
	if newRefunded.Equal(payment.Amount) {
		payment.State = paystate.STATUS_REFUND
	}
	amountCopy := refundMoney
	payment.appendEvent(PaymentEvent{
		Kind:        EventRefunded,
		RequestID:   requestID,
		ProviderRef: result.ProviderRef,
		Amount:      &amountCopy,
	})

	n, err := queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("RefundedAmount", "State", "History", "UpdatedAt").
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("State", priorState),
			expr.Q("RefundedAmount", priorRefunded),
		).
		Update(payment)
	if err != nil {
		return RefundResult{}, err
	}
	if n != 1 {
		return RefundResult{}, errors.New("payment changed concurrently")
	}

	if err := Signals.Refund.Send(ctx, PaymentRefundedSignal{
		Payment:   payment,
		RequestID: requestID,
		Amount:    refundMoney,
		Result:    result,
	}); err != nil {
		return RefundResult{}, err
	}
	if priorState != payment.State {
		if err := Signals.Update.Send(ctx, PaymentUpdatedSignal{
			Payment:       payment,
			PreviousState: priorState,
		}); err != nil {
			return RefundResult{}, err
		}
	}

	return result, tx.Commit(ctx)
}

// Cancel cancels an unsettled payment through the PSP's native Canceller capability.
func Cancel(ctx context.Context, id drivers.ULID, requestID string) error {
	payment, provider, err := operationPayment(ctx, id, requestID)
	if err != nil {
		return err
	}

	canceller, ok := provider.(Canceller)
	if !ok {
		return ErrUnsupportedOperation
	}

	for _, event := range payment.History.Data {
		if event.RequestID == requestID {
			if event.Kind == EventCancelled {
				return nil
			}
			return errors.New("operation request ID already used with a different payload")
		}
	}

	if payment.State != paystate.STATUS_INIT && payment.State != paystate.STATUS_PENDING && payment.State != paystate.STATUS_VERIFY {
		return errors.New("payment cannot be cancelled")
	}

	if err := canceller.CancelPayment(ctx, CancelRequest{
		ProviderRef: payment.TransactionID.String,
		RequestID:   requestID,
	}); err != nil {
		return err
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	priorState := payment.State
	payment.State = paystate.STATUS_CANCEL
	payment.appendEvent(PaymentEvent{
		Kind:      EventCancelled,
		RequestID: requestID,
	})

	n, err := queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("State", "History", "UpdatedAt").
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("State", priorState),
		).
		Update(payment)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("payment changed concurrently")
	}

	if err := Signals.Cancel.Send(ctx, PaymentCancelledSignal{
		Payment:   payment,
		RequestID: requestID,
	}); err != nil {
		return err
	}
	if priorState != payment.State {
		if err := Signals.Update.Send(ctx, PaymentUpdatedSignal{
			Payment:       payment,
			PreviousState: priorState,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func operationPayment(ctx context.Context, id drivers.ULID, requestID string) (*Payment, Provider, error) {
	if !queries.IsCommitContext(ctx) {
		return nil, nil, errors.New("payment operations require commits")
	}
	if queries.IsTransactionContext(ctx) {
		return nil, nil, errors.New("payment operations require a context outside an existing transaction")
	}
	if strings.TrimSpace(requestID) == "" || len(requestID) > 128 {
		return nil, nil, errors.New("operation request ID is required and must fit 128 bytes")
	}

	payment, err := getAttempt(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if payment.IsFreeCheckout() {
		return nil, nil, ErrUnsupportedOperation
	}
	if !payment.TransactionID.Valid || strings.TrimSpace(payment.TransactionID.String) == "" {
		return nil, nil, errors.New("payment has no provider transaction reference")
	}

	provider, err := payment.Account.Provider(ctx)
	return payment, provider, err
}

func getAttempt(ctx context.Context, id drivers.ULID) (*Payment, error) {
	row, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("*", "Account.*").
		Filter("ID", id).
		Get()
	if err != nil {
		return nil, err
	}

	if row.Object.IsFreeCheckout() {
		return row.Object, nil
	}
	if row.Object.Account == nil || row.Object.Account.ID.IsZero() || row.Object.Account.InternalName != row.Object.Provider {
		return nil, errors.New("payment account is missing or its driver has changed")
	}
	if row.Object.CurrencyUnit() == (currency.Unit{}) {
		return nil, errors.New("payment currency is invalid")
	}

	return row.Object, nil
}

func reconcileWebhook(ctx context.Context, account *ProviderAccount, status PaymentStatus) error {
	if status.ProviderRef == "" {
		return ErrInvalidWebhook
	}

	row, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Filter("Account", account.ID).
		Filter("TransactionID", status.ProviderRef).
		Get()
	if err != nil {
		return err
	}
	if row.Object.Provider != account.InternalName {
		return errors.New("payment account's driver has changed")
	}

	_, err = applyStatus(ctx, row.Object, status)
	return err
}

func (payment *Payment) validateStatus(status PaymentStatus) error {
	existingRef := ""
	if payment.TransactionID.Valid {
		existingRef = payment.TransactionID.String
	}
	return validateStatus(payment.Amount, payment.CurrencyUnit(), existingRef, status)
}

func validateStatus(expectedAmount decimal.Decimal, expectedUnit currency.Unit, expectedRef string, status PaymentStatus) error {
	switch status.State {
	case paystate.STATUS_INIT,
		paystate.STATUS_PENDING,
		paystate.STATUS_CANCEL,
		paystate.STATUS_EXPIRED,
		paystate.STATUS_DENIED,
		paystate.STATUS_FAILURE,
		paystate.STATUS_PAID_CHECKAMOUNT,
		paystate.STATUS_PARTIAL_PAYMENT,
		paystate.STATUS_VERIFY,
		paystate.STATUS_AUTHORIZE,
		paystate.STATUS_PARTLY_CAPTURED,
		paystate.STATUS_PAID,
		paystate.STATUS_CHARGEBACK,
		paystate.STATUS_REFUNDING,
		paystate.STATUS_REFUND,
		paystate.STATUS_PARTIAL_REFUND:
	default:
		return errors.New("unknown provider payment state")
	}

	if status.ProviderRef == "" || status.State == "" ||
		(expectedRef != "" && status.ProviderRef != expectedRef) ||
		expectedUnit == (currency.Unit{}) ||
		status.Amount.Currency != expectedUnit ||
		!status.Amount.Amount.Equal(expectedAmount) {
		return errors.New("provider status does not match payment")
	}

	if fee := status.ProviderFee; fee != nil {
		if fee.Currency == (currency.Unit{}) || fee.Amount.IsNegative() {
			return errors.New("provider processing fee requires a nonnegative amount and currency unit")
		}
	}

	return nil
}

func applyStatus(ctx context.Context, payment *Payment, status PaymentStatus) (*Payment, error) {
	if err := payment.validateStatus(status); err != nil {
		return nil, err
	}

	sameStatus := payment.State == status.State
	fee, savedFee := status.ProviderFee, payment.ProviderFee.Data
	feeChanged := fee != nil && (savedFee == nil || fee.Currency != savedFee.Currency || !fee.Amount.Equal(savedFee.Amount))
	if sameStatus && !feeChanged {
		return payment, nil
	}
	if !sameStatus && !canTransition(payment.State, status.State) {
		return payment, nil
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	prior := payment.State
	payment.State = status.State
	event := PaymentEvent{Kind: EventStatusUpdated}
	if feeChanged {
		payment.ProviderFee = drivers.JSON[*price.Money]{Data: copyProviderFee(fee)}
		event.ProviderFee = copyProviderFee(fee)
	}
	if sameStatus {
		event.Kind = EventProviderFeeUpdated
	}
	payment.appendEvent(event)

	n, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("State", "ProviderFee", "History", "UpdatedAt").
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("State", prior),
		).
		Update(payment)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("payment status updated concurrently; retry reconciliation")
	}

	if !sameStatus {
		if err := Signals.Update.Send(ctx, PaymentUpdatedSignal{
			Payment:       payment,
			PreviousState: prior,
		}); err != nil {
			return nil, err
		}
	}

	return payment, tx.Commit(ctx)
}

func canTransition(from, to paystate.PayState) bool {
	if to == paystate.STATUS_NO_PAYMENT_REQUIRED {
		return false
	}
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

func copyProviderFee(fee *price.Money) *price.Money {
	if fee == nil {
		return nil
	}
	return &price.Money{
		Amount:   fee.Amount.Copy(),
		Currency: fee.Currency,
	}
}
