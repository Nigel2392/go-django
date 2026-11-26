# Shop HTTP API

This document describes the `shared/api` request/response helpers and every public and admin HTTP endpoint exposed by `contrib/shop`.

---

## Route Registration & CSRF Rules

### Public API Routes (`SHOP.Ready`)

In `contrib/shop/app.go`, `SHOP.Ready` mounts all public endpoints under the `/api` route group (`"api"`):

| Route Name | Method | URL Pattern | Handler |
|---|---|---|---|
| `api:carts` | `GET` | `/api/carts` | `public.CartAPI.GetOrCreateCart` |
| `api:carts:clear` | `POST` | `/api/carts/clear` | `public.CartAPI.ClearCart` |
| `api:carts:add_item` | `POST` | `/api/carts/items/add` | `public.CartAPI.AddCartItem` |
| `api:carts:remove_item` | `POST` | `/api/carts/items/remove` | `public.CartAPI.RemoveCartItem` |
| `api:coupons:redeem` | `POST` | `/api/coupons/redeem` | `public.RedemptionAPI.RedeemCode` |
| `api:payments:options` | `GET` | `/api/payments/options` | `public.PaymentsAPI.PaymentOptions` |
| `api:payments:checkout` | `POST` | `/api/payments/checkout` | `public.CheckoutAPI.Checkout` |
| `api:payments:webhook` | `POST` | `/api/payments/webhook/<<account>>` | `payments.ServeWebhook` |

### Admin API Routes (`app.Routing`)

In `contrib/shop/app.go`, `app.Routing` mounts the shop admin API under the admin multiplexer:

| Route Name | Method | URL Pattern (under admin) | Handler |
|---|---|---|---|
| `admin:ext:shop:generate_coupon` | `POST` | `/shop/coupon/generate` | `adminviews.GenerateCouponCode` |

### CSRF & Authentication Summary

- **CSRF (`nosurf`)**: All state-changing (`POST`) endpoints are protected by `nosurf` CSRF validation by default (send `X-CSRF-Token` header or form field).
- **CSRF Exemption**: `SHOP.Ready` registers a `django.HOOK_SETUP_NOSURF` hook that exempts the payment provider webhook path:
  ```regex
  ^/api/payments/webhook/[0-9A-HJKMNP-TV-Z]{26}/?$
  ```
- **Session Middleware**: Cart endpoints (`/api/carts*`) and coupon redemption (`/api/coupons/redeem`) require session middleware (`sessions.Retrieve(r)`). Carts are scoped to the active session token and/or authenticated user (`carts.Query().CartForRequest(r)`).
- **Authentication**:
  - Cart, coupon redemption, and payment options endpoints work for both anonymous (session-backed) and authenticated users.
  - `POST /api/payments/checkout` **requires** an authenticated user (`401 Unauthorized` otherwise).
  - `POST .../shop/coupon/generate` **requires** the `"coupons:generate"` object permission on `&coupons.Coupon{}` (redirects/renders `admin.ReLogin` otherwise).

---

## `shared/api` Package

Package `github.com/Nigel2392/go-django/contrib/shop/shared/api` provides shared JSON decoding, encoding, error formatting, and request lifecycle hooks.

### `APIRequest` & `BaseAPIRequest`

```go
type APIRequest interface {
	Started() error
	Finished() error
	HttpRequest() *http.Request
}

type BaseAPIRequest struct {
	*http.Request
}
```

- `BaseAPIRequest.Started()` returns `nil`.
- `BaseAPIRequest.Finished()` returns `nil`.
- `BaseAPIRequest.HttpRequest()` returns `s.Request`.

### `RequestRegistry`

```go
type RequestRegistry struct {
	fallback func(*http.Request, string) APIRequest
	registry map[string]func(*http.Request, string) APIRequest
	mu       *sync.RWMutex
}

func NewRequestRegistry(fallback func(*http.Request, string) APIRequest) RequestRegistry
func (reg *RequestRegistry) RequestStarted(r *http.Request, apiName string) (APIRequest, error)
func (reg *RequestRegistry) RegisterNewAPIRequest(apiName string, newRequest func(*http.Request, string) APIRequest)
```

