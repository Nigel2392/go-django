package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidWebhook       = errors.New("invalid payment webhook")
	ErrUnsupportedOperation = errors.New("payment provider does not support operation")
	ErrOperationPending     = errors.New("payment operation requires reconciliation")
)

type OperationRecord struct {
	PreviousState string
	Kind          OperationKind
	Resource      string
	Amount        Money
	Phase         OperationState
	Result        OperationResult
}

func Refund(ctx context.Context, id drivers.ULID, request RefundRequest) (OperationResult, error) {
	payment, provider, err := operationPayment(ctx, id, request.OperationRequest)
	if err != nil {
		return OperationResult{}, err
	}

	refunder, ok := provider.(Refunder)
	if !ok {
		return OperationResult{}, ErrUnsupportedOperation
	}

	request.OperationRequest, err = bindOperationReference(
		request.OperationRequest,
		operationReferences(payment, provider).Refund,
	)
	if err != nil {
		return OperationResult{}, err
	}

	if err = request.Validate(); err != nil {
		return OperationResult{}, err
	}

	record := OperationRecord{
		Kind:     OperationRefund,
		Resource: request.ProviderRef,
		Amount:   request.Amount,
	}

	result, replay, err := replayOperation(
		payment,
		request.RequestID,
		record,
	)
	if replay || err != nil {
		return result, err
	}

	if payment.State != paystate.STATUS_PAID && payment.State != paystate.STATUS_PARTIAL_REFUND {
		return OperationResult{}, errors.New("payment is not refundable")
	}

	refunded := decimal.Zero
	for _, op := range payment.Operations.Data {
		if op.Kind == OperationRefund && op.Phase == OperationSucceeded {
			refunded = refunded.Add(op.Amount.Amount)
		}
	}

	if request.Amount.Currency != payment.Currency || refunded.Add(request.Amount.Amount).GreaterThan(payment.Amount) {
		return OperationResult{}, errors.New("refund exceeds payment balance or currency differs")
	}

	err = reserveOperation(ctx, payment, request.RequestID, record, paystate.STATUS_REFUNDING)
	if err != nil {
		return OperationResult{}, err
	}

	result, err = refunder.RefundPayment(ctx, request)
	if err != nil {
		return OperationResult{}, failOperation(ctx, payment, request.RequestID, err)
	}

	return finishOperation(ctx, id, request.RequestID, result)
}

func Void(ctx context.Context, id drivers.ULID, request OperationRequest) error {
	payment, provider, err := operationPayment(ctx, id, request)
	if err != nil {
		return err
	}

	voider, ok := provider.(Voider)
	if !ok {
		return ErrUnsupportedOperation
	}

	request, err = bindOperationReference(
		request,
		operationReferences(payment, provider).Void,
	)
	if err != nil {
		return err
	}

	record := OperationRecord{
		Kind:     OperationVoid,
		Resource: request.ProviderRef,
	}

	_, replay, err := replayOperation(
		payment,
		request.RequestID,
		record,
	)
	if replay || err != nil {
		return err
	}

	if payment.State != paystate.STATUS_AUTHORIZE {
		return errors.New("payment is not authorized")
	}

	err = reserveOperation(
		ctx, payment,
		request.RequestID,
		record,
		payment.State,
	)
	if err != nil {
		return err
	}

	if err = voider.VoidPayment(ctx, request); err != nil {
		return failOperation(ctx, payment, request.RequestID, err)
	}

	_, err = finishOperation(ctx, id, request.RequestID, OperationResult{
		ProviderRef: request.ProviderRef,
		State:       OperationSucceeded,
		Status:      OperationStatusVoided,
	})
	return err
}

func Capture(ctx context.Context, id drivers.ULID, request OperationRequest) (OperationResult, error) {
	payment, provider, err := operationPayment(ctx, id, request)
	if err != nil {
		return OperationResult{}, err
	}

	capturer, ok := provider.(Capturer)
	if !ok {
		return OperationResult{}, ErrUnsupportedOperation
	}

	request, err = bindOperationReference(
		request,
		operationReferences(payment, provider).Capture,
	)
	if err != nil {
		return OperationResult{}, err
	}

	record := OperationRecord{
		Kind:     OperationCapture,
		Resource: request.ProviderRef,
	}

	result, replay, err := replayOperation(
		payment,
		request.RequestID,
		record,
	)
	if replay || err != nil {
		return result, err
	}

	if payment.State != paystate.STATUS_AUTHORIZE {
		return OperationResult{}, errors.New(
			"payment is not authorized")
	}

	err = reserveOperation(
		ctx, payment,
		request.RequestID,
		record,
		payment.State,
	)
	if err != nil {
		return OperationResult{}, err
	}

	result, err = capturer.CapturePayment(ctx, request)
	if err != nil {
		return OperationResult{}, failOperation(
			ctx, payment,
			request.RequestID,
			err,
		)
	}

	return finishOperation(ctx, id, request.RequestID, result)
}

