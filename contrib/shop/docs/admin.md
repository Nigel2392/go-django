# Shop Admin

`contrib/shop` registers a `"shop"` admin application (`Commerce`) in [`contrib/shop/app.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/app.go), providing model CRUD, custom supplier-aware product views, order inspection views with payment history and coupon redemption panels, a coupon code generator endpoint, and reusable generic list/filter helpers.

---

## App Registration, Hooks, and Choosers

In `NewAppConfig()` / `SHOP.Init` ([`contrib/shop/app.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/app.go)):

- **Admin App**: `admin.RegisterApp("shop", admin.AppOptions{...})`
  - `AppLabel`: `"Shop"`
  - `MenuLabel`: `"Commerce"`
  - `FullAdminMenu`: `true`
  - `RegisterToAdminMenu`: `true`
  - `EnableIndexView`: `true`
  - `AppDescription`: `"The commerce module is intended to manage your webshop, such as product stock, orders, refunds, etc."`
  - `MenuIcon`: inline Bootstrap `bi-cart4` SVG
- **Custom Admin Menu (`SHOP.ADMIN_MENU`)**: Exposes `index`, `products`, `orders`, `coupons`, and permission-gated (`admin:list`) links for `customers`, `provider_accounts`, `payments`, `carts`, `redemptions`, and `order_states`.
- **Product Chooser (`chooser.Register`)**: Registers `*products.Product` as `"Product Chooser"` with list fields `["Title", "Supplier", "Price", "TaxRate", "DiscountPrice"]`, ordered by `"Title", "-Price"`, and `icontains` search on `Title`.
- **Custom Admin Routes (`admin.RegisterRouteHook`)**: Mounts `commerce/products/add/<<supplier>>` (named `admin:commerce:new_with_supplier`) handled by `ShopHandler{Handle: adminviews.ViewAddProduct}`.
- **Extension Routing (`app.Routing`)**: Mounts `POST /shop/coupon/generate` (named `admin:ext:shop:generate_coupon`) handled by `adminviews.GenerateCouponCode`.
- **Home Page Display Panel (`admin.RegisterHomePageDisplayPanelHook`)**: Registers a display panel counting `&products.Product{}`.
- **Settings Menu Hook (`settings.SettingsMenuHook`)**: Adds a `"Payment Providers"` link to `provider_accounts` when the user has permission `"shop:payments:providers"`.
- **Shared CRUD Helpers**: `createModel`, `updateModel`, and `deleteModels` (transactional) are used by `Customer`, `ProviderAccount`, and `OrderState`.

---

## Registered Admin Models

