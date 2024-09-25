package features

import (
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
			"editorjs/js/vendor/tools/text-alignment.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string { return d.Data["text"].(string) }
			return fb
		},
	},
	TuneFunc: tuneBlocks,
}

func tuneBlocks(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
	fb.Attribute("data-text-align", data.(map[string]interface{})["alignment"])
	return fb
}