func Cancel(ctx context.Context, id drivers.ULID, request OperationRequest) error {
	payment, provider, err := operationPayment(ctx, id, request)
	if err != nil {
		return err
	}

	canceller, ok := provider.(Canceller)
	if !ok {
		return ErrUnsupportedOperation
	}

	request, err = bindOperationReference(
		request,
		operationReferences(payment, provider).Cancel,
	)
	if err != nil {
		return err
	}

	record := OperationRecord{
		Kind:     OperationCancel,
		Resource: request.ProviderRef,
	}

	_, replay, err := replayOperation(
		payment,
		request.RequestID,
		record,
	)
	if replay || err != nil {
		return err
	}

	if payment.State != paystate.STATUS_INIT && payment.State != paystate.STATUS_PENDING && payment.State != paystate.STATUS_VERIFY {
		return errors.New("payment cannot be cancelled")
	}

	err = reserveOperation(
		ctx, payment,
		request.RequestID,
		record,
		payment.State,
	)
	if err != nil {
		return err
	}

	if err = canceller.CancelPayment(ctx, request); err != nil {
		return failOperation(ctx, payment, request.RequestID, err)
	}

	_, err = finishOperation(ctx, id, request.RequestID, OperationResult{
		ProviderRef: request.ProviderRef,
		State:       OperationSucceeded,
		Status:      OperationStatusCancelled,
	})
	return err
}

func operationPayment(ctx context.Context, id drivers.ULID, request OperationRequest) (*Payment, Provider, error) {
	if !queries.IsCommitContext(ctx) {
		return nil, nil, errors.New(
			"payment operations require commits")
	}

	if queries.IsTransactionContext(ctx) {
		return nil, nil, errors.New(
			"payment operations require a context outside an existing transaction")
	}

	if strings.TrimSpace(request.RequestID) == "" || len(request.RequestID) > 128 {
		return nil, nil, errors.New(
			"operation request ID is required and must fit 128 bytes")
	}

	payment, err := getAttempt(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	provider, err := payment.Account.provider(ctx)
	return payment, provider, err
}

func operationReferences(payment *Payment, provider Provider) OperationReferences {
	if resolver, ok := provider.(OperationReferenceResolver); ok {
		return resolver.OperationReferences(payment.status())
	}

	ref := payment.TransactionID.String
	return OperationReferences{
		Refund:  ref,
		Void:    ref,
		Capture: ref,
		Cancel:  ref,
	}
}

func bindOperationReference(request OperationRequest, resource string) (OperationRequest, error) {
	if resource == "" || (request.ProviderRef != "" && request.ProviderRef != resource) {
		return request, errors.New("operation resource does not match payment")
	}

	request.ProviderRef = resource
	return request, nil
}

func replayOperation(payment *Payment, key string, record OperationRecord) (OperationResult, bool, error) {
	if old, ok := payment.Operations.Data[key]; ok {
		if old.Kind != record.Kind || old.Resource != record.Resource || old.Amount.Currency != record.Amount.Currency || !old.Amount.Amount.Equal(record.Amount.Amount) {
			return OperationResult{}, false, errors.New("operation request ID already has a different payload")
		}

		if old.Phase == OperationSucceeded {
			return old.Result, true, nil
		}

		return old.Result, false, ErrOperationPending
	}

	if payment.OperationKey != "" {
		return OperationResult{}, false, ErrOperationPending
	}

	return OperationResult{}, false, nil
}

// reserveOperation commits the durable request before the adapter is called.
func reserveOperation(ctx context.Context, payment *Payment, key string, record OperationRecord, state string) error {
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	priorState := payment.State
	record.PreviousState = priorState
	record.Phase = OperationRequested
	payment.State = state
	if payment.Operations.Data == nil {
		payment.Operations.Data = make(map[string]OperationRecord)
	}

	payment.Operations.Null = false
	payment.Operations.Data[key] = record
	payment.OperationKey = key
	version := payment.OperationVersion
	payment.OperationVersion++

	n, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("Operations", "OperationKey", "OperationVersion", "State", "UpdatedAt").
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("State", priorState),
			expr.Or(
				expr.Q("OperationKey", ""),
				expr.Q("OperationKey__isnull", true),
			),
			expr.Or(
				expr.Q("OperationVersion", version),
				expr.Q("OperationVersion__isnull", true),
			),
		).
		Update(payment)
	if err != nil {
		return err
	}

	if n != 1 {
		return errors.New("payment changed concurrently")
	}

	err = Signals.OperationRequested.Send(ctx, PaymentOperationSignal{
		Payment:   payment,
		RequestID: key,
		Operation: record,
	})
	if err == nil && priorState != payment.State {
		err = Signals.Update.Send(ctx, PaymentUpdatedSignal{Payment: payment, PreviousState: priorState})
	}

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// A failed response may follow a successful remote mutation. Keep the reservation.
func failOperation(ctx context.Context, payment *Payment, key string, cause error) error {
	record := payment.Operations.Data[key]
	record.Phase = OperationUncertain
	record.Result = OperationResult{}
	payment.Operations.Data[key] = record

	_, saveErr := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("Operations", "UpdatedAt").
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("OperationKey", key),
		).
		Update(payment)

	signalErr := Signals.OperationFailed.Send(ctx, PaymentOperationSignal{
		Payment:   payment,
		RequestID: key,
		Operation: record,
		Error:     cause,
	})

	return errors.Join(cause, saveErr, signalErr)
}

