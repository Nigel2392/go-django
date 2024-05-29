package fields

import "github.com/Nigel2392/django/forms/widgets"

type Field interface {
	Attrs() map[string]string
	SetAttrs(attrs map[string]string)
	Hide(hidden bool)
	SetLabel(label func() string)
	SetName(name string)
	SetWidget(widget widgets.Widget)
	SetValidators(validators ...func(interface{}) error)
	SetRequired(b bool)
	Name() string
	Label() string
	Validate(value interface{}) []error
	Widget() widgets.Widget
	Clean(value interface{}) (interface{}, error)
	ValueToForm(value interface{}) interface{}
	ValueToGo(value interface{}) (interface{}, error)
}
