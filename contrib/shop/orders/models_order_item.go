package orders

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
)

var (
	_ queries.ActsBeforeCreate = (*OrderItem)(nil)
	_ queries.ActsAfterQuery   = (*OrderItem)(nil)
)

type OrderItem struct {
	ID           drivers.ULID
	Order        *Order
	Product      drivers.JSON[ProductData]
	Pricing      drivers.JSON[*OrderItemPricing]
	Supplier     products.SupplierValue // product also embeds this, but we cannot query the JSON easily
	SupplierData *blocks.StructBlockValue
	ItemPrice    decimal.Decimal
	TaxRate      decimal.Decimal
	Amount       int

	supplierBlock blocks.Block `json:"-"`
}

// OrderItemPricing captures the totals and coupon allocations used at checkout.
// The map is keyed by coupon redemption ID.
type OrderItemPricing struct {
	Totals            price.Totals
	RedemptionAmounts map[uint64]decimal.Decimal
}

func (m *OrderItem) AfterQuery(ctx context.Context) (err error) {
	if m.Supplier == "" && m.Product.Data.Product != nil {
		m.Supplier = m.Product.Data.Product.Supplier
	}
	supplier, ok := RetrieveSupplier(m.Supplier)
	if !ok {
		if !m.SupplierData.HasData() {
			return nil
		}
		return errors.CorruptedData.Wrapf("supplier %q does not exist", m.Supplier)
	}

	m.supplierBlock, err = supplier.OrderBlock(ctx, m.Order, m)
	if err != nil {
		return errors.Wrapf(err, "OrderItem.AfterQuery(%q)", m.Supplier)
	}

	if m.supplierBlock == nil || m.SupplierData == nil {
		return nil
	}

	return m.SupplierData.BindToBlock(m.supplierBlock)
}

func (m *OrderItem) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *OrderItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*OrderItem, drivers.ULID] {
			return fattrs.PtrFieldConfig[*OrderItem, drivers.ULID]{
				Primary: true,
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*OrderItem, drivers.JSON[ProductData]] {
			return fattrs.PtrFieldConfig[*OrderItem, drivers.JSON[ProductData]]{
				Null:  false,
				Blank: false,
			}
		}),
		fattrs.Field(m, "Pricing", &m.Pricing, func() fattrs.PtrFieldConfig[*OrderItem, drivers.JSON[*OrderItemPricing]] {
			return fattrs.PtrFieldConfig[*OrderItem, drivers.JSON[*OrderItemPricing]]{
				Null:     true,
				Blank:    true,
				ReadOnly: true,
			}
		}),
		fattrs.Field(m, "ItemPrice", &m.ItemPrice, func() fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal]{
				Precision: 13,
				Scale:     4,
				Null:      false,
				Blank:     false,
			}
		}),
		fattrs.Field(m, "TaxRate", &m.TaxRate, func() fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*OrderItem, decimal.Decimal]{
				Null:  false,
				Blank: false,
			}
		}),
		fattrs.Field(m, "Amount", &m.Amount, func() fattrs.PtrFieldConfig[*OrderItem, int] {
			return fattrs.PtrFieldConfig[*OrderItem, int]{
				Null:  false,
				Blank: false,
			}
		}),
		fattrs.Field(m, "Supplier", &m.Supplier, func() fattrs.PtrFieldConfig[*OrderItem, products.SupplierValue] {
			return fattrs.PtrFieldConfig[*OrderItem, products.SupplierValue]{
				Null:  false,
				Blank: false,
			}
		}),
		fattrs.Field(m, "SupplierData", &m.SupplierData, func() fattrs.PtrFieldConfig[*OrderItem, *blocks.StructBlockValue] {
			return fattrs.PtrFieldConfig[*OrderItem, *blocks.StructBlockValue]{
				Null:  true,
				Blank: true,
				Attributes: map[string]any{
					blocks.AttrKeyBlockControl: true,
				},
			}
		}),
		fattrs.Field(m, "Order", &m.Order, func() fattrs.PtrFieldConfig[*OrderItem, *Order] {
			return fattrs.PtrFieldConfig[*OrderItem, *Order]{
				Column:        "order_id",
				ReverseAlias:  "OrderItems",
				RelForeignKey: attrs.Relate(&Order{}, "", nil),
				Null:          false,
				Blank:         false,
			}
		}),
	)
}