// ResolveOperation records a server-verified PSP outcome after an ambiguous or
// asynchronous result. Call only after provider reconciliation or operator review.
// A failed outcome releases the reservation without claiming money moved.
func ResolveOperation(ctx context.Context, id drivers.ULID, requestID string, result OperationResult) (OperationResult, error) {
	if result.State != OperationSucceeded && result.State != OperationFailed {
		return OperationResult{}, errors.New("reconciliation requires a terminal outcome")
	}

	return finishOperation(ctx, id, requestID, result)
}

func finishOperation(ctx context.Context, id drivers.ULID, key string, result OperationResult) (OperationResult, error) {
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return OperationResult{}, err
	}

	defer tx.Rollback(ctx)

	payment, err := getAttempt(ctx, id)
	if err != nil {
		return OperationResult{}, err
	}

	record, ok := payment.Operations.Data[key]
	if !ok {
		return OperationResult{}, errors.New("unknown payment operation")
	}

	if record.Phase == OperationSucceeded || record.Phase == OperationFailed {
		if result.State != record.Phase {
			return OperationResult{}, errors.New("operation already has a different terminal outcome")
		}

		return record.Result, nil
	}

	if payment.OperationKey != key {
		return OperationResult{}, errors.New("payment operation reservation changed")
	}

	if result.State != OperationSucceeded && result.State != OperationFailed {
		result.State = OperationPending
	}

	if result.State == OperationSucceeded && strings.TrimSpace(result.ProviderRef) == "" {
		return OperationResult{}, errors.New("successful operation requires a provider reference")
	}

	record.Result = result
	record.Phase = result.State
	payment.Operations.Data[key] = record
	prior := payment.State

	if result.State == OperationFailed && record.PreviousState != "" {
		payment.State = record.PreviousState
	}

	if result.State == OperationSucceeded {
		switch record.Kind {
		case OperationVoid, OperationCancel:
			payment.State = paystate.STATUS_CANCEL

		case OperationCapture:
			payment.State = paystate.STATUS_PAID
			payment.CaptureRef = result.ProviderRef

		case OperationRefund:
			refunded := decimal.Zero
			for _, op := range payment.Operations.Data {
				if op.Kind == OperationRefund && op.Phase == OperationSucceeded {
					refunded = refunded.Add(op.Amount.Amount)
				}
			}

			payment.State = paystate.STATUS_PARTIAL_REFUND
			if refunded.Equal(payment.Amount) {
				payment.State = paystate.STATUS_REFUND
			}
		}
	}

	if result.State != OperationPending {
		payment.OperationKey = ""
	}

	n, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select("Operations", "OperationKey", "State", "CaptureRef", "UpdatedAt").
		Filter("ID", id).
		Filter("OperationKey", key).
		Update(payment)
	if err != nil {
		return OperationResult{}, err
	}

	if n != 1 {
		return OperationResult{}, errors.New("operation updated concurrently")
	}

	sig := PaymentOperationSignal{Payment: payment, RequestID: key, Operation: record}
	switch result.State {
	case OperationPending:
		err = Signals.OperationPending.Send(ctx, sig)

	case OperationFailed:
		err = Signals.OperationFailed.Send(ctx, sig)

	case OperationSucceeded:
		switch record.Kind {
		case OperationRefund:
			err = Signals.Refund.Send(ctx, PaymentRefundedSignal{
				Payment:   payment,
				RequestID: key,
				Amount:    record.Amount,
				Result:    result,
			})

		case OperationVoid:
			err = Signals.Void.Send(ctx, PaymentVoidedSignal{
				Payment:   payment,
				RequestID: key,
			})

		case OperationCapture:
			err = Signals.Capture.Send(ctx, sig)

		case OperationCancel:
			err = Signals.Cancel.Send(ctx, sig)
		}
	}

	if err == nil && prior != payment.State {
		err = Signals.Update.Send(ctx, PaymentUpdatedSignal{
			Payment:       payment,
			PreviousState: prior,
		})
	}

	if err != nil {
		return result, fmt.Errorf("payment operation signal: %w", err)
	}

	return result, tx.Commit(ctx)
}

