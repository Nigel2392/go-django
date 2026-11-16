package orders

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

var _ queries.ActsBeforeCreate = (*Order)(nil)

type Order struct {
	ID drivers.ULID
	// State      *OrderState
	State      InternalOrderState
	Payment    *payments.Payment
	OrderItems *queries.RelRevFK[*OrderItem]
	Completed  bool
	Processed  bool
}

func (m *Order) BeforeCreate(ctx context.Context) error {
	m.ID = drivers.NewULID()
	return nil
}

func (m *Order) AddPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("State"),
		&admin.ModelFormPanel[*payments.Payment, modelforms.ModelForm[*payments.Payment]]{
			TargetType:     &payments.Payment{},
			FieldName:      "Payment",
			Classname:      "collapsible",
			MinNum:         1,
			DisallowAdd:    true,
			DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("Amount"),
				// admin.FieldPanel("Provider"),
				admin.FieldPanel("State"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("UpdatedAt"),
				//	Amount
				//	Provider
				//	State
				//	CreatedAt
				//	UpdatedAt
			},
		},
		&admin.ModelFormPanel[*OrderItem, modelforms.ModelForm[*OrderItem]]{
			TargetType: &OrderItem{},
			FieldName:  "OrderItems",
			Classname:  "collapsible",
			// SubClassname: "collapsed",
			MinNum: 1,
			// DisallowAdd:    true,
			// DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Amount"),
			},
		},
	}
}

func (m *Order) EditPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("ID"),
		admin.FieldPanel("State"),
		&admin.ModelFormPanel[*payments.Payment, modelforms.ModelForm[*payments.Payment]]{
			TargetType:     &payments.Payment{},
			FieldName:      "Payment",
			Classname:      "collapsible",
			MinNum:         1,
			DisallowAdd:    true,
			DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("Amount"),
				// admin.FieldPanel("Provider"),
				admin.FieldPanel("State"),
				admin.FieldPanel("CreatedAt"),
				admin.FieldPanel("UpdatedAt"),
				//	Amount
				//	Provider
				//	State
				//	CreatedAt
				//	UpdatedAt
			},
		},
		&admin.ModelFormPanel[*OrderItem, modelforms.ModelForm[*OrderItem]]{
			TargetType: &OrderItem{},
			FieldName:  "OrderItems",
			Classname:  "collapsible",
			// SubClassname: "collapsed",
			MinNum: 1,
			// DisallowAdd:    true,
			// DisallowRemove: true,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Amount"),
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
		//	fattrs.Field(m, "State", &m.State, func() fattrs.PtrFieldConfig[*Order, *OrderState] {
		//		return fattrs.PtrFieldConfig[*Order, *OrderState]{
		//			Config: attrs.FieldConfig{
		//				Column:        "state_id",
		//				RelForeignKey: attrs.Relate(&OrderState{}, "", nil),
		//			},
		//		}
		//	}),
		fattrs.Field(m, "State", &m.State, func() fattrs.PtrFieldConfig[*Order, InternalOrderState] {
			return fattrs.PtrFieldConfig[*Order, InternalOrderState]{
				Config: attrs.FieldConfig{
					FormWidget: func(fc attrs.FieldConfig) widgets.Widget {
						return options.NewSelectInput(
							fc.WidgetAttrs,
							stateActionOptions,
						)
					},
				},
			}
		}),
		fattrs.Field(m, "Completed", &m.Completed, func() fattrs.PtrFieldConfig[*Order, bool] {
			return fattrs.PtrFieldConfig[*Order, bool]{
				Config: attrs.FieldConfig{
					Blank:   true,
					Default: false,
				},
			}
		}),
		fattrs.Field(m, "Processed", &m.Processed, func() fattrs.PtrFieldConfig[*Order, bool] {
			return fattrs.PtrFieldConfig[*Order, bool]{
				Config: attrs.FieldConfig{
					Blank:   true,
					Default: false,
				},
			}
		}),
		fattrs.Field(m, "Payment", &m.Payment, func() fattrs.PtrFieldConfig[*Order, *payments.Payment] {
			return fattrs.PtrFieldConfig[*Order, *payments.Payment]{
				Config: attrs.FieldConfig{
					Column:        "payment_id",
					RelForeignKey: attrs.Relate(&payments.Payment{}, "", nil),
					Null:          true,
					Blank:         false,
				},
			}
		}),
		fattrs.Field(m, "OrderItems", &m.OrderItems, func() fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]] {
			return fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]]{}
		}),
	)
}
