package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Category struct {
	ID             uint64
	ParentCategory *Category
	Children       *queries.RelRevFK[*Category]
	Name           string
	Slug           string
}

func (c *Category) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*Category, uint64] {
			return fattrs.PtrFieldConfig[*Category, uint64]{
				Config: attrs.FieldConfig{
					Primary: true,
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
					Column:        "parent_id",
					RelForeignKey: attrs.Relate(&Category{}, "", nil),
				},
			}
		}),
		fattrs.Field(c, "Children", &c.Children, func() fattrs.PtrFieldConfig[*Category, *queries.RelRevFK[*Category]] {
			return fattrs.PtrFieldConfig[*Category, *queries.RelRevFK[*Category]]{
				Config: attrs.FieldConfig{
					Label:    trans.S("Child Categories."),
					HelpText: trans.S("Subcategories bound to this category."),
					ReadOnly: true,
				},
			}
		}),
	)
}
