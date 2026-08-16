package forms

import (
	"context"

	"github.com/Nigel2392/go-django/internal/forms"
	"github.com/Nigel2392/go-django/src/core/filesystem"
)

type (
	FormValuer             = forms.FormValuer
	FormValueConverter     = forms.FormValueConverter
	FormValueOmitter       = forms.FormValueOmitter
	FormValueGetter        = forms.FormValueGetter
	FormRenderer           = forms.FormRenderer
	Cleaner                = forms.Cleaner
	Validator              = forms.Validator
	Option                 = forms.Option
	ErrorAdder             = forms.ErrorAdder
	FieldError             = forms.FieldError
	Widget                 = forms.Widget
	Field                  = forms.Field
	Form                   = forms.Form
	Minimum                = forms.Minimum
	BoundForm              = forms.BoundForm
	BoundField             = forms.BoundField
	IsValidDefiner         = forms.IsValidDefiner
	BinderWidget           = forms.BinderWidget
	BoundFieldBinderWidget = forms.BoundFieldBinderWidget
	FormWrapper[T any]     = forms.FormWrapper[T]
	WithDataDefiner        = forms.WithDataDefiner
	ErrorDefiner           = forms.ErrorDefiner
	FormFieldDefiner       = forms.FormFieldDefiner
	FieldLabelRenderer     = forms.FieldLabelRenderer
	FieldHelpTextRenderer  = forms.FieldHelpTextRenderer
	FormData               = forms.FormData

	PrevalidatorMixin = forms.PrevalidatorMixin
	ValidatorMixin    = forms.ValidatorMixin
	FullCleanMixin    = forms.FullCleanMixin
)

func IsValid[T any](ctx context.Context, f T) bool {
	return forms.IsValid(ctx, f)
}

func FullClean(ctx context.Context, f Form) (invalid, defaults, cleaned map[string]any) {
	return forms.FullClean(ctx, f)
}

func ValueFromDataDict[T any](ctx context.Context, form FormFieldDefiner, name string, data map[string][]string, files map[string][]filesystem.FileHeader) (T, bool, []error) {
	return forms.FormValueFromDataDict[T](ctx, form, name, data, files)
}

func HasErrors(f ErrorDefiner) bool {
	return forms.HasErrors(f)
}

type SaveableForm interface {
	Form
	Save() (map[string]interface{}, error)
}
