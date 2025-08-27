package forms

import "github.com/Nigel2392/go-django/src/internal/forms"

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
)

type SaveableForm interface {
	Form
	Save() (map[string]interface{}, error)
}

func HasErrors(form Form) bool {
	var errs = form.BoundErrors()
	return errs != nil && errs.Len() > 0 || len(form.ErrorList()) > 0
}
