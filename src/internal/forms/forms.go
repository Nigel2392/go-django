package forms

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type FormValueConverter interface {
	// Convert the forms' string value to the appropriate GO type.
	ValueToGo(value interface{}) (interface{}, error)

	// Convert the GO type to the forms' string value.
	ValueToForm(value interface{}) interface{}
}

type FormValueOmitter interface {
	// Check if the value is omitted from the data provided.
	ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool
}

type FormValueGetter interface {
	// Get the value from the provided data.
	ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error)
}

type FormValuer interface {
	FormValueConverter
	FormValueOmitter
	FormValueGetter
}

type Cleaner interface {
	Clean(ctx context.Context, value interface{}) (interface{}, error)
}

type Validator interface {
	Validate(ctx context.Context, value interface{}) []error
}

type Option interface {
	Label() string
	Value() string
}

type ErrorAdder interface {
	AddFormError(errorList ...error)
	AddError(name string, errorList ...error)
}

type ErrorDefiner interface {
	ErrorList() []error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
}

type FieldError interface {
	Name() string
	Field() string
	Errors() []error
}

type Widget interface {
	IsHidden() bool
	Hide(hidden bool)
	FormType() string
	Field() Field
	BindField(field Field)
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(ctx context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string, context ctx.Context) error

	// Render is basically the same as RenderWithErrors, except that it does not take a context.
	// The widget should always be able to generate some sort of context itself based on the provided parameters.
	Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error
	Validate(ctx context.Context, value interface{}) []error

	FormValuer
	media.MediaDefiner
}

type BinderWidget interface {
	Widget
	BoundField(ctx context.Context, w Widget, f Field, name string, value interface{}, errors []error) BoundField
}

type Field interface {
	FormValueConverter
	Attrs() map[string]string
	SetAttrs(attrs map[string]string)
	Hide(hidden bool)

	SetName(name string)
	SetLabel(label func(ctx context.Context) string)
	SetHelpText(helpText func(ctx context.Context) string)
	SetValidators(validators ...func(interface{}) error)
	SetWidget(widget Widget)

	Name() string
	Label(ctx context.Context) string
	HelpText(ctx context.Context) string
	Validate(ctx context.Context, value interface{}) []error
	Widget() Widget
	HasChanged(initial, data interface{}) bool

	Clean(ctx context.Context, value interface{}) (interface{}, error)
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

type WithDataDefiner interface {
	WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request)
	Data() (url.Values, map[string][]filesystem.FileHeader)
}

type FormFieldDefiner interface {
	Field(name string) (Field, bool)
	Widget(name string) (Widget, bool)
	PrefixName(fieldName string) string
}

type Form interface {
	WithDataDefiner
	FullCleanMixin
	ErrorAdder
	ErrorDefiner

	AsP() template.HTML
	AsUL() template.HTML
	Media() media.Media

	Context() context.Context
	WithContext(ctx context.Context)
	Prefix() string
	SetPrefix(prefix string)
	SetInitial(initial map[string]interface{})
	SetValidators(validators ...func(Form, map[string]interface{}) []error)
	Validators() []func(f Form, cleanedData map[string]interface{}) []error
	Ordering([]string)
	FieldOrder() []string

	Field(name string) (Field, bool)
	Fields() []Field
	Widgets() []Widget
	AddField(name string, field Field)
	AddWidget(name string, widget Widget)
	DeleteField(name string) bool
	BoundForm() BoundForm
	BoundFields() *orderedmap.OrderedMap[string, BoundField]

	InitialData() map[string]interface{}
	CleanedData() map[string]interface{}

	OnValid(...func(Form))
	OnInvalid(...func(Form))
	OnFinalize(...func(Form))

	WasCleaned() bool
	HasChanged() bool
	CallbackOnValid() []func(f Form)
	CallbackOnInvalid() []func(f Form)
	CallbackOnFinalize() []func(f Form)
	CleanedDataUnsafe() map[string]interface{}
}

type BoundForm interface {
	AsP() template.HTML
	AsUL() template.HTML
	Media() media.Media
	Fields() []BoundField
	FieldMap() map[string]BoundField // map of field name to BoundField
	ErrorList() []error
	UnpackErrors() []FieldError
	Errors() *orderedmap.OrderedMap[string, []error]
}

type BoundField interface {
	ID() string
	Name() string
	Widget() Widget
	Hidden() bool
	Input() Field
	Label() template.HTML
	HelpText() template.HTML
	Field() template.HTML
	HTML() template.HTML
	Context() context.Context
	Attrs() map[string]string
	Value() interface{}
	Errors() []error
}
