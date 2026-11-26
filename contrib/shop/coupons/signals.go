package coupons

import (
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// CouponCreatedSignal is sent within the coupon transaction by CreateCoupon.
type CouponCreatedSignal struct {
	signals.BaseSignal
	Coupon *Coupon
}

type CouponUpdatedSignal struct {
	signals.BaseSignal
	Coupon *Coupon
}

type CouponRedeemedSignal struct {
	signals.BaseSignal
	Coupon *Coupon
	Cart   attrs.Definer
}

type couponSignalManager struct {
	Created  signals.Signal[CouponCreatedSignal]
	Updated  signals.Signal[CouponUpdatedSignal]
	Redeemed signals.Signal[CouponRedeemedSignal]
}

var Signals = couponSignalManager{
	Created:  signals.Pool.Get[CouponCreatedSignal]("shop.coupons.Coupon.Created"),
	Updated:  signals.Pool.Get[CouponUpdatedSignal]("shop.coupons.Coupon.Updated"),
	Redeemed: signals.Pool.Get[CouponRedeemedSignal]("shop.coupons.Coupon.Redeemed"),
}
