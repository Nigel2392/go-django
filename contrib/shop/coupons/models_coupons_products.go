package coupons

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

type CouponProductThrough struct {
	Coupon  *Coupon
	Product *products.Product
}

func (m *CouponProductThrough) BeforeSave(ctx context.Context) error {
	return nil
}

func (m *CouponProductThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*CouponProductThrough, *Coupon] {
			return fattrs.PtrFieldConfig[*CouponProductThrough, *Coupon]{
				Config: attrs.FieldConfig{
					Column: "coupon_id",
					// ReadOnly:      true,
					// Null:          false,
					// Blank:         false,
					// RelForeignKey: attrs.Relate(&Coupon{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*CouponProductThrough, *products.Product] {
			return fattrs.PtrFieldConfig[*CouponProductThrough, *products.Product]{
				Config: attrs.FieldConfig{
					Column: "product_id",
					// ReadOnly:      true,
					// Null:          true,
					// Blank:         true,
					// RelForeignKey: attrs.Relate(&products.Product{}, "", nil),
				},
			}
		}),
	)
}
