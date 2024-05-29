package fields

import (
	"maps"
	"reflect"

	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/widgets"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type BaseField struct {
	FieldName  string
	Required   bool
	Attributes map[string]string
	Validators []func(interface{}) error
	FormLabel  func() string
	FormWidget widgets.Widget
	Caser      *cases.Caser
}

func NewField(type_ func() string, opts ...func(Field)) *BaseField {
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
	i.Required = b
}

func (i *BaseField) Attrs() map[string]string {
	return i.Attributes
}

func (i *BaseField) Label() string {
	return i.FormLabel()
}

func (i *BaseField) Clean(value interface{}) (interface{}, error) {
	return value, nil
}

func (i *BaseField) Validate(value interface{}) []error {

	for _, validator := range i.Validators {
		if err := validator(value); err != nil {
			return []error{errs.NewValidationError(i.FieldName, err.Error())}
		}
	}

	if i.Required {
		if value == nil {
			return []error{errs.ErrFieldRequired}
		}
		var (
			t = reflect.TypeOf(value)
			v = reflect.ValueOf(value)
		)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
			v = v.Elem()
		}
		switch t.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			if v.Len() == 0 {
				return []error{errs.ErrFieldRequired}
			}
		case reflect.Ptr:
			if v.IsNil() {
				return []error{errs.ErrFieldRequired}
			}
		default:
			if v.IsZero() {
				return []error{errs.ErrFieldRequired}
			}
		}
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