// CreateAttempt snapshots the server-calculated amount and the account's fee.
func CreateAttempt(ctx context.Context, order attrs.Definer, accountID drivers.ULID, subtotal Money) (*Payment, error) {
	return createAttempt(ctx, order, accountID, subtotal, drivers.ULID{})
}

// CreateAttemptWithID saves a new attempt under a caller-supplied ID. A repeat
// checkout can look up that ID before it prepares another order.
func CreateAttemptWithID(ctx context.Context, order attrs.Definer, accountID drivers.ULID, subtotal Money, id drivers.ULID) (*Payment, error) {
	if id.IsZero() {
		return nil, errors.New("payment attempt ID is required")
	}
	return createAttempt(ctx, order, accountID, subtotal, id)
}

func createAttempt(ctx context.Context, order attrs.Definer, accountID drivers.ULID, subtotal Money, id drivers.ULID) (*Payment, error) {
	if order == nil {
		return nil, errors.New("payment requires an order")
	}

	row, err := queries.
		GetQuerySet(&ProviderAccount{}).
		WithContext(ctx).
		Filter("ID", accountID).
		Filter("Enabled", true).
		Get()
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

	return queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Create(&Payment{
			ID:       id,
			Order:    order,
			Account:  row.Object,
			Provider: row.Object.Driver,
			Amount:   quote.Total,
			Fee:      quote.Fee,
			Currency: quote.Currency,
			State:    paystate.STATUS_INIT,
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
		AttemptID: id,
		Amount: Money{
			Amount:   payment.Amount,
			Currency: payment.Currency,
		},
		ReturnURL:  returnURL,
		WebhookURL: webhookURL,
	})
	if err != nil {
		return Checkout{}, err
	}

	if err := payment.validateStatus(checkout.PaymentStatus); err != nil {
		return Checkout{}, err
	}

	payment.TransactionID = sql.NullString{
		String: checkout.ProviderRef,
		Valid:  true,
	}

	payment.CheckoutURL, payment.State = checkout.URL, checkout.State
	payment.AuthorizationRef = checkout.AuthorizationRef
	payment.CaptureRef = checkout.CaptureRef
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return Checkout{}, err
	}

	defer tx.Rollback(ctx)
	n, err := queries.GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select(
			"TransactionID",
			"CheckoutURL",
			"State",
			"AuthorizationRef",
			"CaptureRef",
			"UpdatedAt",
		).
		Filter(
			expr.Q("ID", id),
			expr.Q("TransactionID__isnull", true),
		).
		Update(payment)

	if err == nil && n == 0 {
		payment, err = getAttempt(ctx, id)
		if err == nil {
			return Checkout{
				PaymentStatus: payment.status(),
				URL:           payment.CheckoutURL,
			}, nil
		}
	}

	if err == nil {
		err = Signals.Start.Send(ctx, PaymentStartedSignal{
			Payment: payment,
		})
	}

	if err == nil {
		err = tx.Commit(ctx)
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

	status, err := provider.GetPayment(
		ctx, payment.TransactionID.String,
	)
	if err != nil {
		return nil, err
	}

	return applyStatus(ctx, payment, status)
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

	if row.Object.Account == nil || row.Object.Account.ID.IsZero() || row.Object.Account.Driver != row.Object.Provider {
		return nil, errors.New("payment account is missing or its driver has changed")
	}

	return row.Object, nil
}

