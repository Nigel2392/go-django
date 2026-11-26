# Build a shop checkout with GO Django

Run a checkout with a database supplier and a simulated payment provider. Then use the same interfaces to connect your own catalogue, pricing, and payment accounts.

## Run the checkout

From the repository root, use the Go version in `go.work` and a C compiler for the SQLite driver. Run the tutorial with CGO enabled.

```sh
go run ./examples/commerce/tutorial
```

The program uses an in-memory database. It creates a category, a notebook, a customer, a cart with two notebooks, and a payment account. The notebooks cost EUR 10.00 each. A 20% tax and a EUR 0.50 payment fee bring the checkout total to EUR 24.50.

```text
Prepared order <id> with payment <id> for EUR 24.50
Opened checkout in STATUS_PENDING at https://checkout.example.test/pay/<id>
Reconciled payment in STATUS_PAID. Order is order_pending.
```

Change the cart quantity in `seedCheckout` and run it again to see the recalculated amount. The simulation marks the payment as paid and leaves order acceptance to your application. [Runnable checkout](main.go)

## Install the shop in your application

Add `shop.NewAppConfig` to `django.Apps` with its dependencies. Set `shop.APPVAR_SHOP_CURRENCY` to the currency you use for cart pricing, and configure a database and media backend. Create and apply migrations for your installed apps and supplier models before serving. The tutorial creates its disposable schema during initialization. [App setup](main.go) · [Commerce application](../src/main.go)

## Add a category and a supplier

1. Create a `models.Category` for your products. Use `models.Categories().ChildrenOf(nil)` for the root catalogue categories or pass a parent to find its children.
2. Implement `products.ProductModel` for the supplier-specific product. The database supplier embeds `*products.Product` and adds a `SupplierID` field. Register the concrete type in your app's models so its content type can be resolved when orders are loaded.
3. Implement `orders.SupplierBackend` and call `orders.RegisterSupplier` during app configuration. Use a stable `InternalName` as the value of `products.Product.Supplier`. The product admin calls the backend's `ProductType`, `ProductAddPanel`, and `ProductEditPanel`. The shop syncs registered suppliers to `orders.SupplierModel` at startup.

Use `orders.Supplier` when callbacks are enough. For a dedicated backend, add a compile-time check like `var _ orders.SupplierBackend = (*MySupplier)(nil)`. Use `OrderBlock` to describe the fields stored in `OrderItem.SupplierData`. The `OrderAccepted` callback is available for an acceptance workflow that you wire explicitly. [Database supplier implementation](../src/suppliers/dbsupplier/app.go) · [Concrete product model](../src/suppliers/dbsupplier/models.go)

## Build the cart and apply coupons

Resolve a `customers.Customer` for the buyer. In a storefront request, use `carts.Query().WithContext(r.Context()).CartForRequest(r)` to find the active cart from the authenticated user or session. The public cart endpoints handle cart creation and item changes. Use `CalcOne()` when you need the totals and `CalcOneWithProducts()` when you also need complete products for order snapshots. [Cart API](../../../contrib/shop/views/public/cart.go) · [Cart queries](../../../contrib/shop/carts/models_cart_query.go)

To try a coupon in a running storefront, create a `coupons.Coupon` in the admin, add an eligible product to the cart, and send `{"coupon_code":"WELCOME"}` to the URL for `api:coupons:redeem`. The response includes the recalculated cart. A coupon can target particular products through `CouponProductThrough`. Use `generators.GenerateCouponCode` to generate an available code for a new coupon. For example, a single static part can supply a recognizable code in a development database. [Coupon models](../../../contrib/shop/coupons) · [Redemption handler](../../../contrib/shop/views/public/coupons.go)

```go
code, err := generators.GenerateCouponCode(ctx, []generators.CodePart{
    {Type: generators.CPT_STATIC, Value: "WELCOME"},
})
```

