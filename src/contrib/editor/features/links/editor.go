package links

import (
	"embed"
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/mux"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

//go:embed static/**/*
var linksFs embed.FS

var _ editor.InlineFeature = (*ObjectLinkFeatureBlock)(nil)

type linkedFeatureModel struct {
	identifier string
	model      attrs.Definer
	getURL     func(object attrs.Definer) string
}

type ObjectLinkFeatureBlock struct {
	features.InlineFeature
	models map[string]*linkedFeatureModel
}

func (i *ObjectLinkFeatureBlock) RegisterModel(identifier string, model attrs.Definer, getURL func(object attrs.Definer) string) {
	if i.models == nil {
		i.models = make(map[string]*linkedFeatureModel)
	}
	i.models[identifier] = &linkedFeatureModel{
		identifier: identifier,
		model:      model,
		getURL:     getURL,
	}
}

func (i *ObjectLinkFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = i.InlineFeature.Config(widgetContext)
	cfg["pageListURL"] = django.Reverse(
		"admin:apps:model:chooser:list",
		pages.AdminPagesAppName,
		pages.AdminPagesModelPath,
		chooser.DEFAULT_KEY,
	)
	cfg["pageViewURL"] = django.Reverse("pages_redirect", "<<id>>")
	return cfg
}

func (i *ObjectLinkFeatureBlock) Media() media.Media {
	var m = media.NewMedia()
	m.AddCSS(
		media.CSS(django.Static("pages/admin/css/chooser.css")),
	)
	m.AddJS(
		media.JS(django.Static("links/editorjs/index.js")),
	)
	return m
}

var ObjectLinkFeature *ObjectLinkFeatureBlock

func init() {
	ObjectLinkFeature = &ObjectLinkFeatureBlock{
		InlineFeature: features.InlineFeature{
			TagName: "a",
			Class:   "object-link",
			Attributes: []features.InlineFeatureAttribute{
				{Name: "data-object-id", Required: true},
				{Name: "data-object-key", Required: true},
			},
			BaseFeature: features.BaseFeature{
				Type:          "object-link",
				JSConstructor: "PageLinkTool",
				Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
					fb.GetString = func(d editor.BlockData) string {
						return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
					}
					return fb
				},
				Register: func(m mux.Multiplexer) {
					staticfiles.AddFS(
						filesystem.Sub(linksFs, "static"),
						filesystem.MatchPrefix("links/editorjs"),
					)
				},
			},
			RebuildElementsFn: func(li []*features.InlineFeatureElement) error {

				var objectIds = make([]string, 0)
				for _, el := range li {
					var attrMap = make(map[string]string)
					for _, attr := range el.Node.Attr {
						attrMap[attr.Key] = attr.Val
					}

					var objectIDStr = attrMap["data-object-id"]
					if objectIDStr == "" {
						return errors.New("object ID not found")
					}

					var objectKey = attrMap["data-object-key"]
					if objectKey == "" {
						return errors.New("object key not found")
					}

					objectIds = append(objectIds, objectIDStr)
				}

				var objectList, err = pages.NewPageQuerySet().
					Filter("PK__in", objectIds).
					AllNodes()
				if err != nil {
					return errors.Wrap(
						err, "failed to get objects by ids",
					)
				}

				var idMap = make(map[string]attrs.Definer)
				for _, object := range objectList {
					var pk = attrs.PrimaryKey(object)
					idMap[attrs.ToString(pk)] = object
				}

				for _, el := range li {
					var attrMap = make(map[string]string)
					for _, attr := range el.Node.Attr {
						attrMap[attr.Key] = attr.Val
					}

					var objectIdStr = attrMap["data-object-id"]
					var objectKey = attrMap["data-object-key"]
					if objectIdStr == "" {
						return errors.New("object ID not found")
					}
					if objectKey == "" {
						return errors.New("object key not found")
					}

					var linkedModel, ok = ObjectLinkFeature.models[objectKey]
					if !ok {
						return errors.Errorf("model not registered for key %s", objectKey)
					}

					var object = idMap[objectIdStr]
					el.Node.Attr = []html.Attribute{
						{
							Key: "class",
							Val: "object-link",
						},
						{
							Key: "data-object-id",
							Val: objectIdStr,
						},
						{
							Key: "href",
							Val: linkedModel.getURL(object),
						},
					}
				}

				return nil
			},
		},
	}

	editor.Register(ObjectLinkFeature)
}

//
//	func renderLink(fb editor.FeatureBlock, c context.Context, w io.Writer) error {
//		var pageID = fb.Data().Data["id"]
//		var text = fb.Data().Data["text"]
//		if pageID == nil {
//			return errors.New("link's page ID not found")
//		}
//
//		var idStr = fmt.Sprintf("%v", pageID)
//		var id, err = strconv.Atoi(idStr)
//		if err != nil {
//			return errors.Wrap(
//				err, "failed to convert page id to int",
//			)
//		}
//
//		page, err := pages.QuerySet().GetNodeByID(
//			c, int64(id),
//		)
//		if err != nil {
//			return errors.Wrap(
//				err, "failed to get page by id",
//			)
//		}
//
//		if text == nil || text == "" {
//			text = page.Title
//		}
//
//		var pageUrl = pages.URLPath(&page)
//		fmt.Fprintf(w,
//			"<a href=\"%s\">%s</a>",
//			pageUrl,
//			text,
//		)
//
//		//fmt.Fprintf(w,
//		//	"<a href=\"%s\">%s</a>",
//		//	django.Reverse("pages_redirect", pageID),
//		//	text,
//		//)
//
//		return nil
//	}
//
