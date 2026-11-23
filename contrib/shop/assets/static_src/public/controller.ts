import { Controller } from "@hotwired/stimulus";
import { CartRow, CartItemRequest, CartItemResponse, RedemptionRequest, PaymentOption, APIPromise } from "./api";

type ShopElement = HTMLElement & {
    SHOP: ShopController;
};

export class ShopController extends Controller<ShopElement> {
    /*
        <div
            data-controller="shop"
            data-shop-get-cart-url-value="{{ url "api:carts" }}"
            data-shop-clear-cart-url-value="{{ url "api:carts:clear" }}"
            data-shop-add-cart-item-url-value="{{ url "api:carts:add_item" }}"
            data-shop-remove-cart-item-url-value="{{ url "api:carts:remove_item" }}"
            data-shop-redeem-coupon-url-value="{{ url "api:coupons:redeem" }}"
            data-shop-payment-options-url-value="{{ url "api:payments:options" }}"
            >
        
        </div>
    */

    static values = {
        getCartUrl:        {type: String, default: "/api/carts"},
        clearCartUrl:      {type: String, default: "/api/carts/clear"},
        addCartItemUrl:    {type: String, default: "/api/carts/items/add"},
        removeCartItemUrl: {type: String, default: "/api/carts/items/remove"},
        redeemCouponUrl:   {type: String, default: "/api/coupons/redeem"},
        paymentOptionsUrl: {type: String, default: "/api/payments/options"},
        csrfToken: String,
    };

    declare getCartUrlValue:        string;
    declare clearCartUrlValue:      string;
    declare addCartItemUrlValue:    string;
    declare removeCartItemUrlValue: string;
    declare redeemCouponUrlValue:   string;
    declare paymentOptionsUrlValue: string;
    declare csrfTokenValue: string;

    declare hasGetCartUrlValue:        boolean;
    declare hasClearCartUrlValue:      boolean;
    declare hasAddCartItemUrlValue:    boolean;
    declare hasRemoveCartItemUrlValue: boolean;
    declare hasRedeemCouponUrlValue:   boolean;
    declare hasPaymentOptionsUrlValue: boolean;

    connect(): void {
        if (!this.hasGetCartUrlValue) {
            throw new Error("element is missing `data-shop-get-cart-url-value`");
        }

        if (!this.hasClearCartUrlValue) {
            throw new Error("element is missing `data-shop-clear-cart-url-value`");
        }

        if (!this.hasAddCartItemUrlValue) {
            throw new Error("element is missing `data-shop-add-cart-item-url-value`");
        }

        if (!this.hasRemoveCartItemUrlValue) {
            throw new Error("element is missing `data-shop-remove-cart-item-url-value`");
        }

        if (!this.hasRedeemCouponUrlValue) {
            throw new Error("element is missing `data-shop-redeem-coupon-url-value`");
        }

        if (!this.hasPaymentOptionsUrlValue) {
            throw new Error("element is missing `data-shop-payment-options-url-value`");
        }

        this.element.SHOP = this;
    }

    async getCart(): APIPromise<CartRow> {
        return this.request<CartRow>(this.getCartUrlValue, "GET");
    }

    async clearCart(): APIPromise<CartRow> {
        return this.request<CartRow>(this.clearCartUrlValue, "POST");
    }

    async addCartItem(req: CartItemRequest): APIPromise<CartItemResponse> {
        return this.request<CartItemResponse>(this.addCartItemUrlValue, "POST", req);
    }

    async removeCartItem(req: CartItemRequest): APIPromise<CartItemResponse> {
        return this.request<CartItemResponse>(this.removeCartItemUrlValue, "POST", req);
    }

    async redeemCoupon(req: RedemptionRequest): APIPromise<CartRow> {
        return this.request<CartRow>(this.redeemCouponUrlValue, "POST", req);
    }

    async paymentOptions(): APIPromise<PaymentOption[]> {
        return this.request<PaymentOption[]>(this.paymentOptionsUrlValue, "GET");
    }

    private async request<T>(url: string, method: "GET" | "POST", body?: unknown): APIPromise<T> {
        const headers = new Headers({ Accept: "application/json" });

        if (method === "POST") {
            const token = this.csrfTokenValue
                || document.querySelector<HTMLMetaElement>('meta[name="csrf-token"]')?.content
                || document.querySelector<HTMLInputElement>('input[name="csrf_token"]')?.value;

            if (token) {
                headers.set("X-CSRF-Token", token);
            }
        }

        if (body !== undefined) {
            headers.set("Content-Type", "application/json");
        }

        const response = await fetch(url, {
            method,
            headers,
            credentials: "same-origin",
            body: body === undefined ? undefined : JSON.stringify(body),
        });

        if (!response.ok) {
            const data = await response.json().catch(() => null);

            return {
                message: typeof data?.message === "string"
                    ? data.message
                    : `Shop request failed (HTTP ${response.status})`,
            };
        }

        return response.json() as Promise<T>;
    }
}
