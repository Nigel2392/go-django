package store

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const (
	inStock      = "in_stock"
	backordered  = "backordered"
	discontinued = "discontinued"
)

// Product extends the shop catalog with the availability used to decide orders.
type Product struct {
	*products.Product
	Availability string
}

func (p *Product) ProductPrice(context.Context) price.Money {
	return price.Money{Currency: shopCurrency, Amount: p.Price}
}

func (p *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*Product, any](ctx, p,
		fields.Embed(ctx, &p.Product, fields.EmbedOptions{
			EmbedFields: []any{
				"ID", "Image", "Title", "Slug", "Editor", "Category", "Supplier",
				"Price", "TaxRate", "DiscountPrice", "Ranking",
			},
		}),
		attrs.NewField(p, "Availability", &attrs.Config{MaxLength: 20}),
	).WithTableName("tutorial_products")
}

// Delivery tracks the receipt for one order line. An empty receipt means delivery is pending.
type Delivery struct {
	ID      uint64
	Line    *orders.OrderItem
	Receipt string
}

func (d *Delivery) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, d,
		attrs.NewField(d, "ID", &attrs.Config{Primary: true}),
		attrs.NewField(d, "Line", &attrs.Config{
			Column:      "line_id",
			RelOneToOne: attrs.Relate(&orders.OrderItem{}, "", nil),
			Attributes:  map[string]any{attrs.AttrUniqueKey: true},
		}),
		attrs.NewField(d, "Receipt", &attrs.Config{Blank: true, MaxLength: 120}),
	).WithTableName("tutorial_deliveries")
}
