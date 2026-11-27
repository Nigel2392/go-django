# Configuration & Models

This document covers the `contrib/shop` application configuration, HTTP adapter handlers, shared pricing and relation utilities, payment state constants, form helpers, and all data models and interfaces.

---

## Application Configuration (`shop`)

### `ShopAppConfig`

Defined in `contrib/shop/app.go`:

```go
type ShopAppConfig struct {
	*apps.DBRequiredAppConfig
	ADMIN_ROUTE *mux.Route
	ADMIN_MENU  func(r *http.Request) []menu.MenuItem
}
```

- `BuildRouting(m mux.Multiplexer)`: Panics if `ADMIN_ROUTE` is `nil`; otherwise invokes `a.Routing(a.ADMIN_ROUTE)`.
- `OnReady() error`: Delegates to `a.DBRequiredAppConfig.OnReady()`.
- `TranslationsFS(settings django.Settings) []fs.FS`: Returns the embedded `translations.yml` filesystem.

### Singleton & Factory

- `SHOP *ShopAppConfig`: Pre-initialized `apps.NewDBAppConfig("shop")` with `ADMIN_ROUTE` mounted at `admin.AdminSite.Route.Any("shop/", nil, "shop")`.
- `NewAppConfig() (django.AppConfig, error)`: Configures `SHOP` (returning early if `SHOP == nil || SHOP.IsReady()`):
  - Registers `contenttypes` aliases for legacy `SupplierModel` paths.
  - Declares dependencies: `[]string{"session", "admin", "editor", "blocks", "revisions", "settings"}`.
  - Registers CLI commands in `SHOP.Cmd`: `sync-psp` (`command_sync_psp.go`, runs `payments.ProvidersQuery().WithContext(ctx).Sync()`) and `sync-suppliers` (`command_sync_suppliers.go`, runs `orders.SuppliersQuery().WithContext(ctx).Sync()`).
  - Registers embedded static assets (`assets/static`), templates (`assets/templates`), and the `shop/admin/icons/funnel.svg` icon.
  - Calls `handlers.RegisterAuditLogs()`.
  - Registers all 14 models in `SHOP.ModelObjects`.
  - Configures `SHOP.ADMIN_MENU`, the `products.Product` admin chooser, `admin.RegisterApp("shop", ...)` with model definitions from `contrib/shop/adminsite` (`CategoryAdmin`, `ProductAdmin`, `OrdersAdmin`, `CouponsAdmin`, `CustomersAdmin`), custom route `admin:commerce:new_with_supplier`, and admin extension route `admin:ext:shop:generate_coupon` (`POST /shop/coupon/generate`).
  - In `SHOP.Ready`: mounts `/api/carts*` (including `POST /api/carts/customer`), `/api/coupons/redeem`, `/api/payments/options`, `/api/payments/checkout`, and `/api/payments/webhook/<<account>>`, resolves `APPVAR_SHOP_CURRENCY` via `resolveShopCurrency()`, and registers the `nosurf` CSRF exemption `^/api/payments/webhook/[0-9A-HJKMNP-TV-Z]{26}/?$`.

### App Variables

| Constant | Key | Usage |
| :--- | :--- | :--- |
| `APPVAR_SHOP_CURRENCY` | `"SHOP_CURRENCY"` | Accepts a `currency.Unit` (e.g. `currency.EUR`), an ISO currency code `string` (e.g. `"EUR"`), or a dynamic `func(context.Context) currency.Unit` resolver passed to `public.PaymentsAPI` and `public.CheckoutAPI`. |
| `APPVAR_SHOP_CHECKOUT_ORIGIN` | `"SHOP_CHECKOUT_ORIGIN"` | Origin URL used to build checkout return and webhook URLs. |
| `APPVAR_SHOP_CHECKOUT_RETURN_PATH` | `"SHOP_CHECKOUT_RETURN_PATH"` | Return path appended to the origin (defaults to `"/"`). |

---

## View Adapter Handlers (`contrib/shop/http.go`, `contrib/shop/adminsite/http.go`)

