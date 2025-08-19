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
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/mux"
)

func init() {
	editor.Register(ImageFeature)
	editor.Register(ImagesFeature)
}

//go:embed static/**/*
var imagesFS embed.FS

var (
	_ editor.PrefetchableFeature  = (*ImageFeatureBlock)(nil)
	_ editor.FeatureBlockRenderer = (*ImageFeatureBlock)(nil)
)

// Use a prefetchable feature for images
//
// This is to ensure that the actual serve view for the images
// does not have do make a query for each image.
//
// This can immensely speed up the process of fetching images.
// If the prefetched data is not available, we will fall back to
// having to make individual queries for each image.
type ImageFeatureBlock features.PrefetchableFeature

func (i *ImageFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = (*features.PrefetchableFeature)(i).Config(widgetContext)
	cfg["chooserURL"] = django.Reverse(
		"admin:apps:model:chooser:list",
		"images", "image", chooser.DEFAULT_KEY,
	)
	cfg["serveUrl"] = django.Reverse("images:serve_id", "<<id>>")
	return cfg
}

func (i *ImageFeatureBlock) PrefetchData(ctx context.Context, data []editor.BlockData) (map[string]editor.BlockData, error) {
	return (*features.PrefetchableFeature)(i).PrefetchData(ctx, data)
}

func (b *ImageFeatureBlock) Render(d editor.BlockData) editor.FeatureBlock {
	return (*features.PrefetchableFeature)(b).Render(d)
}

func (b *ImageFeatureBlock) RenderBlock(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	return (*features.PrefetchableFeature)(b).RenderBlock(fb, c, w)
}

var onFeatureRegisterOnce sync.Once

func onFeatureRegister(m mux.Multiplexer) {
	onFeatureRegisterOnce.Do(func() {
		staticfiles.AddFS(
			filesystem.Sub(imagesFS, "static"),
			filesystem.MatchPrefix("images/editorjs"),
		)

		if !django.AppInstalled("editor") {
			panic(errors.New(
				"required apps are not installed: editor",
			))
		}
		if !django.AppInstalled("images") {
			panic(errors.New(
				"required apps are not installed: images",
			))
		}
	})
}

var ImageFeature = &ImageFeatureBlock{
	Block: features.Block{
		BaseFeature: features.BaseFeature{
			Type:          "image",
			JSConstructor: "GoDjangoImageTool",
			CSSFles: []string{
				"images/editorjs/image.css",
			},
			JSFiles: []string{
				"chooser/js/index.js",
				"images/editorjs/image.js",
			},
			Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
				fb.GetString = func(d editor.BlockData) string {
					caption, ok := d.Data["caption"].(string)
					if !ok {
						caption = ""
					}
					return fmt.Sprintf("[%q](%s)", caption, d.Data["id"])
				}
				return fb
			},
			Validate: func(bd editor.BlockData) error {
				if _, ok := bd.Data["id"]; !ok {
					return errors.New("image id not found")
				}
				return nil
			},
			Register: onFeatureRegister,
		},
		RenderFunc: renderImage,
	},
	//	Prefetch: func(ctx context.Context, data []editor.BlockData) (map[string]editor.BlockData, error) {
	//		var dataMap = make(map[uint32][]*editor.BlockData)
	//		var idList = make([]uint32, 0, len(data))
	//
	//		for _, d := range data {
	//			var intVal, err = strconv.Atoi(d.Data["id"].(string))
	//			if err != nil {
	//				return nil, err
	//			}
	//
	//			idList = append(idList, uint32(intVal))
	//			var slice, ok = dataMap[uint32(intVal)]
	//			if !ok {
	//				slice = make([]*editor.BlockData, 0)
	//			}
	//			slice = append(slice, &d)
	//			dataMap[uint32(intVal)] = slice
	//		}
	//
	//		var rowCnt, rowIter, err = queries.GetQuerySet(&images.Image{}).
	//			Filter("ID__in", idList).
	//			IterAll()
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		var objMap = make(map[string]editor.BlockData, rowCnt)
	//		for row, err := range rowIter {
	//			if err != nil {
	//				return nil, err
	//			}
	//
	//			var dataObjs, ok = dataMap[row.Object.ID]
	//			if !ok {
	//				return nil, errors.New("data object not found in dataMap")
	//			}
	//
	//			// Map the data object to the image
	//			for _, dataObj := range dataObjs {
	//
	//				if caption := dataObj.Data["caption"]; caption == nil || caption == "" || caption == "undefined" {
	//					dataObj.Data["caption"] = row.Object.Title
	//				}
	//
	//				dataObj.Data["serve_url"] = django.Reverse("images:serve", row.Object.Path)
	//				objMap[dataObj.ID] = *dataObj
	//			}
	//		}
	//
	//		return objMap, nil
	//	},
}

func renderImage(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var caption = fb.Data().Data["caption"]
	var id = fb.Data().Data["id"]
	var serveUrlFace = fb.Data().Data["serve_url"]

	if caption == nil {
		caption = ""
	}

	var serveURL string
	if serveUrlFace != nil {
		serveURL = serveUrlFace.(string)
	} else {
		serveURL = django.Reverse("images:serve_id", id)
	}

	fmt.Fprintf(w,
		"<img data-block-id=\"%s\" src=\"%s\" alt=\"%s\" data-id=\"%v\" />",
		fb.ID(), serveURL, caption, id,
	)

	return nil
}

