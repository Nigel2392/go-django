package coupons

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

type Redemption struct {
	ID             uint64
	Coupon         *Coupon
	Customer       *customers.Customer
	Product        *products.Product
	Cart           attrs.Definer
	Order          attrs.Definer
	DiscountAmount decimal.Decimal
	RedeemedAt     time.Time
}

func (m *Redemption) BeforeSave(ctx context.Context) error {
	return nil
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
		fattrs.Field(m, "DiscountAmount", &m.DiscountAmount, func() fattrs.PtrFieldConfig[*Redemption, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Redemption, decimal.Decimal]{
				Config: attrs.FieldConfig{
					Attributes: map[string]interface{}{
						attrs.AttrPrecisionKey: 13,
						attrs.AttrScaleKey:     4,
					},
					Null:  false,
					Blank: false,
				},
			}
		}),
		fattrs.Field(m, "Order", &m.Order, func() fattrs.PtrFieldConfig[*Redemption, attrs.Definer] {
			return fattrs.PtrFieldConfig[*Redemption, attrs.Definer]{
				Config: attrs.FieldConfig{
					Column:   "order_id",
					ReadOnly: true,
					Null:     true,
					Blank:    true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						modelkeys.OrderModelKey,
						"", nil,
					),
					Attributes: map[string]interface{}{
						attrs.AttrReverseAliasKey: "Redemptions",
					},
				},
			}
		}),
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*Redemption, *Coupon] {
			return fattrs.PtrFieldConfig[*Redemption, *Coupon]{
				Config: attrs.FieldConfig{
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
				Config: attrs.FieldConfig{
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
				Config: attrs.FieldConfig{
					Column:   "cart_id",
					ReadOnly: true,
					Null:     true,
					Blank:    true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						modelkeys.OrderModelKey,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*Redemption, *products.Product] {
			return fattrs.PtrFieldConfig[*Redemption, *products.Product]{
				Config: attrs.FieldConfig{
					Column:        "product_id",
					ReadOnly:      true,
					Null:          false,
					Blank:         false,
					RelForeignKey: attrs.Relate(&products.Product{}, "", nil),
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
