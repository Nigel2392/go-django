# Signals, Order Flow & Audit Logs

> [!IMPORTANT]
> **The `contrib/shop` order flow is primarily meant to be executed through signals.**
> Rather than hardcoding fulfillment, supplier notifications, inventory updates, or post-payment state transitions into core shop packages, `contrib/shop` exposes a shared typed signal pool (`shared/signals.Pool`) across `products`, `carts`, `coupons`, `orders`, and `payments`. Applications and extensions subscribe to these signals (such as `payments.Signals.Update` and `orders.Signals.Created`) to drive order acceptance, processing, and completion (`orders.Signals.Accepted`, `orders.Signals.Processed`, `orders.Signals.Completed`), while `handlers.RegisterAuditLogs()` demonstrates the built-in listener pattern.

---

## `shared/signals` (`contrib/shop/shared/signals/signals.go`)

```go
type BaseSignal struct {
	Request *http.Request
}

type Signal[T any] = signals.Signal[T]

type Changed[TYP any] struct {
	Old TYP
	New TYP
}

type ModelSignals[SIGNAL_T any] struct {
	Created signals.Signal[SIGNAL_T]
	Updated signals.Signal[SIGNAL_T]
	Deleted signals.Signal[SIGNAL_T]
}

var Pool = signals.NewGPool()
```

All shop signal managers retrieve typed `signals.Signal[T]` instances from `signals.Pool` using dotted string names. Returning an error from a listener registered on a signal that fires inside a database transaction aborts and rolls back that transaction.

---

## Complete Signal Reference

### `products.Signals` (`contrib/shop/products/signals.go`)

| Field | Pool Name | Payload Struct | When It Fires & Transaction Scope |
| --- | --- | --- | --- |
| `Created` | `"shop.products.Product.Created"` | `ProductCreatedSignal{ BaseSignal; Product *Product; Specific ProductModel }` | Admin product creation (`view_product.go` `SuccessFn`), **outside** the save transaction |
| `Updated` | `"shop.products.Product.Updated"` | `ProductUpdatedSignal{ BaseSignal; Product signals.Changed[*Product]; Specific signals.Changed[ProductModel] }` | Admin product edit (`view_product.go` `SuccessFn`), **outside** the save transaction |

### `carts.Signals` (`contrib/shop/carts/signals.go`)

Note: `CartAPISignal` and `CartItemAPISignal` do not embed `signals.BaseSignal`; they declare `Cart *Cart`, `Request *http.Request`, and `User authentication.User` directly.

| Field | Pool Name | Payload Struct | When It Fires & Transaction Scope |
| --- | --- | --- | --- |
| `Created` | `"shop.carts.Cart.Created"` | `CartAPISignal{ Cart *Cart; Request *http.Request; User authentication.User }` | `getOrCreateCart` (**outside** tx, after `CalcOneOrCreate` commits) and `AddToCart` (**inside** tx before `tx.Commit`) when a new cart is created |
| `Cleared` | `"shop.carts.Cart.Cleared"` | `CartAPISignal{ Cart *Cart; Request *http.Request; User authentication.User }` | `ClearCart` (**inside** tx after deactivating the cart and invalidating unattached coupon redemptions) |
| `Item.Add` | `"shop.carts.CartItem.Add"` | `CartItemAPISignal{ CartAPISignal; CartItem *CartItem }` | `AddToCart` (**inside** tx before `tx.Commit`) |
| `Item.Remove` | `"shop.carts.CartItem.Remove"` | `CartItemAPISignal{ CartAPISignal; CartItem *CartItem }` | `RemoveFromCart` (**inside** tx before `tx.Commit`) |

### `coupons.Signals` (`contrib/shop/coupons/signals.go`)

| Field | Pool Name | Payload Struct | When It Fires & Transaction Scope |
| --- | --- | --- | --- |
| `Created` | `"shop.coupons.Coupon.Created"` | `CouponCreatedSignal{ BaseSignal; Coupon *Coupon }` | Declared for coupon creation listeners (registered in `handlers`) |
| `Updated` | `"shop.coupons.Coupon.Updated"` | `CouponUpdatedSignal{ BaseSignal; Coupon *Coupon }` | Declared for coupon update listeners (registered in `handlers`) |
| `Redeemed` | `"shop.coupons.Coupon.Redeemed"` | `CouponRedeemedSignal{ BaseSignal; Coupon *Coupon; Cart attrs.Definer }` | `RedemptionAPI.RedeemCode` (`POST /api/coupons/redeem`), **inside** the redemption transaction before `tx.Commit` |

