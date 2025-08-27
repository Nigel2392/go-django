package forms

import (
	"context"

	"github.com/Nigel2392/go-django/src/internal/forms"

	_ "unsafe"
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
)

//go:linkname IsValid github.com/Nigel2392/go-django/src/internal/forms.IsValid
func IsValid(ctx context.Context, f Form) bool

//go:linkname FullClean github.com/Nigel2392/go-django/src/internal/forms.FullClean
func FullClean(ctx context.Context, f Form) (invalid, defaults, cleaned map[string]any)

type SaveableForm interface {
	Form
	Save() (map[string]interface{}, error)
}

func HasErrors(form Form) bool {
	var errs = form.BoundErrors()
	return errs != nil && errs.Len() > 0 || len(form.ErrorList()) > 0
}
