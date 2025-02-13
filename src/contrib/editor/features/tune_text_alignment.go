package features

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/contrib/editor"
)

func init() {
	editor.Register(AlignmentBlockTune)
	editor.Tune("text-align")
}

var AlignmentBlockTune = &BlockTune{
	BaseFeature: BaseFeature{
		Type:          "text-align",
		JSConstructor: "AlignmentBlockTune",
		JSFiles: []string{
			"editorjs/js/deps/tools/text-alignment.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string { return d.Data["text"].(string) }
			return fb
		},
	},
	TuneFunc: tuneBlocks,
}

func tuneBlocks(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
	var (
		d  map[string]interface{}
		a  interface{}
		ok bool
	)
	if d, ok = data.(map[string]interface{}); !ok {
		return fb
	}
	if a, ok = d["alignment"]; ok {
		fb.Class(fmt.Sprintf("text-align-%s", a))
		fb.Attribute("data-text-align", a)
	}
	return fb
}
