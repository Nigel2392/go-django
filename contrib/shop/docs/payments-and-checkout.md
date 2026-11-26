# Payments & Checkout

The `broker` (`contrib/shop/broker`) and `payments` (`contrib/shop/payments`) packages, together with the public HTTP views (`contrib/shop/views/public/payment_options.go`, `payment_checkout.go`), coordinate idempotent checkout preparation, free (zero-total) checkouts, provider redirects, webhook reconciliation, and post-payment operations (capture, void, cancel, refund).

---

## `broker.Broker` (`contrib/shop/broker/checkout.go`)

`Broker` coordinates order creation and payment attempt creation in a single transaction, using a caller-supplied durable `AttemptID` for idempotent retries.

### Types & Sentinel Errors

```go
var (
	ErrCheckoutConflict           = errors.New("checkout attempt ID is already in use")
	ErrCheckoutEmpty              = errors.New("checkout cart is empty")
	ErrCheckoutAccountRequired    = errors.New("payment account is required")
	ErrCheckoutAccountUnavailable = errors.New("payment account is unavailable")
)

type CheckoutPricer interface {
	CheckoutSubtotal(context.Context, *carts.CartRow) (payments.Money, error)
}

type Broker struct {
	Currency string
	Pricer   CheckoutPricer
}

type CheckoutRequest struct {
	Customer  *customers.Customer
	CartID    drivers.ULID
	AccountID drivers.ULID
	AttemptID drivers.ULID
}

type PreparedCheckout struct {
	Order   *orders.Order
	Payment *payments.Payment
	Cart    *carts.CartRow // Priced cart on initial preparation; nil on retries
}
```

### `PrepareCheckout`

```go
func (b Broker) PrepareCheckout(r *http.Request, request CheckoutRequest) (*PreparedCheckout, error)
```

1. Validates `r`, `Customer` (non-nil, non-zero ID), `CartID`, `AttemptID`, and `queries.IsCommitContext(r.Context())`.
2. Starts/joins a transaction and locks the active cart row (`ID = request.CartID`, `Active = true`) via `BulkUpdate(expr.As("Active", expr.Value(true)))` so pricing serializes with concurrent coupon reservations.
3. Checks if a `payments.Payment` already exists with `ID = request.AttemptID`:
   - If found and owned by `request.Customer.ID`:
     - If `prepared.Payment.IsFreeCheckout()`, commits and returns `prepared` (`Cart == nil`).
     - Otherwise validates `request.AccountID` against `prepared.Payment.Currency` and verifies `prepared.Payment.Account.ID == request.AccountID` (else `ErrCheckoutConflict`).
   - If owned by a different customer, returns `ErrCheckoutConflict`.
4. If no attempt exists yet:
   - Loads and prices the cart via `activeCart.CalcOneWithProducts()`. Returns `ErrCheckoutEmpty` if `len(cart.Items) == 0`.
   - Computes `subtotal` as `payments.Money{Amount: cart.Totals.Payable(), Currency: b.Currency}` (or `b.Pricer.CheckoutSubtotal(ctx, cart)` if `b.Pricer != nil`).
   - Rounds `subtotal` via `(&payments.ProviderAccount{Currency: subtotal.Currency}).Quote(subtotal)`.
   - If `quote.Total.IsZero()` (`free == true`), `AccountID` may be omitted; otherwise `validateCheckoutAccount` verifies the `ProviderAccount` is `Enabled`, matches `subtotal.Currency`, and has a registered provider driver (`payments.ProviderRegistered`).
   - Calls `orders.CreateOrderFromCart(r.WithContext(ctx), request.Customer, cart)`. Maps `orders.ErrCartRedemptionsChanged` to `ErrCheckoutConflict`.
   - Calls `payments.CreateFreeCheckoutWithID(ctx, order, subtotal.Currency, request.AttemptID)` when `free`, or `payments.CreateAttemptWithID(ctx, order, request.AccountID, subtotal, request.AttemptID)` otherwise.
   - Commits and returns `&PreparedCheckout{Order: order, Payment: attempt, Cart: cart}`.

### `StartCheckout`

```go
func (b Broker) StartCheckout(ctx context.Context, customerID, attemptID drivers.ULID, returnURL, webhookURL string) (payments.Checkout, error)
```

- Requires a committed context outside any active transaction (`!queries.IsTransactionContext(ctx) && queries.IsCommitContext(ctx)`).
- Verifies `attemptID` belongs to `customerID` (returning `dberrors.NoRows` if ownership mismatches).
- Delegates to `payments.Start(ctx, prepared.Payment.ID, returnURL, webhookURL)`.

