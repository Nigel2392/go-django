package images

import (
	"context"
	"fmt"
	"io"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/mux"
)

func init() {
	editor.Register(ImageFeature)
}

type ImageFeatureBlock features.Block

func (i *ImageFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = (*features.Block)(i).Config(widgetContext)
	cfg["apiUrl"] = django.Reverse("editor:upload-image")
	return cfg
}

var ImageFeature = &ImageFeatureBlock{
	BaseFeature: features.BaseFeature{
		Type:          "image",
		JSConstructor: "Image",
		JSFiles: []string{
			"editorjs/js/deps/tools/image.js",
		},
		Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["caption"], d.Data["url"])
			}
			return fb
		},
		Register: func(m django.Mux) {
			m.Post(
				"/upload-image",
				mux.NewHandler(uploadImage),
				"upload-image",
			)
		},
	},
	RenderFunc: renderImage,
}

func renderImage(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var url = fb.Data().Data["url"]
	var caption = fb.Data().Data["caption"]
	fmt.Fprintf(w,
		"<img data-block-id=\"%s\" src=\"%s\" alt=\"%s\" />",
		fb.ID(), url, caption,
	)

	return nil
}
