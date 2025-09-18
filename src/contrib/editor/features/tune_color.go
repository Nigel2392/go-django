package features

import (
	"fmt"

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
	TuneFunc: colorTune("text-color"),
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
	TuneFunc: colorTune("background-color"),
}

func colorTune(cssAttr string) func(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
	return func(fb editor.FeatureBlock, data interface{}) editor.FeatureBlock {
		var (
			d  map[string]interface{}
			a  interface{}
			ok bool
		)
		if d, ok = data.(map[string]interface{}); !ok {
			return fb
		}

		if a, ok = d["color"]; !ok || django_reflect.IsZero(a) {
			return fb
		}

		wrapped, ok := fb.(*AttributeWrapperBlock)
		if !ok {
			wrapped = &AttributeWrapperBlock{
				FeatureBlock: fb,
				Attrs:        make(map[string]interface{}),
				Classes:      make([]string, 0, 1),
			}
		}

		wrapped.Classes = append(wrapped.Classes, cssAttr)

		if stretched, _ := d["stretched"].(bool); stretched {
			wrapped.Classes = append(wrapped.Classes, "background-stretched")
		}

		var atts = wrapped.Attributes()
		var style string
		if atts != nil {
			style, _ = atts["style"].(string)
		}

		style += fmt.Sprintf(
			"--%s: %s;", cssAttr, a,
		)

		wrapped.Attrs["style"] = style

		return wrapped
	}
}