---

## Predefined Checkout Flow (`views/public/`)

### 1. Quote Options (`GET /api/payments/options`)

Handled by `public.PaymentsAPI.PaymentOptions` (`contrib/shop/views/public/payment_options.go`):
- Validates `PaymentsAPI.Currency` as an ISO currency (`503` if invalid).
- Calculates the caller's active cart via `cartQuery(ctx, r, false).CalcOne()` (`404` if no cart).
- Computes `subtotal := payments.Money{Amount: cart.Totals.Payable(), Currency: c.Currency}`.
- **Free checkout check**: if `len(cart.Items) > 0` and a zero-fee `ProviderAccount` quote has `quote.Total.IsZero()`, returns a single `PaymentOption`:
  ```go
  PaymentOption{
      Label:             trans.T(ctx, "No payment required"),
      NoPaymentRequired: true,
      Quote:             quote,
  }
  ```
- Otherwise queries enabled `ProviderAccount` rows matching `Currency`, skips any whose `Driver` is not registered via `payments.ProviderRegistered`, calls `account.Quote(subtotal)`, and returns `[]PaymentOption`.

### 2. Start Checkout (`POST /api/payments/checkout`)

Handled by `public.CheckoutAPI.Checkout` (`contrib/shop/views/public/payment_checkout.go`):
- Sets `Cache-Control: no-store`. Enforces `POST` (`405`), authenticated user (`401`), and decodes `PaymentCheckoutRequest{ AccountID string `json:"account_id"` }` from `application/json` (strict, single object, `DisallowUnknownFields`, max 8 KiB) or `application/x-www-form-urlencoded`.
- Validates `CheckoutAPI.Currency`, `CheckoutAPI.Origin` (`http`/`https`, no userinfo/query/fragment/non-root path), and `CheckoutAPI.ReturnPath` (relative path starting with `/`, not `//`, defaulting to `"/"`).
- Starts a transaction, locks the caller's active cart, resolves or creates the `customers.Customer` record (requires a valid email, else `422`), and finds either the active non-empty cart or the most recent converted cart whose `ID` already has a `Payment` for that customer.
- Uses **durable `AttemptID = cart.ID`** when calling `flow.PrepareCheckout`.
- If `prepared.Cart != nil` (initial preparation):
  - Deactivates the cart (`Active = false`, assigns `User = user`) and invalidates any remaining unattached `coupons.Redemption` rows on that cart (`Order__isnull = true`, `Invalidated = false` -> `Invalidated = true`).
- Commits the preparation transaction before any external provider call.
- Appends `payment_id=<prepared.Payment.ID>` to `returnURL`:
  - **Free checkout (`prepared.Payment.IsFreeCheckout()`)**: also appends `order_id=<prepared.Order.ID>` and responds `303 See Other` directly to `returnURL` without calling any provider.
  - **Paid checkout**: builds `webhookURL` from `Origin` + `django.Reverse("api:payments:webhook", prepared.Payment.Account.ID)` and calls `flow.StartCheckout(...)` (`502` on error).
  - If `checkout.State` is already terminal/resolved (`PAID`, `CANCEL`, `EXPIRED`, `DENIED`, `FAILURE`, `REFUND`, `PARTIAL_REFUND`, `CHARGEBACK`), redirects `303 See Other` to `returnURL`.
  - Otherwise validates `checkout.URL` (`https`, non-empty host, no userinfo; else `502`) and redirects `303 See Other` to `checkout.URL`.

### 3. Provider Webhooks (`POST /api/payments/webhook/<<account>>`)

Calls `payments.ServeWebhook(w, r, accountID)` (exempt from CSRF middleware).

---

## `payments` Service Functions (`contrib/shop/payments/service.go`)

### Sentinel Errors

```go
var (
	ErrInvalidWebhook       = errors.New("invalid payment webhook")
	ErrUnsupportedOperation = errors.New("payment provider does not support operation")
	ErrOperationPending     = errors.New("payment operation requires reconciliation")
	ErrNonPositiveTotal     = errors.New("payment total must be positive")
)
```

### Attempt Creation

