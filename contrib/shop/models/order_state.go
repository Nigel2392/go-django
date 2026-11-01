package models

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
	return fn(ctx)
}

const (
	orderStateAccepted InternalOrderState = "order_accepted"
	orderStateRejected InternalOrderState = "order_rejected"
	orderStateWorkflow InternalOrderState = "order_workflow"
)

var orderStateLabels = map[InternalOrderState]func(context.Context) string{
	orderStateAccepted: trans.S("Order Accepted"),
	orderStateRejected: trans.S("Order Rejected"),
	orderStateWorkflow: trans.S("Order Workflow"),
}

func stateActionOptions() []widgets.Option {
	return []widgets.Option{
		&widgets.FormOption{OptLabel: orderStateLabels[orderStateAccepted], OptValue: string(orderStateAccepted)},
		&widgets.FormOption{OptLabel: orderStateLabels[orderStateRejected], OptValue: string(orderStateRejected)},
		&widgets.FormOption{OptLabel: orderStateLabels[orderStateWorkflow], OptValue: string(orderStateWorkflow)},
	}
}

func (m *OrderState) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*OrderState, int64] {
			return fattrs.PtrFieldConfig[*OrderState, int64]{
				Config: attrs.FieldConfig{
					Primary: true,
				},
			}
		}),
		fattrs.Field(m, "Label", &m.Label, func() fattrs.PtrFieldConfig[*OrderState, string] {
			return fattrs.PtrFieldConfig[*OrderState, string]{
				Config: attrs.FieldConfig{
					MinLength: 1,
					MaxLength: 30,
					Blank:     false,
					Null:      false,
				},
			}
		}),
		fattrs.Field(m, "Action", &m.Action, func() fattrs.PtrFieldConfig[*OrderState, InternalOrderState] {
			return fattrs.PtrFieldConfig[*OrderState, InternalOrderState]{
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
		fattrs.Field(m, "WorkflowID", &m.WorkflowID, func() fattrs.PtrFieldConfig[*OrderState, drivers.ULID] {
			return fattrs.PtrFieldConfig[*OrderState, drivers.ULID]{
				Config: attrs.FieldConfig{
					MinLength: 1,
					MaxLength: 30,
					Blank:     false,
					Null:      false,
				},
			}
		}),
	)
}
