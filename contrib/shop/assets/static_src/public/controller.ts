import { Controller } from "@hotwired/stimulus";
import { CartRow, CartItemRequest, CartItemResponse, RedemptionRequest, PaymentOption } from "./api";

type ShopElement = HTMLElement & {
    SHOP: ShopController
}

class ShopController extends Controller<ShopElement> {
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
    };

    declare getCartUrlValue:        string;
    declare clearCartUrlValue:      string;
    declare addCartItemUrlValue:    string;
    declare removeCartItemUrlValue: string;
    declare redeemCouponUrlValue:   string;
    declare paymentOptionsUrlValue: string;

    declare hasGetCartUrlValue:        boolean;
    declare hasClearCartUrlValue:      boolean;
    declare hasAddCartItemUrlValue:    boolean;
    declare hasRemoveCartItemUrlValue: boolean;
    declare hasRedeemCouponUrlValue:   boolean;
    declare hasPaymentOptionsUrlValue: boolean;

    connect(): void {
        if (!this.hasGetCartUrlValue) {
            throw new Error("element is missing `data-shop-get-cart-url-value`")
        }
        if (!this.hasClearCartUrlValue) {
            throw new Error("element is missing `data-shop-clear-cart-url-value`")
        }
        if (!this.hasAddCartItemUrlValue) {
            throw new Error("element is missing `data-shop-add-cart-item-url-value`")
        }
        if (!this.hasRemoveCartItemUrlValue) {
            throw new Error("element is missing `data-shop-remove-cart-item-url-value`")
        }
        if (!this.hasRedeemCouponUrlValue) {
            throw new Error("element is missing `data-shop-redeem-coupon-url-value`")
        }
        if (!this.hasPaymentOptionsUrlValue) {
            throw new Error("element is missing `data-shop-payment-options-url-value`")
        }

        this.element.SHOP = this
    }

    getCart(): CartRow {
        return null
    }

    clearCart(): CartRow {
        return null
    }

    addCartItem(req: CartItemRequest): CartItemResponse {
        return null
    }

    removeCartItem(req: CartItemRequest): CartItemResponse {
        return null
    }

    redeemCoupon(req: RedemptionRequest): CartRow {
        return null
    }

    paymentOptions(): PaymentOption[] {
        return null
    }
}