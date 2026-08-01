package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*Order)(nil)

type Order struct {
	ID         drivers.ULID
	OrderItems *queries.RelRevFK[*OrderItem]
}

func (m *Order) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Order, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Order, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "OrderItems", &m.OrderItems, func() fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]] {
			return fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]]{}
		}),
	)
}
