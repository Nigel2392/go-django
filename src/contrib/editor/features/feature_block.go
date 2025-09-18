package features

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/a-h/templ"
)

var (
	_ editor.FeatureBlock = (*FeatureBlock)(nil)

	ErrRenderNotImplemented = errors.New("feature does not implement RenderBlock")
)

type BlockRenderer interface {
	RenderBlock(fb editor.FeatureBlock, c context.Context, w io.Writer) error
}

type WrapperBlock struct {
	editor.FeatureBlock
	Wrap func(editor.FeatureBlock) func(context.Context, io.Writer) error
}

func (w *WrapperBlock) Render(ctx context.Context, wr io.Writer) error {
	if w.Wrap != nil {
		var component = w.Wrap(w.FeatureBlock)
		return component(ctx, wr)
	}
	return fmt.Errorf(
		"feature '%s' (%T) does not implement RenderBlock: %w",
		w.Type(),
		w.Feature(),
		ErrRenderNotImplemented,
	)
}

type AttributeWrapperBlock struct {
	editor.FeatureBlock
	Attrs   map[string]interface{}
	Classes []string
}

func (a *AttributeWrapperBlock) Attribute(key string, value interface{}) {
	if a.Attrs == nil {
		a.Attrs = make(map[string]interface{})
	}
	a.Attrs[key] = value
}

func (a *AttributeWrapperBlock) Attributes() map[string]interface{} {
	return a.Attrs
}

func (a *AttributeWrapperBlock) Render(ctx context.Context, w io.Writer) error {
	if a.FeatureBlock == nil {
		return fmt.Errorf("AttributeWrapperBlock has no FeatureBlock")
	}

	var atts = make(map[string]interface{}, len(a.Attrs)+1)
	for k, v := range a.Attrs {
		atts[k] = v
	}
	if len(a.Classes) > 0 {
		var classList = atts["class"]
		if classListStr, ok := classList.(string); ok && classListStr != "" {
			classListStr = classListStr + " " + strings.Join(a.Classes, " ")
			atts["class"] = classListStr
		} else {
			atts["class"] = strings.Join(a.Classes, " ")
		}
	}

	fmt.Fprintf(w, "<div ")
	err := templ.RenderAttributes(ctx, w, templ.Attributes(atts))
	if err != nil {
		return err
	}
	fmt.Fprint(w, ">")
	err = a.FeatureBlock.Render(ctx, w)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "</div>")
	return nil
}

type FeatureBlock struct {
	Attrs      map[string]interface{}
	Classes    []string
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

func (b *FeatureBlock) Class(key string) {
	if b.Classes == nil {
		b.Classes = make([]string, 0)
	}
	b.Classes = append(b.Classes, key)
}

func (b *FeatureBlock) ClassName() string {
	return strings.Join(b.Classes, " ")
}

func (b *FeatureBlock) Render(ctx context.Context, w io.Writer) error {
	if r, ok := b.FeatureObject.(BlockRenderer); ok {
		return r.RenderBlock(b, ctx, w)
	}
	return fmt.Errorf(
		"feature '%s' (%T) does not implement RenderBlock: %w",
		b.FeatureName,
		b.FeatureObject,
		ErrRenderNotImplemented,
	)
}

func (b *FeatureBlock) Data() editor.BlockData {
	return b.FeatureData
}
