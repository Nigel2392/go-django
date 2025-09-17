package features

import (
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

func init() {
	editor.Register(TextColorTune)
	editor.Tune("text-color")

	editor.Register(BackgroundColorTune)
	editor.Tune("background-color")
}

var TextColorTune = &BlockTune{
	BaseFeature: BaseFeature{
		Type:          "text-color",
		JSConstructor: "TextColorTune",
		CSSFiles: []string{
			"editorjs/css/color-tune.css",
		},
		JSFiles: []string{
			"editorjs/js/deps/tools/color-tune.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string { return d.Data["text"].(string) }
			return fb
		},
	},
	TuneFunc: colorTune("text-color", nil),
}

var BackgroundColorTune = &BlockTune{
	BaseFeature: BaseFeature{
		Type:          "background-color",
		JSConstructor: "BackgroundColorTune",
		CSSFiles: []string{
			"editorjs/css/color-tune.css",
		},
		JSFiles: []string{
			"editorjs/js/deps/tools/color-tune.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string { return d.Data["text"].(string) }
			return fb
		},
	},
	TuneFunc: colorTune("background-color", func(fb editor.FeatureBlock, data map[string]interface{}) {
		if a, ok := data["stretched"]; ok && !django_reflect.IsZero(a) {
			fb.Class("background-stretched")
			fb.Attribute("data-stretched", a)
		}
	}),
}

func colorTune(cssAttr string, extra func(fb editor.FeatureBlock, data map[string]interface{})) func(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
	return func(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
		var (
			d  map[string]interface{}
			a  interface{}
			ok bool
		)
		if d, ok = data.(map[string]interface{}); !ok {
			return fb
		}
		if a, ok = d["color"]; ok {
			fb.Class(cssAttr)
			fb.Attribute("data-"+cssAttr, a)
		}
		if extra != nil {
			extra(fb, d)
		}
		return fb
	}
}
