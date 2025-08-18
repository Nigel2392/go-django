package images

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

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
	editor.Register(ImagesFeature)
}

//go:embed static/**/*
var imagesFS embed.FS

type ImageFeatureBlock features.Block

func (i *ImageFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = (*features.Block)(i).Config(widgetContext)
	cfg["uploadUrl"] = django.Reverse("editor:upload-image")
	cfg["serveUrl"] = django.Reverse("images:serve_id", "<<id>>")
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

var onFeatureRegisterOnce sync.Once

func onFeatureRegister(m mux.Multiplexer) {
	onFeatureRegisterOnce.Do(func() {
		staticfiles.AddFS(
			filesystem.Sub(imagesFS, "static"),
			filesystem.MatchPrefix("images/editorjs"),
		)

		m.Any(
			"/upload-image",
			mux.NewHandler(uploadImage),
			"upload-image",
		)
	})
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
		Validate: func(bd editor.BlockData) error {
			var rImage = reflect.ValueOf(bd.Data)
			if rImage.Kind() != reflect.Map {
				return errors.New("image data is not a map")
			}
			if rImage.MapIndex(reflect.ValueOf("id")).IsNil() {
				return errors.New("image id not found")
			}
			if rImage.MapIndex(reflect.ValueOf("filePath")).IsNil() {
				return errors.New("image filePath not found")
			}
			return nil
		},
		Register: onFeatureRegister,
	},
	RenderFunc: renderImage,
}

func renderImage(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var caption = fb.Data().Data["caption"]
	var id = fb.Data().Data["id"]

	if caption == nil {
		caption = ""
	}

	fmt.Fprintf(w,
		"<img data-block-id=\"%s\" src=\"%s\" alt=\"%s\" data-id=\"%v\" />",
		fb.ID(), django.Reverse("images:serve_id", id), caption, id,
	)

	return nil
}

var ImagesFeature = &ImageFeatureBlock{
	BaseFeature: features.BaseFeature{
		Type:          "images",
		JSConstructor: "GoDjangoImagesTool",
		Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				var rImages = reflect.ValueOf(d.Data["images"])
				var imagesLen int
				if rImages.Kind() == reflect.Slice {
					imagesLen = rImages.Len()
				}
				return fmt.Sprintf(
					"%d images",
					imagesLen,
				)
			}
			return fb
		},
		Validate: func(bd editor.BlockData) error {
			var rImages = reflect.ValueOf(bd.Data["images"])
			if rImages.Kind() != reflect.Slice {
				return errors.New("images data is not a slice")
			}

			for i := 0; i < rImages.Len(); i++ {
				var img = rImages.Index(i).Interface().(map[string]interface{})
				if img["id"] == nil {
					return errors.New("image id not found")
				}
				if img["filePath"] == nil {
					return errors.New("image filePath not found")
				}
			}
			return nil
		},
		Register: onFeatureRegister,
	},
	RenderFunc: renderImages,
}

func renderImages(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var imagesData = fb.Data().Data["images"]
	if imagesData == nil {
		return errors.New("images data not found")
	}

	var rImages = reflect.ValueOf(imagesData)
	if rImages.Kind() != reflect.Slice {
		return errors.New("images data is not a slice")
	}

	fmt.Fprint(w, "<div class=\"multi-images\" data-block-id=\"")
	fmt.Fprint(w, fb.ID())
	fmt.Fprint(w, "\">")

	for i := 0; i < rImages.Len(); i++ {
		var img = rImages.Index(i).Interface().(map[string]interface{})
		var id = img["id"].(string)

		fmt.Fprintf(
			w, "<img src=\"%s\" alt=\"%s\" data-index=\"%d\" data-id=\"%v\" />",
			django.Reverse("images:serve_id", id), img["caption"].(string), i, id,
		)
	}

	fmt.Fprint(w, "</div>")

	return nil
}
