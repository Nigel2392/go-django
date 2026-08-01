package models

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

type ProductSku struct {
	ID      uint64
	Product *Product
}

func (m *ProductSku) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ProductSku, uint64] {
			return fattrs.PtrFieldConfig[*ProductSku, uint64]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*ProductSku, *Product] {
			return fattrs.PtrFieldConfig[*ProductSku, *Product]{
				Config: attrs.FieldConfig{
					Column:        "product_id",
					RelForeignKey: attrs.Relate(&Product{}, "", nil),
					Attributes: map[string]interface{}{
						attrs.AttrReverseAliasKey: "Skus",
					},
					Null:  false,
					Blank: false,
				},
			}
		}),
	)
}
