# Coupons, Redemptions & Discount Calculation

The `coupons` package (`contrib/shop/coupons`) and its `generators` subpackage (`contrib/shop/coupons/generators`) implement promotional coupon codes, code generation, cart-level reservation and `UsageLimit` accounting, deterministic discount calculation in `carts.CartQuery.Calc()`, and order-level redemption reconciliation.

---

## Models

### `Coupon` (`coupons/models_coupon.go`)

`CouponCode` is a defined string type (`type CouponCode string`) with package-level length variables:
- `MIN_CODE_LENGTH int64 = 6`
- `MAX_CODE_LENGTH int64 = 32`

| Field | Type | Constraints / Notes |
| --- | --- | --- |
| `ID` | `uint64` | Primary key, read-only |
| `Title` | `string` | `MinLength: 2`, `MaxLength: 75`, `Blank: false` |
| `ShortDescription` | `string` | `Blank: true` |
| `Code` | `CouponCode` | `MinLength: 6`, `MaxLength: 32`, `Blank: false`; registered with `CouponGeneratorWidget` |
| `IsPercentage` | `bool` | `Blank: true`; `false` = fixed currency discount, `true` = percentage discount |
| `MaxDiscount` | `decimal.Decimal` | `Precision: 13`, `Scale: 4`, `Null: false`, `Blank: false` |
| `MinOrderAmount` | `decimal.Decimal` | `Precision: 13`, `Scale: 4`, `Null: false`, `Blank: true` |
| `UsageLimit` | `uint64` | `Blank: true`; `0` means unlimited |
| `UsageCount` | `uint64` | Read-only; incremented when completed orders consume reservations |
| `ValidFrom` | `time.Time` | `Null: true`, `Blank: true` |
| `ValidUntil` | `time.Time` | `Null: true`, `Blank: true` |
| `CreatedAt` | `time.Time` | Read-only, `AutoNowAdd: true` |
| `UpdatedAt` | `time.Time` | Read-only, `AutoNow: true` |
| `CreatedBy` | `users.User` | Column `user_id`, read-only, `Null: true`, FK to `users.MODEL_KEY` |
| `RedemptionSet` | `*queries.RelRevFK[*Redemption]` | Reverse relation to `Redemption` |
| `Products` | `*queries.RelM2M[*products.Product, *CouponProductThrough]` | `Blank: true`, `ReverseAlias: "Coupons"`, through `CouponProductThrough`. Empty = cart-wide coupon; non-empty = product-specific coupon |

### `CouponProductThrough` (`coupons/models_coupons_products.go`)

Through model linking `*Coupon` (`coupon_id`, read-only, `Null: false`, `Blank: false`) and `*products.Product` (`product_id`, uses `chooser.NewChooserWidget` in forms).

### `Redemption` (`coupons/models_redemption.go`)

Represents a coupon reservation on a cart and, once converted, an applied discount on an order.

| Field | Type | Constraints / Notes |
| --- | --- | --- |
| `ID` | `uint64` | Primary key, read-only |
| `Coupon` | `*Coupon` | Column `coupon_id`, read-only, `Null: false`, `Blank: false` |
| `Customer` | `*customers.Customer` | Column `customer_id`, read-only, `Null: true`, `Blank: true` |
| `Cart` | `attrs.Definer` | Column `cart_id`, read-only, `Null: true`, `Blank: true`, `ReverseAlias: "Redemptions"`, deferred FK to `modelkeys.CartModelKey` |
| `Order` | `attrs.Definer` | Column `order_id`, read-only, `Null: true`, `Blank: true`, `ReverseAlias: "Redemptions"`, deferred FK to `modelkeys.OrderModelKey` |
| `DiscountAmount` | `decimal.Decimal` | `Precision: 19`, `Scale: 10`, read-only, `Null: false`, `Blank: false` |
| `Invalidated` | `bool` | Read-only, `Blank: true`; `true` when released (e.g., cart cleared, unapplied coupon at checkout, or order rejected) |
| `Counted` | `bool` | Read-only, `Blank: true`, `Default: false`; `true` once included in `Coupon.UsageCount` |
| `RedeemedAt` | `time.Time` | Read-only |

