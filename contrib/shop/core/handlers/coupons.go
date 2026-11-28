package handlers

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func registerCouponAudits() {
	listenAudit(coupons.Signals.Created, "shop:coupons:create", couponCreatedDefinition{}, func(s coupons.CouponCreatedSignal) auditEvent {
		return couponAudit(s.Coupon)
	})
	listenAudit(coupons.Signals.Updated, "shop:coupons:update", couponUpdatedDefinition{}, func(s coupons.CouponUpdatedSignal) auditEvent {
		return couponAudit(s.Coupon)
	})
	listenAudit(coupons.Signals.Redeemed, "shop:coupons:redeem", couponRedeemedDefinition{}, func(s coupons.CouponRedeemedSignal) auditEvent {
		entry := couponAudit(s.Coupon)
		if !attrs.IsZero(s.Cart) {
			entry.data["cart_id"] = attrs.Define(context.Background(), s.Cart).Primary().GetValue()
		}
		return entry
	})
}

func couponAudit(coupon *coupons.Coupon) auditEvent {
	data := make(map[string]any)
	if coupon != nil {
		data["title"] = coupon.Title
	}
	return auditEvent{object: coupon, level: logger.INF, data: data}
}