- `RegisterNewAPIRequest` registers a factory for `apiName`. Panics if `apiName` is already registered.
- `RequestStarted` looks up `apiName` under a read lock. If found and the factory returns a non-nil `APIRequest`, it calls and returns `req, req.Started()`. Otherwise it returns `&BaseAPIRequest{Request: r}, nil`.

### `Decode`, `Respond`, `ErrorResponse`, `RespondError`

- **`Decode(r *http.Request, dst any) error`**: Closes `r.Body` on return and decodes JSON into `dst` with `dec.DisallowUnknownFields()`. Unknown JSON fields cause a decode error.
- **`Respond(w http.ResponseWriter, data any) error`**: Sets `Content-Type: application/json` and writes pretty-printed JSON (2-space indentation) of `data`.
- **`ErrorResponse`**:
  ```go
  type ErrorResponse struct {
      Message string `json:"message"`
  }
  ```
- **`RespondError(w http.ResponseWriter, statusCode int, message string, args ...any)`**: Writes `statusCode`, sets `Content-Type: application/json`, formats `message` with `fmt.Sprintf` if `args` are provided, and encodes `ErrorResponse{Message: message}`. Calls `except.Fail(http.StatusInternalServerError, ...)` if JSON encoding fails.

---

## Common JSON Schemas

### `Totals` (`price.Totals`)

Decimal values are serialized as strings (`shopspring/decimal` default).

```json
{
  "total_price": "100.00",
  "total_fee": "0.00",
  "total_tax": "21.00",
  "total_discount": "10.00",
  "left_over_discount": "0.00"
}
```

### `Cart` (`carts.Cart`)

```json
{
  "id": "01JABCDEF...",
  "created_at": "2026-01-01T12:00:00Z",
  "active": true
}
```
*(Fields `User`, `Session`, `CartItems`, `Redemptions` have `json:"-"`.)*

### `Product` (`products.Product`)

```json
{
  "id": "01JABCDEF...",
  "title": "Example Product",
  "image": { ... },
  "slug": "example-product",
  "ranking": 1,
  "price": "50.00",
  "additional_fee": "0.00",
  "tax_rate": "21.00",
  "discount_price": "0.00",
  "supplier": "default",
  "category": { ... },
  "editor": { ... }
}
```
*(Most fields use `omitempty`; cart preloads select `ID`, `Title`, `Image`, `Price`, `TaxRate`, `DiscountPrice`.)*

### `CartItem` (`carts.CartItem`)

```json
{
  "id": "01JABCDEF...",
  "product": { ... },
  "amount": 2
}
```

### `CartItemRow` (`carts.CartItemRow`)

```json
{
  "item": { ... },
  "coupons": [ ... ],
  "total_price": { ... }
}
```

### `CartRow` (`carts.CartRow`)

```json
{
  "cart": { ... },
  "items": [ ... ],
  "coupons": [ ... ],
  "total_price": { ... }
}
```

> **Note on `Coupon` and `Redemption` JSON casing:** `coupons.Coupon` and `coupons.Redemption` structs do not define lowercase `json:` struct tags, so their Go field names (`ID`, `Title`, `Code`, `DiscountAmount`, `Invalidated`, `RedeemedAt`, etc.) are preserved in JSON output.

---

## Public Endpoints

### 1. Get or Create Active Cart

