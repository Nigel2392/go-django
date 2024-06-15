package forms

import (
	"html/template"
	"io"
	"net/http"
	"net/url"

	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
)

type Cleaner interface {
	Clean(value interface{}) (interface{}, error)
}

type Validator interface {
	Validate(value interface{}) []error
}

type BoundField interface {
	ID() string
	Name() string
	Widget() widgets.Widget
	Input() fields.Field
	Label() template.HTML
	HelpText() template.HTML
	Field() template.HTML
	HTML() template.HTML
	Attrs() map[string]string
	Value() interface{}
	Errors() []error
}

type FormRenderer interface {
	AsP() template.HTML
	AsUL() template.HTML
	Media() media.Media
}

type ErrorAdder interface {
	AddFormError(errorList ...error)
	AddError(name string, errorList ...error)
}

type Form interface {
	FormRenderer

	Prefix() string
	SetPrefix(prefix string)
	SetInitial(initial map[string]interface{})
	SetValidators(validators ...func(Form) []error)
	Ordering([]string)
	FieldOrder() []string

	Field(name string) fields.Field
	Widget(name string) widgets.Widget
	Fields() []fields.Field
	Widgets() []widgets.Widget
	AddField(name string, field fields.Field)
	AddWidget(name string, widget widgets.Widget)
	BoundForm() BoundForm
	BoundFields() *orderedmap.OrderedMap[string, BoundField]
	BoundErrors() *orderedmap.OrderedMap[string, []error]
	ErrorList() []error

	WithData(data url.Values, files map[string][]io.ReadCloser, r *http.Request) Form
	CleanedData() map[string]interface{}

	FullClean()
	Validate()
	HasChanged() bool
	IsValid() bool
	Close() error

	OnValid(...func(Form))
	OnInvalid(...func(Form))
	OnFinalize(...func(Form))
}