---

## Code Generation (`coupons/generators`)

### `CodePartType` & `CodePart` (`coupons/generators/generators.go`)

`CodePartType` (`uint`) defines building blocks for composable coupon codes:

| Constant | String() | Description |
| --- | --- | --- |
| `CPT_RANDOM` | `"CPT_RANDOM"` | Random uppercase ASCII letters (`A-Z`) of length `Length` |
| `CPT_STATIC` | `"CPT_STATIC"` | Literal `Value` string |
| `CPT_YEAR` | `"CPT_YEAR"` | Current year (`time.Now().Year()`) |
| `CPT_MONTH` | `"CPT_MONTH"` | Current month (`int(t.Month())`) |
| `CPT_WEEKDAY` | `"CPT_WEEKDAY"` | Current weekday (`int(t.Weekday())`) |
| `CPT_DAY` | `"CPT_DAY"` | Current day of month (`t.Day()`) |
| `CPT_HOUR` | `"CPT_HOUR"` | Current hour (`t.Hour()`) |
| `CPT_MINUTE` | `"CPT_MINUTE"` | Current minute (`t.Minute()`) |
| `CPT_DASH` | `"CPT_DASH"` | Literal `"-"` delimiter |
| `CPT_UUID` | `"CPT_UUID"` | Random UUID v4 with dashes stripped, optionally truncated to `Length` |

```go
type CodePart struct {
	Type   CodePartType `json:"type"`
	Length int          `json:"length"` // used for CPT_RANDOM and CPT_UUID
	Value  string       `json:"value"`  // used for CPT_STATIC
}
```

### `GenerateCouponCode`

```go
func GenerateCouponCode(ctx context.Context, parts []CodePart) (code string, err error)
```

- Builds a candidate code from `parts` and validates length against `coupons.MIN_CODE_LENGTH` (6) and `coupons.MAX_CODE_LENGTH` (32).
- Checks `coupons.Coupon` existence in the database (`Filter("Code", code).Exists()`).
- If the code already exists and `parts` contains at least one dynamic part (`CPT_RANDOM` with `Length > 0`, or `CPT_UUID`), it retries until a unique code is found; if all parts are static, it returns `ErrCouponExists`.
- Sentinel errors (`errs.Error`): `ErrInvalidPartType`, `ErrCouponCodeTooShort`, `ErrCouponCodeTooLong`, `ErrCouponExists`.

### `CouponGeneratorWidget` (`coupons/generators/widget.go`)

- `init()` registers `CouponGeneratorWidget` as the default form field widget for `coupons.CouponCode` via `attrs.RegisterFormFieldType`.
- Renders template `shop/admin/widgets/coupon_generator.tmpl` with CSS `shop/admin/css/widgets/coupon-generator.css` and JS `shop/admin/js/coupon-generator.js`.
- `GetContextData` provides `generator.url` (`django.Reverse("admin:ext:shop:generate_coupon")`), current `generator.time` components, and translated `generator.i18n` labels for all `CPT_*` part types.

---

## Coupon Redemption (`POST /api/coupons/redeem`)

Handled by `public.RedemptionAPI.RedeemCode` in `contrib/shop/views/public/coupons.go`.

### Request & Anti-Enumeration

- Decodes `RedemptionRequest{ CouponCode string `json:"coupon_code"` }` via `api.Decode`. Returns `400 Bad Request` on decode failure.
- To make coupon code enumeration harder, failed coupon lookups and unexpected redemption query errors invoke `laggy()` (random sleep between `5ms` and `100ms`) before returning `404 Not Found`.

### Transactional Reservation & `UsageLimit` Accounting

`RedeemCode` executes inside a database transaction with deterministic row locking to prevent oversubscription:

