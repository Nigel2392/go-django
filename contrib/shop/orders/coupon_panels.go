package orders

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func (m *Order) couponPanels(ctx context.Context) ([]admin.Panel, error) {
	redemptions, err := queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(ctx).
		Select("*", "Coupon.Code", "Coupon.Title").
		Filter(
			expr.Q("Order", m.ID),
		).
		OrderBy("ID").
		All()
	if err != nil || len(redemptions) == 0 {
		return nil, err
	}

	if m.OrderItems == nil {
		attrs.Define(ctx, m).Get("OrderItems")
	}
	items := m.OrderItems.AsList()
	if len(items) == 0 {
		rows, err := m.OrderItems.
			Objects().
			WithContext(ctx).
			OrderBy("ID").
			All()
		if err != nil {
			return nil, err
		}
		items = make([]*OrderItem, len(rows))
		for i, row := range rows {
			items[i] = row.Object
		}
	}

	panels := make([]admin.Panel, 0, len(redemptions))
	for _, row := range redemptions {
		redemption := row.Object
		label := trans.S("Redemption %d", redemption.ID)
		if redemption.Coupon != nil && redemption.Coupon.Code != "" {
			label = trans.S("Coupon %s", string(redemption.Coupon.Code))
		}

		status := admin.ALERT_INFO
		var html strings.Builder
		escape := template.HTMLEscapeString
		if redemption.Coupon != nil && redemption.Coupon.Title != "" {
			fmt.Fprintf(&html, "<p>%s</p>", escape(redemption.Coupon.Title))
		}

		var state string
		switch {
		case redemption.Invalidated:
			status = admin.ALERT_WARNING
			state = trans.T(ctx, "Reservation released")
		case redemption.Counted:
			status = admin.ALERT_SUCCESS
			state = trans.T(ctx, "Counted toward coupon usage")
		default:
			state = trans.T(ctx, "Reserved for this order")
		}
		fmt.Fprintf(&html, "<p>%s</p>", escape(state))

		if !redemption.RedeemedAt.IsZero() {
			at := redemption.RedeemedAt.UTC()
			fmt.Fprintf(&html, `<p>%s <time datetime="%s">%s</time></p>`,
				escape(trans.T(ctx, "Redeemed at")), at.Format(time.RFC3339), at.Format("2006-01-02 15:04:05 UTC"))
		}

		if redemption.DiscountAmount.IsPositive() {
			fmt.Fprintf(&html, "<p>%s</p>", escape(trans.T(ctx, "%s discount", redemption.DiscountAmount.String())))
		} else {
			fmt.Fprintf(&html, "<p>%s</p>", escape(trans.T(ctx, "Discount amount not recorded")))
		}

		hasAllocations := false
		for _, item := range items {
			pricing := item.Pricing.Data
			if pricing == nil {
				continue
			}
			amount, applied := pricing.RedemptionAmounts[redemption.ID]
			if !applied {
				continue
			}
			if !hasAllocations {
				html.WriteString("<ul>")
				hasAllocations = true
			}
			line := item.ID.String()
			if product := item.Product.Data.Product; product != nil && product.Title != "" {
				line = product.Title
			}
			fmt.Fprintf(&html, "<li>%s</li>", escape(trans.T(ctx, "%s · %s discount · %s line total",
				line, amount.String(), pricing.Totals.Payable().String())))
		}
		if hasAllocations {
			html.WriteString("</ul>")
		} else if redemption.DiscountAmount.IsPositive() {
			fmt.Fprintf(&html, "<p>%s</p>", escape(trans.T(ctx, "Line allocations not recorded")))
		}

		panels = append(panels, &admin.AlertPanel{
			Type:  status,
			Label: label,
			HTML:  template.HTML(html.String()),
		})
	}
	return panels, nil
}
