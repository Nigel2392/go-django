package coupons

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

var (
	MIN_CODE_LENGTH int64 = 6
	MAX_CODE_LENGTH int64 = 32
)

type CouponCode string

type Coupon struct {
	ID               uint64
	Title            string
	ShortDescription string
	Code             CouponCode

	// price related
	IsPercentage   bool
	MaxDiscount    decimal.Decimal
	MinOrderAmount decimal.Decimal

	// usage limits
	UsageLimit uint64
	UsageCount uint64

	// valid time range
	ValidFrom  time.Time
	ValidUntil time.Time

	// simple audit information metadata
	CreatedAt time.Time
	UpdatedAt time.Time

	// the user that created this coupon code
	CreatedBy users.User

	// The redemptions related to this coupon
	RedemptionSet *queries.RelRevFK[*Redemption]

	// The products this coupon can be used for
	Products *queries.RelM2M[*products.Product, *CouponProductThrough]
}

func (c *Coupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Title", &c.Title, func() fattrs.PtrFieldConfig[*Coupon, string] {
			return fattrs.PtrFieldConfig[*Coupon, string]{
				Config: attrs.Config{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "Code", &c.Code, func() fattrs.PtrFieldConfig[*Coupon, CouponCode] {
			return fattrs.PtrFieldConfig[*Coupon, CouponCode]{
				Config: attrs.Config{
					MinLength: MIN_CODE_LENGTH,
					MaxLength: MAX_CODE_LENGTH,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "ShortDescription", &c.ShortDescription, func() fattrs.PtrFieldConfig[*Coupon, string] {
			return fattrs.PtrFieldConfig[*Coupon, string]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "UsageLimit", &c.UsageLimit, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "UsageCount", &c.UsageCount, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.Config{
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "IsPercentage", &c.IsPercentage, func() fattrs.PtrFieldConfig[*Coupon, bool] {
			return fattrs.PtrFieldConfig[*Coupon, bool]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "MaxDiscount", &c.MaxDiscount, func() fattrs.PtrFieldConfig[*Coupon, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Coupon, decimal.Decimal]{
				Config: attrs.Config{
					Precision: 13,
					Scale:     4,
					Null:      false,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "MinOrderAmount", &c.MinOrderAmount, func() fattrs.PtrFieldConfig[*Coupon, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Coupon, decimal.Decimal]{
				Config: attrs.Config{
					Precision: 13,
					Scale:     4,
					Null:      false,
					Blank:     true,
				},
			}
		}),
		fattrs.Field(c, "ValidFrom", &c.ValidFrom, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "ValidUntil", &c.ValidUntil, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "CreatedAt", &c.CreatedAt, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.Config{
					ReadOnly:   true,
					AutoNowAdd: true,
				},
			}
		}),
		fattrs.Field(c, "UpdatedAt", &c.UpdatedAt, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.Config{
					ReadOnly: true,
					AutoNow:  true,
				},
			}
		}),
		fattrs.Field(c, "CreatedBy", &c.CreatedBy, func() fattrs.PtrFieldConfig[*Coupon, users.User] {
			return fattrs.PtrFieldConfig[*Coupon, users.User]{
				Config: attrs.Config{
					Column:   "user_id",
					ReadOnly: true,
					Null:     true,
					RelForeignKey: attrs.RelatedDeferred(
						attrs.RelManyToOne,
						users.MODEL_KEY,
						"", nil,
					),
				},
			}
		}),
		fattrs.Field(c, "RedemptionSet", &c.RedemptionSet, func() fattrs.PtrFieldConfig[*Coupon, *queries.RelRevFK[*Redemption]] {
			return fattrs.PtrFieldConfig[*Coupon, *queries.RelRevFK[*Redemption]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
			}
		}),
		fattrs.Field(c, "Products", &c.Products, func() fattrs.PtrFieldConfig[*Coupon, *queries.RelM2M[*products.Product, *CouponProductThrough]] {
			return fattrs.PtrFieldConfig[*Coupon, *queries.RelM2M[*products.Product, *CouponProductThrough]]{
				Config: attrs.Config{
					Blank: true,
					RelManyToMany: attrs.Relate(
						&products.Product{},
						"", &attrs.ThroughModel{
							This:   &CouponProductThrough{},
							Source: "Coupon",
							Target: "Product",
						},
					),
					ReverseAlias: "Coupons",
				},
			}
		}),
	)
}
