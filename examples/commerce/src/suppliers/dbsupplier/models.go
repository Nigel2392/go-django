package dbsupplier

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"golang.org/x/text/currency"
)

type Product struct {
	*products.Product
	SupplierID string
}

func (p *Product) ProductPrice(ctx context.Context) products.Price {
	return products.Price{
		Unit:   currency.EUR,
		Amount: p.Price,
	}
}

func (p *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*Product, any](ctx, p,
		fields.Embed(ctx, &p.Product, fields.EmbedOptions{
			EmbedFields: []any{
				"ID",
				"Image",
				"Title",
				"Slug",
				"Editor",
				"Category",
				"Supplier",
				"Price",
				"TaxRate",
				"DiscountPrice",
				"Ranking",
			},
		}),
		fattrs.Field(p, "SupplierID", &p.SupplierID, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.Config{
					Blank: false,
				},
			}
		}),
	)
}