Read `cart.Totals.Payable()` for the amount after cart discounts, including product fees and tax. Use `ProviderAccount.Quote` for the payment account fee and currency rounding. The public payment options endpoint already combines these for the active cart. [Price calculation](../../../contrib/shop/shared/price/price.go) · [Payment options handler](../../../contrib/shop/views/public/payment_options.go)

## Register a payment provider

Implement `payments.Provider` in your application's payment provider package and register a factory with `payments.RegisterProvider`. Set the account's `Driver` to that registration key and its `Currency` to the currency it accepts. When the factory needs credentials, use `ProviderAccount.CredentialRef` to look up the secret. [Runnable provider](paymentproviders/demo.go) · [Provider interface](../../../contrib/shop/payments/payments.go)

| Method | Adapter responsibility |
| --- | --- |
| `CreatePayment` | Open checkout with the supplied amount and URLs. Use `AttemptID` as the provider idempotency key. |
| `GetPayment` | Look up a payment by provider reference and return its normalized state, amount, and currency. |
| `HandleWebhook` | Authenticate the provider's callback or verify the reported status with the provider before returning it. |

Implement `payments.Capturer`, `payments.Voider`, `payments.Refunder`, or `payments.Canceller` when your provider supports those operations. Keep the same `OperationRequest.RequestID` when retrying an operation. Use `payments.ResolveOperation` with the verified outcome when the provider reports it later. [Payment service](../../../contrib/shop/payments/service.go)

## Connect checkout to the storefront

For the built-in checkout, set `shop.APPVAR_SHOP_CHECKOUT_ORIGIN` to your public shop origin and optionally set `shop.APPVAR_SHOP_CHECKOUT_RETURN_PATH` to your order confirmation page. Use `django.Reverse("api:payments:checkout")` as the action of a POST form. Include the framework CSRF token and an `account_id` field with the ID from the payment options response. The buyer must be signed in with an email address on their user or linked customer record. The view also accepts a JSON body. It converts the cart once and returns a 303 to the provider. If the provider call fails, submit the same account again to resume the saved attempt. The return URL includes `payment_id` for looking up the payment. [Checkout handler](../../../contrib/shop/views/public/payment_checkout.go)

For a custom checkout, call the broker from your storefront handler.

1. Resolve the customer and authorize access to the cart in your handler. Let the buyer select an eligible payment account. Issue one stable attempt ID for this checkout submission and keep it for retries.
2. Call `broker.Broker.PrepareCheckout` with the authorized inputs. The broker prices the cart on the server, saves the order and product snapshots, and creates a payment attempt in the same transaction. `orders.Signals.Created` runs inside the order transaction. Use that context for database writes that should roll back with the order.
3. Call `Broker.StartCheckout` after commit. Redirect the buyer to the returned checkout URL when the provider supplies one. Build the return and webhook URLs from your configured storefront origin. Use `django.Reverse("api:payments:webhook", account.ID)` for the webhook path.
4. If checkout fails to open, retry the saved attempt using the same ID and URLs. If the submission response was lost, repeat preparation with the original customer, account, and attempt ID to recover the saved order and payment.
5. Use `payments.Refresh` or the registered webhook endpoint to reconcile payment status. Use that verified status in your application's acceptance policy.

The broker takes its default amount from the cart. To include a EUR 4.00 shipping charge in this example, implement `broker.CheckoutPricer` and set `Pricer: shippingPricer{}` on the broker used in `run`. [Broker source](../../../contrib/shop/broker/checkout.go)

```go
type shippingPricer struct{}

func (shippingPricer) CheckoutSubtotal(_ context.Context, cart *carts.CartRow) (payments.Money, error) {
    return payments.Money{
        Amount:   cart.Totals.Payable().Add(decimal.NewFromInt(4)),
        Currency: "EUR",
    }, nil
}
```

## Customize the shop admin

