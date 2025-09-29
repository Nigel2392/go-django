package forms

import (
	"context"

	"github.com/Nigel2392/go-django/src/internal/forms"
)

type (
	FormValuer         = forms.FormValuer
	FormValueConverter = forms.FormValueConverter
	FormValueOmitter   = forms.FormValueOmitter
	FormValueGetter    = forms.FormValueGetter
	Cleaner            = forms.Cleaner
	Validator          = forms.Validator
	Option             = forms.Option
	ErrorAdder         = forms.ErrorAdder
	FieldError         = forms.FieldError
	Widget             = forms.Widget
	Field              = forms.Field
	Form               = forms.Form
	BoundForm          = forms.BoundForm
	BoundField         = forms.BoundField
	IsValidDefiner     = forms.IsValidDefiner
	BinderWidget       = forms.BinderWidget
	FormWrapper[T any] = forms.FormWrapper[T]
	WithDataDefiner    = forms.WithDataDefiner
	ErrorDefiner       = forms.ErrorDefiner

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

type SaveableForm interface {
	Form
	Save() (map[string]interface{}, error)
}

func HasErrors(form ErrorDefiner) bool {
	var errs = form.BoundErrors()
	return errs != nil && errs.Len() > 0 || len(form.ErrorList()) > 0
}
