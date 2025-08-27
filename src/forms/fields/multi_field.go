package fields

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type MultiField struct {
	*BaseField
	Fields []Field
}

func NewMultiField(fields []Field, opts ...func(Field)) *MultiField {
	var w = &MultiField{
		BaseField: NewField(),
		Fields:    fields,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

func (m *MultiField) SetRequired(b bool) {
	m.Required_ = b
	for _, field := range m.Fields {
		field.SetRequired(b)
	}
}

func (m *MultiField) SetReadOnly(b bool) {
	m.ReadOnly_ = b
	for _, field := range m.Fields {
		field.SetReadOnly(b)
	}
}

func (m *MultiField) Hide(hidden bool) {
	m.FormWidget.Hide(hidden)
	for _, field := range m.Fields {
		field.Hide(hidden)
	}
}

func (m *MultiField) IsEmpty(value interface{}) bool {
	var valMap, ok = value.(map[string]interface{})
	if !ok {
		return true
	}

	for _, field := range m.Fields {
		if !field.IsEmpty(valMap[field.Name()]) {
			return false
		}
	}

	return true
}

func (m *MultiField) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	var cleaned = make(map[string]interface{})
	var valMap, ok = value.(map[string]interface{})
	if !ok {
		return nil, errors.TypeMismatch.Wrapf(
			"expected map[string]interface{} when cleaning, got %T",
			value,
		)
	}

	for _, field := range m.Fields {
		var fieldValue, err = field.Clean(ctx, valMap[field.Name()])
		if err != nil {
			return nil, err
		}
		cleaned[field.Name()] = fieldValue
	}

	return cleaned, nil
}

func (m *MultiField) Validate(ctx context.Context, value interface{}) []error {

	if errs := m.BaseField.Validate(ctx, value); len(errs) > 0 {
		return errs
	}

	if value == nil {
		return []error{}
	}

	var errs []error
	var valMap, ok = value.(map[string]interface{})
	if !ok {
		return []error{errors.TypeMismatch.Wrapf(
			"expected map[string]interface{} when validating, got %T",
			value,
		)}
	}

	for _, field := range m.Fields {
		var fieldValue = valMap[field.Name()]
		var fieldErrs = field.Validate(ctx, fieldValue)
		errs = append(errs, fieldErrs...)
	}

	return errs
}

func (m *MultiField) HasChanged(initial interface{}, data interface{}) bool {
	var initialMap, ok1 = initial.(map[string]interface{})
	var dataMap, ok2 = data.(map[string]interface{})
	if !ok1 || !ok2 {
		return true
	}

	for _, field := range m.Fields {
		var initialValue = initialMap[field.Name()]
		var dataValue = dataMap[field.Name()]
		if field.HasChanged(initialValue, dataValue) {
			return true
		}
	}

	return false
}

func (m *MultiField) Widget() widgets.Widget {
	if m.FormWidget != nil {
		return m.FormWidget
	}

	var multi = widgets.NewMultiWidget(m.Attrs())
	for _, field := range m.Fields {
		multi.AddWidget(field.Name(), field.Widget())
	}
	return multi
}