`ShopHandler` and `ShopInstanceHandler[T]` (defined in both `contrib/shop/http.go` and `contrib/shop/adminsite/http.go`) adapt plain `http.HandlerFunc`-style functions into Go-Django `views.View` implementations that take control of request handling:

```go
type ShopHandler struct {
	Handle func(w http.ResponseWriter, r *http.Request)
}

func (s ShopHandler) ServeXXX(w http.ResponseWriter, r *http.Request) {}
func (s ShopHandler) TakeControl(w http.ResponseWriter, r *http.Request, _ views.View)

type ShopInstanceHandler[T attrs.Definer] struct {
	Instance T
	Handle   func(w http.ResponseWriter, r *http.Request, instance T)
}

func (s ShopInstanceHandler[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}
func (s ShopInstanceHandler[T]) TakeControl(w http.ResponseWriter, r *http.Request, _ views.View)
```

---

## Shared Pricing (`contrib/shop/shared/price`)

The `price` package uses `github.com/shopspring/decimal` and `golang.org/x/text/currency` for exact monetary and currency-rounded fee calculations.

### Types & Helpers

```go
// Money pairs a decimal amount with a developer-controlled ISO currency unit.
// Marshals/unmarshals JSON as {"amount": "<decimal>", "currency": "<ISO>"}.
type Money struct {
	Amount   decimal.Decimal
	Currency currency.Unit
}

func (m Money) Quote(fixedFee, percentageFee decimal.Decimal) (Quote, error)

// Quote represents a currency-rounded subtotal, payment fee, and total.
// Marshals/unmarshals JSON as {"subtotal": ..., "fee": ..., "total": ..., "currency": "<ISO>"}.
type Quote struct {
	Subtotal decimal.Decimal
	Fee      decimal.Decimal
	Total    decimal.Decimal
	Currency currency.Unit
}

// QuoteFee rounds subtotal.Amount to subtotal.Currency's standard ISO scale and adds
// fixedFee + base * percentageFee / 100 (rounded to scale) when base is positive.
func QuoteFee(subtotal Money, fixedFee, percentageFee decimal.Decimal) (Quote, error)

type AmountValue struct {
	Amount       decimal.Decimal
	IsPercentage bool
}

type Price struct {
	ItemPrice     decimal.Decimal `json:"item_price"`     // currency
	ItemFee       AmountValue     `json:"item_fee"`       // currency
	TaxPercentage decimal.Decimal `json:"tax_percentage"` // percentage
}

type Totals struct {
	TotalPrice       decimal.Decimal `json:"total_price"`        // currency
	TotalFee         decimal.Decimal `json:"total_fee"`          // currency
	TotalTax         decimal.Decimal `json:"total_tax"`          // currency
	TotalDiscount    decimal.Decimal `json:"total_discount"`     // currency
	LeftOverDiscount decimal.Decimal `json:"left_over_discount"` // currency
}

// Payable returns the amount after discounts, including product fees and tax.
func (t Totals) Payable() decimal.Decimal

type PriceCalculation struct {
	ItemCount int
	Discount  AmountValue
}
```

### `(p Price) Calculate(opts PriceCalculation) (totalPriceInclTax decimal.Decimal, calculatedPrice Totals)`

1. Multiplies `p.ItemPrice` by `opts.ItemCount`.
2. Applies `opts.Discount`:
   - **Percentage** (`IsPercentage == true`): caps percentage at `100`, subtracts from `totalPrice`, and records `totalDiscount`.
   - **Fixed** (`IsPercentage == false`): subtracts up to `totalPrice`; any excess becomes `leftOverDiscount`.
3. Computes `totalFee` on the **post-discount** `totalPrice` (either percentage of `totalPrice` or `p.ItemFee.Amount * itemCount`).
4. If a fixed discount had `leftOverDiscount > 0`, applies the remainder against `totalFee`.
5. Computes `totalTax` on `totalPrice + totalFee` using `p.TaxPercentage / 100`.
6. Returns `totalPrice + totalFee + totalTax` and the populated `Totals` struct.

