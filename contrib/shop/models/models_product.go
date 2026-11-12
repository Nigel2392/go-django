package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/editor"
	_ "github.com/Nigel2392/go-django/contrib/editor/features"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/specific"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

type ProductQuery struct {
	*queries.WrappedQuerySet[*Product, *ProductQuery, *specific.SpecificQuerySet[*Product, supplier.ProductModel, *queries.QuerySet[*Product]]]
}

func Products() *ProductQuery {
	productQuerySet := &ProductQuery{}
	productQuerySet.WrappedQuerySet = queries.WrapQuerySet[*Product](
		specific.GetSpecificQuerySet(
			queries.GetQuerySet(&Product{}),
			specific.SpecificQuerySetOptions[*Product, supplier.ProductModel]{},
		),
		productQuerySet,
	)
	return productQuerySet
}

func (qs *ProductQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Product, *ProductQuery, *specific.SpecificQuerySet[*Product, supplier.ProductModel, *queries.QuerySet[*Product]]]) *ProductQuery {
	return &ProductQuery{
		WrappedQuerySet: wrapped,
	}
}

type Product struct {
	specific.Specific[*Product]
	ID       drivers.ULID
	Title    string
	Slug     string
	Ranking  uint64
	Supplier supplier.SupplierValue
	Editor   *editor.EditorJSBlockData
}

func (m *Product) ProductBase() *Product {
	return m
}

func (m *Product) AddPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")).
			WithOutputFields("Slug"),
		admin.FieldPanel("Slug"),
		admin.FieldPanel("Supplier"),
		admin.FieldPanel("Ranking"),
		admin.FieldPanel("Editor").Class("fullsize"),
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
		admin.FieldPanel("Supplier"),
		admin.FieldPanel("Ranking"),
		admin.FieldPanel("Editor").Class("fullsize"),
	}
}

func (m *Product) BeforeSave(ctx context.Context) error {
	if m.ID.IsZero() {
		m.ID = drivers.NewULID()
	}
	return nil
}

func (m *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	embedded := attrs.ContextHasFlag(ctx, attrs.CtxFlagEmbedded)
	var key []string
	if embedded {
		key = []string{"embedded"}
	}

	return attrs.Make[*Product, any](ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Product, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Product, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.FieldConfig{
					Embedded:  embedded,
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}, key...),
		fattrs.Field(m, "Slug", &m.Slug, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.FieldConfig{
					Embedded:  embedded,
					MinLength: 2,
					MaxLength: 75,
					FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
						return widgets.NewTextInput(map[string]string{"readonly": ""})
					},
				},
			}
		}, key...),
		editor.NewField(m, "Editor", editor.FieldConfig{
			Embedded: embedded,
			Label:    trans.S("Description"),
			HelpText: trans.S("Describe your product"),
			Default:  &editor.EditorJSBlockData{},
			Features: []string{
				"paragraph",
				"text-align",
				"list",
			},
		}),
		fattrs.Field(m, "Ranking", &m.Ranking, func() fattrs.PtrFieldConfig[*Product, uint64] {
			return fattrs.PtrFieldConfig[*Product, uint64]{
				Config: attrs.FieldConfig{
					Embedded: embedded,
					ReadOnly: true,
				},
			}
		}, key...),
		fattrs.Field(m, "Supplier", &m.Supplier, func() fattrs.PtrFieldConfig[*Product, supplier.SupplierValue] {
			return fattrs.PtrFieldConfig[*Product, supplier.SupplierValue]{
				Config: attrs.FieldConfig{
					Embedded: embedded,
					ReadOnly: true,
					FormField: func(opts ...func(fields.Field)) fields.Field {
						var widget = options.NewSelectInput(nil, func(ctx context.Context) []widgets.Option {
							var wo = make([]widgets.Option, 0)
							for _, backend := range supplier.IterBackends(true) {
								wo = append(wo, widgets.NewOption(backend.InternalName(), backend.Label(ctx), backend.InternalName()))
							}
							return wo
						})

						return fields.CharField(append(opts, fields.Widget(widget))...)
					},
				},
			}
		}, key...),
		fattrs.Field(m, "SpecificID", &m.SpecificID, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{Config: attrs.FieldConfig{
				Embedded:  embedded,
				ReadOnly:  true,
				FormField: func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}, key...),
		fattrs.Field(m, "SpecificContentType", &m.SpecificContentType, func() fattrs.PtrFieldConfig[*Product, *contenttypes.BaseContentType[attrs.Definer]] {
			return fattrs.PtrFieldConfig[*Product, *contenttypes.BaseContentType[attrs.Definer]]{Config: attrs.FieldConfig{
				Embedded:  embedded,
				ReadOnly:  true,
				FormField: func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}, key...),
	)
}