### `orders.Signals` (`contrib/shop/orders/signals.go`)

| Field | Pool Name | Payload Struct | When It Fires & Transaction Scope |
| --- | --- | --- | --- |
| `Created` | `"shop.orders.Order.Created"` | `OrderCreatedSignal{ BaseSignal; Order *Order; Lines []*OrderItem }` | `orders.CreateOrder` / `CreateOrderFromCart` (**inside** tx before `tx.Commit`) and admin order add (`view_order.go`, **inside** form save tx) |
| `Updated` | `"shop.orders.Order.Updated"` | `OrderUpdatedSignal{ BaseSignal; Order *Order; Lines []*OrderItem }` | Admin order edit (`view_order.go`, **inside** form save tx after `ReconcileCouponRedemptions`) |
| `Accepted` | `"shop.orders.Order.Accepted"` | `OrderAcceptedSignal{ BaseSignal; Order *Order; Lines []*OrderItem }` | Workflow signal for order acceptance (listened to by `handlers`, emitted by application/workflow handlers) |
| `Processed` | `"shop.orders.Order.Processed"` | `OrderProcessedSignal{ BaseSignal; Order *Order }` | Workflow signal for order processing (listened to by `handlers`, emitted by application/workflow handlers) |
| `Completed` | `"shop.orders.Order.Completed"` | `OrderCompletedSignal{ BaseSignal; Order *Order }` | Workflow signal for order completion (listened to by `handlers`, emitted by application/workflow handlers) |

### `payments.Signals` (`contrib/shop/payments/signals.go`)

| Field | Pool Name | Payload Struct | When It Fires & Transaction Scope |
| --- | --- | --- | --- |
| `Start` | `"shop.payments.Payment.Start"` | `PaymentStartedSignal{ BaseSignal; Payment *Payment }` | `payments.Start` (**inside** tx when provider checkout opens) |
| `Update` | `"shop.payments.Payment.Update"` | `PaymentUpdatedSignal{ BaseSignal; Payment *Payment; PreviousState string }` | `reserveOperation` (if state changed, **inside** tx), `finishOperation` (if state changed, **inside** tx), and `applyStatus` via `Refresh` / `ServeWebhook` (if status/refs changed, **inside** tx) |
| `Refund` | `"shop.payments.Payment.Refund"` | `PaymentRefundedSignal{ BaseSignal; Payment *Payment; RequestID string; Amount Money; Result OperationResult }` | `finishOperation` when `OperationRefund` reaches `OperationSucceeded` (**inside** tx) |
| `Void` | `"shop.payments.Payment.Void"` | `PaymentVoidedSignal{ BaseSignal; Payment *Payment; RequestID string }` | `finishOperation` when `OperationVoid` reaches `OperationSucceeded` (**inside** tx) |
| `Capture` | `"shop.payments.Payment.Capture"` | `PaymentOperationSignal{ BaseSignal; Payment *Payment; RequestID string; Operation OperationRecord; Error error }` | `finishOperation` when `OperationCapture` reaches `OperationSucceeded` (**inside** tx) |
| `Cancel` | `"shop.payments.Payment.Cancel"` | `PaymentOperationSignal{ BaseSignal; Payment *Payment; RequestID string; Operation OperationRecord; Error error }` | `finishOperation` when `OperationCancel` reaches `OperationSucceeded` (**inside** tx) |
| `OperationRequested` | `"shop.payments.Payment.OperationRequested"` | `PaymentOperationSignal` | `reserveOperation` before calling the provider (**inside** tx) |
| `OperationPending` | `"shop.payments.Payment.OperationPending"` | `PaymentOperationSignal` | `finishOperation` when `result.State == OperationPending` (**inside** tx) |
| `OperationFailed` | `"shop.payments.Payment.OperationFailed"` | `PaymentOperationSignal` | `failOperation` (`Phase: OperationUncertain`, **outside** tx after update) and `finishOperation` when `result.State == OperationFailed` (**inside** tx) |

