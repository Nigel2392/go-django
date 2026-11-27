/** ULIDs and RFC 3339 timestamps are serialized as JSON strings. */
export type ULID = string;
export type Timestamp = string;

/** Decimal values use the server's default string JSON representation. */
export type Decimal = string;

export interface Response {
    status_code: number;
}

export interface ErrorResponse extends Response {
    message: string;
}

export interface DataResponse<T> extends Response {
    data: T
}

export function isErrorResponse(resp: any): resp is ErrorResponse {
    return resp.status_code >= 400;
}

export function isResponseT<T>(resp: any): resp is T {
    return resp.status_code == 200 && resp.data != undefined;
}

export type APIPromise<T> = Promise<DataResponse<T> | ErrorResponse>

export interface Totals {
    total_price: Decimal;
    total_fee: Decimal;
    total_tax: Decimal;
    total_discount: Decimal;
    left_over_discount: Decimal;
}

export interface EmailAddress {
    Name: string;
    Address: string;
}

/** These model fields have no JSON tags, so their Go casing is preserved. */
export interface Customer {
    ID: ULID;
    Email: EmailAddress | null;
    FirstName: string;
    LastName: string;
    User?: unknown;
    FirstSeen: Timestamp;
    Orders?: unknown;
}

export interface Cart {
    id: ULID;
    customer?: Customer | null;
    created_at: Timestamp;
    active: boolean;
}

export interface Category {
    ID: number;
    Name: string;
    Slug: string;
    ParentCategory: Category | null;
    Children: Category[] | null;
}

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

export interface CartItem {
    id: ULID;
    product?: Product;
    amount: number;
}

/** These model fields have no JSON tags, so their Go casing is preserved. */
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

export interface CartItemRow {
    item: CartItem | null;
    coupons: Redemption[] | null;
    total_price: Totals;
}

/** Returned by cart retrieval, cart clearing and coupon redemption. */
export interface CartRow {
    cart: Cart | null;
    items: CartItemRow[] | null;
    coupons: Redemption[] | null;
    total_price: Totals;
}

/** Matches the Go CartResponse declaration. Retrieval returns CartRow. */
export interface CartResponse {
    cart: Cart | null;
    items: CartItem[] | null;
}

export interface CartCustomerRequest {
    email: string;
    first_name: string;
    last_name: string;
}

export interface CartItemRequest {
    product_id: ULID;
    amount: number;
}

export interface CartItemResponse {
    cart: Cart | null;
    item: CartItem | null;
    product: Product | null;
}

export interface RedemptionRequest {
    coupon_code: string;
}

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

/** Request/response contracts keyed by the URL definitions in SHOP_API_URLS. */
export interface ShopAPI {
    getCart: {
        request: undefined;
        response: CartRow;
    };
    clearCart: {
        request: undefined;
        response: CartRow;
    };
    setCustomer: {
        request: CartCustomerRequest;
        response: CartRow;
    };
    addCartItem: {
        request: CartItemRequest;
        response: CartItemResponse;
    };
    removeCartItem: {
        request: CartItemRequest;
        response: CartItemResponse;
    };
    redeemCoupon: {
        request: RedemptionRequest;
        response: CartRow;
    };
    paymentOptions: {
        request: undefined;
        response: PaymentOption[];
    };
}
