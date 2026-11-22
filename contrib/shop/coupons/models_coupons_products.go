package coupons

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type CouponProductThrough struct {
	// ID      uint64
	Coupon  *Coupon
	Product *products.Product
}

func (m *CouponProductThrough) BeforeSave(ctx context.Context) error {
	return nil
}

func (m *CouponProductThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		//	 fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*CouponProductThrough, uint64] {
		//	 	return fattrs.PtrFieldConfig[*CouponProductThrough, uint64]{}
		//	 }),
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*CouponProductThrough, *Coupon] {
			return fattrs.PtrFieldConfig[*CouponProductThrough, *Coupon]{
				Config: attrs.Config{
					Column:   "coupon_id",
					ReadOnly: true,
					Null:     false,
					Blank:    false,
					//	RelForeignKey: attrs.Relate(&Coupon{}, "", nil),
					//	Attributes: map[string]interface{}{
					//		attrs.AttrReverseAliasKey: "Products",
					//	},
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*CouponProductThrough, *products.Product] {
			return fattrs.PtrFieldConfig[*CouponProductThrough, *products.Product]{
				Config: attrs.Config{
					Column: "product_id",
					FormField: func(opts ...func(fields.Field)) fields.Field {
						var widget = chooser.NewChooserWidget(&products.Product{}, map[string]string{}, chooser.DEFAULT_KEY)
						return &formfields.ForeignKeyFormField{
							BaseRelationField: formfields.BaseRelationField{
								BaseField: fields.NewField(append(opts, fields.Widget(widget))...),
								Relation:  attrs.Relate(&products.Product{}, "", nil),
							},
						}
					},
					// ReadOnly:      true,
					// Null:          true,
					// Blank:         true,
					// RelForeignKey: attrs.Relate(&products.Product{}, "", nil),
				},
			}
		}),
	)
}