---

## Order Creation (`orders.CreateOrder` & `orders.CreateOrderFromCart`)

Implemented in `contrib/shop/orders/models_order_create.go`:

- `CreateOrder(r *http.Request, customer *customers.Customer, cart *carts.Cart, items ...*carts.CartItem) (*Order, []*OrderItem, error)` creates an order from a cart and optional explicit item list.
- `CreateOrderFromCart(r *http.Request, customer *customers.Customer, cart *carts.CartRow) (*Order, []*OrderItem, error)` requires a non-nil priced `*carts.CartRow` (typically from `CartQuery.CalcOneWithProducts()`) with at least one item.

Inside a database transaction:
1. Initializes `&Order{ State: orderStatePending, Customer: customer }`.
2. Loads each item's specific `products.ProductModel` via `SpecificContentType` and `SpecificID`.
3. Builds `OrderItem` rows with a `ProductData` JSON snapshot (`Product` + `Specific` `ProductModel`), `ItemPrice`, `TaxRate`, `Amount`, `Supplier`, and (when `pricedCart != nil`) `Pricing.Data = &OrderItemPricing{ Totals: line.CalculatedPrice, RedemptionAmounts: maps.Clone(line.RedemptionAmounts) }`.
4. Persists the `Order` and bulk-creates the `OrderItem` rows.
5. If `pricedCart != nil && len(pricedCart.RedemptionAmounts) != 0`, iterates `pricedCart.RedemptionAmounts` in **sorted redemption ID order** and updates each `coupons.Redemption` (`ID = id`, `Cart = cart.ID`, `Invalidated = false`, `Order__isnull = true`) with `Order`, `Customer`, and `DiscountAmount`. If `count != 1`, aborts with `orders.ErrCartRedemptionsChanged`.
6. Fires `orders.Signals.Created` (`OrderCreatedSignal{ Request: r.WithContext(ctx), Order: order, Lines: orderItems }`) **inside the transaction** before `tx.Commit(ctx)`.

---

## Order States & Transitions (`contrib/shop/orders/models_order_state.go`)

`Order.State` has type `InternalOrderState` (`string`), and `Order` also tracks boolean flags `Processed` and `Completed`. `OrderState` is a model with `ID`, `Label`, `Action InternalOrderState`, and `WorkflowID drivers.ULID`.

`InternalOrderState` constants (package-internal, exposed in admin options and labels):
- `orderStatePending` (`"order_pending"` — `"Order Pending"`): initial state set by `createOrder`.
- `orderStateAccepted` (`"order_accepted"` — `"Order Accepted"`).
- `orderStateRejected` (`"order_rejected"` — `"Order Rejected"`).
- `orderStateWorkflow` (`"order_workflow"` — `"Order Workflow"`).

Transitions happen via:
- **Signal handlers** listening to `payments.Signals.Update` / `orders.Signals.*` to advance `Order.State`, `Processed`, and `Completed`, and emitting `orders.Signals.Accepted`, `orders.Signals.Processed`, and `orders.Signals.Completed`.
- **Admin order edit** (`view_order.go`), which saves `Order`, runs `orders.ReconcileCouponRedemptions(ctx, p.ID)`, and fires `orders.Signals.Updated` in the same transaction.
  - When `Order.State == orderStateRejected`, `ReconcileCouponRedemptions` invalidates pending coupon reservations (`Invalidated = true`).
  - When `Order.Completed == true`, `ReconcileCouponRedemptions` marks pending coupon reservations `Counted = true` and increments `Coupon.UsageCount`.

---

## Audit Logs (`contrib/shop/handlers/`)

`ShopAppConfig.Init` in `contrib/shop/app.go` calls `handlers.RegisterAuditLogs()`, which uses `sync.Once` to register all `auditlogs.Definition` types and attach listeners to every shop signal. `logAudit` silently ignores `auditlogs.ErrLogsNotReady`.

### Registered Audit Log Definitions & Listeners

