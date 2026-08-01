package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*OrderItem)(nil)

type OrderItem struct {
	ID    drivers.ULID
	Order *Order
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
