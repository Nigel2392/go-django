package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
)

type CategoryQuery struct {
	*queries.WrappedQuerySet[*Product, *CategoryQuery, *queries.QuerySet[*Category]]
}

func Categories() *CategoryQuery {
	base := &CategoryQuery{}
	base.WrappedQuerySet = queries.WrapQuerySet[*Product](
		queries.GetQuerySet(&Category{}),
		base,
	)
	return base
}

func (c *CategoryQuery) ChildrenOf(cat *Category) *CategoryQuery {
	return c.Filter("ParentCategory", cat.ParentCategory.ID)
}

type Category struct {
	ID             uint64
	Name           string
	Slug           string
	ParentCategory *Category
	Children       *queries.RelRevFK[*Category]
}

func (c *Category) String() string {
	return c.Name
}

func (c *Category) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*Category, uint64] {
			return fattrs.PtrFieldConfig[*Category, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Name", &c.Name, func() fattrs.PtrFieldConfig[*Category, string] {
			return fattrs.PtrFieldConfig[*Category, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "Slug", &c.Slug, func() fattrs.PtrFieldConfig[*Category, string] {
			return fattrs.PtrFieldConfig[*Category, string]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "ParentCategory", &c.ParentCategory, func() fattrs.PtrFieldConfig[*Category, *Category] {
			return fattrs.PtrFieldConfig[*Category, *Category]{
				Config: attrs.FieldConfig{
					Null:          true,
					Blank:         true,
					Column:        "parent_id",
					RelForeignKey: attrs.Relate(&Category{}, "", nil),
					FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
						return formfields.ModelSelectWidget(
							true, "-----",
							chooser.BaseChooserOptions{TargetObject: &Category{}},
							nil,
						)
					},
				},
			}
		}),
		fattrs.Field(c, "Children", &c.Children, func() fattrs.PtrFieldConfig[*Category, *queries.RelRevFK[*Category]] {
			return fattrs.PtrFieldConfig[*Category, *queries.RelRevFK[*Category]]{
				Config: attrs.FieldConfig{
					ReadOnly:             true,
					Label:                trans.S("Child Categories."),
					HelpText:             trans.S("Subcategories bound to this category."),
					RelForeignKeyReverse: attrs.Relate(&Category{}, "", nil),
					FormField: func(opts ...func(fields.Field)) fields.Field {
						return nil
					},
				},
			}
		}),
	)
}
