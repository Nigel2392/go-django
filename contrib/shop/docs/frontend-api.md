# Shop Frontend & TypeScript API

This document describes the TypeScript client, Stimulus controllers, and custom web components under `contrib/shop/assets/static_src/`.

---

## Source Layout

| File | Purpose |
|---|---|
| `assets/static_src/public/api/types.ts` | TypeScript interfaces, enums, and `ShopAPI` contract map |
| `assets/static_src/public/api/index.ts` | Re-exports everything from `./types` |
| `assets/static_src/public/controller.ts` | `ShopController` Stimulus controller for storefront API calls |
| `assets/static_src/public/index.ts` | Registers `ShopController` on `window.Stimulus` as `"shop"` |
| `assets/static_src/public/menu-icon.ts` | `<django-menu-icon>` custom web component (`MenuIcon`) |
| `assets/static_src/admin/widgets/CouponGenerator.ts` | `CouponGeneratorController` Stimulus controller for admin coupon widget |
| `assets/static_src/admin/index.ts` | Registers `CouponGeneratorController` on `window.Stimulus` as `'coupon-generator'` |

---

## Public API Types (`public/api/types.ts`)

Re-exported from `public/api/index.ts`.

### Scalar & Utility Types

- **`ULID`**: `string` — ULID serialized as a JSON string.
- **`Timestamp`**: `string` — RFC 3339 timestamp serialized as a JSON string.
- **`Decimal`**: `string` — `shopspring/decimal` serialized as a JSON string.
- **`ErrorResponse`**:
  ```ts
  export interface ErrorResponse {
      message: string;
  }
  ```
- **`APIPromise<T>`**: `Promise<T | ErrorResponse>` — Return type of all `ShopController` API methods.

### Domain & Response Interfaces

- **`Totals`**:
  ```ts
  export interface Totals {
      total_price: Decimal;
      total_fee: Decimal;
      total_tax: Decimal;
      total_discount: Decimal;
      left_over_discount: Decimal;
  }
  ```
- **`Cart`**:
  ```ts
  export interface Cart {
      id: ULID;
      created_at: Timestamp;
      active: boolean;
  }
  ```
- **`Category`**:
  ```ts
  export interface Category {
      ID: number;
      Name: string;
      Slug: string;
      ParentCategory: Category | null;
      Children: Category[] | null;
  }
  ```
- **`EditorBlock`** & **`EditorData`**:
  ```ts
  export interface EditorBlock {
      id: string;
      type: string;
      data: Record<string, unknown>;
      tunes?: Record<string, unknown>;
  }

  export interface EditorData {
      time: number;
      blocks: EditorBlock[] | null;
      version: string;
  }
  ```
- **`Product`**:
  ```ts
  export interface Product {
      id: ULID;
      title: string;
      image?: string;
      slug?: string;
      ranking?: number;
      price?: Decimal;
      additional_fee?: Decimal;
      tax_rate?: Decimal;
      discount_price?: Decimal;
      supplier?: string;
      category?: Category;
      editor?: EditorData;
      specific_id?: string;
      specific_content_type?: string;
      Coupons: unknown[] | null;
  }
  ```
- **`CartItem`**:
  ```ts
  export interface CartItem {
      id: ULID;
      product?: Product;
      amount: number;
  }
  ```
- **`Coupon`** *(Go casing preserved because the Go struct has no JSON tags)*:
  ```ts
  export interface Coupon {
      ID: number;
      Title: string;
      ShortDescription: string;
      Code: string;
      IsPercentage: boolean;
      MaxDiscount: Decimal;
      MinOrderAmount: Decimal;
      UsageLimit: number;
      UsageCount: number;
      ValidFrom: Timestamp;
      ValidUntil: Timestamp;
      CreatedAt: Timestamp;
      UpdatedAt: Timestamp;
      CreatedBy: unknown;
      RedemptionSet: Redemption[] | null;
      Products: Product[] | null;
  }
  ```
- **`Redemption`**:
  ```ts
  export interface Redemption {
      ID: number;
      Coupon: Coupon | null;
      Customer: unknown;
      Cart: Cart | null;
      Order: unknown;
      DiscountAmount: Decimal;
      Invalidated: boolean;
      RedeemedAt: Timestamp;
  }
  ```
- **`CartItemRow`**:
  ```ts
  export interface CartItemRow {
      item: CartItem | null;
      coupons: Redemption[] | null;
      total_price: Totals;
  }
  ```
- **`CartRow`** *(returned by `getCart`, `clearCart`, `redeemCoupon`)*:
  ```ts
  export interface CartRow {
      cart: Cart | null;
      items: CartItemRow[] | null;
      coupons: Redemption[] | null;
      total_price: Totals;
  }
  ```
