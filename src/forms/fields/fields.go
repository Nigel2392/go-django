package fields

import (
	"context"

	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type Field interface {
	Attrs() map[string]string
	SetAttrs(attrs map[string]string)
	Hide(hidden bool)

	SetName(name string)
	SetLabel(label func(ctx context.Context) string)
	SetHelpText(helpText func(ctx context.Context) string)
	SetValidators(validators ...func(interface{}) error)
	SetWidget(widget widgets.Widget)

	Name() string
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Validate(ctx context.Context, value interface{}) []error
	Widget() widgets.Widget
	HasChanged(initial, data interface{}) bool

	Clean(ctx context.Context, value interface{}) (interface{}, error)
	ValueToForm(value interface{}) interface{}
	ValueToGo(value interface{}) (interface{}, error)
	Required() bool
	SetRequired(b bool)
	ReadOnly() bool
	SetReadOnly(b bool)
	IsEmpty(value interface{}) bool
}

type SaveableField interface {
	Field
	Save(value interface{}) (interface{}, error)
}
