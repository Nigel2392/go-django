package editor

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ fields.Field = (*EditorJSFormField)(nil)

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

func (e *EditorJSFormField) Validate(ctx context.Context, value interface{}) []error {
	var widget = e.Widget()
	return widget.Validate(ctx, value)
}

func (e *EditorJSFormField) HasChanged(initial, data interface{}) bool {
	initialV, ok1 := initial.(*EditorJSBlockData)
	dataV, ok2 := data.(*EditorJSBlockData)
	if !ok1 || !ok2 {
		logger.Warnf("EditorJSFormField HasChanged: both initial and data are not *EditorJSBlockData (initial: %T, data: %T)", initial, data)
		return false
	}
	if initialV == nil && dataV == nil {
		logger.Warnf("EditorJSFormField HasChanged: both initial and data are nil")
		return false
	}
	if initialV == nil || dataV == nil {
		logger.Errorf("EditorJSFormField HasChanged: one of initial or data is nil (initial: %v, data: %v)", initialV, dataV)
		return true
	}

	// Time check should be removed.
	// Time will be updated on every form save, thus not being reliable.
	// if initialV.Time != dataV.Time {
	// return true
	// }

	if initialV.Version != dataV.Version {
		return true
	}
	if len(initialV.Blocks) != len(dataV.Blocks) {
		return true
	}
	for i, block := range initialV.Blocks {
		if !reflect.DeepEqual(block.Data(), dataV.Blocks[i].Data()) {
			return true
		}
	}
	return false
}

func EditorJSField(features []string, opts ...func(fields.Field)) fields.Field {
	return &EditorJSFormField{
		JSONFormField: fields.JSONField[EditorJSData](opts...),
		Features:      features,
	}
}