- **`CartResponse`**:
  ```ts
  export interface CartResponse {
      cart: Cart | null;
      items: CartItem[] | null;
  }
  ```
- **`CartItemRequest`** & **`CartItemResponse`**:
  ```ts
  export interface CartItemRequest {
      product_id: ULID;
      amount: number;
  }

  export interface CartItemResponse {
      cart: Cart | null;
      item: CartItem | null;
      product: Product | null;
  }
  ```
- **`RedemptionRequest`**:
  ```ts
  export interface RedemptionRequest {
      coupon_code: string;
  }
  ```
- **`Quote`** & **`PaymentOption`**:
  ```ts
  export interface Quote {
      subtotal: Decimal;
      fee: Decimal;
      total: Decimal;
      currency: string;
  }

  export interface PaymentOption extends Quote {
      account_id: ULID;
      label: string;
  }
  ```
  *(Note: When the payable total is zero, the server may also include `no_payment_required: true` on the `PaymentOption` object.)*
- **`CodePartType`**, **`CodePart`**, **`CouponGenerationRequest`**, **`CouponGenerationResponse`**:
  ```ts
  export enum CodePartType {
      Random = 0,
      Static = 1,
      Year = 2,
      Month = 3,
      Weekday = 4,
      Day = 5,
      Hour = 6,
      Minute = 7,
      Dash = 8,
      UUID = 9,
  }

  export interface CodePart {
      type: CodePartType;
      length: number;
      value: string;
  }

  export interface CouponGenerationRequest {
      code_format_parts: CodePart[] | null;
  }

  export interface CouponGenerationResponse {
      coupon_code: string;
  }
  ```
- **`ShopAPI`**:
  Maps operation keys (`getCart`, `clearCart`, `addCartItem`, `removeCartItem`, `redeemCoupon`, `paymentOptions`) to `{ request, response }` pairs.

---

## Public Stimulus Controller (`ShopController`)

- **Registered Name:** `"shop"` (registered in `assets/static_src/public/index.ts` on `window.Stimulus`).
- **Class:** `export class ShopController extends Controller<ShopElement>`

### Stimulus Values & HTML Usage

| Value | Type | Default | Data Attribute |
|---|---|---|---|
| `getCartUrl` | `String` | `"/api/carts"` | `data-shop-get-cart-url-value` |
| `clearCartUrl` | `String` | `"/api/carts/clear"` | `data-shop-clear-cart-url-value` |
| `addCartItemUrl` | `String` | `"/api/carts/items/add"` | `data-shop-add-cart-item-url-value` |
| `removeCartItemUrl` | `String` | `"/api/carts/items/remove"` | `data-shop-remove-cart-item-url-value` |
| `redeemCouponUrl` | `String` | `"/api/coupons/redeem"` | `data-shop-redeem-coupon-url-value` |
| `paymentOptionsUrl` | `String` | `"/api/payments/options"` | `data-shop-payment-options-url-value` |
| `csrfToken` | `String` | *(none)* | `data-shop-csrf-token-value` |

Example markup:
```html
<div
    data-controller="shop"
    data-shop-get-cart-url-value="{{ url "api:carts" }}"
    data-shop-clear-cart-url-value="{{ url "api:carts:clear" }}"
    data-shop-add-cart-item-url-value="{{ url "api:carts:add_item" }}"
    data-shop-remove-cart-item-url-value="{{ url "api:carts:remove_item" }}"
    data-shop-redeem-coupon-url-value="{{ url "api:coupons:redeem" }}"
    data-shop-payment-options-url-value="{{ url "api:payments:options" }}">
</div>
```

### Lifecycle & Behavior

- **`connect()`**:
  - Verifies `hasGetCartUrlValue`, `hasClearCartUrlValue`, `hasAddCartItemUrlValue`, `hasRemoveCartItemUrlValue`, `hasRedeemCouponUrlValue`, and `hasPaymentOptionsUrlValue`; throws an `Error` if any is missing.
  - Attaches itself to the host DOM element as `this.element.SHOP = this`.
- **Methods:**
  - `getCart(): APIPromise<CartRow>` — `GET` to `getCartUrlValue`.
  - `clearCart(): APIPromise<CartRow>` — `POST` to `clearCartUrlValue`.
  - `addCartItem(req: CartItemRequest): APIPromise<CartItemResponse>` — `POST` JSON `req` to `addCartItemUrlValue`.
  - `removeCartItem(req: CartItemRequest): APIPromise<CartItemResponse>` — `POST` JSON `req` to `removeCartItemUrlValue`.
  - `redeemCoupon(req: RedemptionRequest): APIPromise<CartRow>` — `POST` JSON `req` to `redeemCouponUrlValue`.
  - `paymentOptions(): APIPromise<PaymentOption[]>` — `GET` to `paymentOptionsUrlValue`.