All models below are registered under the `"shop"` admin app in [`contrib/shop/app.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/app.go):

| Name | Model | `MenuOrder` | `MenuLabel` | In Menu | Restrictions | List / Form Configuration |
| --- | --- | --- | --- | --- | --- | --- |
| *(default)* | `&models.Category{}` | `100` | `Category` | Yes | — | List: `Name`, `ParentCategory`; `icontains` search on `Name`. Add/Edit: `Name`, `Slug`, `ParentCategory`, and collapsible `ModelFormPanel` for `Children` (`MinNum: 1`, fields `Name`, `Slug`). |
| `products` | `&products.Product{}` | `200` | `Products` | Yes | — | List: `adminviews.ViewProductList`. Add: `ShopHandler{Handle: adminviews.ViewAddNewProductSupplier}`. Edit: `ShopInstanceHandler[*products.Product]{Handle: adminviews.ViewEditProduct}`. |
| `orders` | `&orders.Order{}` | `300` | `Orders` | Yes | — | List: `adminviews.ViewOrderList`. Add: `ShopHandler{Handle: adminviews.ViewAddOrder}`. Edit: `ShopHandler{Handle: adminviews.ViewEditOrder}`. |
| `customers` | `&customers.Customer{}` | `500` | `Customers` | Yes | — | List: `Email`, `FirstName`, `LastName`, `FirstSeen`; ordered by `-FirstSeen`; `icontains` search on `Email`, `FirstName`, `LastName`. Add/Edit: `Email` + row(`FirstName`, `LastName`). Delete: `deleteModels`. |
| `provider_accounts` | `&payments.ProviderAccount{}` | `600` | `Payment providers` | No *(in Settings menu)* | — | List: `Label`, `Driver`, `Enabled`, `Currency`, `FixedFee`, `PercentageFee`; ordered by `Label`; `icontains` search on `Label`, `Driver`. Add/Edit: `TitlePanel(Label)`, row(`Driver`, `Enabled`), `CredentialRef`, `Currency`, row(`FixedFee`, `PercentageFee`). Delete: `deleteModels`. |
| `payments` | `&payments.Payment{}` | `700` | `Payments` | Yes | `DisallowCreate`, `DisallowEdit`, `DisallowDelete` | List: `ID`, `Order`, `Account`, `Provider`, `TransactionID`, `Amount`, `Fee`, `Currency`, `State`, `CreatedAt`; ordered by `-CreatedAt`; `icontains` search on `ID`, `TransactionID`, `Provider`, `State`. |
| `carts` | `&carts.Cart{}` | `800` | `Carts` | Yes | `DisallowCreate`, `DisallowEdit`, `DisallowDelete` | List: `ID`, `Active`, `CreatedAt`; ordered by `-CreatedAt`. |
| `coupons` | `&coupons.Coupon{}` | `900` | `Coupons` | Yes | `DisallowDelete` | Custom labels (`ShortDescription` -> `Description`, `UsageCount` -> `Uses`, `UsageLimit` -> `Limit`, `ValidFrom` -> `Valid From`, `ValidUntil` -> `Valid Until`, `CreatedAt` -> `Created`). List: `ID`, `Code`, `Title`, `ShortDescription`, `UsageCount`, `UsageLimit`, `ValidFrom`, `ValidUntil`, `CreatedAt`; `icontains` search on `Code`, `Title`, `ShortDescription`, `CreatedBy.Email`. Add/Edit: 3 tabs (`Coupon`, `Constraints`, `Products` via collapsible `CouponProductThrough` `ModelFormPanel` with `MinNum: 1`); Edit tab also shows `CreatedBy`, `UsageCount`, `CreatedAt`, `UpdatedAt`. |
| `redemptions` | `&coupons.Redemption{}` | `900` | `Coupon redemptions` | Yes | `DisallowCreate`, `DisallowEdit`, `DisallowDelete` | List: `ID`, `Coupon`, `DiscountAmount`, `Invalidated`, `RedeemedAt`; ordered by `-RedeemedAt`. |
| `order_states` | `&orders.OrderState{}` | `1000` | `Order states` | Yes | — | List: `ID`, `Label`, `Action`, `WorkflowID`. Add: `ID`, `Label`, `Action`, `WorkflowID`. Edit: `Label`, `Action`, `WorkflowID`. Delete: `deleteModels`. |

---

## Custom Admin Views

### `home.go`

[`contrib/shop/views/adminviews/home.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/home.go) defines `Home(w http.ResponseWriter, r *http.Request)` as an empty placeholder. The shop admin index is served by the core `admin` app (`EnableIndexView: true`), supplemented by `RegisterHomePageDisplayPanelHook`.

### Product Views (`view_product.go`)

Defined in [`contrib/shop/views/adminviews/view_product.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/view_product.go):

- **`ViewProductList`**: A `generic.ListViewConfig[*products.Product]` requiring permission `"products:list"`.
  - Preloads `SelectRelated("Category")`.
  - `search` filter matches `Title`, `Slug`, `Supplier`, or `Category.Name` (`icontains`, `Distinct()`).
  - Header actions: `generic.AmountHeaderAction` and an `"Add"` link to the product add route.
  - Columns: `Title` (edit link), `ID`, `Price`, `DiscountPrice`, `Category`, `Supplier`.
- **`ViewAddNewProductSupplier`**: Requires `"products:add"`. Queries `orders.SuppliersQuery().OrderBy("Enabled")`, wraps each row in `orders.BoundSupplierBackend` (falling back to `&orders.Supplier{Name: ..., TextLabel: ...}` if not in the in-memory registry), sorts enabled suppliers first then by `Ordering()` / `InternalName()`, and renders `shop/products/admin/suppliers.tmpl` with `definitions` and `BackURL`.
- **`ViewAddProduct`**: Requires `"products:add"`. Resolves `mux.Vars(r).Get("supplier")` via `orders.RetrieveSupplier` (404 if missing), opens a transaction, constructs a form from `supplierVal.ProductType` and `supplierVal.ProductAddPanel`, saves both the specific `ProductModel` and the base `*products.Product` (setting `product.Supplier`), fires `products.Signals.Created`, commits the transaction, and redirects to the product list (`shop/products/admin/add.tmpl`).
- **`ViewEditProduct`**: Requires `"products:edit"`. Loads the specific product via `instance.Specific.Specific(ctx, true)`, clones `originalProduct` and `originalInstance` on `POST` for change tracking, opens a transaction, resolves `orders.RetrieveSupplier(instance.Supplier)`, builds the form with `backend.ProductEditPanel`, saves both models, fires `products.Signals.Updated`, commits the transaction, and redirects (`shop/products/admin/edit.tmpl`).

### Order Views (`view_order.go`)

Defined in [`contrib/shop/views/adminviews/view_order.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/view_order.go):

