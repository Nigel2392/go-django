package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*Cart)(nil)

type Cart struct {
	ID        drivers.ULID
	CartItems *queries.RelRevFK[*CartItem]
}

func (m *Cart) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Cart) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Cart, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Cart, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "CartItems", &m.CartItems, func() fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]] {
			return fattrs.PtrFieldConfig[*Cart, *queries.RelRevFK[*CartItem]]{}
		}),
	)
}
