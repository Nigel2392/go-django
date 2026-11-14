package settings

import (
	"context"
	"encoding/json"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

type RegisteredSetting struct {
	ID           uint64
	InternalName string
	Data         json.RawMessage
}

func (m *RegisteredSetting) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*RegisteredSetting, uint64] {
			return fattrs.PtrFieldConfig[*RegisteredSetting, uint64]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "InternalName", &m.InternalName, func() fattrs.PtrFieldConfig[*RegisteredSetting, string] {
			return fattrs.PtrFieldConfig[*RegisteredSetting, string]{
				Config: attrs.FieldConfig{
					Null:      false,
					Blank:     false,
					MinLength: 2,
					MaxLength: 75,
				},
			}
		}),
		fattrs.Field(m, "Data", &m.Data, func() fattrs.PtrFieldConfig[*RegisteredSetting, json.RawMessage] {
			return fattrs.PtrFieldConfig[*RegisteredSetting, json.RawMessage]{
				Config: attrs.FieldConfig{
					Null:  true,
					Blank: true,
				},
			}
		}),
	)
}