- **`ViewOrderList`**: A `generic.ListViewConfig[*orders.Order]` requiring permission `"orders:list"`.
  - Preloads `Payments` via `queries.NoJoins("Payments")`.
  - `search` filter matches `Order.ID` (`icontains`) or orders linked to `payments.Payment` rows whose `ID` or `Provider` matches (`icontains`).
  - Header action: `generic.AmountHeaderAction`.
  - Columns: `ID` (`Order ID`, edit link) and a computed `Payment IDs` column joining `order.Payments.AsList()` IDs.
- **`ViewAddOrder`**: Requires `"orders:add"`. Uses `order.AddPanels`, saves the `*orders.Order` and its `OrderItems` formset, sends `orders.Signals.Created`, and renders `shop/orders/admin/add.tmpl`.
- **`ViewEditOrder`**: Requires `"orders:edit"`. Queries the order by `model_id` with `Preload(queries.NoJoins("OrderItems"))`, builds `order.EditPanels(r.Context())`, saves the order and `OrderItems`, calls `orders.ReconcileCouponRedemptions(ctx, p.ID)`, sends `orders.Signals.Updated`, and renders `shop/orders/admin/edit.tmpl`.

### Coupon Code Generator (`view_coupon_generate.go`)

Defined in [`contrib/shop/views/adminviews/view_coupon_generate.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/view_coupon_generate.go):

- **`GenerateCouponCode`**: Checks object permission `"coupons:generate"` on `&coupons.Coupon{}` (calling `admin.ReLogin` if unauthorized). Decodes `CouponGenerationRequest` (`code_format_parts: []generators.CodePart`), calls `generators.GenerateCouponCode(r.Context(), request.Parts)`, and returns `200 OK` with `CouponGenerationResponse` (`coupon_code: string`), or `400 Bad Request` on decode/generation errors (`500` on response encoding failure).

---

## Order Edit Panels: Payment History and Coupon Panels

In [`contrib/shop/orders/models_order.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/orders/models_order.go), `Order.EditPanels(ctx)` builds a `TabbedPanel` with four tabs:
1. **Product Details**: `ID`, `State`, `Processed`, `Completed`
2. **Order Items**: Collapsible `ModelFormPanel` for `OrderItems` (`ID`, `Amount`)
3. **Payment Details**: One `*admin.AlertPanel` per associated `payments.Payment` (or an `ALERT_ERROR` panel if none exist)
4. **Redeemed Coupons**: One `*admin.AlertPanel` per `coupons.Redemption` from `m.couponPanels(ctx)` (or an `ALERT_INFO` panel if none exist)

### `paymentHistoryHTML` (`orders/payment_history.go`)