---

## Model Keys (`contrib/shop/shared/relations/modelkeys`)

`modelkeys` defines content-type key constants in a leaf package to avoid import cycles:

```go
const (
	ProductModelKey  = "shop.Product"
	OrderModelKey    = "shop.Order"
	CartModelKey     = "shop.Cart"
	CategoryModelKey = "shop.Category"
	PaymentModelKey  = "shop.Payment"
	CustomerModelKey = "shop.Customer"
)
```

`registry.go` registers `contenttypes.NewLazyRegistry` entries for `ProductModelKey`, `OrderModelKey`, `CartModelKey`, `CategoryModelKey`, and `PaymentModelKey`, matching models owned by the `"shop"` app with the corresponding model name.

---

## Payment States (`contrib/shop/util/paystate`)

`paystate` defines `type PayState = string`, a `PAYSTATE strich.Check[PayState]` checker (`Delimiter: ':'`, `Display.Delimiter: " | "`), and 17 `STATUS_*` constants:

| Constant | Value | Display Label | Meaning |
| :--- | :--- | :--- | :--- |
| `STATUS_INIT` | `"STATUS_INIT"` | `Initialised` | Transaction started; no payment method selected yet. |
| `STATUS_PENDING` | `"STATUS_PENDING"` | `Pending` | Awaiting payer input, synchronous bank/third-party response, or fund settlement. |
| `STATUS_CANCEL` | `"STATUS_CANCEL"` | `Cancelled` | Cancelled by the user. |
| `STATUS_EXPIRED` | `"STATUS_EXPIRED"` | `Expired` | Payment or authorization expired. |
| `STATUS_DENIED` | `"STATUS_DENIED"` | `Denied` | Rejected by processor (e.g., failed BNPL credit check). |
| `STATUS_FAILURE` | `"STATUS_FAILURE"` | `Failure` | Processing error at payment method, processor, or issuer. |
| `STATUS_PAID_CHECKAMOUNT` | `"STATUS_PAID_CHECKAMOUNT"` | `Check Amount` | Amount paid by customer does not match the original order amount. |
| `STATUS_PARTIAL_PAYMENT` | `"STATUS_PARTIAL_PAYMENT"` | `Partial Payment` | Partial payment (e.g., gift card) not covering the full order amount. |
| `STATUS_VERIFY` | `"STATUS_VERIFY"` | `Manual Verify` | Flagged as suspicious; requires manual verification. |
| `STATUS_AUTHORIZE` | `"STATUS_AUTHORIZE"` | `Authorize` | Funds reserved/authorized and ready to capture (credit card, BNPL). |
| `STATUS_PARTLY_CAPTURED` | `"STATUS_PARTLY_CAPTURED"` | `Partly Captured` | Partially captured; remaining authorized amount can be captured or voided. |
| `STATUS_PAID` | `"STATUS_PAID"` | `Paid` | Payment succeeded. |
| `STATUS_NO_PAYMENT_REQUIRED` | `"STATUS_NO_PAYMENT_REQUIRED"` | `No payment required` | Zero-total checkout requiring no external payment. |
| `STATUS_CHARGEBACK` | `"STATUS_CHARGEBACK"` | `Chargeback` | Credit card chargeback initiated. |
| `STATUS_REFUNDING` | `"STATUS_REFUNDING"` | `Refunding` | Refund in progress (may still be cancellable). |
| `STATUS_REFUND` | `"STATUS_REFUND"` | `Refunded` | Payment has been fully refunded. |
| `STATUS_PARTIAL_REFUND` | `"STATUS_PARTIAL_REFUND"` | `Partial Refund` | Payment has been partially refunded. |

---

## Form Helpers (`contrib/shop/util/forms`)

```go
func NewAdminForm[MODEL attrs.Definer](
	r *http.Request,
	instance MODEL,
	getPanels func(MODEL) []admin.Panel,
) *admin.AdminForm[*modelforms.BaseModelForm[MODEL], MODEL]
```

