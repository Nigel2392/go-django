package editor_test

import (
	"context"
	"io"

	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/media"
)

type MockFeatureBlock struct {
	IDVal   string
	TypeVal string
}

func (m MockFeatureBlock) ID() string {
	return m.IDVal
}

func (m MockFeatureBlock) Type() string {
	return m.TypeVal
}

func (m MockFeatureBlock) Feature() editor.BaseFeature {
	return MockBaseFeature{}
}

func (m MockFeatureBlock) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.TypeVal))
	return err
}

func (m MockFeatureBlock) Attribute(key string, value any) {}

func (m MockFeatureBlock) Attributes() map[string]interface{} {
	return map[string]interface{}{}
}

func (m MockFeatureBlock) Data() editor.BlockData {
	return editor.BlockData{
		ID:    m.IDVal,
		Type:  m.TypeVal,
		Data:  map[string]interface{}{},
		Tunes: map[string]interface{}{},
	}
}

type MockBaseFeature struct {
	NameVal string
}

func (m MockBaseFeature) Name() string {
	return m.NameVal
}

func (m MockBaseFeature) Config(widgetContext ctx.Context) map[string]interface{} {
	return map[string]interface{}{}
}

func (m MockBaseFeature) Constructor() string {
	return "MockConstructor"
}

func (m MockBaseFeature) Media() media.Media {
	return media.NewMedia()
}
