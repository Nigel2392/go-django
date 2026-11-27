package handlers

import (
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/stretchr/testify/require"
)

func TestShopAuditDefinitions(t *testing.T) {
	RegisterAuditLogs()
	req := httptest.NewRequest("GET", "/audit", nil)
	seen := make(map[reflect.Type]string)
	for kind, title := range map[string]string{
		"shop:product:add":                  "Product added",
		"shop:product:edit":                 "Product edited",
		"shop:carts:create":                 "Cart created",
		"shop:carts:clear":                  "Cart cleared",
		"shop:carts:items:add":              "Cart item added",
		"shop:carts:items:remove":           "Cart item removed",
		"shop:coupons:create":               "Coupon created",
		"shop:coupons:update":               "Coupon updated",
		"shop:coupons:redeem":               "Coupon redeemed",
		"shop:orders:create":                "Order created",
		"shop:orders:update":                "Order updated",
		"shop:orders:accept":                "Order accepted",
		"shop:orders:process":               "Order processed",
		"shop:orders:complete":    "Order completed",
		"shop:payments:start":     "Payment started",
		"shop:payments:update":    "Payment status updated",
		"shop:payments:refund":    "Payment refunded",
		"shop:payments:cancel":    "Payment cancelled",
	} {
		t.Run(kind, func(t *testing.T) {
			def := auditlogs.DefinitionForType(kind)
			require.NotNil(t, def)
			defType := reflect.TypeOf(def)
			require.Empty(t, seen[defType], "%s reuses the definition for %s", kind, seen[defType])
			seen[defType] = kind
			entry := &auditlogs.Entry{
				Typ:   drivers.String(kind),
				ObjID: drivers.JSON[any]{Data: "audit-fixture"},
				Src: drivers.JSON[map[string]any]{Data: map[string]any{
					"title": "Fixture", "state": paystate.STATUS_PAID, "previous_state": paystate.STATUS_PENDING,
					"line_count": 1, "cart_id": "cart-fixture", "amount": "12", "currency": "EUR",
					"refund_amount": "2", "refund_currency": "EUR", "request_id": "request-fixture",
				}},
			}
			require.Equal(t, title, def.TypeLabel(req, kind))
			require.NotEmpty(t, def.GetLabel(req, entry))
			message := fmt.Sprint(def.FormatMessage(req, entry))
			require.Contains(t, message, "audit-fixture")
			require.NotContains(t, message, "%!")
			require.Empty(t, def.GetActions(req, entry))
			if kind == "shop:payments:update" {
				require.Contains(t, message, "changed from STATUS_PENDING to STATUS_PAID")
				entry.Data()["previous_state"] = paystate.STATUS_PAID
				require.Contains(t, def.FormatMessage(req, entry), "updated while in status STATUS_PAID")
			}
		})
	}
}

func TestPaymentAuditLevels(t *testing.T) {
	for _, tc := range []struct {
		state string
		level logger.LogLevel
	}{
		{paystate.STATUS_PAID, logger.INF},
		{paystate.STATUS_EXPIRED, logger.INF},
		{paystate.STATUS_VERIFY, logger.WRN},
		{paystate.STATUS_DENIED, logger.WRN},
		{paystate.STATUS_PAID_CHECKAMOUNT, logger.WRN},
		{paystate.STATUS_CHARGEBACK, logger.WRN},
		{paystate.STATUS_FAILURE, logger.ERR},
	} {
		t.Run(tc.state, func(t *testing.T) {
			require.Equal(t, tc.level, paymentStateLevel(&payments.Payment{State: tc.state}))
		})
	}
}