var ImagesFeature = &ImageFeatureBlock{
	Block: features.Block{
		BaseFeature: features.BaseFeature{
			Type:          "images",
			JSConstructor: "GoDjangoImagesTool",
			CSSFles: []string{
				"images/editorjs/image.css",
			},
			JSFiles: []string{
				"chooser/js/index.js",
				"images/editorjs/images.js",
			},
			Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
				fb.GetString = func(d editor.BlockData) string {
					var rImages = reflect.ValueOf(d.Data["ids"])
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
				var rImages = reflect.ValueOf(bd.Data["ids"])
				if rImages.Kind() != reflect.Slice {
					return errors.New("images data is not a slice")
				}

				for i := 0; i < rImages.Len(); i++ {
					var idFace = rImages.Index(i).Interface()
					if idFace == nil {
						return fmt.Errorf(
							"image id at index %d is nil", i,
						)
					}

					var id, ok = idFace.(string)
					if id == "" || !ok {
						return fmt.Errorf(
							"image id at index %d is not valid: %v",
							i, idFace,
						)
					}
				}
				return nil
			},
			Register: onFeatureRegister,
		},
		RenderFunc: renderImages,
	},
	//	Prefetch: func(ctx context.Context, data []editor.BlockData) (map[string]editor.BlockData, error) {
	//		var dataMap = make(map[uint32][]*editor.BlockData)
	//		var idList = make([]uint32, 0, len(data))
	//
	//		for _, d := range data {
	//
	//			if _, ok := d.Data["ids"]; !ok {
	//				return nil, errors.New("ids not found in block data")
	//			}
	//
	//			var rImages = reflect.ValueOf(d.Data["ids"])
	//			if rImages.Kind() != reflect.Slice {
	//				return nil, errors.New("images data is not a slice")
	//			}
	//
	//			for i := 0; i < rImages.Len(); i++ {
	//				var idFace = rImages.Index(i).Interface()
	//				if idFace == nil {
	//					return nil, fmt.Errorf("image id at index %d is nil", i)
	//				}
	//
	//				var id, ok = idFace.(string)
	//				if id == "" || !ok {
	//					return nil, fmt.Errorf("image id at index %d is not valid: %v", i, idFace)
	//				}
	//
	//				var intVal, err = strconv.Atoi(id)
	//				if err != nil {
	//					return nil, err
	//				}
	//
	//				idList = append(idList, uint32(intVal))
	//				slice, ok := dataMap[uint32(intVal)]
	//				if !ok {
	//					slice = make([]*editor.BlockData, 0)
	//				}
	//				slice = append(slice, &d)
	//				dataMap[uint32(intVal)] = slice
	//			}
	//		}
	//
	//		var rowCnt, rowIter, err = queries.GetQuerySet(&images.Image{}).
	//			Filter("ID__in", idList).
	//			IterAll()
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		var objMap = make(map[string]editor.BlockData, rowCnt)
	//		for row, err := range rowIter {
	//			if err != nil {
	//				return nil, err
	//			}
	//
	//			var dataObjs, ok = dataMap[row.Object.ID]
	//			if !ok {
	//				return nil, errors.New("data object not found in dataMap")
	//			}
	//
	//			// Map the data object to the image
	//			for _, dataObj := range dataObjs {
	//
	//				var serveURLs, ok = dataObj.Data["serve_urls"].([]string)
	//				if !ok {
	//					serveURLs = []string{django.Reverse("images:serve", row.Object.Path)}
	//				} else {
	//					serveURLs = append(serveURLs, django.Reverse("images:serve", row.Object.Path))
	//				}
	//
	//				dataObj.Data["serve_urls"] = serveURLs
	//
	//				objMap[dataObj.ID] = *dataObj
	//			}
	//		}
	//
	//		return objMap, nil
	//	},
}

func renderImages(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
	var imagesData = fb.Data().Data["ids"]
	if imagesData == nil {
		return errors.New("images data not found")
	}

	var usingServeURLs = false
	if serveURLs, ok := fb.Data().Data["serve_urls"].([]string); ok {
		usingServeURLs = true
		imagesData = serveURLs
	}

	var rImages = reflect.ValueOf(imagesData)
	if rImages.Kind() != reflect.Slice {
		return errors.New("images data is not a slice")
	}

	fmt.Fprint(w, "<div class=\"multi-images\" data-block-id=\"")
	fmt.Fprint(w, fb.ID())
	fmt.Fprint(w, "\">")

	for i := 0; i < rImages.Len(); i++ {
		var img = rImages.Index(i).Interface().(string)

		var url string
		if usingServeURLs {
			url = img
		} else {
			url = django.Reverse("images:serve_id", img)
		}

		fmt.Fprintf(
			w, "<img src=\"%s\" alt=\"editor image %s\" data-index=\"%d\" data-id=\"%v\" />",
			url, img, i, img,
		)
	}

	fmt.Fprint(w, "</div>")

	return nil
}
