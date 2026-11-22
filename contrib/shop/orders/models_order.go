package orders

import (
	"context"
	"fmt"
	"html/template"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
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
	Customer   *customers.Customer
	State      InternalOrderState
	Payments   *queries.RelRevFK[*payments.Payment]
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
	panels := []admin.Panel{
		admin.FieldPanel("ID"),
		admin.FieldPanel("State"),
	}
	for _, payment := range m.Payments.AsList() {
		panels = append(panels, &admin.AlertPanel{
			Label: "Payment " + payment.ID.String(),
			HTML: template.HTMLEscapeString(fmt.Sprintf("%s · %s %s · %s",
				payment.Provider, payment.Amount, payment.Currency, payment.State)),
		})
	}
	return append(panels,
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
	)
}

func (m *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Order, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Order, drivers.ULID]{
				Config: attrs.Config{
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
				Config: attrs.Config{
					FormWidget: func(fc attrs.Config) widgets.Widget {
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
				Config: attrs.Config{
					Blank:   true,
					Default: false,
				},
			}
		}),
		fattrs.Field(m, "Processed", &m.Processed, func() fattrs.PtrFieldConfig[*Order, bool] {
			return fattrs.PtrFieldConfig[*Order, bool]{
				Config: attrs.Config{
					Blank:   true,
					Default: false,
				},
			}
		}),
		fattrs.Field(m, "Customer", &m.Customer, func() fattrs.PtrFieldConfig[*Order, *customers.Customer] {
			return fattrs.PtrFieldConfig[*Order, *customers.Customer]{
				Config: attrs.Config{
					Column:        "customer_id",
					RelForeignKey: attrs.Relate(&customers.Customer{}, "", nil),
					Null:          false,
					Blank:         false,
					ReverseAlias:  "Orders",
				},
			}
		}),
		fattrs.Field(m, "Payments", &m.Payments, func() fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*payments.Payment]] {
			return fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*payments.Payment]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
			}
		}),
		fattrs.Field(m, "OrderItems", &m.OrderItems, func() fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]] {
			return fattrs.PtrFieldConfig[*Order, *queries.RelRevFK[*OrderItem]]{}
		}),
	)
}