- `CreateAttempt(ctx, order, accountID, subtotal Money) (*Payment, error)`: creates a `Payment` with a new ULID, `State: STATUS_INIT`, quoted `Amount`/`Fee`, and `EventPrepared` history entry. Rejects non-positive `quote.Total` with `ErrNonPositiveTotal`.
- `CreateAttemptWithID(ctx, order, accountID, subtotal Money, id drivers.ULID) (*Payment, error)`: same as `CreateAttempt`, requiring a non-zero caller-supplied `id`.
- `CreateFreeCheckoutWithID(ctx, order, code string, id drivers.ULID) (*Payment, error)`: creates a `Payment` with `Provider: "shop:free-checkout"`, `State: paystate.STATUS_NO_PAYMENT_REQUIRED`, `Amount: 0`, `Fee: 0`, and `EventNoPaymentRequired` history entry. `Payment.IsFreeCheckout()` returns `true` for these records.

### Checkout, Refresh & Webhooks

- `Start(ctx, id, returnURL, webhookURL) (Checkout, error)`:
  - Returns immediately for `IsFreeCheckout()` or if `TransactionID.Valid` is already set (idempotent retry).
  - Requires `Account.Enabled == true`. Calls `provider.CreatePayment(ctx, CreatePaymentRequest{...})`.
  - On failure or invalid status, appends `EventCheckoutFailed` to `History` via `recordCheckoutFailure`.
  - On success, validates status, stores `TransactionID`, `CheckoutURL`, `State`, `AuthorizationRef`, `CaptureRef`, `ProviderFee`, appends `EventCheckoutOpened`, updates optimistically with `OperationVersion` + `TransactionID__isnull`, fires `payments.Signals.Start` (`PaymentStartedSignal`) inside the transaction, and commits.
- `Refresh(ctx, id) (*Payment, error)`: loads the attempt (even if the account is disabled), calls `provider.GetPayment(ctx, payment.TransactionID.String)` if `TransactionID.Valid`, and applies the status via `applyStatus`.
- `ServeWebhook(w, request, accountID)`: parses `accountID` as ULID (`404`), loads `ProviderAccount` (`404` if missing, `503` on DB/provider error), reads up to `1 MiB` of body (`400`), calls `provider.HandleWebhook` (`400` on `ErrInvalidWebhook`), reconciles the `Payment` matching `(Account, TransactionID)` via `applyStatus` (`503` on error), and invokes `result.Acknowledge(w)` or writes `200 OK`.

### Post-Payment Operations (`Refund`, `Void`, `Capture`, `Cancel`, `ResolveOperation`)

All four mutation operations (`Refund`, `Void`, `Capture`, `Cancel`) share the same durable two-phase flow:
1. `operationPayment` enforces `queries.IsCommitContext(ctx)` and `!queries.IsTransactionContext(ctx)`, validates `RequestID` (non-empty, `<= 128` bytes), rejects `IsFreeCheckout()` with `ErrUnsupportedOperation`, and resolves the `Provider`.
2. Checks the corresponding capability interface (`Refunder`, `Voider`, `Capturer`, `Canceller`); returns `ErrUnsupportedOperation` if not implemented.
3. Resolves the target resource ID via `OperationReferenceResolver` (or `payment.TransactionID.String` by default) and binds `request.ProviderRef`.
4. `replayOperation` checks `payment.Operations.Data[RequestID]`:
   - Different payload for the same `RequestID` -> error.
   - `OperationSucceeded` -> returns stored `OperationResult` immediately.
   - Any other existing phase or an in-flight `payment.OperationKey != ""` -> returns `ErrOperationPending`.
5. Precondition state checks:
   - `Refund`: `STATUS_PAID` or `STATUS_PARTIAL_REFUND`; currency must match and `refunded + amount <= payment.Amount`; reserves with state `STATUS_REFUNDING`.
   - `Void`: `STATUS_AUTHORIZE`; reserves keeping `payment.State`.
   - `Capture`: `STATUS_AUTHORIZE`; reserves keeping `payment.State`.
   - `Cancel`: `STATUS_INIT`, `STATUS_PENDING`, or `STATUS_VERIFY`; reserves keeping `payment.State`.
6. `reserveOperation` commits `OperationRecord` (`Phase: OperationRequested`), `OperationKey`, `History` (`EventOperation`), and `OperationVersion` before calling the provider, firing `Signals.OperationRequested` (and `Signals.Update` if state changed).
7. Calls the provider method. If it errors, `failOperation` marks the record `OperationUncertain` (keeping `OperationKey` reserved) and fires `Signals.OperationFailed`.
8. `finishOperation` records `OperationSucceeded`, `OperationFailed` (restoring `PreviousState`), or `OperationPending`:
   - On `OperationSucceeded`: `Void`/`Cancel` -> `STATUS_CANCEL`; `Capture` -> `STATUS_PAID` + `CaptureRef`; `Refund` -> `STATUS_PARTIAL_REFUND` or `STATUS_REFUND` (when total succeeded refunds equal `payment.Amount`). Clears `OperationKey`.
   - Fires `Signals.Refund`, `Signals.Void`, `Signals.Capture`, `Signals.Cancel`, `Signals.OperationPending`, or `Signals.OperationFailed`, plus `Signals.Update` if `State` changed, inside the commit transaction.