func (payment *Payment) status() PaymentStatus {
	return PaymentStatus{
		ProviderRef:      payment.TransactionID.String,
		State:            payment.State,
		AuthorizationRef: payment.AuthorizationRef,
		CaptureRef:       payment.CaptureRef,
		Amount: Money{
			Amount:   payment.Amount,
			Currency: payment.Currency,
		},
	}
}

func (payment *Payment) validateStatus(status PaymentStatus) error {
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
		(payment.TransactionID.Valid && status.ProviderRef != payment.TransactionID.String) ||
		status.Amount.Currency != payment.Currency || !status.Amount.Amount.Equal(payment.Amount) {
		return errors.New("provider status does not match payment")
	}

	return nil
}

func applyStatus(ctx context.Context, payment *Payment, status PaymentStatus) (*Payment, error) {
	if err := payment.validateStatus(status); err != nil {
		return nil, err
	}

	sameAuthorization := status.AuthorizationRef == "" || payment.AuthorizationRef == status.AuthorizationRef
	sameCapture := status.CaptureRef == "" || payment.CaptureRef == status.CaptureRef
	if payment.State == status.State && sameAuthorization && sameCapture {
		return payment, nil
	}

	if payment.State != status.State && !canTransition(payment.State, status.State) {
		return payment, nil
	}

	if payment.OperationKey != "" {
		return nil, ErrOperationPending
	}

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)
	prior := payment.State
	payment.State = status.State
	if status.AuthorizationRef != "" {
		payment.AuthorizationRef = status.AuthorizationRef
	}

	if status.CaptureRef != "" {
		payment.CaptureRef = status.CaptureRef
	}

	n, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Select(
			"State",
			"AuthorizationRef",
			"CaptureRef",
			"UpdatedAt",
		).
		Filter(
			expr.Q("ID", payment.ID),
			expr.Q("State", prior),
			expr.Or(
				expr.Q("OperationKey", ""),
				expr.Q("OperationKey__isnull", true),
			),
		).
		Update(payment)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, errors.New("payment status updated concurrently; retry reconciliation")
	}

	err = Signals.Update.Send(ctx, PaymentUpdatedSignal{
		Payment:       payment,
		PreviousState: prior,
	})

	if err != nil {
		return nil, err
	}

	return payment, tx.Commit(ctx)
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
		http.Error(w,
			"payment account unavailable",
			http.StatusServiceUnavailable)
		return
	}

	provider, err := account.Object.provider(ctx)
	if err != nil {
		http.Error(w,
			"payment provider unavailable",
			http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, request.Body, 1<<20))
	if err != nil {
		http.Error(w,
			"invalid webhook body",
			http.StatusBadRequest)
		return
	}

	result, err := provider.HandleWebhook(ctx, WebhookRequest{
		Method: request.Method,
		Header: request.Header,
		URL:    request.URL,
		Body:   body,
	})
	if errors.Is(err, ErrInvalidWebhook) {
		http.Error(w,
			"invalid webhook",
			http.StatusBadRequest)
		return
	}

	if err == nil {
		err = reconcileWebhook(ctx, account.Object, result.Payment)
	}

	if err != nil {
		http.Error(w,
			"payment reconciliation failed",
			http.StatusServiceUnavailable)
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

	row, err := queries.
		GetQuerySet(&Payment{}).
		WithContext(ctx).
		Filter("Account", account.ID).
		Filter("TransactionID", status.ProviderRef).
		Get()
	if err != nil {
		return err
	}

	if row.Object.Provider != account.Driver {
		return errors.New("payment account's driver has changed")
	}

	_, err = applyStatus(ctx, row.Object, status)
	return err
}
