package features

import (
	"context"
	"io"
	"maps"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/mux"
	"github.com/PuerkitoBio/goquery"
	"github.com/a-h/templ"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

var _ editor.BaseFeature = (*BaseFeature)(nil)
var _ editor.BlockTuneFeature = (*BlockTune)(nil)
var _ editor.FeatureBlockRenderer = (*BaseFeature)(nil)

type BaseFeature struct {
	Type          string
	Extra         map[string]interface{}
	JSConstructor string
	JSFiles       []string
	CSSFles       []string
	Validate      func(editor.BlockData) error
	Build         func(*FeatureBlock) *FeatureBlock
	Register      func(mux.Multiplexer)
}

// Name returns the name of the feature.
func (b *BaseFeature) Name() string {
	return b.Type
}

// OnRegister is called when the feature is registered.
//
// It is allowed to add custom routes or do other setup here.
func (b *BaseFeature) OnRegister(m mux.Multiplexer) error {
	if b.Register != nil {
		b.Register(m)
	}
	return nil
}

// OnValidate is called when the feature is validated.
func (b *BaseFeature) OnValidate(data editor.BlockData) error {
	if b.Validate != nil {
		return b.Validate(data)
	}
	return nil
}

// Config returns the configuration of the feature.
func (b *BaseFeature) Config(widgetContext ctx.Context) map[string]interface{} {
	var config = make(map[string]interface{})
	maps.Copy(config, b.Extra)
	return config
}

// Constructor returns the JS class name of the feature.
func (b *BaseFeature) Constructor() string {
	return b.JSConstructor
}

// Media return's the feature's static / media files.
func (b *BaseFeature) Media() media.Media {
	var m = media.NewMedia()
	for _, js := range b.JSFiles {
		m.AddJS(&media.JSAsset{URL: django.Static(js)})
	}
	for _, css := range b.CSSFles {
		m.AddCSS(media.CSS(
			django.Static(css),
		))
	}
	return m
}

// Render returns the feature block.
func (b *BaseFeature) Render(d editor.BlockData) editor.FeatureBlock {
	var block = &FeatureBlock{
		FeatureObject: b,
		FeatureData:   d,
		FeatureName:   d.Type,
		Identifier:    d.ID,
	}
	if b.Build != nil {
		block = b.Build(block)
	}
	return block
}

type Block struct {
	BaseFeature
	RenderFunc func(b editor.FeatureBlock, c context.Context, w io.Writer) error
}

func (b *Block) RenderBlock(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	if b.RenderFunc != nil {
		return b.RenderFunc(fb, c, w)
	}
	return ErrRenderNotImplemented
}

func (b *Block) Render(d editor.BlockData) editor.FeatureBlock {
	var block = b.BaseFeature.Render(d).(*FeatureBlock)
	block.FeatureObject = b
	return block
}

type BlockTune struct {
	BaseFeature
	TuneFunc func(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock
}

func (b *BlockTune) Tune(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
	if b.TuneFunc != nil {
		return b.TuneFunc(fb, data)
	}
	return fb
}

type WrapperTune struct {
	BaseFeature
	Wrap func(editor.FeatureBlock) func(context.Context, io.Writer) error
}

func (b *WrapperTune) Render(d editor.BlockData) editor.FeatureBlock {
	var block = b.BaseFeature.Render(d).(*FeatureBlock)
	block.FeatureObject = b
	return &WrapperBlock{
		FeatureBlock: block,
		Wrap:         b.Wrap,
	}
}

type InlineFeatureAttribute struct {
	Required bool    // if true, the attribute must be present
	Name     string  // the name of the attribute to search for
	Value    *string // nil means any value is allowed, or that just the presence of the attribute is enough
}

type InlineFeatureElement struct {
	Node *html.Node
	Data map[string]*string // the data of the element, equivalent to the parsed data of InlineFeatureAttribute
}

type InlineFeature struct {
	BaseFeature
	TagName           string
	Class             string
	Attributes        []InlineFeatureAttribute
	RebuildElementsFn func([]*InlineFeatureElement) error
	RebuildElementFn  func(*InlineFeatureElement) error
}

func (b *InlineFeature) ParseInlineData(doc *goquery.Selection) error {
	var elements = doc.Find(b.TagName).FilterFunction(func(i int, s *goquery.Selection) bool {
		if b.Class != "" && !s.HasClass(b.Class) {
			return false
		}

		for _, attr := range b.Attributes {
			var value, exists = s.Attr(attr.Name)
			if !exists && attr.Required {
				return false
			}

			if attr.Value != nil && value != *attr.Value {
				return false
			}
		}
		return true
	})

	if elements.Length() == 0 {
		return nil
	}

	var matches = make([]*InlineFeatureElement, 0)
	elements.Each(func(i int, s *goquery.Selection) {
		var node = s.Get(0)
		var data = make(map[string]*string)
		for _, attr := range b.Attributes {
			if value, exists := s.Attr(attr.Name); exists {
				data[attr.Name] = &value
			} else {
				data[attr.Name] = nil
			}
		}

		matches = append(matches, &InlineFeatureElement{
			Node: node,
			Data: data,
		})
	})

	return b.RebuildElements(matches)
}

func (b *InlineFeature) RebuildElements(elements []*InlineFeatureElement) error {
	if b.RebuildElementsFn != nil {
		return b.RebuildElementsFn(elements)
	}

	if b.RebuildElementFn == nil {
		return errors.Wrap(
			errs.ErrNotImplemented,
			"RebuildElementFn is not implemented",
		)
	}

	for _, element := range elements {
		if err := b.RebuildElementFn(element); err != nil {
			return err
		}
	}
	return nil
}

func templRender(fn func(fb editor.FeatureBlock, c context.Context, w io.Writer) templ.Component) func(b editor.FeatureBlock, c context.Context, w io.Writer) error {
	return func(b editor.FeatureBlock, c context.Context, w io.Writer) error {
		return fn(b, c, w).Render(c, w)
	}
}
