package fields

import (
	"context"
	"database/sql/driver"
	"maps"
	"reflect"

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
	FormLabel    func(ctx context.Context) string
	FormHelpText func(ctx context.Context) string
	FormWidget   widgets.Widget
	GetDefault   func() interface{}
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
	if i.FormWidget != nil {
		i.FormWidget.Hide(hidden)
		return
	}

	i.FormWidget = i.Widget()
	i.FormWidget.Hide(hidden)
}

func (i *BaseField) SetLabel(label func(ctx context.Context) string) {
	i.FormLabel = label
}

func (i *BaseField) SetDefault(defaultValue func() interface{}) {
	i.GetDefault = defaultValue
}

func (i *BaseField) SetHelpText(helpText func(ctx context.Context) string) {
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

func (i *BaseField) Label(ctx context.Context) string {
	if i.FormLabel != nil {
		return i.FormLabel(ctx)
	}
	if i.Caser == nil {
		var titleCaser = cases.Title(language.English)
		i.Caser = &titleCaser
	}
	return i.Caser.String(i.FieldName)
}

func (i *BaseField) HelpText(ctx context.Context) string {
	if i.FormHelpText != nil {
		return i.FormHelpText(ctx)
	}
	return ""
}

func (i *BaseField) Default() interface{} {
	if i.GetDefault != nil {
		return i.GetDefault()
	}
	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.Func, reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.Invalid:
		return true
	}
	return v.IsZero()
}

func (i *BaseField) HasChanged(initial, data interface{}) bool {
	var rA = reflect.ValueOf(initial)
	var rB = reflect.ValueOf(data)
	if isZero(rA) && isZero(rB) {
		return false
	}

	if isZero(rA) != isZero(rB) {
		return true
	}

	if valuerA, ok := initial.(driver.Valuer); ok {
		var valA, err = valuerA.Value()
		if err != nil {
			return true
		}
		if valuerB, ok := data.(driver.Valuer); ok {
			var valB, err = valuerB.Value()
			if err != nil {
				return true
			}
			return !reflect.DeepEqual(valA, valB)
		}
	}

	if rA.Kind() == reflect.Ptr {
		rA = rA.Elem()
	}
	if rB.Kind() == reflect.Ptr {
		rB = rB.Elem()
	}

	if rA.IsValid() != rB.IsValid() {
		return true
	}

	if !rA.IsValid() && !rB.IsValid() {
		return false
	}

	var aType = rA.Type()
	var bType = rB.Type()
	if aType != bType && !aType.ConvertibleTo(bType) && !bType.ConvertibleTo(aType) {
		return true
	}

	if aType != bType {
		switch {
		case aType.ConvertibleTo(bType):
			rA = rA.Convert(bType)
		case bType.ConvertibleTo(aType):
			rB = rB.Convert(aType)
		}
	}

	var valuerType = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	if rA.Type().Implements(valuerType) && rB.Type().Implements(valuerType) {
		var vA = rA.Interface().(driver.Valuer)
		var vB = rB.Interface().(driver.Valuer)
		var valA, errA = vA.Value()
		var valB, errB = vB.Value()
		if errA != nil || errB != nil {
			return true
		}

		rA = reflect.ValueOf(valA)
		rB = reflect.ValueOf(valB)

		if rA.Kind() == reflect.Ptr {
			rA = rA.Elem()
		}

		if rB.Kind() == reflect.Ptr {
			rB = rB.Elem()
		}
	}

	return !reflect.DeepEqual(rA.Interface(), rB.Interface())
}

func (i *BaseField) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	return value, nil
}

func (i *BaseField) Validate(ctx context.Context, value interface{}) []error {
	var errors = make([]error, 0)
	for _, validator := range i.Validators {
		if err := validator(value); err != nil {
			errors = append(errors, err)
		}
	}

	var widget = i.Widget()
	if widget != nil {
		var errs = widget.Validate(ctx, value)
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
