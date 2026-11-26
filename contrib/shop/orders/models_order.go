package orders

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
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

func (m *Order) EditPanels(ctx context.Context) ([]admin.Panel, error) {
	if m.Payments == nil {
		attrs.Define(ctx, m).Get("Payments")
	}

	n, paymentRows, err := m.Payments.
		Objects().
		WithContext(ctx).
		OrderBy("CreatedAt", "ID").
		IterAll()
	if err != nil {
		return nil, errors.Wrap(err, "EditPanels")
	}

	var paymentPanels = make([]admin.Panel, 0, n)
	for row, err := range paymentRows {
		if err != nil {
			return nil, errors.Wrap(err, "EditPanels/paymentRows")
		}

		var alertType = admin.ALERT_INFO
		switch row.Object.State {
		case paystate.STATUS_CANCEL,
			paystate.STATUS_EXPIRED,
			paystate.STATUS_DENIED,
			paystate.STATUS_FAILURE,
			paystate.STATUS_CHARGEBACK,
			paystate.STATUS_PARTIAL_REFUND:
			alertType = admin.ALERT_ERROR

		case paystate.STATUS_VERIFY,
			paystate.STATUS_AUTHORIZE,
			paystate.STATUS_REFUNDING,
			paystate.STATUS_REFUND,
			paystate.STATUS_PARTLY_CAPTURED:
			alertType = admin.ALERT_WARNING

		case paystate.STATUS_PAID,
			paystate.STATUS_NO_PAYMENT_REQUIRED:
			alertType = admin.ALERT_SUCCESS
		}

		paymentPanels = append(paymentPanels, &admin.AlertPanel{
			Type:  alertType,
			Label: trans.S("Payment %s", row.Object.ID.String()),
			HTML:  paymentHistoryHTML(ctx, row.Object),
		})
	}

	if len(paymentPanels) == 0 {
		paymentPanels = append(paymentPanels, &admin.AlertPanel{
			Type:  admin.ALERT_ERROR,
			Label: trans.S("No payments"),
			HTML: template.HTMLEscapeString(trans.T(
				ctx, "No payments on record for order %q",
				m.ID.String(),
			)),
		})
	}

	tabs := []admin.TabPanel{
		admin.PanelTab(
			trans.S("Product Details"),
			admin.FieldPanel("ID"),
			admin.FieldPanel("State"),
			admin.FieldPanel("Processed"),
			admin.FieldPanel("Completed"),
		),
		admin.PanelTab(
			trans.S("Order Items"),
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
		),
		admin.PanelTab(
			trans.S("Payment Details"),
			paymentPanels...,
		),
	}

	couponPanels, err := m.couponPanels(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "EditPanels/coupons")
	}
	if len(couponPanels) > 0 {
		tabs = append(tabs, admin.PanelTab(trans.S("Coupons"), couponPanels...))
	}

	return []admin.Panel{admin.TabbedPanel(tabs...)}, nil
}

func (m *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*Order, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Order, drivers.ULID]{
				Primary:  true,
				ReadOnly: true,
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