| Audit Kind | Signal Listened | Level | `Definition` Struct | Data Keys Recorded |
| --- | --- | --- | --- | --- |
| `"shop:product:add"` | `products.Signals.Created` | `INF` | `productAddedDefinition` | `title` |
| `"shop:product:edit"` | `products.Signals.Updated` | `INF` | `productEditedDefinition` | `title`, `changes` |
| `"shop:carts:create"` | `carts.Signals.Created` | `INF` | `cartCreatedDefinition` | _(none)_ |
| `"shop:carts:clear"` | `carts.Signals.Cleared` | `INF` | `cartClearedDefinition` | _(none)_ |
| `"shop:carts:items:add"` | `carts.Signals.Item.Add` | `INF` | `cartItemAddedDefinition` | `cart_id` |
| `"shop:carts:items:remove"` | `carts.Signals.Item.Remove` | `INF` | `cartItemRemovedDefinition` | `cart_id` |
| `"shop:coupons:create"` | `coupons.Signals.Created` | `INF` | `couponCreatedDefinition` | `title` |
| `"shop:coupons:update"` | `coupons.Signals.Updated` | `INF` | `couponUpdatedDefinition` | `title` |
| `"shop:coupons:redeem"` | `coupons.Signals.Redeemed` | `INF` | `couponRedeemedDefinition` | `title`, `cart_id` |
| `"shop:orders:create"` | `orders.Signals.Created` | `INF` | `orderCreatedDefinition` | `state`, `line_count` |
| `"shop:orders:update"` | `orders.Signals.Updated` | `INF` | `orderUpdatedDefinition` | `state`, `line_count` |
| `"shop:orders:accept"` | `orders.Signals.Accepted` | `INF` | `orderAcceptedDefinition` | `state`, `line_count` |
| `"shop:orders:process"` | `orders.Signals.Processed` | `INF` | `orderProcessedDefinition` | `state` |
| `"shop:orders:complete"` | `orders.Signals.Completed` | `INF` | `orderCompletedDefinition` | `state` |
| `"shop:payments:start"` | `payments.Signals.Start` | `paymentStateLevel` (`ERR` for `FAILURE`, `WRN` for `DENIED`/`VERIFY`/`PAID_CHECKAMOUNT`/`CHARGEBACK`, else `INF`) | `paymentStartedDefinition` | `state`, `amount`, `currency`, `provider`, `account_id`, `order_id` |
| `"shop:payments:update"` | `payments.Signals.Update` | `paymentStateLevel` | `paymentUpdatedDefinition` | `paymentAudit` keys + `previous_state` |
| `"shop:payments:refund"` | `payments.Signals.Refund` | `INF` | `paymentRefundedDefinition` | `paymentAudit` keys + `request_id`, `refund_amount`, `refund_currency` |
| `"shop:payments:void"` | `payments.Signals.Void` | `INF` | `paymentVoidedDefinition` | `paymentAudit` keys + `request_id` |
| `"shop:payments:capture"` | `payments.Signals.Capture` | `INF` | `paymentCapturedDefinition` | `paymentAudit` keys + `request_id`, `operation`, `phase`, `operation_amount`, `operation_currency` |
| `"shop:payments:cancel"` | `payments.Signals.Cancel` | `INF` | `paymentCancelledDefinition` | `paymentOperationAudit` keys |
| `"shop:payments:operation:requested"` | `payments.Signals.OperationRequested` | `INF` | `paymentOperationRequestedDefinition` | `paymentOperationAudit` keys |
| `"shop:payments:operation:pending"` | `payments.Signals.OperationPending` | `DBG` | `paymentOperationPendingDefinition` | `paymentOperationAudit` keys |
| `"shop:payments:operation:failed"` | `payments.Signals.OperationFailed` | `WRN` if `OperationUncertain`, else `ERR` | `paymentOperationFailedDefinition` | `paymentOperationAudit` keys |

Shared action helpers in `handlers/auditlogs_definition.go`:
- `shopEditAuditAction`: builds an `"admin:apps:model:edit"` action with `?next=<r.URL.Path>` when the user has `admin:edit` and any extra required permissions (`"products:edit"`, `"orders:edit"`).
- `shopListAuditAction`: builds an `"admin:apps:model"` action when the user has `admin:view_list` (used by cart and payment audit entries).
