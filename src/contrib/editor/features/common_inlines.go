package features

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/contrib/editor"
	"golang.org/x/net/html"
)

func init() {
	editor.Register(MarkerFeature)
	editor.Register(InlineCodeFeature)
	editor.Register(UnderlineFeature)
	editor.Register(StrikeThroughFeature)
	editor.Register(HotkeyFeature)
}

var MarkerFeature = &InlineFeature{
	TagName:    "mark",
	Class:      "cdx-marker",
	Attributes: []InlineFeatureAttribute{},
	BaseFeature: BaseFeature{
		Type:          "marker",
		JSConstructor: "Marker",
		JSFiles: []string{
			"editorjs/js/deps/tools/marker.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
	},
	RebuildElementsFn: func(li []*InlineFeatureElement) error {
		for _, el := range li {
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "marker",
				},
			}
		}

		return nil
	},
}

var InlineCodeFeature = &InlineFeature{
	TagName:    "span",
	Class:      "inline-code",
	Attributes: []InlineFeatureAttribute{},
	BaseFeature: BaseFeature{
		Type:          "inline-code",
		JSConstructor: "InlineCode",
		JSFiles: []string{
			"editorjs/js/deps/tools/inline-code.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
	},
	RebuildElementsFn: func(li []*InlineFeatureElement) error {
		for _, el := range li {
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "inline-code",
				},
			}
		}

		return nil
	},
}

var UnderlineFeature = &InlineFeature{
	TagName:    "u",
	Class:      "cdx-underline",
	Attributes: []InlineFeatureAttribute{},
	BaseFeature: BaseFeature{
		Type:          "underline",
		JSConstructor: "Underline",
		JSFiles: []string{
			"editorjs/js/deps/tools/underline.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
	},
	RebuildElementsFn: func(li []*InlineFeatureElement) error {
		for _, el := range li {
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "underline",
				},
			}
		}

		return nil
	},
}

var StrikeThroughFeature = &InlineFeature{
	TagName:    "s",
	Class:      "cdx-strikethrough",
	Attributes: []InlineFeatureAttribute{},
	BaseFeature: BaseFeature{
		Type:          "strikethrough",
		JSConstructor: "Strikethrough",
		JSFiles: []string{
			"editorjs/js/deps/tools/strikethrough.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
	},
	RebuildElementsFn: func(li []*InlineFeatureElement) error {
		for _, el := range li {
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "strikethrough",
				},
			}
		}

		return nil
	},
}

var HotkeyFeature = &InlineFeature{
	TagName:    "kbd",
	Class:      "editorjs-inline-hotkey",
	Attributes: []InlineFeatureAttribute{},
	BaseFeature: BaseFeature{
		Type:          "hotkey",
		JSConstructor: "EditorJSInlineHotkey",
		JSFiles: []string{
			"editorjs/js/deps/tools/hotkey.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
	},
	RebuildElementsFn: func(li []*InlineFeatureElement) error {
		for _, el := range li {
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "hotkey",
				},
			}
		}

		return nil
	},
}
