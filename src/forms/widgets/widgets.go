package widgets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Nigel2392/go-django/internal/forms"
	"github.com/Nigel2392/go-django/src/core/trans"
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

type FileObject struct {
	Name string
	File *bytes.Buffer
}

func (f FileObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Name)
}

type FormOption struct {
	OptLabel any
	OptValue string
}

func (o *FormOption) Label() string {
	label, ok := trans.GetText(context.Background(), o.OptLabel)
	if ok {
		return label
	}
	return fmt.Sprint(o.OptLabel)
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

func WrapOptions(ctx context.Context, options []Option, selectedValues []string) []WrappedOption {
	var wrappedOptions []WrappedOption
	var selectedVals = make(map[string]struct{})
	for _, s := range selectedValues {
		selectedVals[s] = struct{}{}
	}
	for _, option := range options {
		_, selected := selectedVals[option.Value()]
		wrappedOptions = append(wrappedOptions, WrappedOption{Option: option, Selected: selected})
	}
	return wrappedOptions
}
