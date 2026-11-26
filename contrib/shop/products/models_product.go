package products

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/editor"
	_ "github.com/Nigel2392/go-django/contrib/editor/features"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/specific"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type Price struct {
	Unit   currency.Unit
	Amount decimal.Decimal
}

type ProductModel interface {
	specific.Model
	ProductPrice(ctx context.Context) Price
}

type ProductQuery struct {
	*queries.WrappedQuerySet[*Product, *ProductQuery, *specific.SpecificQuerySet[*Product, ProductModel, *queries.QuerySet[*Product]]]
}

func Query() *ProductQuery {
	productQuerySet := &ProductQuery{}
	productQuerySet.WrappedQuerySet = queries.WrapQuerySet[*Product](
		specific.GetSpecificQuerySet(
			queries.GetQuerySet(&Product{}),
			specific.SpecificQuerySetOptions[*Product, ProductModel]{},
		),
		productQuerySet,
	)
	return productQuerySet
}

func (qs *ProductQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Product, *ProductQuery, *specific.SpecificQuerySet[*Product, ProductModel, *queries.QuerySet[*Product]]]) *ProductQuery {
	return &ProductQuery{
		WrappedQuerySet: wrapped,
	}
}

//	type ProductCategory struct {
//		Category *Category
//		Product  *Product
//	}
//
//	func (m *ProductCategory) FieldDefs(ctx context.Context) attrs.Definitions {
//		return attrs.Make[*ProductCategory, any](ctx, m,
//			fattrs.Field(m, "Category", &m.Category, func() fattrs.PtrFieldConfig[*ProductCategory, *Category] {
//				return fattrs.PtrFieldConfig[*ProductCategory, *Category]{
//					Config: attrs.FieldConfig{
//						RelOneToOne: attrs.Relate(&Category{}, "", nil),
//						Column:      "category_id",
//					},
//				}
//			}),
//			fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*ProductCategory, *Product] {
//				return fattrs.PtrFieldConfig[*ProductCategory, *Product]{
//					Config: attrs.FieldConfig{
//						RelForeignKey: attrs.Relate(&Product{}, "", nil),
//						Column:        "product_id",
//					},
//				}
//			}),
//		)
//	}

type SupplierValue string