Builds a `modelforms.BaseModelForm[MODEL]` for `instance` using `r.Context()` and wraps it in `admin.NewAdminForm(r, form, getPanels(instance)...)`.

---

## Models & Interfaces

### Categories (`contrib/shop/models`)

#### `Category`

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `uint64` | Primary key, read-only. |
| `Name` | `string` | Required (`MinLength: 2`, `MaxLength: 75`). |
| `Slug` | `string` | Required (`MinLength: 2`, `MaxLength: 75`). |
| `ParentCategory` | `*Category` | Nullable self-referential FK (`parent_id`), reverse alias `"Children"`. |
| `Children` | `*queries.RelRevFK[*Category]` | Reverse FK to child categories. |

- `String() string`: Returns `c.Name`.
- `Categories() *CategoryQuery`: Returns a `*CategoryQuery` with `ChildrenOf(cat *Category) *CategoryQuery` (filters `ParentCategory__isnull=true` when `cat == nil`, or `ParentCategory=cat.ID`).

---

### Customers (`contrib/shop/customers`)

#### `Customer`

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `drivers.ULID` | Primary key, read-only; auto-generated in `BeforeSave` if zero. |
| `Email` | `*drivers.Email` | `email`, `MinLength: 3`, `MaxLength: 255`. |
| `FirstName` | `string` | `first_name`, `MaxLength: 75`. |
| `LastName` | `string` | `last_name`, `MaxLength: 75`. |
| `User` | `users.User` | Nullable read-only deferred FK (`user_id`, `users.MODEL_KEY`). |
| `FirstSeen` | `time.Time` | `AutoNowAdd: true`, read-only. |
| `Orders` | `*queries.RelRevFK[attrs.Definer]` | Reverse relation (excluded from DB migrations via `migrator.AttrUseInDBKey: false`). |

---

### Products (`contrib/shop/products`)

#### `ProductModel` and `SupplierValue`

```go
type ProductModel interface {
	specific.Model
	ProductPrice(ctx context.Context) price.Money
}

type SupplierValue string
```

- `ProductModel` is implemented by custom product subtype models integrated via `queries/src/specific`.
- `SupplierValue` stores the registered supplier backend identifier on a `Product` or `OrderItem`.

#### `Product`

Embeds `specific.Specific[*Product]`. `BeforeSave` generates a `drivers.ULID` if `ID` is zero. `Query() *ProductQuery` returns a `specific.SpecificQuerySet`-backed queryset.

| Field | Type | JSON Tag | Notes |
| :--- | :--- | :--- | :--- |
| `Image` | `*mediafiles.SimpleStoredObject` | `image,omitempty` | Product image. |
| `ID` | `drivers.ULID` | `id` | Primary key, read-only. |
| `Title` | `string` | `title` | Required (`MinLength: 2`, `MaxLength: 75`). |
| `Slug` | `string` | `slug,omitempty` | Read-only text input (`MinLength: 2`, `MaxLength: 75`). |
| `Ranking` | `uint64` | `ranking,omitempty` | Read-only ranking (1-10). |
| `Price` | `decimal.Decimal` | `price,omitempty` | `Precision: 13`, `Scale: 4`. |
| `AdditionalFee` | `decimal.Decimal` | `additional_fee,omitempty` | Per-product fee (`Precision: 13`, `Scale: 4`). |
| `TaxRate` | `decimal.Decimal` | `tax_rate,omitempty` | Tax percentage (`Precision: 13`, `Scale: 4`). |
| `DiscountPrice` | `decimal.Decimal` | `discount_price,omitempty` | Optional discount price (`Precision: 13`, `Scale: 4`, `Blank: true`). |
| `Supplier` | `SupplierValue` | `supplier,omitempty` | Read-only supplier backend key. |
| `Category` | `*models.Category` | `category,omitempty` | FK (`category_id`). |
| `Editor` | `*editor.EditorJSBlockData` | `editor,omitempty` | Rich-text description (`paragraph`, `text-align`, `list`). |
| `SpecificID` | `string` | `specific_id,omitempty` | Embedded from `specific.Specific[*Product]`; read-only, omitted from forms (`FormField` returns `nil`). |
| `SpecificContentType` | `*contenttypes.BaseContentType[attrs.Definer]` | `specific_content_type,omitempty` | Embedded from `specific.Specific[*Product]`; read-only, omitted from forms (`FormField` returns `nil`). |
| `Coupons` | `*queries.RelM2M[attrs.Definer, attrs.Definer]` | — | Reverse M2M to coupons. |

