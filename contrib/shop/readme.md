# Shop (`contrib/shop`)

`contrib/shop` is a modular e-commerce application for Go-Django (`github.com/Nigel2392/go-django/contrib/shop`). It provides product catalogs with polymorphic product subtypes, hierarchical categories, session- and user-scoped shopping carts, coupon codes and redemptions, supplier fulfillment backends, customer records, signal-driven order creation, payment provider orchestration, public JSON/HTTP APIs, a TypeScript storefront client, and full `contrib/admin` integration.

---

## Architecture

`contrib/shop` is organized around domain packages backed by Go-Django's `queries` ORM, `attrs` field definitions, `specific` model inheritance, and a typed signal bus (`shared/signals`):

1. **Catalog & Suppliers (`products`, `models`, `orders/supplier*.go`)**:
   - `products.Product` uses `queries/src/specific` so applications can define custom product types implementing `products.ProductModel` alongside shared base fields (`Price`, `AdditionalFee`, `TaxRate`, `DiscountPrice`, `Category`, `Supplier`).
   - Delivery and fulfillment are delegated to registered `orders.SupplierBackend` implementations, synchronized into `orders.SupplierModel` rows on startup.
2. **Carts & Coupons (`carts`, `coupons`)**:
   - `carts.Cart` and `carts.CartItem` track active shopping carts scoped to the authenticated user or HTTP session.
   - `coupons.Coupon`, `coupons.CouponProductThrough`, and `coupons.Redemption` support percentage and fixed-amount discounts, product-targeted or cart-wide rules, minimum order amounts, validity windows, and usage limits.
   - `carts.CartQuery.Calc()` / `CalcOne()` / `CalcOneWithProducts()` compute line-level and cart-level `price.Totals` inside a database transaction.