type Product struct {
	specific.Specific[*Product]
	Image         *mediafiles.SimpleStoredObject `json:"image,omitempty"`
	ID            drivers.ULID                   `json:"id"`
	Title         string                         `json:"title"`
	Slug          string                         `json:"slug,omitempty"`
	Ranking       uint64                         `json:"ranking,omitempty"`
	Price         decimal.Decimal                `json:"price,omitempty"`
	AdditionalFee decimal.Decimal                `json:"additional_fee,omitempty"`
	TaxRate       decimal.Decimal                `json:"tax_rate,omitempty"`
	DiscountPrice decimal.Decimal                `json:"discount_price,omitempty"`
	Supplier      SupplierValue                  `json:"supplier,omitempty"`
	Category      *models.Category               `json:"category,omitempty"`
	Editor        *editor.EditorJSBlockData      `json:"editor,omitempty"`
	Coupons       *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (m *Product) ActualPrice() decimal.Decimal {
	// 	productprice:     50.00 €
	// 	service cost:      2.50 €
	// 				    --------- +
	// 				      52.50 €
	//  tax rate:	      21.00 %
	// 				    --------- *
	// 				    1102.50 €
	// 				     100.00 €
	//                  --------- /
	// 				     11.025 €
	//                   52.50  €
	// 				    --------- +
	// 			        [63.525] = €63.53
	price := m.Price.Add(m.AdditionalFee)
	return price.Add(price.
		Mul(m.TaxRate).
		Div(decimal.NewFromInt(100)))
}

func (m *Product) AddPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")).
			WithOutputFields("Slug"),
		admin.FieldPanel("Slug"),
		admin.FieldPanel("Supplier"),
		admin.FieldPanel("Price"),
		admin.FieldPanel("TaxRate"),
		admin.FieldPanel("DiscountPrice"),
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
		admin.FieldPanel("Price"),
		admin.FieldPanel("TaxRate"),
		admin.FieldPanel("DiscountPrice"),
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
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Image", &m.Image, func() fattrs.PtrFieldConfig[*Product, *mediafiles.SimpleStoredObject] {
			return fattrs.PtrFieldConfig[*Product, *mediafiles.SimpleStoredObject]{
				Config: attrs.Config{
					IsEmbedded: embedded,
					HelpText:   trans.S("This image should depict the product you are selling."),
				},
			}
		}, key...),
		fattrs.Field(m, "Title", &m.Title, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.Config{
					HelpText:   trans.S("The name of your product as visible to users."),
					IsEmbedded: embedded,
					MinLength:  2,
					MaxLength:  75,
					Blank:      false,
				},
			}
		}, key...),
		fattrs.Field(m, "Slug", &m.Slug, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{
				Config: attrs.Config{
					HelpText:   trans.S("The path part of this product in the browsers' search bar."),
					IsEmbedded: embedded,
					MinLength:  2,
					MaxLength:  75,
					FormWidget: func(fc attrs.Config) widgets.Widget {
						return widgets.NewTextInput(map[string]string{"readonly": ""})
					},
				},
			}
		}, key...),
		fattrs.Field(m, "Category", &m.Category, func() fattrs.PtrFieldConfig[*Product, *models.Category] {
			return fattrs.PtrFieldConfig[*Product, *models.Category]{
				Config: attrs.Config{
					HelpText:      trans.S("The category this product falls under."),
					IsEmbedded:    embedded,
					RelForeignKey: attrs.Relate(&models.Category{}, "", nil),
					Column:        "category_id",
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
		fattrs.Field(m, "Price", &m.Price, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.Config{
					HelpText:   trans.S("The price you are selling this product for."),
					IsEmbedded: embedded,
					Precision:  13,
					Scale:      4,
					Null:       false,
					Blank:      false,
				},
			}
		}, key...),
		fattrs.Field(m, "AdditionalFee", &m.AdditionalFee, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.Config{
					HelpText:   trans.S("The fee to add to the price per product."),
					IsEmbedded: embedded,
					Precision:  13,
					Scale:      4,
					Null:       false,
					Blank:      false,
				},
			}
		}, key...),
		fattrs.Field(m, "TaxRate", &m.TaxRate, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.Config{
					HelpText:   trans.S("The additional tax fee to add to the purchase price."),
					IsEmbedded: embedded,
					Precision:  13,
					Scale:      4,
					Null:       false,
					Blank:      false,
				},
			}
		}, key...),
		fattrs.Field(m, "DiscountPrice", &m.DiscountPrice, func() fattrs.PtrFieldConfig[*Product, decimal.Decimal] {
			return fattrs.PtrFieldConfig[*Product, decimal.Decimal]{
				Config: attrs.Config{
					HelpText:   trans.S("The discount price you are selling this product for. (Precedence over price)"),
					IsEmbedded: embedded,
					Precision:  13,
					Scale:      4,
					Null:       false,
					Blank:      true,
				},
			}
		}, key...),
		fattrs.Field(m, "Ranking", &m.Ranking, func() fattrs.PtrFieldConfig[*Product, uint64] {
			return fattrs.PtrFieldConfig[*Product, uint64]{
				Config: attrs.Config{
					HelpText:   trans.S("How well this product ranks generally measured across all products. (1-10)"),
					IsEmbedded: embedded,
					ReadOnly:   true,
				},
			}
		}, key...),
		fattrs.Field(m, "Supplier", &m.Supplier, func() fattrs.PtrFieldConfig[*Product, SupplierValue] {
			return fattrs.PtrFieldConfig[*Product, SupplierValue]{
				HelpText:   trans.S("The supplier that handles delivery of this product."),
				IsEmbedded: embedded,
				ReadOnly:   true,
			}
		}, key...),
		fattrs.Field(m, "SpecificID", &m.SpecificID, func() fattrs.PtrFieldConfig[*Product, string] {
			return fattrs.PtrFieldConfig[*Product, string]{Config: attrs.Config{
				IsEmbedded: embedded,
				ReadOnly:   true,
				FormField:  func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}, key...),
		fattrs.Field(m, "SpecificContentType", &m.SpecificContentType, func() fattrs.PtrFieldConfig[*Product, *contenttypes.BaseContentType[attrs.Definer]] {
			return fattrs.PtrFieldConfig[*Product, *contenttypes.BaseContentType[attrs.Definer]]{Config: attrs.Config{
				IsEmbedded: embedded,
				ReadOnly:   true,
				FormField:  func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}, key...),
		fattrs.Field(m, "Coupons", &m.Coupons, func() fattrs.PtrFieldConfig[*Product, *queries.RelM2M[attrs.Definer, attrs.Definer]] {
			return fattrs.PtrFieldConfig[*Product, *queries.RelM2M[attrs.Definer, attrs.Definer]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.Config{
					Attributes: map[string]interface{}{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	)
}