- `ActualPrice() decimal.Decimal`: Computes `(Price + AdditionalFee) * (1 + TaxRate/100)`.
- `AddPanels(ctx)` / `EditPanels(ctx)`: Return admin panel layouts.

---

### Carts (`contrib/shop/carts`)

#### `Cart`

Implements `queries.ActsBeforeCreate` (`BeforeCreate` assigns a new `drivers.ULID` to `ID`).

| Field | Type | JSON Tag | Notes |
| :--- | :--- | :--- | :--- |
| `ID` | `drivers.ULID` | `id` | Primary key, read-only. |
| `User` | `users.User` | `-` | Nullable read-only deferred FK (`user_id`, `users.MODEL_KEY`). |
| `Customer` | `*customers.Customer` | `customer,omitempty` | Nullable FK (`customer_id`), used for guest or explicit checkout customer details. |
| `Session` | `string` | `-` | Nullable read-only session token (`session_id`). |
| `CreatedAt` | `time.Time` | `created_at` | `AutoNowAdd: true`, read-only. |
| `Active` | `bool` | `active` | Whether the cart is active. |
| `CartItems` | `*queries.RelRevFK[*CartItem]` | `-` | Reverse FK to `CartItem`. |
| `Redemptions` | `*queries.RelRevFK[*coupons.Redemption]` | `-` | Reverse relation to applied redemptions (non-DB). |

`carts/models_cart.go` also defines guest/cart customer attachment:
- `ErrInvalidCustomerEmail`: returned when an email address fails `mail.ParseAddress` or is shorter than 3 characters.
- `CustomerDetails{ Email string; FirstName string; LastName string; User users.User }`.
- `SetCustomer(ctx context.Context, cart *Cart, details CustomerDetails) (*customers.Customer, error)`: updates the cart's existing `Customer` (preserving/attaching an authenticated `User` if unset) or creates a new `Customer` and links `cart.Customer` inside a transaction.

#### `CartItem`

Implements `queries.ActsBeforeCreate` (`BeforeCreate` assigns `ID = drivers.NewULID()`) and `UniqueTogether()` on `{"Cart", "Product"}`.

| Field | Type | JSON Tag | Notes |
| :--- | :--- | :--- | :--- |
| `ID` | `drivers.ULID` | `id` | Primary key, read-only. |
| `Cart` | `*Cart` | `-` | Required FK (`cart_id`), reverse alias `"CartItems"`. |
| `Product` | `*products.Product` | `product,omitempty` | Required FK (`product_id`), reverse alias `"CartItems"`. |
| `Amount` | `uint64` | `amount` | Quantity (`MinValue: 0`). |

#### `CartQuery`, `CartRow`, and `CartItemRow`

```go
type CartItemRow struct {
	CartItem          *CartItem                  `json:"item"`
	Redemptions       []*coupons.Redemption      `json:"coupons"`
	CalculatedPrice   price.Totals               `json:"total_price"`
	RedemptionAmounts map[uint64]decimal.Decimal `json:"-"`
}

type CartRow struct {
	Object            *Cart                      `json:"cart"`
	Items             []*CartItemRow             `json:"items"`
	Coupons           []*coupons.Redemption      `json:"coupons"`
	Totals            price.Totals               `json:"total_price"`
	RedemptionAmounts map[uint64]decimal.Decimal `json:"-"`
}
```

