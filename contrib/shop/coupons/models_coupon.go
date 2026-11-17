package coupons

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

type Coupon struct {
	ID               uint64
	Title            string
	ShortDescription string
	Code             string

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
}

func (c *Coupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Title", &c.Title, func() fattrs.PtrFieldConfig[*Coupon, string] {
			return fattrs.PtrFieldConfig[*Coupon, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "Code", &c.Code, func() fattrs.PtrFieldConfig[*Coupon, string] {
			return fattrs.PtrFieldConfig[*Coupon, string]{
				Config: attrs.FieldConfig{
					MinLength: 6,
					MaxLength: 32,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "ShortDescription", &c.ShortDescription, func() fattrs.PtrFieldConfig[*Coupon, string] {
			return fattrs.PtrFieldConfig[*Coupon, string]{
				Config: attrs.FieldConfig{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "UsageLimit", &c.UsageLimit, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.FieldConfig{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "UsageCount", &c.UsageCount, func() fattrs.PtrFieldConfig[*Coupon, uint64] {
			return fattrs.PtrFieldConfig[*Coupon, uint64]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "IsPercentage", &c.IsPercentage, func() fattrs.PtrFieldConfig[*Coupon, bool] {
			return fattrs.PtrFieldConfig[*Coupon, bool]{
				Config: attrs.FieldConfig{
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "MaxDiscount", &c.MaxDiscount, func() fattrs.PtrFieldConfig[*Coupon, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Coupon, decimal.Decimal]{
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
		fattrs.Field(c, "MinOrderAmount", &c.MinOrderAmount, func() fattrs.PtrFieldConfig[*Coupon, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Coupon, decimal.Decimal]{
				Config: attrs.FieldConfig{
					Attributes: map[string]interface{}{
						attrs.AttrPrecisionKey: 13,
						attrs.AttrScaleKey:     4,
					},
					Null:  false,
					Blank: true,
				},
			}
		}),
		fattrs.Field(c, "ValidFrom", &c.ValidFrom, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.FieldConfig{},
			}
		}),
		fattrs.Field(c, "ValidUntil", &c.ValidUntil, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.FieldConfig{},
			}
		}),
		fattrs.Field(c, "CreatedAt", &c.CreatedAt, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
					Attributes: map[string]interface{}{
						attrs.AttrAutoNowAddKey: true,
					},
				},
			}
		}),
		fattrs.Field(c, "UpdatedAt", &c.UpdatedAt, func() fattrs.PtrFieldConfig[*Coupon, time.Time] {
			return fattrs.PtrFieldConfig[*Coupon, time.Time]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
					Attributes: map[string]interface{}{
						attrs.AttrAutoNowKey: true,
					},
				},
			}
		}),
		fattrs.Field(c, "CreatedBy", &c.CreatedBy, func() fattrs.PtrFieldConfig[*Coupon, users.User] {
			return fattrs.PtrFieldConfig[*Coupon, users.User]{
				Config: attrs.FieldConfig{
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
			return fattrs.PtrFieldConfig[*Coupon, *queries.RelRevFK[*Redemption]]{}
		}),
	)
}
