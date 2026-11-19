package carts

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*CartItem)(nil)

type CartItem struct {
	ID      drivers.ULID      `json:"id"`
	Cart    *Cart             `json:"-"`
	Product *products.Product `json:"product,omitempty"`
	Amount  uint64            `json:"amount"`
}

func (m *CartItem) UniqueTogether() [][]string {
	return [][]string{
		{"Cart", "Product"},
	}
}

func (m *CartItem) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *CartItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*CartItem, drivers.ULID] {
			return fattrs.PtrFieldConfig[*CartItem, drivers.ULID]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Cart", &m.Cart, func() fattrs.PtrFieldConfig[*CartItem, *Cart] {
			return fattrs.PtrFieldConfig[*CartItem, *Cart]{
				Config: attrs.Config{
					Column:        "cart_id",
					ReadOnly:      true,
					ReverseAlias:  "CartItems",
					RelForeignKey: attrs.Relate(&Cart{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*CartItem, *products.Product] {
			return fattrs.PtrFieldConfig[*CartItem, *products.Product]{
				Config: attrs.Config{
					Column:        "product_id",
					ReadOnly:      true,
					ReverseAlias:  "CartItems",
					RelForeignKey: attrs.Relate(&products.Product{}, "", nil),
					Null:          false,
					Blank:         false,
				},
			}
		}),
		fattrs.Field(m, "Amount", &m.Amount, func() fattrs.PtrFieldConfig[*CartItem, uint64] {
			return fattrs.PtrFieldConfig[*CartItem, uint64]{
				Config: attrs.Config{
					MinValue: 0,
				},
			}
		}),
	)
}