Key `CartQuery` methods (created via `carts.Query()`):
- `CartForRequest(r *http.Request) *CartQuery`: Scopes to active carts for the request's session or authenticated user, ordered by `CreatedAt`.
- `CartsForRequest(r *http.Request) *CartQuery`: Same scoping, including inactive carts.
- `PreloadProducts(productFields ...string) *CartQuery`: Preloads `CartItems` with selected `Product.*` fields.
- `Calc() (int, iter.Seq2[*CartRow, error], error)`: Calculates line-level and cart-level totals and coupon discounts in a transaction.
- `CalcOne() (*CartRow, error)`: Returns a single calculated `*CartRow` (or `errors.NoRows` / `errors.MultipleRows`).
- `CalcOneWithProducts() (*CartRow, error)`: Like `CalcOne()`, but selects `Product.*` for order snapshot creation.
- `CalcOneOrCreate(create *Cart) (row *CartRow, created bool, err error)`: Atomically fetches and calculates an existing cart or creates a new one.

---

### Coupons (`contrib/shop/coupons`)

Package variables: `MIN_CODE_LENGTH int64 = 6`, `MAX_CODE_LENGTH int64 = 32`.

```go
type CouponCode string
```

#### `Coupon`

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `uint64` | Primary key, read-only. |
| `Title` | `string` | Required (`MinLength: 2`, `MaxLength: 75`). |
| `ShortDescription` | `string` | Optional description (`Blank: true`). |
| `Code` | `CouponCode` | Required (`MinLength: MIN_CODE_LENGTH`, `MaxLength: MAX_CODE_LENGTH`). |
| `IsPercentage` | `bool` | Whether `MaxDiscount` is a percentage (`true`) or fixed amount (`false`). |
| `MaxDiscount` | `decimal.Decimal` | `Precision: 13`, `Scale: 4`. |
| `MinOrderAmount` | `decimal.Decimal` | `Precision: 13`, `Scale: 4`, `Blank: true`. |
| `UsageLimit` | `uint64` | Maximum redemptions (`Blank: true`; `0` = unlimited). |
| `UsageCount` | `uint64` | Read-only counter of counted redemptions. |
| `ValidFrom` | `time.Time` | Nullable validity start. |
| `ValidUntil` | `time.Time` | Nullable validity end. |
| `CreatedAt` | `time.Time` | `AutoNowAdd: true`, read-only. |
| `UpdatedAt` | `time.Time` | `AutoNow: true`, read-only. |
| `CreatedBy` | `users.User` | Nullable read-only deferred FK (`user_id`, `users.MODEL_KEY`). |
| `RedemptionSet` | `*queries.RelRevFK[*Redemption]` | Reverse FK to redemptions. |
| `Products` | `*queries.RelM2M[*products.Product, *CouponProductThrough]` | Optional M2M restricting the coupon to specific products (`ReverseAlias: "Coupons"`). |

#### `CouponProductThrough`

Through-model for `Coupon.Products`:

| Field | Type | Notes |
| :--- | :--- | :--- |
| `Coupon` | `*Coupon` | Required read-only FK (`coupon_id`). |
| `Product` | `*products.Product` | FK (`product_id`) with product chooser form field. |

#### `Redemption`

Tracks a coupon applied to a cart and/or order:

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `uint64` | Primary key, read-only. |
| `Coupon` | `*Coupon` | Required read-only FK (`coupon_id`). |
| `Customer` | `*customers.Customer` | Nullable read-only FK (`customer_id`). |
| `Cart` | `attrs.Definer` | Nullable read-only deferred FK (`cart_id`, `modelkeys.CartModelKey`, reverse `"Redemptions"`). |
| `Order` | `attrs.Definer` | Nullable read-only deferred FK (`order_id`, `modelkeys.OrderModelKey`, reverse `"Redemptions"`). |
| `DiscountAmount` | `decimal.Decimal` | Read-only (`Precision: 19`, `Scale: 10`). |
| `Invalidated` | `bool` | Read-only flag marking invalidated redemptions. |
| `Counted` | `bool` | Read-only flag indicating whether this redemption is included in `Coupon.UsageCount`. |
| `RedeemedAt` | `time.Time` | Read-only timestamp. |

