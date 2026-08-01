package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

var _ queries.ActsBeforeCreate = (*Payment)(nil)

type Payment struct {
	ID    drivers.ULID
	State paystate.PayState
}

func (m *Payment) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Payment) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Payment, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Payment, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "State", &m.State, func() fattrs.PtrFieldConfig[*Payment, paystate.PayState] {
			return fattrs.PtrFieldConfig[*Payment, paystate.PayState]{
				Config: attrs.FieldConfig{
					MinLength: 2,
					Null:      false,
					Blank:     false,
				},
			}
		}),
	)
}