[`contrib/shop/orders/payment_history.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/orders/payment_history.go) renders HTML for each payment's `AlertPanel`:
- Alert severity in `Order.EditPanels` maps `paystate` constants: `ALERT_SUCCESS` (`STATUS_PAID`, `STATUS_NO_PAYMENT_REQUIRED`), `ALERT_WARNING` (`STATUS_VERIFY`, `STATUS_AUTHORIZE`, `STATUS_REFUNDING`, `STATUS_REFUND`, `STATUS_PARTLY_CAPTURED`), `ALERT_ERROR` (`STATUS_CANCEL`, `STATUS_EXPIRED`, `STATUS_DENIED`, `STATUS_FAILURE`, `STATUS_CHARGEBACK`, `STATUS_PARTIAL_REFUND`), and `ALERT_INFO` otherwise.
- Summary line: `Provider · Amount Currency · State` (omitting `Provider` when `payment.IsFreeCheckout()`).
- Fee breakdown: subtotal before payment fee (`Amount.Sub(Fee)`), customer payment fee (`Fee`), and optional `ProviderFee`.
- Ordered list of `payment.History.Data` events (`EventPrepared`, `EventNoPaymentRequired`, `EventCheckoutOpened`, `EventCheckoutFailed`, `EventStatusUpdated`, `EventProviderFeeUpdated`, `EventOperation` formatted via `paymentOperationLabel`), including UTC timestamp, state, optional `RequestID`, and optional `ProviderFee`.
- Fallback section (`Earlier operations without timestamps`) for entries in `payment.Operations.Data` not already present in `payment.History.Data`.

### `Order.couponPanels` (`orders/coupon_panels.go`)

[`contrib/shop/orders/coupon_panels.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/orders/coupon_panels.go) queries `coupons.Redemption` rows for `Order = m.ID` (selecting `Coupon.Code` and `Coupon.Title`) and loads `m.OrderItems`:
- Alert type and state label per redemption:
  - `Invalidated` -> `ALERT_WARNING` (`"Reservation released"`)
  - `Counted` -> `ALERT_SUCCESS` (`"Counted toward coupon usage"`)
  - Default -> `ALERT_INFO` (`"Reserved for this order"`)
- Displays coupon title, `RedeemedAt` UTC timestamp, total `DiscountAmount`, and a list of per-line allocations from `item.Pricing.Data.RedemptionAmounts[redemption.ID]` (showing product title or item ID, discount amount, and `pricing.Totals.Payable()`).

---

## Generic Admin Helpers (`views/adminviews/generic/*`)

### `generic/listview.go`

[`contrib/shop/views/adminviews/generic/listview.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/generic/listview.go) defines `ListViewConfig[T attrs.Definer]`, which implements `TakeControl(w, r, _)`:
- Enforces `permissions.HasObjectPermission(r, l.Model, l.Permissions...)` (redirects via `admin.ReLogin` on failure).
- Wraps columns from `GetColumns(r)` with `columns.NewSortableColumnBuilder(l.Model)` and sorts the queryset by `r.URL.Query()["sort"]`.
- Optionally prepends `columns.ListActionsColumn[T]` from `GetListActions(r)` and wraps column `0` with `list.TitleFieldColumn` using `GetEditLink(r, row)`.
- Applies `filters.NewFilters[T](ctx, "filters")` when `Filters` is non-empty; if `UsePanelFilters` is `true`, appends `admin.SidePanelFilters` to `PageOptions.SidePanels`.
- Delegates rendering to `list.View[T]` using `BaseTemplateKey: "shop"` and `TemplateName: "shop/admin/models/list.tmpl"`, with `DefaultAmount: 25` and `MaxAmount: 100` (`AMOUNT_OPTIONS`).

### `generic/filters.go`

[`contrib/shop/views/adminviews/generic/filters.go`](file:///c:/Users/PvPBe/Desktop/dev/agents/godjango/contrib/shop/views/adminviews/generic/filters.go) provides pagination size helpers:
- `AMOUNT_OPTIONS = []int{25, 50, 75, 100}`
- `FilterSpecAmount[T attrs.Definer](name string, amounts []int, opts ...func(*filterSpecOpts)) filters.FilterSpec[T]`: Defaults `name` to `"amount"` and default value to `"10"`. Configurable via `OptFilterSpecFormField`, `OptFilterSpecApplyAmount`, and `OptFilterSpecDefaultAmount`. (Note: in the current implementation, `OptFilterSpecApplyAmount(b bool)` sets `fso.useApply = false` regardless of `b`.)
- `AmountHeaderAction(r, currentAmount, reverseURL, amounts, reverseArgs...)`: Returns a `components.ShowableComponent` rendering a `GET` `<form>` that preserves existing query parameters (excluding `page` and `amount`) as hidden inputs and submits on `<select name="amount">` change.
