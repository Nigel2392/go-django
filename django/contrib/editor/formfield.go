package editor

import (
	"reflect"

	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type EditorJSFormField struct {
	*fields.JSONFormField[EditorJSData]
	widgetOverride func(...string) widgets.Widget
	Features       []string
}

func (e *EditorJSFormField) ValueToForm(value interface{}) interface{} {
	return e.Widget().ValueToForm(value)
}

func (e *EditorJSFormField) ValueToGo(value interface{}) (interface{}, error) {
	return e.Widget().ValueToGo(value)
}

func (e *EditorJSFormField) SetWidget(widget widgets.Widget) {
	e.widgetOverride = func(...string) widgets.Widget {
		return widget
	}
}

func (e *EditorJSFormField) Widget() widgets.Widget {
	if e.widgetOverride != nil {
		return e.widgetOverride(e.Features...)
	}
	return NewEditorJSWidget(e.Features...)
}

func (e *EditorJSFormField) HasChanged(initial, data interface{}) bool {
	return reflect.DeepEqual(initial, data)
}

func EditorJSField(features []string, opts ...func(fields.Field)) fields.Field {
	return &EditorJSFormField{
		JSONFormField: fields.JSONField[EditorJSData](opts...),
		Features:      features,
	}
}
