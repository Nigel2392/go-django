package dbsupplier

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type Product struct {
	*models.Product
	Price decimal.Decimal
}

func (p *Product) ProductPrice(ctx context.Context) supplier.Price {
	return supplier.Price{
		Unit:   currency.EUR,
		Amount: p.Price,
	}
}

func (p *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &models.Product{Instance: p}
	}
	return attrs.Make[*Product, any](ctx, p,
		fields.Embed(ctx, &p.Product, fields.EmbedOptions{
			EmbedFields: []any{"ID", "Title", "Slug", "Editor", "Supplier", "Ranking"},
		}),
		fattrs.Field(p, "Price", &p.Price, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.FieldConfig{
					Attributes: map[string]interface{}{
						attrs.AttrPrecisionKey: 13,
						attrs.AttrScaleKey:     4,
					},
				},
			}
		}),
	)
}
