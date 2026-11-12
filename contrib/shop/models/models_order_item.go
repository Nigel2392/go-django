package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/revisions"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

var _ queries.ActsBeforeCreate = (*OrderItem)(nil)

type OrderItem struct {
	ID        drivers.ULID
	Order     *Order
	Product   *revisions.Revision
	ItemPrice decimal.Decimal
	Amount    int
}

func (m *OrderItem) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *OrderItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*OrderItem, drivers.ULID] {
			return fattrs.PtrFieldConfig[*OrderItem, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*OrderItem, *revisions.Revision] {
			return fattrs.PtrFieldConfig[*OrderItem, *revisions.Revision]{
				Config: attrs.FieldConfig{
					Null:  false,
					Blank: false,
				},
			}
		}),
		fattrs.Field(m, "ItemPrice", &m.ItemPrice, func() fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal]{
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
		fattrs.Field(m, "Amount", &m.Amount, func() fattrs.PtrFieldConfig[*OrderItem, int] {
			return fattrs.PtrFieldConfig[*OrderItem, int]{
				Config: attrs.FieldConfig{
					Null:  false,
					Blank: false,
				},
			}
		}),
		fattrs.Field(m, "Order", &m.Order, func() fattrs.PtrFieldConfig[*OrderItem, *Order] {
			return fattrs.PtrFieldConfig[*OrderItem, *Order]{
				Config: attrs.FieldConfig{
					Column: "order_id",
					Attributes: map[string]interface{}{
						attrs.AttrReverseAliasKey: "OrderItems",
					},
					RelForeignKey: attrs.Relate(&Order{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
	)
}
