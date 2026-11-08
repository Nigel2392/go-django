package supplier

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var (
	_ migrator.IndexDefiner = (*SupplierModel)(nil)
)

type SupplierModel struct {
	InternalName string
	Enabled      bool
}

func (u *SupplierModel) UniqueTogether() [][]string {
	return [][]string{
		{"InternalName"},
	}
}

func (u *SupplierModel) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "supplier_internal_name_idx", Fields: []string{"InternalName"}, Unique: true},
	}
}

func (s *SupplierModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, s,
		fattrs.Field(s, "InternalName", &s.InternalName, func() fattrs.PtrFieldConfig[*SupplierModel, string] {
			return fattrs.PtrFieldConfig[*SupplierModel, string]{
				Config: attrs.FieldConfig{
					Primary:   true,
					MinLength: 2,
					MaxLength: 75,
					Attributes: map[string]interface{}{
						attrs.AttrUniqueKey: true,
					},
				},
				IsZero: func(s string) bool {
					return s == ""
				},
			}
		}),
		fattrs.Field(s, "Enabled", &s.Enabled, func() fattrs.PtrFieldConfig[*SupplierModel, bool] {
			return fattrs.PtrFieldConfig[*SupplierModel, bool]{
				Config: attrs.FieldConfig{
					Blank:   true,
					Default: true,
				},
			}
		}),
	)
}