- **Internal Request Handling (`request<T>`)**:
  - Sets `Accept: application/json` and `credentials: "same-origin"`.
  - For `POST` requests, resolves the CSRF token in order:
    1. `this.csrfTokenValue`
    2. `meta[name="csrf-token"]` (`content`)
    3. `input[name="csrf_token"]` (`value`)
    If found, sets header `X-CSRF-Token`.
  - When `body !== undefined`, sets `Content-Type: application/json` and serializes with `JSON.stringify(body)`.
  - On non-`2xx` responses, parses JSON and returns `{ message: data.message }` if `data?.message` is a string, otherwise `{ message: "Shop request failed (HTTP <status>)" }`.

---

## `<django-menu-icon>` Web Component (`public/menu-icon.ts`)

- **Custom Element Tag:** `django-menu-icon` (auto-registered if not already defined).
- **Exports:**
  - `export interface MenuToggleDetail { expanded: boolean; }`
  - `export class MenuIcon extends HTMLElement`

### Attributes, Properties & Events

- **`observedAttributes`**: `["expanded", "label", "disabled"]`.
- **`expanded` (getter/setter)**: Reflects presence of the `expanded` boolean attribute.
- **`label` attribute**: Overrides the button's `aria-label`. Defaults to `"Close menu"` when `expanded` is true, or `"Open menu"` when false.
- **`disabled` attribute**: Disables the inner `<button>` element.
- **Click behavior & `menu-toggle` event**: Clicking the shadow-DOM `<button>` toggles `this.expanded` and dispatches:
  ```ts
  new CustomEvent<MenuToggleDetail>("menu-toggle", {
      bubbles: true,
      composed: true,
      detail: { expanded: this.expanded },
  })
  ```
- **Styling (Shadow DOM CSS custom properties):**
  - `--menu-icon-size` (default `44px`)
  - `--menu-icon-radius` (default `8px`)
  - `--menu-icon-background` (default `transparent`)
  - `--menu-icon-hover-background` (default `rgb(127 127 127 / 14%)`)
  - Honors `@media (prefers-reduced-motion: reduce)` by disabling bar transitions.

---

## Admin Stimulus Controller (`CouponGeneratorController`)

- **Registered Name:** `'coupon-generator'` (registered in `assets/static_src/admin/index.ts` on `window.Stimulus`).
- **Class:** `export { CouponGeneratorController }` (`assets/static_src/admin/widgets/CouponGenerator.ts`).
- **Template:** `assets/templates/shop/admin/widgets/coupon_generator.tmpl`.

### Values & Targets

- **Values:** `url` (`String`, required) — `data-coupon-generator-url-value` pointing to `admin:ext:shop:generate_coupon`.
- **Targets:**
  - `input` (required): Container holding the active sequence of `.coupon-generator-part` elements.
  - `output`: `<input>` element whose `.value` is updated with the generated `coupon_code`.
  - `part`: Individual part elements (with `data-part-value` set to `"CPT_RANDOM"`, `"CPT_STATIC"`, `"CPT_YEAR"`, `"CPT_MONTH"`, `"CPT_WEEKDAY"`, `"CPT_DAY"`, `"CPT_HOUR"`, `"CPT_MINUTE"`, `"CPT_DASH"`, or `"CPT_UUID"`).
  - `controls` (required): Palette container of draggable `.coupon-generator-part` templates.
  - `refresh` (required): Button element that triggers regeneration on click.

### Behavior

- **`connect()`**:
  - Validates that `url`, `input`, `controls`, and `refresh` targets/values are present (throws `Error` if missing).
  - Initializes two `Sortable` instances scoped to `sortableGroup-${this.outputTarget.id}`:
    - `this.sortable` on `controlsTarget`: clones `.coupon-generator-part` items (`pull: 'clone'`, `put: false`, `sort: false`).
    - `this.out` on `inputTarget`: accepts cloned/reordered `.coupon-generator-part` items and calls `compileParts()` on `onChange`.
  - Binds `refreshTarget` `"click"` event to `compileParts()`.
- **`compileParts()`**:
  - Iterates over `.coupon-generator-part` elements inside `this.inputTarget`.
  - Reads `target.dataset.partValue` (mapped via `CPT_CONV` to numeric `0..9`), optional `input[type=number]` (`length`, defaulting to `0`), and optional `input[type=text]` (`value`, defaulting to `""`).
  - Reads the CSRF token from `document.querySelector("input[name=csrf_token]")`.
  - Sends a `POST` request to `this.urlValue` with JSON body `{ code_format_parts: l }` and header `X-CSRF-Token`.
  - Sets `this.outputTarget.value = data.coupon_code` on success, or logs to `console.error` on fetch failure.
