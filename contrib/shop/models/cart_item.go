package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*CartItem)(nil)

type CartItem struct {
	ID   drivers.ULID
	Cart *Cart
}

func (m *CartItem) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *CartItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*CartItem, drivers.ULID] {
			return fattrs.PtrFieldConfig[*CartItem, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "Cart", &m.Cart, func() fattrs.PtrFieldConfig[*CartItem, *Cart] {
			return fattrs.PtrFieldConfig[*CartItem, *Cart]{
				Config: attrs.FieldConfig{
					Column: "cart_id",
					Attributes: map[string]interface{}{
						attrs.AttrReverseAliasKey: "CartItems",
					},
					RelForeignKey: attrs.Relate(&Cart{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
	)
}
