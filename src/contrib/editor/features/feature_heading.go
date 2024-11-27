package features

import (
	"context"
	"fmt"
	"io"

	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/a-h/templ"
)

func init() {
	editor.Register(HeadingFeature)
}

var HeadingFeature = &Block{
	BaseFeature: BaseFeature{
		Type:          "header",
		JSConstructor: "Header",
		JSFiles: []string{
			"editorjs/js/deps/tools/header.js",
		},
		Build: func(fb *FeatureBlock) *FeatureBlock {
			fb.GetString = func(d editor.BlockData) string { return d.Data["text"].(string) }
			return fb
		},
	},
	RenderFunc: renderHeading,
}

func renderHeading(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var headingLevel = fb.Data().Data["level"]
	if headingLevel == nil || headingLevel == 0.0 {
		headingLevel = 1.0
	}
	var tag = fmt.Sprintf("h%.0f", headingLevel)
	fmt.Fprintf(w, "<%s data-block-id=\"%s\"", tag, fb.ID())
	if err := templ.RenderAttributes(c, w, templ.Attributes(fb.Attributes())); err != nil {
		return err
	}
	fmt.Fprintf(w, ">%s</%s>", fb.Data().Data["text"], tag)
	return nil
}
