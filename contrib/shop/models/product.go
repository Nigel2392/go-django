package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/shopspring/decimal"
)

type Product struct {
	ID       uint64
	Title    string
	Slug     string
	Ranking  uint64
	PriceMax decimal.Decimal
	Skus     *queries.RelRevFK[*ProductSku]
}

func (m *Product) AddPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")).
			WithOutputFields("Slug"),
		admin.FieldPanel("Slug"),
		admin.FieldPanel("Ranking"),

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
	}
}

func (m *Product) EditPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")).
			WithOutputFields("Slug"),
		admin.RowPanel(
			admin.FieldPanel("Slug"),
			admin.FieldPanel("ID"),
		),
		admin.FieldPanel("Ranking"),

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
	}
}

func (m *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*Product, attrs.Field](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Product, uint64] {
			return fattrs.PtrFieldConfig[*Product, uint64]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
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
		// fields.NewVirtualField[decimal.Decimal](m, &m.PriceMax, "PriceMax", expr.AVG("Skus.Price")),
	)
}