3. **Checkout Broker & Payments (`broker`, `payments`)**:
   - `broker.Broker` coordinates checkout preparation (`PrepareCheckout`) and execution (`StartCheckout`) using a `CheckoutPricer` (defaulting to the cart's payable total) and a `payments.ProviderAccount`.
   - `payments` manages `ProviderAccount` configuration, `Payment` attempts, durable `AttemptID` idempotency, free checkouts (`IsFreeCheckout()`), webhooks, and post-payment operations (`Refund`, `Void`, `Capture`, `Cancel`).
4. **Signal-Driven Order Lifecycle (`orders`, `shared/signals`, `handlers`)**:
   - Domain packages emit typed signals on `shared/signals.Pool` during cart, coupon, product, order, and payment transitions.
   - `orders.CreateOrder` / `orders.CreateOrderFromCart` snapshot cart lines and pricing into `Order` and `OrderItem` records inside a transaction and emit `orders.Signals.Created`.
   - Application workflows subscribe to these signals to advance order states (`Accepted`, `Processed`, `Completed`), trigger supplier fulfillment, and record audit logs (`handlers.RegisterAuditLogs()`).
5. **HTTP APIs, Admin & Frontend (`views/public`, `views/adminviews`, `assets`)**:
   - Public JSON endpoints under `/api/carts`, `/api/coupons`, and `/api/payments`.
   - Admin views and panels registered under `admin.RegisterApp("shop", ...)` plus an admin coupon-generator endpoint.
   - Embedded Go templates, static assets, and a TypeScript storefront API client (`assets/static_src`).

---

## Package Map

| Package | Purpose |
| :--- | :--- |
| `contrib/shop` (`app.go`, `http.go`) | `ShopAppConfig` (`SHOP`, `NewAppConfig()`), model/admin/route registration, `ShopHandler`, `ShopInstanceHandler`. |
| `contrib/shop/models` | `Category` model and `CategoryQuery`. |
| `contrib/shop/products` | `Product` base model, `ProductModel` interface, `SupplierValue`, `ProductQuery`, and product signals. |
| `contrib/shop/customers` | `Customer` model linked optionally to `users.User`. |
| `contrib/shop/carts` | `Cart`, `CartItem`, `CartQuery` (`Calc`, `CalcOne`, `CartForRequest`), `CartRow`, `CartItemRow`, and cart signals. |
| `contrib/shop/coupons` | `Coupon`, `CouponCode`, `CouponProductThrough`, `Redemption`, and coupon signals. |
| `contrib/shop/coupons/generators` | Configurable coupon code generation (`CodePart`, `GenerateCouponCode`, `CouponGeneratorWidget`). |
| `contrib/shop/orders` | `Order`, `OrderItem`, `OrderItemPricing`, `ProductData` snapshot, `OrderState`, `InternalOrderState`, `OrderQuery`, `CreateOrder`/`CreateOrderFromCart`, coupon reconciliation, supplier registry/backends, admin panels, and order signals. |
| `contrib/shop/payments` | `ProviderAccount`, `Quote`, `Payment`, `PaymentEvent`, `Provider` & capability interfaces, service lifecycle (`Start`, `Refresh`, `ServeWebhook`, `Refund`, `Void`, `Capture`, `Cancel`), and payment signals. |
| `contrib/shop/broker` | `Broker`, `CheckoutPricer`, `CheckoutRequest`, `PreparedCheckout`, `PrepareCheckout`, `StartCheckout`. |
| `contrib/shop/shared/api` | Request/response helpers (`APIRequest`, `BaseAPIRequest`, `RequestRegistry`, `Decode`, `Respond`, `ErrorResponse`, `RespondError`). |
| `contrib/shop/shared/price` | Decimal pricing calculator (`AmountValue`, `Price`, `Totals`, `PriceCalculation`). |
| `contrib/shop/shared/relations/modelkeys` | Content-type keys and lazy registries (`ProductModelKey`, `OrderModelKey`, `CartModelKey`, `CategoryModelKey`, `PaymentModelKey`, `CustomerModelKey`). |
| `contrib/shop/shared/signals` | Typed signal pool (`Pool`, `Signal[T]`, `BaseSignal`, `Changed[TYP]`, `ModelSignals`). |
| `contrib/shop/util/paystate` | `PayState` type, `PAYSTATE` checker, and 17 `STATUS_*` constants. |
| `contrib/shop/util/forms` | `NewAdminForm` helper for building `admin.AdminForm` instances. |
| `contrib/shop/handlers` | Signal listeners and `audit_logs` definitions (`RegisterAuditLogs()`). |
| `contrib/shop/views/public` | Public HTTP handlers for carts, coupon redemption, payment options, and checkout. |
| `contrib/shop/views/adminviews` | Admin list/add/edit views for products and orders, coupon generator endpoint, and generic list/filter helpers. |
| `contrib/shop/assets` | Embedded templates (`assets/templates`), built static assets (`assets/static`), and TypeScript sources (`assets/static_src`). |

---

## Installation & App Registration

Register `contrib/shop` in your Go-Django application using `shop.NewAppConfig()` (which configures and returns the `shop.SHOP` `*ShopAppConfig` singleton).

### Dependencies

`SHOP.Deps` requires the following apps to be registered before `shop`:

- `"session"`
- `"admin"`
- `"editor"`
- `"blocks"`
- `"revisions"`
- `"settings"`

`shop` embeds `*apps.DBRequiredAppConfig`, so `django.APPVAR_DATABASE` must also be configured.

### App Variables

Configure `contrib/shop` behavior via `django.Settings` / `djangoapps.CONFIG`:

| Constant | Setting Key | Default | Description |
| :--- | :--- | :--- | :--- |
| `shop.APPVAR_SHOP_CURRENCY` | `"SHOP_CURRENCY"` | `""` | ISO currency code (e.g. `"EUR"`, `"USD"`) passed to `public.PaymentsAPI` and `public.CheckoutAPI`. |
| `shop.APPVAR_SHOP_CHECKOUT_ORIGIN` | `"SHOP_CHECKOUT_ORIGIN"` | `""` | Base origin URL used by `public.CheckoutAPI` to build return and webhook URLs. |
| `shop.APPVAR_SHOP_CHECKOUT_RETURN_PATH` | `"SHOP_CHECKOUT_RETURN_PATH"` | `"/"` | Path appended to the origin for post-checkout customer redirects. |

### Example Registration

```go
package main

import (
	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/contrib/shop"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/session"
)

func setupApp() *django.Application {
	return django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE: db,
			shop.APPVAR_SHOP_CURRENCY:             "EUR",
			shop.APPVAR_SHOP_CHECKOUT_ORIGIN:      "https://shop.example.com",
			shop.APPVAR_SHOP_CHECKOUT_RETURN_PATH: "/checkout/complete",
		}),
		django.Apps(
			session.NewAppConfig,
			admin.NewAppConfig,
			editor.NewAppConfig,
			blocks.NewAppConfig,
			revisions.NewAppConfig,
			settings.NewAppConfig,
			shop.NewAppConfig,
		),
	)
}
```

On startup, `NewAppConfig()` registers:
- Content-type aliases for legacy `SupplierModel` paths, embedded templates/static files, and the funnel icon.
- Audit log definitions and signal listeners via `handlers.RegisterAuditLogs()`.
- All 14 shop models (`CartItem`, `Cart`, `OrderState`, `OrderItem`, `Order`, `ProviderAccount`, `Payment`, `Product`, `Category`, `SupplierModel`, `Coupon`, `CouponProductThrough`, `Redemption`, `Customer`).
- The `products.Product` admin chooser, admin app/model registrations, and `admin:ext:shop:generate_coupon` route.
- Public `/api/carts*`, `/api/coupons/redeem`, and `/api/payments/*` routes, a `nosurf` CSRF exemption for `/api/payments/webhook/<ULID>`, and `orders.SuppliersQuery().Sync()`.

---

## Signal-Driven Order Flow (High Level)

The shop's order lifecycle is **primarily signal-driven** rather than hardcoded into a single monolithic service:

1. **Cart & Coupon Signals**: Public API handlers emit `carts.Signals` (`Created`, `Cleared`, `Item.Add`, `Item.Remove`) and `coupons.Signals` (`Redeemed`, `Invalidated`, etc.) as shoppers build carts and apply promo codes.
2. **Checkout Preparation & Order Creation**: `broker.Broker.PrepareCheckout` (called by `StartCheckout` / `POST /api/payments/checkout`) prices the cart via `carts.Query().CartForRequest(...).CalcOneWithProducts()`, invokes `orders.CreateOrderFromCart`, and persists the `Order`, `OrderItem` snapshots (including `OrderItemPricing` and `ProductData`), and initial `Payment` inside a database transaction. `orders.CreateOrder` emits `orders.Signals.Created` within that transaction.
3. **Payment Start, Webhooks & Reconciliation**: `payments.Start`, `payments.Refresh`, `payments.ServeWebhook`, and post-payment operations (`Refund`, `Void`, `Capture`, `Cancel`) emit `payments.Signals` (`Start`, `Update`, `Refund`, `Void`, `Capture`, `Cancel`, `OperationRequested`, `OperationPending`, `OperationFailed`).
4. **Order State Progression & Fulfillment**: Applications listen to `payments.Signals.Update` and `orders.Signals` (`Created`, `Updated`, `Accepted`, `Processed`, `Completed`) to transition orders from pending to accepted/processed/completed, trigger `SupplierBackend` fulfillment, and log activity—while `handlers.RegisterAuditLogs()` automatically records audit entries for all domain signals.

---

## What You Need to Implement

`contrib/shop` provides the core models, checkout broker, HTTP endpoints, and admin UI, but expects the host application to supply its own domain-specific extensions:

1. **Product Subtypes (`products.ProductModel`)**:
   - Define your concrete product model(s) implementing `products.ProductModel` (`specific.Model` + `ProductPrice(ctx context.Context) products.Price`) and register them as Go-Django models/content types so `products.Product` and `orders.ProductData` snapshots can resolve `SpecificContentType`.
2. **Supplier Backends (`orders.SupplierBackend`)**:
   - Implement `orders.SupplierBackend` (and optionally `orders.CatalogableSupplierBackend`) and register each backend with `orders.RegisterSupplier(...)` before `NewAppConfig()`'s `Ready` hook runs `orders.SuppliersQuery().Sync()`.
3. **Payment Providers (`payments.Provider` & `payments.RegisterProvider`)**:
   - Implement `payments.Provider` (`CreatePayment`, `GetPayment`, `HandleWebhook`) and any optional capability interfaces (`payments.Refunder`, `payments.Voider`, `payments.Capturer`, `payments.Canceller`, `payments.OperationReferenceResolver`), register a `payments.ProviderFactory` via `payments.RegisterProvider(key, factory)`, and create matching `payments.ProviderAccount` records in the admin or database.
4. **Signal-Driven Order Workflow (`orders.Signals` & `payments.Signals`)**:
   - Subscribe to `payments.Signals.Update` (and operation signals) and `orders.Signals` (`Created`, `Updated`, `Accepted`, `Processed`, `Completed`) to advance `Order.State` / `Processed` / `Completed`, trigger supplier fulfillment, and send customer notifications.
5. **App Variables & Checkout Configuration**:
   - Configure `SHOP_CURRENCY`, `SHOP_CHECKOUT_ORIGIN`, and `SHOP_CHECKOUT_RETURN_PATH` in `django.Configure(...)` (or provide a custom `broker.CheckoutPricer` if using `broker.Broker` directly).
6. **Storefront Templates & Frontend Wiring**:
   - Provide your storefront templates (product catalog, cart, checkout pages) and wire the TypeScript storefront API client / Stimulus controller (`assets/static_src/public/*`, built to `/static/shop/public/js/index.js`) to your UI.

---

## Documentation Index

- [Configuration & Models](./docs/configuration-and-models.md) — `ShopAppConfig`, handlers, `shared/price`, `modelkeys`, `paystate`, `util/forms`, and all data models & interfaces.
- [HTTP API](./docs/http-api.md) — `shared/api` helpers and all public/admin HTTP endpoints, schemas, auth/CSRF rules, and status codes.
- [Frontend TypeScript API](./docs/frontend-api.md) — Storefront TypeScript API client, types, Stimulus/UI controllers, and admin widgets.
- [Coupons & Discounts](./docs/coupons.md) — Coupon models, code generators, redemption, usage accounting, `CartQuery.Calc()` discount rules, and order reconciliation.
- [Payments & Checkout](./docs/payments-and-checkout.md) — `broker.Broker`, checkout flow, free checkouts, webhooks, `payments` service operations, and `Provider` capability interfaces.
- [Signals & Order Flow](./docs/signals-and-order-flow.md) — `shared/signals`, all domain signals across `products`/`carts`/`coupons`/`orders`/`payments`, `CreateOrder`, state transitions, and audit logging.
- [Suppliers](./docs/suppliers.md) — `SupplierBackend`, `CatalogableSupplierBackend`, `BoundSupplierBackend`, supplier registry, `SuppliersQuery().Sync()`, and product/order integration.
- [Admin Integration](./docs/admin.md) — Admin model registrations, custom admin views, order coupon/payment panels, and generic list/filter utilities.