- **URL:** `GET /api/carts` (`api:carts`)
- **Auth / Session / CSRF:** Session required (automatically renews a session token via `s.RenewToken()` if empty). Anonymous or authenticated users supported. No CSRF token needed for `GET`.
- **Behavior:** Resolves the active cart for the request (with preloaded product fields `ID`, `Title`, `Image`, `Price`, `TaxRate`, `DiscountPrice`) or creates a new active cart (firing `carts.Signals.Created`), then calculates totals via `CalcOneOrCreate`.
- **Response (`200 OK`):** [`CartRow`](#cartrow-cartscartrow) JSON object.
- **Error Responses:**
  - `500 Internal Server Error` (`ErrorResponse`): Session middleware missing, database error, signal error, or JSON encoding failure.

---

### 2. Clear Active Cart

- **URL:** `POST /api/carts/clear` (`api:carts:clear`)
- **Auth / Session / CSRF:** Session required. CSRF token required (`nosurf`).
- **Request Body:** None.
- **Behavior:** If an active cart exists for the request, starts a transaction, marks the cart `Active = false`, invalidates any unattached coupon redemptions (`Order__isnull = true`, `Invalidated = false`), fires `carts.Signals.Cleared`, creates a fresh active cart (`carts.Signals.Created`), and commits.
- **Response (`200 OK`):** [`CartRow`](#cartrow-cartscartrow) JSON object for the newly created active cart.
- **Error Responses:**
  - `400 Bad Request`: Invalid/missing CSRF token (from `nosurf`).
  - `500 Internal Server Error` (`ErrorResponse`): Database, transaction, signal, or encoding error.

---

### 3. Add Item to Cart

- **URL:** `POST /api/carts/items/add` (`api:carts:add_item`)
- **Auth / Session / CSRF:** Session required (`503 Service Unavailable` if session middleware is missing). CSRF token required (`nosurf`).
- **Request Body (`CartItemRequest`):**
  ```json
  {
    "product_id": "01JABCDEF...",
    "amount": 1
  }
  ```
  - `product_id` (`string`): ULID of the `products.Product`.
  - `amount` (`int`): Quantity to add; must be `> 0`.
  - Unknown fields are rejected by `api.Decode`.
- **Behavior:** Starts a transaction, loads the product, gets or creates the active cart, creates or atomically increments the `CartItem.Amount`, fires `carts.Signals.Created` (if new cart) and `carts.Signals.Item.Add`, and commits.
- **Response (`200 OK`, `CartItemResponse`):**
  ```json
  {
    "cart": { ... },
    "item": { ... },
    "product": { ... }
  }
  ```
- **Error Responses:**
  - `400 Bad Request` (`ErrorResponse`): Invalid JSON / unknown fields, or `amount <= 0` (`"amount cannot be less than 0: ..."`).
  - `404 Not Found` (`ErrorResponse`): Product with `product_id` not found.
  - `503 Service Unavailable` (`ErrorResponse`): Cart session middleware unavailable (`"cart session unavailable"`).
  - `500 Internal Server Error` (`ErrorResponse`): Transaction, database, or signal error.

---

### 4. Remove Item from Cart

- **URL:** `POST /api/carts/items/remove` (`api:carts:remove_item`)
- **Auth / Session / CSRF:** Session required. CSRF token required (`nosurf`).
- **Request Body (`CartItemRequest`):**
  ```json
  {
    "product_id": "01JABCDEF...",
    "amount": 1
  }
  ```
  - `amount` (`int`): Quantity to subtract; must be `> 0`. If `amount >= item.Amount`, the `CartItem` row is deleted and the returned `item.amount` is `0`.
- **Behavior:** Starts a transaction, looks up the product, active cart, and `CartItem`, deletes or decrements the item, fires `carts.Signals.Item.Remove`, and commits.
- **Response (`200 OK`, `CartItemResponse`):**
  ```json
  {
    "cart": { ... },
    "item": { ... },
    "product": { ... }
  }
  ```
- **Error Responses:**
  - `400 Bad Request` (`ErrorResponse`): Invalid JSON / unknown fields, or `amount <= 0`.
  - `404 Not Found` (`ErrorResponse`): Product not found, active cart does not exist, or cart item does not exist.
  - `500 Internal Server Error` (`ErrorResponse`): Transaction, database, or signal error.

---

### 5. Redeem Coupon Code

- **URL:** `POST /api/coupons/redeem` (`api:coupons:redeem`)
- **Auth / Session / CSRF:** Session required (must already have an active cart). CSRF token required (`nosurf`).
- **Request Body (`RedemptionRequest`):**
  ```json
  {
    "coupon_code": "SUMMER2026"
  }
  ```
- **Behavior:**
  - Looks up the coupon by `Code` and the caller's active cart.
  - Induces a random 5–100 ms delay (`laggy()`) on invalid/missing lookups to hinder code enumeration.
  - Starts a transaction, locks the active cart row and coupon row, checks `UsageLimit` against `UsageCount` plus active uncounted reservations (`Counted = false`, `Invalidated = false`), checks whether the coupon is already redeemed for this cart, and verifies at least one cart item matches `coupon.Products` (if product-restricted).
  - Creates a `coupons.Redemption` reservation, recalculates the cart (`CartRow`), fires `coupons.Signals.Redeemed`, and commits.
- **Response (`200 OK`):** Updated [`CartRow`](#cartrow-cartscartrow) JSON object with applied discounts.
- **Error Responses:**
  - `400 Bad Request` (`ErrorResponse`): Invalid JSON / unknown fields.
  - `404 Not Found` (`ErrorResponse`): Coupon code invalid, no active cart exists, coupon usage limit reached (`"This coupon has reached its usage limit"`), no products in cart match coupon (`"No products in your cart match this coupon"`), or calculation error.
  - `409 Conflict` (`ErrorResponse`): Coupon is already redeemed for this cart.
  - `500 Internal Server Error` (`ErrorResponse`): Transaction, locking, database, or signal error.

---

### 6. List Payment Options

- **URL:** `GET /api/payments/options` (`api:payments:options`)
- **Auth / Session / CSRF:** Session required (must have an active cart). Configured with `APPVAR_SHOP_CURRENCY`.
- **Behavior:**
  - Validates `PaymentsAPI.Currency` as an ISO currency code.
  - Calculates the active cart (`cartQuery(ctx, r, false).CalcOne()`).
  - If the cart has items (`len(cart.Items) > 0`) and `cart.Totals.Payable()` is zero (`quote.Total.IsZero()`), returns a single `"No payment required"` option with `no_payment_required: true` and zeroed `Quote`.
  - Otherwise queries enabled `payments.ProviderAccount` records matching `Currency` (ordered by `Label`, `ID`), filters to accounts whose `Driver` is registered via `payments.ProviderRegistered`, and computes a fee `Quote` for each.
- **Response (`200 OK`):** JSON array of `PaymentOption`:
  ```json
  [
    {
      "account_id": "01JABCDEF...",
      "label": "Stripe EUR",
      "no_payment_required": false,
      "subtotal": "100.00",
      "fee": "1.50",
      "total": "101.50",
      "currency": "EUR"
    }
  ]
  ```
  *(Note: `no_payment_required` has `omitempty` and is omitted when `false`; `account_id` is zero ULID when `no_payment_required` is `true`.)*
- **Error Responses:**
  - `404 Not Found` (`ErrorResponse`): Active cart not found (`"cart not found"`).
  - `503 Service Unavailable` (`ErrorResponse`): Shop currency is missing or invalid (`"shop payment currency is unavailable"`).
  - `500 Internal Server Error` (`ErrorResponse`): Cart calculation or database query failure.

---

### 7. Start Checkout

- **URL:** `POST /api/payments/checkout` (`api:payments:checkout`)
- **Auth / Session / CSRF:**
  - **Authentication required:** Must be an authenticated `users.User` (`401 Unauthorized` if anonymous/missing).
  - **CSRF required:** Protected by `nosurf`.
  - Sets `Cache-Control: no-store` on every response.
  - Body size limited to 8 KiB (`http.MaxBytesReader`).
- **Content-Type:** Accepts `application/json` (or empty `Content-Type` treated as single JSON object, `DisallowUnknownFields()`) or `application/x-www-form-urlencoded`.
- **Request Body (`PaymentCheckoutRequest`):**
  - JSON:
    ```json
    {
      "account_id": "01JABCDEF..."
    }
    ```
  - Form (`application/x-www-form-urlencoded`): `account_id=<ULID>`
  - `account_id` may be empty (`""`) for free checkouts (`total == 0`). If non-empty, must be a valid ULID.
- **Configuration Requirements:**
  - `APPVAR_SHOP_CURRENCY` must be a valid ISO currency code.
  - `APPVAR_SHOP_CHECKOUT_ORIGIN` must be an `http` or `https` origin with hostname, no userinfo, no query, no fragment, and empty/`/` path.
  - `APPVAR_SHOP_CHECKOUT_RETURN_PATH` (defaults to `"/"`) must be a relative path starting with `/` (not `//`).
- **Behavior:**
  - Ensures the caller has a `customers.Customer` record with a valid email address (auto-creates one from the authenticated user's `Email`, `FirstName`, `LastName` attributes if none exists).
  - Resolves the caller's active non-empty cart (or an already-converted cart with a payment attempt for idempotency).
  - Calls `broker.Broker.PrepareCheckout` using `AttemptID: cart.ID`, marks the cart `Active = false` and binds `User`, invalidates unattached coupon redemptions, and commits the transaction.
  - **Free checkout (`prepared.Payment.IsFreeCheckout()`):** Redirects (`303 See Other`) to `<Origin><ReturnPath>?payment_id=<payment_id>&order_id=<order_id>`.
  - **Paid checkout:** Calls `broker.Broker.StartCheckout` with `returnURL` (`?payment_id=<payment_id>`) and `webhookURL` (`<Origin>/api/payments/webhook/<account_id>`).
    - If the provider immediately reports a terminal state (`PAID`, `CANCEL`, `EXPIRED`, `DENIED`, `FAILURE`, `REFUND`, `PARTIAL_REFUND`, `CHARGEBACK`), redirects (`303 See Other`) to `returnURL`.
    - Otherwise validates that `checkout.URL` is an `https://` URL with a hostname and no userinfo, and redirects (`303 See Other`) to `checkout.URL`.
- **Response (`303 See Other`):** HTTP redirect to return URL or provider checkout URL.
- **Error Responses:**
  - `400 Bad Request` (`ErrorResponse`): Invalid JSON/form, non-ULID `account_id`, or `ErrCheckoutAccountRequired` (`"select a payment account"`).
  - `401 Unauthorized` (`ErrorResponse`): User is not authenticated (`"sign in to check out"`).
  - `404 Not Found` (`ErrorResponse`): Cart not found (`"cart not found"`), or `ErrCheckoutAccountUnavailable` (`"payment account is unavailable"`).
  - `405 Method Not Allowed` (`ErrorResponse`): Non-`POST` request (`Allow: POST`).
  - `409 Conflict` (`ErrorResponse`): Cart is empty (`"cart is empty"`), or cart changed / multiple active carts / already has a checkout (`"cart changed or already has a checkout"`).
  - `413 Request Entity Too Large` (`ErrorResponse`): Request body exceeds 8 KiB.
  - `415 Unsupported Media Type` (`ErrorResponse`): Unsupported `Content-Type`.
  - `422 Unprocessable Entity` (`ErrorResponse`): Missing/invalid customer email (`"a customer email address is required"`), or `payments.ErrNonPositiveTotal` (`"payment total must be positive"`).
  - `500 Internal Server Error` (`ErrorResponse`): Endpoint called inside an existing/non-commit transaction context, or unexpected DB/commit error.
  - `502 Bad Gateway` (`ErrorResponse`): `StartCheckout` failed (`"could not open provider checkout"`) or provider returned an invalid/non-HTTPS checkout URL.
  - `503 Service Unavailable` (`ErrorResponse`): Invalid `SHOP_CURRENCY`, `SHOP_CHECKOUT_ORIGIN`, or `SHOP_CHECKOUT_RETURN_PATH`.

---

### 8. Payment Provider Webhook

- **URL:** `POST /api/payments/webhook/<<account>>` (`api:payments:webhook`)
- **Path Parameters:**
  - `account`: ULID of the `payments.ProviderAccount`.
- **Auth / Session / CSRF:**
  - **CSRF Exempt:** Exempted from `nosurf` via `HOOK_SETUP_NOSURF` (`^/api/payments/webhook/[0-9A-HJKMNP-TV-Z]{26}/?$`).
  - No user session required; authenticity is verified by the provider's `HandleWebhook` implementation.
- **Request Body:** Raw provider payload, capped at 1 MiB (`http.MaxBytesReader`).
- **Behavior:**
  - Parses `account` as a ULID and loads `ProviderAccount`.
  - Instantiates the provider and calls `provider.HandleWebhook(ctx, WebhookRequest{Method, Header, URL, Body})`.
  - Reconciles the returned `PaymentStatus` against the matching `Payment` (`Account == account.ID` and `TransactionID == status.ProviderRef`) via `reconcileWebhook` and `applyStatus`.
  - Calls `result.Acknowledge(w)` if non-nil, otherwise writes `200 OK`.
- **Responses:**
  - `200 OK` (or custom response written by `result.Acknowledge(w)`).
  - `400 Bad Request` (plain text via `http.Error`): Body unreadable/too large (`"invalid webhook body"`) or provider returned `payments.ErrInvalidWebhook` (`"invalid webhook"`).
  - `404 Not Found` (via `http.NotFound`): `account` is not a valid ULID or `ProviderAccount` does not exist.
  - `503 Service Unavailable` (plain text via `http.Error`): Database error loading account, provider unavailable, or reconciliation failure.

---

## Admin Endpoints

### 9. Generate Coupon Code

- **URL:** `POST <admin_prefix>/shop/coupon/generate` (`admin:ext:shop:generate_coupon`)
- **Auth / Permissions / CSRF:**
  - Requires `"coupons:generate"` object permission on `&coupons.Coupon{}` (`permissions.HasObjectPermission(r, &coupons.Coupon{}, "coupons:generate")`). If unauthorized, delegates to `admin.ReLogin(w, r, r.URL.Path)`.
  - Protected by `nosurf` CSRF (`X-CSRF-Token`).
- **Request Body (`CouponGenerationRequest`):**
  ```json
  {
    "code_format_parts": [
      { "type": 1, "length": 0, "value": "SALE" },
      { "type": 8, "length": 0, "value": "" },
      { "type": 0, "length": 6, "value": "" }
    ]
  }
  ```
  - `code_format_parts` (`[]generators.CodePart`): Ordered list of code parts:
    - `type` (`uint`): `0` (`CPT_RANDOM`), `1` (`CPT_STATIC`), `2` (`CPT_YEAR`), `3` (`CPT_MONTH`), `4` (`CPT_WEEKDAY`), `5` (`CPT_DAY`), `6` (`CPT_HOUR`), `7` (`CPT_MINUTE`), `8` (`CPT_DASH`), `9` (`CPT_UUID`).
    - `length` (`int`): Used by `CPT_RANDOM` and `CPT_UUID`.
    - `value` (`string`): Used by `CPT_STATIC`.
- **Response (`200 OK`, `CouponGenerationResponse`):**
  ```json
  {
    "coupon_code": "SALE-XKQMPZ"
  }
  ```
- **Error Responses:**
  - `400 Bad Request` (`ErrorResponse`): Invalid JSON / unknown fields, or `generators.GenerateCouponCode` validation/generation error (e.g., invalid part type, code shorter than `MIN_CODE_LENGTH` or longer than `MAX_CODE_LENGTH`, or generated code already exists).
  - `500 Internal Server Error` (`except.Fail`): Failure encoding the JSON response.
