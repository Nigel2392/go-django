package models

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/editor"
	_ "github.com/Nigel2392/go-django/contrib/editor/features"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/shopspring/decimal"
)

type ProductQuery struct {
	*queries.WrappedQuerySet[*Product, *ProductQuery, *queries.QuerySet[*Product]]
}

func Products() *ProductQuery {
	productQuerySet := &ProductQuery{}
	productQuerySet.WrappedQuerySet = queries.WrapQuerySet[*Product](
		queries.GetQuerySet(&Product{}),
		productQuerySet,
	)
	return productQuerySet
}

func (qs *ProductQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Product, *ProductQuery, *queries.QuerySet[*Product]]) *ProductQuery {
	return &ProductQuery{
		WrappedQuerySet: wrapped,
	}
}

type syncData struct {
	skuCount int
	priceMax decimal.Decimal
}

// A helper function in case the PriceMax and SkuCount columns get
// desynchronised.
//
// For the provided products it updates the PriceMax and SkuCount columns,
// but only if there was a difference in those columns compared to the syncData.
//
// Otherwise it will perform an in-db update for all provided objects.
func (qs *ProductQuery) SyncProducts(products ...*Product) error {
	qs = qs.Clone()

	tx, err := qs.GetOrCreateTransaction()
	if err != nil {
		return fmt.Errorf("SyncProducts/GetOrCreateTransaction: %w", err)
	}

	defer tx.Rollback(qs.Context())

	if len(products) > 0 {
		qs = qs.Filter("ID__in", products)
	}

	_, res, err := queries.GetQuerySet(&ProductSku{}).
		Select("ID", "Price", "Product").
		Filter("Product__in", qs.Select("ID")).
		OrderBy("Product").
		IterAll()

	if err != nil {
		return fmt.Errorf("SyncProducts/Skus/IterAll: %w", err)
	}

	var syncMap = make(map[uint64]syncData)
	for row, err := range res {
		if err != nil {
			return fmt.Errorf("SyncProducts/Skus/%d: %w", row.Object.ID, err)
		}

		m := syncMap[row.Object.Product.ID]

		m.skuCount++

		if row.Object.Price.GreaterThan(m.priceMax) {
			m.priceMax = row.Object.Price
		}

		syncMap[row.Object.Product.ID] = m
	}

	var changedList = make([]*Product, 0, len(syncMap))
	for _, product := range products {
		var (
			chngd bool
			m     = syncMap[product.ID]
		)

		if product.PriceMax != m.priceMax {
			product.PriceMax = m.priceMax
			chngd = true
		}

		if product.SkuCount != m.skuCount {
			product.SkuCount = m.skuCount
			chngd = true
		}

		if chngd {
			changedList = append(changedList, product)
		}
	}

	if len(products) > 0 && len(changedList) > 0 {
		_, err = qs.
			Select("PriceMax", "SkuCount").
			BulkUpdate(changedList)

		if err != nil {
			return fmt.Errorf("SyncProducts/Objects/BulkUpdate: %w", err)
		}
	}

	// product list was provided, they were updated above.
	if len(products) > 0 {
		return nil
	}

	for k, d := range syncMap {
		changedList = append(changedList, &Product{
			ID:       k,
			PriceMax: d.priceMax,
			SkuCount: d.skuCount,
		})
	}

	if len(changedList) > 0 {
		_, err = qs.
			Select("PriceMax", "SkuCount").
			BulkUpdate(changedList)

		if err != nil {
			return fmt.Errorf("SyncProducts/QuerySet/BulkUpdate: %w", err)
		}
	}

	return tx.Commit(qs.Context())
}

type Product struct {
	ID      uint64
	Title   string
	Slug    string
	Ranking uint64
	Editor  *editor.EditorJSBlockData
	Skus    *queries.RelRevFK[*ProductSku]

	// Should be synchronised periodically
	PriceMax decimal.Decimal
	SkuCount int

	// Used in product admin list
	LowestStock int
}

