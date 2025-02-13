package editor

import (
	"context"
	"io"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/PuerkitoBio/goquery"
)

type FeatureBlock interface {
	ID() string
	Type() string
	Feature() BaseFeature
	Render(ctx context.Context, w io.Writer) error
	Attribute(key string, value any)
	Attributes() map[string]interface{}
	Class(key string)
	ClassName() string
	Data() BlockData
}

type BaseFeature interface {
	// Name returns the name of the feature.
	Name() string

	// Config returns the configuration of the feature.
	Config(widgetContext ctx.Context) map[string]interface{}

	// OnRegister is called when the feature is registered.
	//
	// It is allowed to add custom routes or do other setup here.
	OnRegister(django.Mux) error

	// OnValidate is called when the feature is validated.
	OnValidate(BlockData) error

	// Constructor returns the JS class name of the feature.
	Constructor() string

	// Media return's the feature's static / media files.
	Media() media.Media
}

// FeatureBlockRenderer is a feature that can render a block.
//
// This is used to render the block after it has been converted from
// JSON to Go.
//
// The render method should return an object based on the provided data.
// This object will be used to render the HTML.
type FeatureBlockRenderer interface {
	BaseFeature
	// Render should return a new block object that can be used to render
	// the HTML.
	Render(BlockData) FeatureBlock
}

// BlockTuneFeature is a feature that can tune a block.
//
// This is used to tune the block after it has been converted from
// JSON to Go.
//
// The tune method should return a new block, or the same block if no
// changes were made.
//
// The tune object should wrap the provided block or make changes to it.
type BlockTuneFeature interface {
	BaseFeature
	Tune(FeatureBlock, interface{}) FeatureBlock
}

type InlineFeature interface {
	BaseFeature
	ParseInlineData(soup *goquery.Document) error
}
