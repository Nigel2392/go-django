package editor

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/media"
)

type FeatureBlock interface {
	json.Marshaler
	ID() string
	Type() string
	Feature() BaseFeature
	Render(ctx context.Context, w io.Writer) error
	Data() map[string]interface{}
}

type BaseFeature interface {
	// Name returns the name of the feature.
	Name() string

	// Config returns the configuration of the feature.
	Config(widgetContext ctx.Context) map[string]interface{}

	// Constructor returns the JS class name of the feature.
	Constructor() string

	// Media return's the feature's static / media files.
	Media() media.Media

	json.Unmarshaler
	json.Marshaler
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
	// Render returns the rendered HTML of the feature.
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
}