func (m *Product) AddPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Product"),
				admin.TitlePanel(admin.FieldPanel("Title")).
					WithOutputFields("Slug"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Ranking"),
				admin.FieldPanel("Editor").Class("fullsize"),
			),
			admin.PanelTab(
				trans.S("SKUs"),
				&admin.ModelFormPanel[*ProductSku, modelforms.ModelForm[*ProductSku]]{
					TargetType: &ProductSku{},
					FieldName:  "Skus",
					Classname:  "collapsible",
					// SubClassname: "collapsed",
					MinNum:         1,
					DisallowAdd:    false,
					DisallowRemove: false,
					Panels: []admin.Panel{
						admin.FieldPanel("Title"),
						admin.FieldPanel("Price"),
						admin.FieldPanel("Stock"),
					},
				},
			),
		),
	}
}

func (m *Product) EditPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Product"),
				admin.TitlePanel(admin.FieldPanel("Title")).
					WithOutputFields("Slug"),
				admin.RowPanel(
					admin.FieldPanel("Slug"),
					admin.FieldPanel("ID"),
				),
				admin.FieldPanel("Ranking"),
				admin.FieldPanel("Editor").Class("fullsize"),
			),
			admin.PanelTab(
				trans.S("SKUs"),
				&admin.ModelFormPanel[*ProductSku, modelforms.ModelForm[*ProductSku]]{
					TargetType: &ProductSku{},
					FieldName:  "Skus",
					Classname:  "collapsible",
					// SubClassname: "collapsed",
					MinNum:         1,
					DisallowAdd:    false,
					DisallowRemove: false,
					Panels: []admin.Panel{
						admin.FieldPanel("Title"),
						admin.FieldPanel("Price"),
						admin.FieldPanel("Stock"),
					},
				},
			),
		),
	}
}

func (m *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*Product, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Product, uint64] {
			return fattrs.PtrFieldConfig[*Product, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(m, "Slug", &m.Slug, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
					ReadOnly:  true,
				},
			}
		}),
		fattrs.Field(m, "PriceMax", &m.PriceMax, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
					Default:  0,
					Attributes: map[string]interface{}{
						attrs.AttrPrecisionKey: 13,
						attrs.AttrScaleKey:     4,
					},
				},
			}
		}),
		editor.NewField(m, "Editor", editor.FieldConfig{
			Label:    trans.S("Description"),
			HelpText: trans.S("Describe your product"),
			Default:  &editor.EditorJSBlockData{},
			Features: []string{
				"paragraph",
				"text-align",
				"list",
			},
		}),
		fattrs.Field(m, "SkuCount", &m.SkuCount, func() fattrs.PtrFieldConfig[*Product, int] {
			return fattrs.PtrFieldConfig[*Product, int]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
					Default:  0,
				},
			}
		}),
		fattrs.Field(m, "Ranking", &m.Ranking, func() fattrs.PtrFieldConfig[*Product, uint64] {
			return fattrs.PtrFieldConfig[*Product, uint64]{
				Config: attrs.FieldConfig{
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Skus", &m.Skus, func() fattrs.PtrFieldConfig[*Product, *queries.RelRevFK[*ProductSku]] {
			return fattrs.PtrFieldConfig[*Product, *queries.RelRevFK[*ProductSku]]{
				Config: attrs.FieldConfig{
					Label:    trans.S("Product SKUs"),
					HelpText: trans.S("Variations on this product."),
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "LowestStock", &m.LowestStock, func() fattrs.PtrFieldConfig[*Product, int] {
			return fattrs.PtrFieldConfig[*Product, int]{
				Config: attrs.FieldConfig{
					Embedded: true,
					Attributes: map[string]interface{}{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
		// fields.NewVirtualField[decimal.Decimal](m, &m.PriceMax, "PriceMax", expr.AVG("Skus.Price")),
	)
}
