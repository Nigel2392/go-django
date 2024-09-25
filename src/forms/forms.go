package forms

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
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

type FieldError interface {
	Field() string
	Errors() []error
}

type Form interface {
	FormRenderer

	Prefix() string
	SetPrefix(prefix string)
	SetInitial(initial map[string]interface{})
	SetValidators(validators ...func(Form) []error)
	Ordering([]string)
	FieldOrder() []string

	Field(name string) (fields.Field, bool)
	Widget(name string) (widgets.Widget, bool)
	Fields() []fields.Field
	Widgets() []widgets.Widget
	AddField(name string, field fields.Field)
	AddWidget(name string, widget widgets.Widget)
	DeleteField(name string) bool
	BoundForm() BoundForm
	BoundFields() *orderedmap.OrderedMap[string, BoundField]
	BoundErrors() *orderedmap.OrderedMap[string, []error]
	ErrorList() []error

	WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) Form
	InitialData() map[string]interface{}
	CleanedData() map[string]interface{}

	FullClean()
	Validate()
	HasChanged() bool
	IsValid() bool

	OnValid(...func(Form))
	OnInvalid(...func(Form))
	OnFinalize(...func(Form))
}