- `ResolveOperation(ctx, id, requestID, result)`: records a terminal `OperationSucceeded` or `OperationFailed` outcome for an ambiguous/pending operation and runs `finishOperation`.

### `canTransition(from, to paystate.PayState) bool`

State transitions enforced by `applyStatus`:
- Never transitions into `STATUS_NO_PAYMENT_REQUIRED`.
- From `STATUS_INIT`: any other state.
- From `STATUS_PENDING`: any state except `STATUS_INIT`.
- From `STATUS_VERIFY`, `STATUS_PARTIAL_PAYMENT`, `STATUS_AUTHORIZE`: any state except `STATUS_INIT` and `STATUS_PENDING`.
- From `STATUS_PARTLY_CAPTURED`: `STATUS_PAID` or `STATUS_VERIFY`.
- From `STATUS_PAID`, `STATUS_REFUNDING`, `STATUS_PARTIAL_REFUND`: `STATUS_REFUNDING`, `STATUS_PARTIAL_REFUND`, `STATUS_REFUND`, or `STATUS_CHARGEBACK`.
- All other states: terminal (`false`).

---

## Provider & Capability Interfaces (`contrib/shop/payments/payments.go`)

### Registration

- `type ProviderFactory func(context.Context, *ProviderAccount) (Provider, error)`
- `RegisterProvider(key string, factory ProviderFactory)` (panics on empty key/nil factory, reserved `"shop:free-checkout"`, or duplicate key)
- `ProviderRegistered(key string) bool`
- `(*ProviderAccount).Provider(ctx) (Provider, error)`

### Core & Optional Capability Interfaces

| Interface | Methods |
| --- | --- |
| `Provider` | `CreatePayment(context.Context, CreatePaymentRequest) (Checkout, error)`<br>`GetPayment(context.Context, string) (PaymentStatus, error)`<br>`HandleWebhook(context.Context, WebhookRequest) (WebhookResult, error)` |
| `Voider` | `VoidPayment(context.Context, OperationRequest) error` |
| `Refunder` | `RefundPayment(context.Context, RefundRequest) (OperationResult, error)` |
| `Capturer` | `CapturePayment(context.Context, OperationRequest) (OperationResult, error)` |
| `Canceller` | `CancelPayment(context.Context, OperationRequest) error` |
| `OperationReferenceResolver` | `OperationReferences(PaymentStatus) OperationReferences` |

### Request, Result, Money & Status Types

- `Money`: `Amount decimal.Decimal`, `Currency string`
- `CreatePaymentRequest`: `AttemptID drivers.ULID`, `Amount Money`, `ReturnURL string`, `WebhookURL string`
- `PaymentStatus`: `ProviderRef string`, `AuthorizationRef string`, `CaptureRef string`, `State paystate.PayState`, `Amount Money`, `ProviderFee *Money`
- `Checkout`: embeds `PaymentStatus`, plus `URL string`
- `WebhookRequest`: `Method string`, `Header http.Header`, `URL *url.URL`, `Body []byte`
- `WebhookResult`: `Payment PaymentStatus`, `Acknowledge func(http.ResponseWriter)`
- `OperationRequest`: `ProviderRef string`, `RequestID string` (`Validate()` requires both non-whitespace)
- `RefundRequest`: embeds `OperationRequest`, plus `Amount Money` (`Validate()` checks positive amount, valid ISO currency, and currency scale)
- `OperationResult`: `ProviderRef string`, `Status string`, `State OperationState`
- `OperationReferences`: `Refund string`, `Void string`, `Capture string`, `Cancel string`
- `OperationState` (`string`): `OperationRequested` (`"requested"`), `OperationUncertain` (`"uncertain"`), `OperationPending` (`"pending"`), `OperationSucceeded` (`"succeeded"`), `OperationFailed` (`"failed"`)
- `OperationKind` (`string`): `OperationRefund` (`"refund"`), `OperationVoid` (`"void"`), `OperationCapture` (`"capture"`), `OperationCancel` (`"cancel"`)
- Status string constants: `OperationStatusVoided = "VOIDED"`, `OperationStatusCancelled = "CANCELLED"`
