package fields

import (
	"maps"

	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type BaseField struct {
	FieldName    string
	Required_    bool
	ReadOnly_    bool
	Attributes   map[string]string
	Validators   []func(interface{}) error
	FormLabel    func() string
	FormHelpText func() string
	FormWidget   widgets.Widget
	Caser        *cases.Caser
}

func NewField(opts ...func(Field)) *BaseField {
	var bf = &BaseField{}

	for _, opt := range opts {
		opt(bf)
	}

	if bf.Caser == nil {
		var titleCaser = cases.Title(language.English)
		bf.Caser = &titleCaser
	}

	return bf
}

func (i *BaseField) ValueToGo(value interface{}) (interface{}, error) {
	return i.Widget().ValueToGo(value)
}

func (i *BaseField) ValueToForm(value interface{}) interface{} {
	return i.Widget().ValueToForm(value)
}

func (i *BaseField) Name() string {
	return i.FieldName
}

func (i *BaseField) SetAttrs(attrs map[string]string) {
	if i.Attributes == nil {
		i.Attributes = make(map[string]string)
	}
	maps.Copy(i.Attributes, attrs)
}

func (i *BaseField) Hide(hidden bool) {
	i.FormWidget.Hide(hidden)
}

func (i *BaseField) SetLabel(label func() string) {
	i.FormLabel = label
}

func (i *BaseField) SetHelpText(helpText func() string) {
	i.FormHelpText = helpText
}

func (i *BaseField) SetName(name string) {
	i.FieldName = name
}

func (i *BaseField) SetWidget(w widgets.Widget) {
	i.FormWidget = w
}

func (i *BaseField) SetValidators(validators ...func(interface{}) error) {
	if i.Validators == nil {
		i.Validators = make([]func(interface{}) error, 0)
	}
	i.Validators = append(i.Validators, validators...)
}

func (i *BaseField) SetRequired(b bool) {
	i.Required_ = b
}

func (i *BaseField) Required() bool {
	return i.Required_
}

func (i *BaseField) SetReadOnly(b bool) {
	i.ReadOnly_ = b
}

func (i *BaseField) ReadOnly() bool {
	return i.ReadOnly_
}

func (i *BaseField) IsEmpty(value interface{}) bool {
	return IsZero(value)
}

func (i *BaseField) Attrs() map[string]string {
	return i.Attributes
}

func (i *BaseField) Label() string {
	if i.FormLabel != nil {
		return i.FormLabel()
	}
	return i.Caser.String(i.FieldName)
}

func (i *BaseField) HelpText() string {
	if i.FormHelpText != nil {
		return i.FormHelpText()
	}
	return ""
}

func (i *BaseField) HasChanged(initial, data interface{}) bool {
	return initial != data
}

func (i *BaseField) Clean(value interface{}) (interface{}, error) {
	return value, nil
}

func (i *BaseField) Validate(value interface{}) []error {
	var errors = make([]error, 0)
	for _, validator := range i.Validators {
		if err := validator(value); err != nil {
			errors = append(errors, err)
		}
	}

	var widget = i.Widget()
	if widget != nil {
		var errs = widget.Validate(value)
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	if len(errors) > 0 {
		return errors
	}

	if i.Required() && i.IsEmpty(value) {
		return []error{errs.NewValidationError(
			i.FieldName, errs.ErrFieldRequired,
		)}
	}
	return nil
}

func (i *BaseField) Widget() widgets.Widget {
	if i.FormWidget != nil {
		return i.FormWidget
	} else {
		return widgets.NewTextInput(nil)
	}
}
