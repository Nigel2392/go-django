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
	*queries.WrappedQuerySet[*Category, *CategoryQuery, *queries.QuerySet[*Category]]
}

func Categories() *CategoryQuery {
	base := &CategoryQuery{}
	base.WrappedQuerySet = queries.WrapQuerySet[*Category](
		queries.GetQuerySet(&Category{}),
		base,
	)
	return base
}

func (c *CategoryQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Category, *CategoryQuery, *queries.QuerySet[*Category]]) *CategoryQuery {
	return &CategoryQuery{
		WrappedQuerySet: wrapped,
	}
}

func (c *CategoryQuery) ChildrenOf(cat *Category) *CategoryQuery {
	if cat == nil {
		return c.Filter("ParentCategory__isnull", true)
	}
	return c.Filter("ParentCategory", cat.ID)
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
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Name", &c.Name, func() fattrs.PtrFieldConfig[*Category, string] {
			return fattrs.PtrFieldConfig[*Category, string]{
				Config: attrs.Config{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "Slug", &c.Slug, func() fattrs.PtrFieldConfig[*Category, string] {
			return fattrs.PtrFieldConfig[*Category, string]{
				Config: attrs.Config{
					MinLength: 2,
					MaxLength: 75,
					Blank:     false,
				},
			}
		}),
		fattrs.Field(c, "ParentCategory", &c.ParentCategory, func() fattrs.PtrFieldConfig[*Category, *Category] {
			return fattrs.PtrFieldConfig[*Category, *Category]{
				Config: attrs.Config{
					Null:          true,
					Blank:         true,
					Column:        "parent_id",
					RelForeignKey: attrs.Relate(&Category{}, "", nil),
					ReverseAlias:  "Children",
					FormWidget: func(fc attrs.Config) widgets.Widget {
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
				Flags:    fattrs.F_IS_REVERSE_FLD,
				ReadOnly: true,
				Label:    trans.S("Child Categories."),
				HelpText: trans.S("Subcategories bound to this category."),
				// RelForeignKeyReverse: attrs.Relate(&Category{}, "ParentCategory", nil),
				ReverseAlias: "ParentCategory",
				FormField: func(opts ...func(fields.Field)) fields.Field {
					return nil
				},
			}
		}),
	)
}
