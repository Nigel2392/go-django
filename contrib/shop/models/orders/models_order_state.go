package orders

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
)

var _ queries.ActsBeforeCreate = (*OrderItem)(nil)

type OrderState struct {
	ID         int64
	Label      string
	Action     InternalOrderState
	WorkflowID drivers.ULID // placeholder
}

type InternalOrderState string

func (i InternalOrderState) Label(ctx context.Context) string {
	fn, ok := orderStateLabels[i]
	if !ok {
		return string(i)
	}
	return fn.T(ctx)
}

const (
	OrderStateAccepted InternalOrderState = "order_accepted"
	OrderStatePending  InternalOrderState = "order_pending"
	OrderStateRejected InternalOrderState = "order_rejected"
	OrderStateWorkflow InternalOrderState = "order_workflow"
)

var orderStateLabels = map[InternalOrderState]trans.Lazy{
	OrderStateAccepted: trans.L("Order Accepted"),
	OrderStatePending:  trans.L("Order Pending"),
	OrderStateRejected: trans.L("Order Rejected"),
	OrderStateWorkflow: trans.L("Order Workflow"),
}

func stateActionOptions(ctx context.Context) []widgets.Option {
	return []widgets.Option{
		&widgets.FormOption{OptLabel: orderStateLabels[OrderStateAccepted], OptValue: string(OrderStateAccepted)},
		&widgets.FormOption{OptLabel: orderStateLabels[OrderStatePending], OptValue: string(OrderStatePending)},
		&widgets.FormOption{OptLabel: orderStateLabels[OrderStateRejected], OptValue: string(OrderStateRejected)},
		// &widgets.FormOption{OptLabel: orderStateLabels[OrderStateWorkflow], OptValue: string(OrderStateWorkflow)},
	}
}

func (m *OrderState) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*OrderState, int64] {
			return fattrs.PtrFieldConfig[*OrderState, int64]{
				Primary:  true,
				ReadOnly: true,
			}
		}),
		fattrs.Field(m, "Label", &m.Label, func() fattrs.PtrFieldConfig[*OrderState, string] {
			return fattrs.PtrFieldConfig[*OrderState, string]{
				Config: attrs.Config{
					MinLength: 1,
					MaxLength: 30,
					Blank:     false,
					Null:      false,
				},
			}
		}),
		fattrs.Field(m, "Action", &m.Action, func() fattrs.PtrFieldConfig[*OrderState, InternalOrderState] {
			return fattrs.PtrFieldConfig[*OrderState, InternalOrderState]{
				Options: []attrs.Option{
					{Label: orderStateLabels[OrderStateAccepted], Val: string(OrderStateAccepted)},
					{Label: orderStateLabels[OrderStatePending], Val: string(OrderStatePending)},
					{Label: orderStateLabels[OrderStateRejected], Val: string(OrderStateRejected)},
				},
				FormWidget: func(fc attrs.Config) widgets.Widget {
					return options.NewSelectInput(
						fc.WidgetAttrs,
						stateActionOptions,
					)
				},
			}
		}),
		fattrs.Field(m, "WorkflowID", &m.WorkflowID, func() fattrs.PtrFieldConfig[*OrderState, drivers.ULID] {
			return fattrs.PtrFieldConfig[*OrderState, drivers.ULID]{
				Config: attrs.Config{
					MinLength: 1,
					MaxLength: 30,
					Blank:     false,
					Null:      false,
				},
			}
		}),
	)
}
