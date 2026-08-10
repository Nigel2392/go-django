package models

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var _ queries.ActsBeforeCreate = (*Order)(nil)

type Order struct {
	ID         drivers.ULID
	Payment    *Payment
	OrderItems *queries.RelRevFK[*OrderItem]
}

func (m *Order) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Order) EditPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("ID"),
		&admin.ModelFormPanel[*OrderItem, modelforms.ModelForm[*OrderItem]]{
			TargetType: &OrderItem{},
			FieldName:  "OrderItems",
			Classname:  "collapsible",
			// SubClassname: "collapsed",
			MinNum:         0,
			DisallowAdd:    true,
			DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
			},
		},
	}
}

func (m *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Order, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Order, drivers.ULID]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Payment", &m.Payment, func() fattrs.PtrFieldConfig[*Order, *Payment] {
			return fattrs.PtrFieldConfig[*Order, *Payment]{
				Config: attrs.FieldConfig{
					Column:        "payment_id",
					RelForeignKey: attrs.Relate(&Payment{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "OrderItems", &m.OrderItems, func() fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]] {
			return fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]]{}
		}),
	)
}
