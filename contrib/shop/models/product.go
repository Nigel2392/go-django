package models

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/gosimple/slug"
)

type Product struct {
	ID    uint64
	Title string
	Slug  string

	Skus *queries.RelRevFK[*ProductSku]
}

func (m *Product) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
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
				},
				Default: func(p *Product) string {
					return slug.Make(p.Title)
				},
			}
		}),
		fattrs.Field(m, "Skus", &m.Skus, func() fattrs.PtrFieldConfig[*Product, *queries.RelRevFK[*ProductSku]] {
			return fattrs.PtrFieldConfig[*Product, *queries.RelRevFK[*ProductSku]]{}
		}),
	)
}
