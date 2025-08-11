package widgets

import (
	"context"
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
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

type Widget interface {
	IsHidden() bool
	Hide(hidden bool)
	FormType() string
	SetAttrs(attrs map[string]string)
	IdForLabel(id string) string
	GetContextData(ctx context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context
	RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string, context ctx.Context) error
	Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error
	Validate(ctx context.Context, value interface{}) []error

	FormValuer
	media.MediaDefiner
}

type Option interface {
	Label() string
	Value() string
}

type FormOption struct {
	OptLabel string
	OptValue string
}

func (o *FormOption) Label() string {
	return o.OptLabel
}

func (o *FormOption) Value() string {
	return o.OptValue
}

func NewOption(name, label, value string) Option {
	return &FormOption{OptLabel: label, OptValue: value}
}

type WrappedOption struct {
	Option
	Selected bool
}

func (w *WrappedOption) Label() string {
	return w.Option.Label()
}

func (w *WrappedOption) Value() string {
	return w.Option.Value()
}

func WrapOptions(options []Option, selectedValues []string) []WrappedOption {
	var wrappedOptions []WrappedOption
	for _, option := range options {
		var selected bool
		for _, selectedValue := range selectedValues {
			if selectedValue == option.Value() {
				selected = true
				break
			}
		}
		wrappedOptions = append(wrappedOptions, WrappedOption{Option: option, Selected: selected})
	}
	return wrappedOptions
}
