package editor

import (
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type EditorJSFormField struct {
	*fields.JSONFormField[EditorJSData]
	Features []string
}

func (e *EditorJSFormField) ValueToForm(value interface{}) interface{} {
	return e.Widget().ValueToForm(value)
}

func (e *EditorJSFormField) ValueToGo(value interface{}) (interface{}, error) {
	return e.Widget().ValueToGo(value)
}

func (e *EditorJSFormField) Widget() widgets.Widget {
	return NewEditorJSWidget(e.Features...)
}

func EditorJSField(features []string, opts ...func(fields.Field)) fields.Field {
	return &EditorJSFormField{
		JSONFormField: fields.JSONField[EditorJSData](opts...),
		Features:      features,
	}
}
