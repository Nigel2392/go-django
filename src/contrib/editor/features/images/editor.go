package images

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/mux"
)

func init() {
	editor.Register(ImageFeature)
}

//go:embed static/**/*
var imagesFS embed.FS

type ImageFeatureBlock features.Block

func (i *ImageFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = (*features.Block)(i).Config(widgetContext)
	cfg["uploadUrl"] = django.Reverse("editor:upload-image")
	var serveURL = strings.TrimSuffix(
		django.Reverse("editor:images"), "/",
	)
	cfg["serveUrl"] = strings.TrimSuffix(serveURL, "*")
	return cfg
}

func (i *ImageFeatureBlock) Media() media.Media {
	var m = media.NewMedia()
	m.AddJS(
		media.JS(django.Static("images/editorjs/image.js")),
	)
	m.AddCSS(
		media.CSS(django.Static("images/editorjs/image.css")),
	)
	return m
}

func (b *ImageFeatureBlock) Render(d editor.BlockData) editor.FeatureBlock {
	return (*features.Block)(b).Render(d)
}

func (b *ImageFeatureBlock) RenderBlock(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	return (*features.Block)(b).RenderBlock(fb, c, w)
}

var ImageFeature = &ImageFeatureBlock{
	BaseFeature: features.BaseFeature{
		Type:          "image",
		JSConstructor: "GoDjangoImageTool",
		Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["caption"], d.Data["filePath"])
			}
			return fb
		},
		Register: func(m django.Mux) {
			staticfiles.AddFS(
				filesystem.Sub(imagesFS, "static"),
				filesystem.MatchPrefix("images/editorjs"),
			)
			m.Any(
				"/upload-image",
				mux.NewHandler(uploadImage),
				"upload-image",
			)
			m.Get(
				"/images/*",
				mux.NewHandler(viewImage),
				"images",
			)
		},
	},
	RenderFunc: renderImage,
}

func renderImage(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var url = fb.Data().Data["filePath"]
	var caption = fb.Data().Data["caption"]
	var serveURL = strings.TrimSuffix(
		django.Reverse("editor:images"), "/",
	)
	serveURL = strings.TrimSuffix(
		serveURL, "*",
	)

	if url == nil {
		return errors.New("image url not found")
	}

	if caption == nil {
		caption = ""
	}

	fmt.Fprintf(w,
		"<img data-block-id=\"%s\" src=\"%s\" alt=\"%s\" />",
		fb.ID(), path.Join(serveURL, url.(string)), caption,
	)

	return nil
}