---

### Orders (`contrib/shop/orders`)

#### `Order`

Implements `queries.ActsBeforeCreate` (`BeforeCreate` assigns `ID = drivers.NewULID()`). `orders.Query()` returns an `*OrderQuery`.

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `drivers.ULID` | Primary key, read-only. |
| `Customer` | `*customers.Customer` | Required FK (`customer_id`), reverse alias `"Orders"`. |
| `State` | `InternalOrderState` | Order workflow state. |
| `Payments` | `*queries.RelRevFK[*payments.Payment]` | Reverse FK to `Payment`. |
| `OrderItems` | `*queries.RelRevFK[*OrderItem]` | Reverse FK to `OrderItem`. |
| `Completed` | `bool` | `Default: false`, `Blank: true`. |
| `Processed` | `bool` | `Default: false`, `Blank: true`. |

#### `OrderItem` & `OrderItemPricing`

Implements `queries.ActsBeforeCreate` and `queries.ActsAfterQuery` (binds `SupplierData` to the supplier's `OrderBlock`).

```go
// OrderItemPricing captures the totals and coupon allocations used at checkout.
// The map is keyed by coupon redemption ID.
type OrderItemPricing struct {
	Totals            price.Totals
	RedemptionAmounts map[uint64]decimal.Decimal
}
```

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `drivers.ULID` | Primary key. |
| `Order` | `*Order` | Required FK (`order_id`), reverse alias `"OrderItems"`. |
| `Product` | `drivers.JSON[ProductData]` | JSON snapshot of the base product and specific product model. |
| `Pricing` | `drivers.JSON[*OrderItemPricing]` | Nullable read-only JSON snapshot of `Totals` and per-redemption amounts. |
| `Supplier` | `products.SupplierValue` | Supplier key copied from the product for querying. |
| `SupplierData` | `*blocks.StructBlockValue` | Nullable block data bound to the supplier's `OrderBlock`. |
| `ItemPrice` | `decimal.Decimal` | `Precision: 13`, `Scale: 4`. |
| `TaxRate` | `decimal.Decimal` | Tax rate at order time. |
| `Amount` | `int` | Quantity ordered. |

#### `ProductData`

```go
type ProductData struct {
	Product  *products.Product
	Specific products.ProductModel
}
```

`UnmarshalJSON` decodes `Product`, instantiates the concrete `products.ProductModel` via `Product.SpecificContentType.New()`, and unmarshals `Specific` into it (with backwards compatibility for older snapshots that stored `*products.Product` directly).

#### `OrderState` & `InternalOrderState`

```go
type OrderState struct {
	ID         int64
	Label      string
	Action     InternalOrderState
	WorkflowID drivers.ULID // placeholder
}

type InternalOrderState string

func (i InternalOrderState) Label(ctx context.Context) string
```

Exported `InternalOrderState` constants in `orders/models_order_state.go`:
- `OrderStateAccepted` (`"order_accepted"`) — `"Order Accepted"`
- `OrderStatePending` (`"order_pending"`) — `"Order Pending"`
- `OrderStateRejected` (`"order_rejected"`) — `"Order Rejected"`
- `OrderStateWorkflow` (`"order_workflow"`) — `"Order Workflow"`

---

### Payments (`contrib/shop/payments`)

#### `ProviderAccount`

`ProviderAccount` tracks which registered `ProviderBackend` implementations are enabled in the database (`BeforeCreate` generates `ID` if zero; unique index `provider_account_internal_name_idx` and `UniqueTogether: [["InternalName"]]`). Provider fees, currency, labels, and credentials stay in Go code via `payments.RegisterProvider` (see [Payments & Checkout](./payments-and-checkout.md)).

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `drivers.ULID` | Primary key, read-only. |
| `InternalName` | `string` | Unique registered `ProviderBackend` key (`MinLength: 2`, `MaxLength: 75`, `attrs.AttrUniqueKey: true`). |
| `Enabled` | `bool` | `Blank: true`, `Default: true`. Whether the account is active for checkout. |

Methods:
- `Provider(ctx context.Context) (Provider, error)`: Resolves the registered `ProviderBackend` for `a.InternalName` and calls `backend.Provider(ctx, a)`.
- `Quote(ctx context.Context, subtotal price.Money) (price.Quote, error)`: Resolves the registered `ProviderBackend`, verifies `backend.Currency(ctx) == subtotal.Currency`, and delegates to `price.QuoteFee(subtotal, backend.FixedFee(ctx), backend.PercentageFee(ctx))`.

#### `Payment`, `PaymentEvent`, & `PaymentEventKind`

`Payment` records a payment attempt (`BeforeCreate` generates `ID` if zero; unique index `shop_payment_account_ref_idx` on `("Account", "TransactionID")`):

| Field | Type | Notes |
| :--- | :--- | :--- |
| `ID` | `drivers.ULID` | Primary key, read-only. |
| `Order` | `attrs.Definer` | Nullable deferred FK (`order_id`, `modelkeys.OrderModelKey`, reverse `"Payments"`). |
| `Account` | `*ProviderAccount` | Nullable FK (`account_id`). |
| `Provider` | `string` | Provider `InternalName` (`MaxLength: 75`), or `"shop:free-checkout"` for zero-total orders. |
| `TransactionID` | `sql.NullString` | Provider reference (`MaxLength: 255`, nullable). |
| `CheckoutURL` | `string` | Hosted checkout redirect URL (`MaxLength: 4096`). |
| `State` | `paystate.PayState` | Current payment state (`MaxLength: 40`). |
| `Amount` | `decimal.Decimal` | Total charge amount including `Fee` (`Precision: 13`, `Scale: 4`). |
| `Fee` | `decimal.Decimal` | Customer-facing payment fee included in `Amount` (`Precision: 13`, `Scale: 4`, `Default: 0`). |
| `RefundedAmount` | `decimal.Decimal` | Cumulative refunded amount in `Currency` (`Precision: 13`, `Scale: 4`, `Default: 0`). |
| `ProviderFee` | `drivers.JSON[*price.Money]` | Latest merchant processing fee reported by the PSP (read-only). |
| `Currency` | `string` | Snapshotted 3-letter ISO currency code (`MaxLength: 3`). |
| `CreatedAt` | `drivers.Timestamp` | `AutoNowAdd: true`. |
| `UpdatedAt` | `drivers.Timestamp` | `AutoNow: true`. |
| `History` | `drivers.JSON[[]PaymentEvent]` | Append-only event log in write order (read-only). |

Methods:
- `CurrencyUnit() currency.Unit`: Parses the snapshotted ISO `Currency` string into a `currency.Unit` (returns zero `currency.Unit{}` if empty or invalid).
- `IsFreeCheckout() bool`: Returns `true` when `Provider == "shop:free-checkout"`, `State == paystate.STATUS_NO_PAYMENT_REQUIRED`, `Account` is nil/zero, `!TransactionID.Valid`, `Amount.IsZero()`, and `Fee.IsZero()`.

```go
// PaymentEvent is a recorded step in a payment attempt, in write order.
type PaymentEvent struct {
	At          time.Time
	Kind        PaymentEventKind
	State       paystate.PayState
	RequestID   string       `json:",omitempty"`
	ProviderRef string       `json:",omitempty"`
	Amount      *price.Money `json:",omitempty"`
	ProviderFee *price.Money `json:",omitempty"`
}

type PaymentEventKind string

const (
	EventCheckoutOpened     PaymentEventKind = "checkout_opened"
	EventStatusUpdated      PaymentEventKind = "status_updated"
	EventProviderFeeUpdated PaymentEventKind = "provider_fee_updated"
	EventNoPaymentRequired  PaymentEventKind = "no_payment_required"
	EventRefunded           PaymentEventKind = "refunded"
	EventCancelled          PaymentEventKind = "cancelled"
)
```
