package store

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	queries "github.com/Nigel2392/go-django/queries/src"
)

const (
	notebookSupplier = "notebook-supplier"
	penSupplier      = "pen-supplier"
)

func registerSuppliers() {
	receiptBlock := blocks.NewStructBlock()
	receiptBlock.AddField("receipt", blocks.CharBlock())

	for _, name := range []string{notebookSupplier, penSupplier} {
		orders.RegisterSupplier(&orders.Supplier{
			Name: name,
			ProductModel: func(context.Context) products.ProductModel {
				return &Product{}
			},
			OrderBlockType: func(context.Context, *orders.Order, *orders.OrderItem) (blocks.Block, error) {
				return receiptBlock, nil
			},
			OrderIsAccepted: func(ctx context.Context, _ *orders.Order, line *orders.OrderItem) (*blocks.StructBlockValue, error) {
				// The delivery view records the receipt before this workflow runs.
				row, err := queries.GetQuerySet(&Delivery{}).WithContext(ctx).Filter("Line", line.ID).Get()
				if err != nil {
					return nil, fmt.Errorf("read delivery for line %s: %w", line.ID, err)
				}
				if row.Object.Receipt == "" {
					return nil, fmt.Errorf("delivery for line %s is unconfirmed", line.ID)
				}
				return blocks.NewBlockValue(receiptBlock, map[string]any{"receipt": row.Object.Receipt}), nil
			},
		})
	}
}