Use the supplier's panels to add product fields. Use the shop's admin views for the existing product and order workflows. For a custom list, start with `generic.ListViewConfig` and pass your model, queryset, filters, and columns. For a custom edit form, `util/forms.NewAdminForm` binds a model and its panels to a request. [Product and order views](../../../contrib/shop/views/adminviews) · [List configuration](../../../contrib/shop/views/adminviews/generic/listview.go) · [Form helper](../../../contrib/shop/util/forms/forms.go)

## Extend the order lifecycle

Use `broker` for workflows that cross package boundaries. Keep order transitions and supplier capabilities in `orders`, and keep provider calls and payment status transitions in `payments`. Use `shared/relations/modelkeys` with deferred relations when two model packages need to refer to each other. [Deferred payment to order relation](../../../contrib/shop/payments/models_payment.go)

When adding acceptance and fulfillment, let the application decide when to accept a verified payment. Account for both `payments.Signals.Start` and `payments.Signals.Update`. The initial provider response can already be paid, and later updates can come from a webhook or refresh. Record the acceptance and enqueue the fulfillment jobs in one database transaction. Give each job a stable key, retry schedule, and lease. Run supplier work in a worker after that transaction commits, and reuse the job key when retrying. This design can run on the shop database. An application that needs an external broker can publish committed jobs through an outbox.

Run the shop integration suite after changing a workflow or an adapter.

```sh
go test -tags "testing_auth test sqlite" ./contrib/shop/... ./examples/commerce/...
```

## Package map

| Shop package | Where to use it |
| --- | --- |
| [`shop`](../../../contrib/shop) | Install the app, configure currency, and register the stock routes and models. |
| [`broker`](../../../contrib/shop/broker) | Coordinate order and payment preparation and start a committed checkout. |
| [`carts`](../../../contrib/shop/carts) | Resolve active carts and calculate totals with products and redemptions. |
| [`coupons`](../../../contrib/shop/coupons) | Define discount rules, product targets, and redemptions. |
| [`coupons/generators`](../../../contrib/shop/coupons/generators) | Generate coupon codes and render the coupon generator widget. |
| [`customers`](../../../contrib/shop/customers) | Store the customer record associated with an order. |
| [`handlers`](../../../contrib/shop/handlers) | Follow the built-in cart and product signal listeners for audit logging. |
| [`models`](../../../contrib/shop/models) | Build the category tree and query its children. |
| [`orders`](../../../contrib/shop/orders) | Create orders and line snapshots, register suppliers, and subscribe to order signals. |
| [`payments`](../../../contrib/shop/payments) | Register providers, quote fees, open checkout, and reconcile payment attempts. |
| [`products`](../../../contrib/shop/products) | Define the common product record and the interface for concrete supplier products. |
| [`shared/api`](../../../contrib/shop/shared/api) | Read JSON and write JSON responses or errors in custom shop HTTP handlers. |
| [`shared/price`](../../../contrib/shop/shared/price) | Calculate line totals, discounts, fees, and taxes with decimal amounts. |
| [`shared/relations/modelkeys`](../../../contrib/shop/shared/relations/modelkeys) | Reference model types through deferred relations across packages. |
| [`shared/signals`](../../../contrib/shop/shared/signals) | Use the shop's typed signal pool and shared request or change payloads. |
| [`util/forms`](../../../contrib/shop/util/forms) | Build a shop admin form from a model and its panels. |
| [`util/paystate`](../../../contrib/shop/util/paystate) | Use the normalized payment states shared by provider adapters and the admin. |
| [`views/adminviews`](../../../contrib/shop/views/adminviews) | See how the stock admin handles supplier selection, order editing, and coupon generation. |
| [`views/adminviews/generic`](../../../contrib/shop/views/adminviews/generic) | Reuse list view configuration, filters, and pagination controls. |
| [`views/public`](../../../contrib/shop/views/public) | Use the cart, coupon redemption, payment options, and checkout HTTP handlers. |
