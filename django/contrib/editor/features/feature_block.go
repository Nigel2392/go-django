package features

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Nigel2392/django/contrib/editor"
)

var (
	_ editor.FeatureBlock = (*FeatureBlock)(nil)

	ErrRenderNotImplemented = errors.New("feature does not implement RenderBlock")
)

type BlockRenderer interface {
	RenderBlock(fb editor.FeatureBlock, c context.Context, w io.Writer) error
}

type FeatureBlock struct {
	Attrs      map[string]interface{}
	Identifier string

	FeatureData   editor.BlockData
	FeatureObject editor.BaseFeature
	FeatureName   string
	GetString     func(editor.BlockData) string
}

func (b *FeatureBlock) ID() string {
	return b.Identifier
}

func (b *FeatureBlock) Type() string {
	return b.FeatureName
}

func (b *FeatureBlock) Feature() editor.BaseFeature {
	return b.FeatureObject
}

func (b *FeatureBlock) String() string {
	if b.GetString != nil {
		return b.GetString(b.FeatureData)
	}
	return b.FeatureName
}

func (b *FeatureBlock) Attribute(key string, value interface{}) {
	if b.Attrs == nil {
		b.Attrs = make(map[string]interface{})
	}
	b.Attrs[key] = value
}

func (b *FeatureBlock) Attributes() map[string]interface{} {
	return b.Attrs
}

func (b *FeatureBlock) Render(ctx context.Context, w io.Writer) error {
	fmt.Println("FeatureBlock.Render", b.FeatureName)
	if r, ok := b.FeatureObject.(BlockRenderer); ok {
		return r.RenderBlock(b, ctx, w)
	}
	return fmt.Errorf(
		"feature '%s' (%T) does not implement RenderBlock %w",
		b.FeatureName,
		b.FeatureObject,
		ErrRenderNotImplemented,
	)
}

func (b *FeatureBlock) Data() editor.BlockData {
	return b.FeatureData
}
