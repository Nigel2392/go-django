package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/shopspring/decimal"
)

type ProductSku struct {
	ID           drivers.ULID             `json:"id"`
	Product      *Product                 `json:"-"`
	Title        string                   `json:"title"`
	Price        decimal.Decimal          `json:"price"`
	Stock        uint64                   `json:"stock"`
	Images       *blocks.ListBlockValue   `json:"images"`
	SupplierData *blocks.StreamBlockValue `json:"supplierData"`
}

func (m *ProductSku) Label(ctx context.Context) string {
	return trans.T(ctx, "Product SKU")
}

func (m *ProductSku) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (b *ProductSku) GetImagesBlock() *blocks.ListBlock {
	var block = blocks.NewListBlock(blocks.ImageBlock(""))
	block.Min = 0
	block.Max = 3
	return block
}

func (b *ProductSku) GetSupplierDataBlock() *supplier.SupplierBlock {
	var streamBlock = blocks.NewStreamBlock()
	streamBlock.Min = 1
	streamBlock.Max = 1
	for _, backend := range supplier.IterBackends(true) {
		widget := backend.ProductWidget()
		if widget == nil {
			widget = blocks.NewStructBlock()
			widget.AddField("enabled", blocks.BooleanBlock())
		}
		widget.SetLabel(backend.Label)

		streamBlock.AddField(backend.InternalName(), widget)
	}
	return &supplier.SupplierBlock{StreamBlock: streamBlock}
}

func (m *ProductSku) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*ProductSku, drivers.ULID] {
			return fattrs.PtrFieldConfig[*ProductSku, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*ProductSku, string] {
			return fattrs.PtrFieldConfig[*ProductSku, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
				},
			}
		}),
		fattrs.Field(m, "Price", &m.Price, func() fattrs.PtrFieldConfig[*ProductSku, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*ProductSku, decimal.Decimal]{
				Config: attrs.FieldConfig{
					Attributes: map[string]interface{}{
						attrs.AttrPrecisionKey: 13,
						attrs.AttrScaleKey:     4,
					},
				},
			}
		}),
		fattrs.Field(m, "Stock", &m.Stock, func() fattrs.PtrFieldConfig[*ProductSku, uint64] {
			return fattrs.PtrFieldConfig[*ProductSku, uint64]{
				Config: attrs.FieldConfig{
					MinValue: 0,
				},
			}
		}),
		fattrs.Field(m, "Images", &m.Images, func() fattrs.PtrFieldConfig[*ProductSku, *blocks.ListBlockValue] {
			return fattrs.PtrFieldConfig[*ProductSku, *blocks.ListBlockValue]{
				Default: &blocks.ListBlockValue{
					V: make([]*blocks.ListBlockData, 0),
				},
			}
		}),
		fattrs.Field(m, "SupplierData", &m.SupplierData, func() fattrs.PtrFieldConfig[*ProductSku, *blocks.StreamBlockValue] {
			return fattrs.PtrFieldConfig[*ProductSku, *blocks.StreamBlockValue]{
				Default: &blocks.StreamBlockValue{
					V: make([]*blocks.StreamBlockData, 0),
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