1. **Pre-lock lookup**: Looks up the candidate `Coupon` ID by `Code` and the active `Cart` ID via `cartQuery(r.Context(), r, false)`. Returns `404` if either is missing.
2. **Lock order**: Starts a transaction, locks the active `Cart` row first (matching checkout's lock order via `BulkUpdate(expr.As("Active", expr.Value(true)))`), then locks the `Coupon` row (`BulkUpdate(expr.As("UsageCount", expr.Field("UsageCount")))`) before reading capacity.
3. **Preload & capacity check**: Loads `Coupon` with `Products`.
   - If `coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit`, returns `404` (`ErrRedeemed`).
   - Counts pending (uncounted, non-invalidated) reservations:
     ```go
     Filter(expr.Q("Invalidated", false), expr.Q("Coupon", coupon), expr.Q("Counted", false)).Count()
     ```
   - If `uint64(totalRedemptions) >= coupon.UsageLimit - coupon.UsageCount`, returns `404` (`ErrRedeemed`).
4. **Duplicate redemption check**: Checks whether an active (`Invalidated == false`) `Redemption` already exists for `(Coupon, Cart)`. If `exists`, returns `409 Conflict` (`ErrInvalid`).
5. **Product matching**: If `len(coupon.Products.AsList()) > 0`, verifies that at least one `CartItem.Product` in the cart matches a product bound to the coupon; otherwise returns `404` (`ErrNoMatchingProducts`).
6. **Create reservation & recalculate**:
   - Inserts `&coupons.Redemption{ Coupon: coupon, Cart: cartRow.Object, RedeemedAt: time.Now() }` (`Counted: false`, `Invalidated: false`).
   - Calls `cartQ.CalcOne()` to return the updated `*carts.CartRow` JSON.
   - Fires `coupons.Signals.Redeemed` (`CouponRedeemedSignal`) **inside the transaction** before `tx.Commit(ctx)`.

---

## Discount Calculation Rules (`CartQuery.Calc()`)

Implemented in `contrib/shop/carts/models_cart_query.go`. `Calc()` and `CalcOne()` / `CalcOneWithProducts()` preload `Customer` (`SelectRelated("Customer")`), load cart items ordered by `Product.Price, ID` (ascending item price), and load active cart redemptions (`Invalidated == false`, `Cart__in`, `Order__isnull == true`) with `Coupon` and `Coupon.Products`.

### 1. Grouping Redemptions

For each redemption whose `Coupon != nil` and `Coupon.MaxDiscount.IsPositive()`:
- **Cart-wide** (`len(coupon.Products.AsList()) == 0`):
  - Added to `cartCouponsPercentage` if `IsPercentage == true`, else `cartCouponsFixed`.
  - Appended to `cartRow.Coupons`.
- **Product-specific** (`len(coupon.Products.AsList()) > 0`):
  - Deduplicated per product ID in the cart; added to `cartItemCouponsPercentage[productID]` or `cartItemCouponsFixed[productID]`.
  - Appended to `cartRow.Coupons` only if at least one bound product is present in the cart.

### 2. Priority Ordering (`compareRedemptionPriority`)

Each of the four groups (`cartCouponsFixed`, `cartCouponsPercentage`, and per-product `cartItemCouponsFixed[pid]` / `cartItemCouponsPercentage[pid]`) is sorted using `compareRedemptionPriority`:
1. Larger `Coupon.MaxDiscount` comes first (`b.Coupon.MaxDiscount.Cmp(a.Coupon.MaxDiscount)`).
2. Ties are broken by ascending `Redemption.ID` (`cmp.Compare(a.ID, b.ID)`).

### 3. Fixed Discounts (Product-Specific Before Cart-Wide)

- A per-cart remaining balance map `cartFixedBalance[redemption.ID] = redemption.Coupon.MaxDiscount` is initialized for `cartCouponsFixed`.
- Items are processed in `Product.Price, ID` ascending order.
- For each line item, `fixedBudget = totalCartFixedBalance + sum(fixedCoupons.MaxDiscount)` is passed to `price.Price.Calculate` (with `ItemCount: int(item.Amount)`).
- `unallocated := calculatedPrice.TotalDiscount` is attributed to redemptions in order:
  1. **Product-specific fixed coupons** (`cartItemCouponsFixed[productID]`): each takes `min(unallocated, redemption.Coupon.MaxDiscount)`.
  2. **Cart-wide fixed coupons** (`cartCouponsFixed`): each takes `min(unallocated, cartFixedBalance[redemption.ID])`, decrementing `cartFixedBalance[redemption.ID]` and `totalCartFixedBalance` across items.

### 4. Percentage Discounts & 100% Cap

- After fixed discounts are applied to the line, percentage discounts apply to the **post-fixed-discount** `calculatedPrice.TotalPrice` (preserving `calculatedPrice.TotalFee` and recalculating tax on both) when `item.Amount > 0`.
- Product-specific percentage coupons (`cartItemCouponsPercentage[productID]`) are processed first, followed by cart-wide percentage coupons (`cartCouponsPercentage`).
- Cumulative percentage is capped at `100` (`percentage = decimal.Min(100, percentage + redemption.Coupon.MaxDiscount)`).
- Each coupon's attributed `applied` amount is computed from cumulative `calculatedPrice.TotalPrice * percentage / 100 - previousCumulativeDiscount` so allocations sum cleanly even when division rounds.
- If `percentage.IsPositive()`, `remainingPrice.Calculate` runs with `ItemCount: 1` and `Discount: {Amount: percentage, IsPercentage: true}`, and the resulting `TotalPrice`, `TotalFee`, `TotalTax` replace the line's values while `TotalDiscount` adds `pctCalculated.TotalDiscount`.

### 5. Per-Line & Cart-Level `RedemptionAmounts`

- Each `CartItemRow.RedemptionAmounts` (`map[uint64]decimal.Decimal`, `json:"-"`) stores the positive discount amount applied by each `Redemption.ID` on that line.
- Each `CartRow.RedemptionAmounts` (`map[uint64]decimal.Decimal`, `json:"-"`) accumulates `CartItemRow.RedemptionAmounts` across all lines in the cart.

---

## Order Conversion & Redemption Reconciliation

### Order Conversion (`orders.CreateOrderFromCart` / `ErrCartRedemptionsChanged`)

In `contrib/shop/orders/models_order_create.go`:
- When `CreateOrderFromCart` converts a priced `*carts.CartRow` into an `*Order`, it stores `OrderItemPricing{ Totals: line.CalculatedPrice, RedemptionAmounts: maps.Clone(line.RedemptionAmounts) }` on each `OrderItem`.
- For each `Redemption.ID` in `pricedCart.RedemptionAmounts` (iterated in sorted ID order to prevent lock-order deadlocks), it updates the `Redemption` row where `ID = id`, `Cart = cart.ID`, `Invalidated = false`, and `Order__isnull = true`, setting `Order`, `Customer`, and `DiscountAmount`.
- If any update affects `count != 1` rows (e.g., the cart's redemptions changed concurrently), order creation aborts and returns `orders.ErrCartRedemptionsChanged`.

### `orders.ReconcileCouponRedemptions` (`contrib/shop/orders/redemptions.go`)

```go
func ReconcileCouponRedemptions(ctx context.Context, orderID drivers.ULID) error
```

Accounts for coupon redemptions on a saved order. It requires a writable transaction (`queries.IsCommitContext(ctx)`), starts/joins a transaction, and locks the `Order` row (`BulkUpdate(expr.As("Completed", expr.Field("Completed")))`) before reading `State` and `Completed`.

It operates on pending redemptions (`Order = orderID`, `Invalidated = false`, `Counted = false`):
- **Rejected orders (`order.State == orders.OrderStateRejected`)**: marks pending redemptions `Invalidated = true`, releasing their reserved capacity.
- **Completed orders (`order.Completed == true`)**: iterates pending redemptions ordered by `Coupon, ID`; for each redemption, locks the `Coupon` row (`BulkUpdate(expr.As("UsageCount", expr.Field("UsageCount")))`), sets `Counted = true` on the redemption, and if updated (`count == 1`), increments `Coupon.UsageCount` by `1`.
- Idempotent: calling `ReconcileCouponRedemptions` again after redemptions are marked `Invalidated` or `Counted` is a no-op.
- Called automatically by both the admin order editor (`view_order.go`) and the built-in `orders.Signals.Completed` fulfillment workflow (`orders/supplier_workflow.go`).
