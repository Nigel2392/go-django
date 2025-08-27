package formsets

import (
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type FormSet interface {
	FullClean()
	Validate()
	HasChanged() bool
	IsValid() bool
	Media() media.Media
	Forms() []forms.Form
	Form(index int) (form forms.Form, ok bool)
	CleanedData() []map[string]any
	DeletedForms() []map[string]any
	ErrorList() [][]error
	BoundErrors() []*orderedmap.OrderedMap[string, []error]
	SetValidators(...func([]forms.Form, []map[string]any) []error)
}

type BaseFormSet struct {
	forms      []forms.Form
	validators []func([]forms.Form, []map[string]any) []error
}
