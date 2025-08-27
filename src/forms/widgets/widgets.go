package widgets

import (
	"github.com/Nigel2392/go-django/src/internal/forms"
)

type (
	Widget             = forms.Widget
	Field              = forms.Field
	Option             = forms.Option
	FormValuer         = forms.FormValuer
	FormValueConverter = forms.FormValueConverter
	FormValueOmitter   = forms.FormValueOmitter
	FormValueGetter    = forms.FormValueGetter
)

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
