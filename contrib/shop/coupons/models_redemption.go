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
	Cart           attrs.Definer
	Order          attrs.Definer
	DiscountAmount decimal.Decimal
	Invalidated    bool
	RedeemedAt     time.Time
}

func (m *Redemption) BeforeSave(ctx context.Context) error {
	return nil
}

func (m *Redemption) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Redemption, uint64] {
			return fattrs.PtrFieldConfig[*Redemption, uint64]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*Redemption, *Coupon] {
			return fattrs.PtrFieldConfig[*Redemption, *Coupon]{
				Config: attrs.Config{
					Column:        "coupon_id",
					ReadOnly:      true,
					Null:          false,
					Blank:         false,
					RelForeignKey: attrs.Relate(&Coupon{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "Customer", &m.Customer, func() fattrs.PtrFieldConfig[*Redemption, *customers.Customer] {
			return fattrs.PtrFieldConfig[*Redemption, *customers.Customer]{
				Config: attrs.Config{
					Column:        "customer_id",
					ReadOnly:      true,
					Null:          false,
					Blank:         false,
					RelForeignKey: attrs.Relate(&customers.Customer{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "Cart", &m.Cart, func() fattrs.PtrFieldConfig[*Redemption, attrs.Definer] {
			return fattrs.PtrFieldConfig[*Redemption, attrs.Definer]{
				Config: attrs.Config{
					Column:       "cart_id",
					ReadOnly:     true,
					Null:         true,
					Blank:        true,
					ReverseAlias: "Redemptions",
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						modelkeys.CartModelKey,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(m, "Order", &m.Order, func() fattrs.PtrFieldConfig[*Redemption, attrs.Definer] {
			return fattrs.PtrFieldConfig[*Redemption, attrs.Definer]{
				Config: attrs.Config{
					Column:   "order_id",
					ReadOnly: true,
					Null:     true,
					Blank:    true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						modelkeys.OrderModelKey,
						"", nil,
					),
					ReverseAlias: "Redemptions",
				},
			}
		}),
		fattrs.Field(m, "DiscountAmount", &m.DiscountAmount, func() fattrs.PtrFieldConfig[*Redemption, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Redemption, decimal.Decimal]{
				Config: attrs.Config{
					Precision: 13,
					Scale:     4,
					ReadOnly:  true,
					Null:      false,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(m, "Invalidated", &m.Invalidated, func() fattrs.PtrFieldConfig[*Redemption, bool] {
			return fattrs.PtrFieldConfig[*Redemption, bool]{
				Config: attrs.Config{
					ReadOnly: true,
					Blank:    true,
				},
			}
		}),
		fattrs.Field(m, "RedeemedAt", &m.RedeemedAt, func() fattrs.PtrFieldConfig[*Redemption, time.Time] {
			return fattrs.PtrFieldConfig[*Redemption, time.Time]{
				Config: attrs.Config{
					ReadOnly: true,
				},
			}
		}),
	)
}
