package editor

import (
	"context"
	"database/sql/driver"
	"fmt"
	"html/template"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/PuerkitoBio/goquery"
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

// The main datatype to use in structs.
//
// This will automatically assign the appropriate form widget for the field.
//
// It will also allow for simple rendering of the field's data by calling the `Render()` method.
//
// If the struct which the field belongs to defines a `Get<FieldName>Features() []string` method,
// then these features will be used to build the editorjs widget.
type EditorJSBlockData struct {
	Time     int64          `json:"time"`
	Blocks   []FeatureBlock `json:"blocks"`
	Version  string         `json:"version"`
	Features []BaseFeature  `json:"-"`
}

func (e *EditorJSBlockData) DBType() dbtype.Type {
	return dbtype.JSON
}

func (e *EditorJSBlockData) Value() (driver.Value, error) {
	return JSONMarshalEditorData(e)
}

func (e *EditorJSBlockData) Scan(src interface{}) error {
	var features = make([]string, len(e.Features))
	for i, f := range e.Features {
		features[i] = f.Name()
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
	return _JSONUnmarshalEditorData(
		e, features, b,
	)
}

func (e *EditorJSBlockData) String() string {
	var b = new(strings.Builder)
	for _, block := range e.Blocks {
		fmt.Fprintf(b, "%s\n", block)
	}
	return b.String()
}

func (e *EditorJSBlockData) Render() (template.HTML, error) {
	var ctx = context.Background()
	var b = new(strings.Builder)
	var prefetchableBlocks = make(map[string]PrefetchableFeature)
	for _, feature := range e.Features {
		if prefetchable, ok := feature.(PrefetchableFeature); ok {
			prefetchableBlocks[feature.Name()] = prefetchable
		}
	}

	var dataForPrefetch = make(map[string][]BlockData)
	var prefetchedData = make(map[string]map[string]BlockData)
	if len(prefetchableBlocks) > 0 {
		for _, block := range e.Blocks {
			if _, ok := prefetchableBlocks[block.Type()]; ok {
				dataForPrefetch[block.Type()] = append(dataForPrefetch[block.Type()], block.Data())
			} else {
				dataForPrefetch[block.Type()] = []BlockData{block.Data()}
			}
		}

		for name, feature := range prefetchableBlocks {
			data, err := feature.PrefetchData(ctx, dataForPrefetch[name])
			if err != nil {
				return "", err
			}

			prefetchedData[name] = data
		}
	}

	for _, block := range e.Blocks {
		if dataMap, ok := prefetchedData[block.Type()]; ok {
			if data, ok := dataMap[block.ID()]; ok {
				prefetchableBlock, ok := block.(PrefetchableFeatureBlock)
				if !ok {
					panic(fmt.Sprintf(
						"block %q is not a PrefetchableFeatureBlock",
						block.ID(),
					))
				}
				prefetchableBlock.WithData(ctx, data)
			}
		}

		if err := block.Render(ctx, b); err != nil && RENDER_ERRORS {
			fmt.Fprintf(b, "Error (%s): %s", block.Type(), err)
		}
	}

	var goQueryDocument, err = goquery.NewDocumentFromReader(
		strings.NewReader(b.String()),
	)
	if err != nil {
		return "", err
	}

	var goQuerySelection = goQueryDocument.Find("body")
	var inlines = make([]InlineFeature, 0)
	for _, feature := range e.Features {
		if inline, ok := feature.(InlineFeature); ok {
			inlines = append(inlines, inline)
		}
	}

	for _, inline := range inlines {
		err = inline.ParseInlineData(goQuerySelection)
		if err != nil {
			return "", err
		}
	}

	html, err := goQuerySelection.Html()
	return template.HTML(html), err
}

func (e *EditorJSBlockData) MustRender() template.HTML {
	html, err := e.Render()
	if err != nil {
		logger.Errorf("Error rendering editorjs block data: %v", err)
		return ""
	}
	return html
}

func FeatureNames(f ...BaseFeature) []string {
	var names = make([]string, 0, len(f))
	for _, feature := range f {
		names = append(names, feature.Name())
	}
	return names
}

type editorRegistry struct {
	features *orderedmap.OrderedMap[string, BaseFeature]
	ft_tunes map[string][]string
	tunes    []string
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
		} else {
			logger.Infof("Feature %q not found in feature registry", name)
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
	var featuresMap = make(map[string]BaseFeature)
	var featuresList = e.Features(features...)
	for _, f := range featuresList {
		featuresMap[f.Name()] = f
	}

	var toolsConfig = make(map[string]interface{})
	for _, f := range featuresList {
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
		var tunes = make([]string, 0, len(e.tunes))
		for _, tune := range e.tunes {
			var _, ok = featuresMap[tune]
			if !ok {
				continue
			}
			tunes = append(tunes, tune)
		}
		config["tunes"] = tunes
	}

	return config
}

func (e *editorRegistry) ValueToForm(data *EditorJSBlockData) *EditorJSData {
	var blocks = data.Blocks
	var blockData = &EditorJSData{
		Time:    data.Time,
		Version: data.Version,
	}

	var blockList = make([]BlockData, 0, len(blocks))
	for _, block := range blocks {
		if block == nil {
			continue
		}
		var _, ok = e.features.Get(block.Type())
		if !ok {
			continue
		}

		blockList = append(blockList, block.Data())
	}

	blockData.Blocks = blockList
	return blockData
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

			var err = tuneFeatureBlock.OnValidate(block)
			if err != nil {
				return blockData, err
			}

			blockObj = tuneFeatureBlock.Tune(blockObj, v)
		}

		blockList = append(blockList, blockObj)
	}

	blockData.Blocks = blockList
	return blockData, nil
}

func (e *editorRegistry) Validate(tools []string, data EditorJSData) []error {
	var blocks = data.Blocks
	var errs []error
	for _, block := range blocks {
		feature, ok := e.features.Get(block.Type)
		if !ok {
			continue
		}

		if err := feature.OnValidate(block); err != nil {
			errs = append(errs, err)
		}

		for k := range block.Tunes {
			var tuneFeature, ok = e.features.Get(k)
			if !ok {
				continue
			}

			var tuneFeatureBlock BlockTuneFeature
			if tuneFeatureBlock, ok = tuneFeature.(BlockTuneFeature); !ok {
				errs = append(errs, fmt.Errorf(
					"feature %q marked as tune but does not implement BlockTuneFeature", k,
				))
			}

			var err = tuneFeatureBlock.OnValidate(block)
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"tune feature %q validation failed: %w", k, err,
				))
			}
		}
	}
	return errs
}

var (
	EditorRegistry = newEditorRegistry()
	ValueToForm    = EditorRegistry.ValueToForm
	ValueToGo      = EditorRegistry.ValueToGo
	Features       = EditorRegistry.Features
	Register       = EditorRegistry.Register
	TuneFeature    = EditorRegistry.TuneFeature
	Tune           = EditorRegistry.Tune
	buildConfig    = EditorRegistry.BuildConfig
)
