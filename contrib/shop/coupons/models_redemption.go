package coupons

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

type Redemption struct {
	ID             uint64
	Coupon         *Coupon
	Customer       *customers.Customer
	Order          attrs.Definer
	DiscountAmount decimal.Decimal
	RedeemedAt     time.Time
}

func (m *Redemption) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Redemption, uint64] {
			return fattrs.PtrFieldConfig[*Redemption, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Order", &m.Order, func() fattrs.PtrFieldConfig[*Redemption, attrs.Definer] {
			return fattrs.PtrFieldConfig[*Redemption, attrs.Definer]{
				Config: attrs.FieldConfig{
					Column:   "order_id",
					ReadOnly: true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						modelkeys.OrderModelKey,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*Redemption, *Coupon] {
			return fattrs.PtrFieldConfig[*Redemption, *Coupon]{
				Config: attrs.FieldConfig{
					Column:        "coupon_id",
					RelForeignKey: attrs.Relate(&Coupon{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
		fattrs.Field(m, "Customer", &m.Customer, func() fattrs.PtrFieldConfig[*Redemption, *customers.Customer] {
			return fattrs.PtrFieldConfig[*Redemption, *customers.Customer]{
				Config: attrs.FieldConfig{
					Column:        "customer_id",
					RelForeignKey: attrs.Relate(&customers.Customer{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
		fattrs.Field(m, "RedeemedAt", &m.RedeemedAt, func() fattrs.PtrFieldConfig[*Redemption, time.Time] {
			return fattrs.PtrFieldConfig[*Redemption, time.Time]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
					Attributes: map[string]interface{}{
						attrs.AttrAutoNowAddKey: true,
					},
				},
			}
		}),
	)
}
