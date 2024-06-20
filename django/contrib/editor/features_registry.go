package editor

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/elliotchance/orderedmap/v2"
)

type BlockData struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Tunes map[string]interface{} `json:"tunes,omitempty"`
}

type EditorJSData struct {
	Time    int64       `json:"time"`
	Blocks  []BlockData `json:"blocks"`
	Version string      `json:"version"`
}

type EditorJSBlockData struct {
	Time     int64          `json:"time"`
	Blocks   []FeatureBlock `json:"blocks"`
	Version  string         `json:"version"`
	Features []BaseFeature  `json:"-"`
}

func (e *EditorJSBlockData) String() string {
	var b = new(strings.Builder)
	for _, block := range e.Blocks {
		fmt.Fprintf(b, "%s\n", block)
	}
	return b.String()
}

func (e *EditorJSBlockData) Render() template.HTML {
	var ctx = context.Background()
	var b = new(strings.Builder)
	for _, block := range e.Blocks {
		if err := block.Render(ctx, b); err != nil {
			fmt.Fprintf(b, "Error (%s): %s", block.Type(), err)
		}
	}
	return template.HTML(b.String())

}

type editorRegistry struct {
	features *orderedmap.OrderedMap[string, BaseFeature]
	ft_tunes map[string][]string
	tunes    []string
}

func init() {
	staticfiles.AddFS(
		editorJS_FS,
		tpl.MatchAnd(
			tpl.MatchPrefix("editorjs"),
			tpl.MatchOr(
				tpl.MatchExt(".js"),
			),
		),
	)
}

func newEditorRegistry() *editorRegistry {
	return &editorRegistry{
		features: orderedmap.NewOrderedMap[string, BaseFeature](),
		ft_tunes: make(map[string][]string),
		tunes:    make([]string, 0),
	}
}

func (e *editorRegistry) Features(f ...string) []BaseFeature {
	if len(f) == 0 {
		f = e.features.Keys()
	}

	var features []BaseFeature = make([]BaseFeature, 0, len(f))
	for _, name := range f {
		if feature, ok := e.features.Get(name); ok {
			features = append(features, feature)
		}
	}

	return features
}

func (e *editorRegistry) Register(feature BaseFeature) {
	e.features.Set(feature.Name(), feature)
}

func (e *editorRegistry) TuneFeature(featureName string, tuneName string) {
	var tunes, ok = e.ft_tunes[featureName]
	if !ok {
		tunes = make([]string, 0)
	}
	tunes = append(tunes, tuneName)
	e.ft_tunes[featureName] = tunes
}

func (e *editorRegistry) Tune(tuneName string) {
	e.tunes = append(e.tunes, tuneName)
}

func (e *editorRegistry) BuildConfig(widgetContext ctx.Context, features ...string) map[string]interface{} {
	var toolsConfig = make(map[string]interface{})
	for _, f := range e.Features(features...) {
		var featureCfg = f.Config(widgetContext)
		var jsClass = f.Constructor()
		var fullCfg = map[string]interface{}{
			"class": jsClass,
		}
		if len(featureCfg) > 0 {
			fullCfg["config"] = featureCfg
		}
		if tunes, ok := e.ft_tunes[f.Name()]; ok {
			fullCfg["tunes"] = tunes
		}
		toolsConfig[f.Name()] = fullCfg
	}

	var config = map[string]interface{}{
		"tools": toolsConfig,
	}

	if len(e.tunes) > 0 {
		config["tunes"] = e.tunes
	}

	return config
}

func (e *editorRegistry) ValueToGo(tools []string, data EditorJSData) (*EditorJSBlockData, error) {
	var blocks = data.Blocks
	var blockData = &EditorJSBlockData{
		Time:     data.Time,
		Version:  data.Version,
		Features: e.Features(tools...),
	}

	var blockList = make([]FeatureBlock, 0, len(blocks))
	for _, block := range blocks {

		feature, ok := e.features.Get(block.Type)
		if !ok {
			continue
		}

		var b FeatureBlockRenderer
		if b, ok = feature.(FeatureBlockRenderer); !ok {
			return blockData, fmt.Errorf("feature %q marked as feature but does not implement FeatureBlockRenderer", block.Type)
		}

		var blockObj = b.Render(block)
		for k, v := range block.Tunes {
			var tuneFeature, ok = e.features.Get(k)
			if !ok {
				continue
			}

			var tuneFeatureBlock BlockTuneFeature
			if tuneFeatureBlock, ok = tuneFeature.(BlockTuneFeature); !ok {
				return blockData, fmt.Errorf("feature %q marked as tune but does not implement BlockTuneFeature", k)
			}

			blockObj = tuneFeatureBlock.Tune(blockObj, v)
		}

		blockList = append(blockList, blockObj)
	}

	blockData.Blocks = blockList
	return blockData, nil
}

var (
	EditorRegistry = newEditorRegistry()
	ValueToGo      = EditorRegistry.ValueToGo
	Features       = EditorRegistry.Features
	Register       = EditorRegistry.Register
	TuneFeature    = EditorRegistry.TuneFeature
	Tune           = EditorRegistry.Tune
	buildConfig    = EditorRegistry.BuildConfig
)
