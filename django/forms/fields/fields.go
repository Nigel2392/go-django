package fields

import "github.com/Nigel2392/django/forms/widgets"

type Field interface {
	Attrs() map[string]string
	SetAttrs(attrs map[string]string)
	Hide(hidden bool)

	SetName(name string)
	SetLabel(label func() string)
	SetHelpText(helpText func() string)
	SetValidators(validators ...func(interface{}) error)
	SetWidget(widget widgets.Widget)

	Name() string
	Label() string
	HelpText() string
	Validate(value interface{}) []error
	Widget() widgets.Widget

	Clean(value interface{}) (interface{}, error)
	ValueToForm(value interface{}) interface{}
	ValueToGo(value interface{}) (interface{}, error)
	Required() bool
	SetRequired(b bool)
	IsEmpty(value interface{}) bool
}
