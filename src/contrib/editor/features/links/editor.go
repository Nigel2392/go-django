package links

import (
	"context"
	"embed"
	"fmt"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	"github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/mux"
	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

func init() {
	editor.Register(PageLinkFeature)
}

//go:embed static/**/*
var linksFs embed.FS

var _ editor.InlineFeature = (*PageLinkFeatureBlock)(nil)

type PageLinkFeatureBlock features.InlineFeature

func (i *PageLinkFeatureBlock) Config(widgetContext ctx.Context) map[string]interface{} {
	var cfg = (*features.InlineFeature)(i).Config(widgetContext)
	cfg["pageListURL"] = django.Reverse("editor:links:list-pages")
	cfg["pageListQueryVar"] = pages.PageIDVariableName
	return cfg
}

func (i *PageLinkFeatureBlock) Media() media.Media {
	var m = media.NewMedia()
	m.AddJS(
		media.JS(django.Static("links/editorjs/index.js")),
	)
	return m
}

func (b *PageLinkFeatureBlock) Render(d editor.BlockData) editor.FeatureBlock {
	return (*features.InlineFeature)(b).Render(d)
}

func (b *PageLinkFeatureBlock) ParseInlineData(doc *goquery.Document) error {
	return (*features.InlineFeature)(b).ParseInlineData(doc)
}

var PageLinkFeature = &PageLinkFeatureBlock{
	TagName: "a",
	Class:   "page-link",
	Attributes: []features.InlineFeatureAttribute{
		{Name: "data-page-id", Required: true},
	},
	BaseFeature: features.BaseFeature{
		Type:          "pagelink",
		JSConstructor: "PageLinkTool",
		Build: func(fb *features.FeatureBlock) *features.FeatureBlock {
			fb.GetString = func(d editor.BlockData) string {
				return fmt.Sprintf("[%s](%s)", d.Data["text"], d.Data["id"])
			}
			return fb
		},
		Register: func(m django.Mux) {
			staticfiles.AddFS(
				filesystem.Sub(linksFs, "static"),
				filesystem.MatchPrefix("links/editorjs"),
			)
			var linkRoute = m.Any(
				"links",
				nil, "links",
			)
			linkRoute.Get(
				"/list-pages",
				mux.NewHandler(listPages),
				"list-pages",
			)
		},
	},
	RebuildElementsFn: func(li []*features.InlineFeatureElement) error {

		var pageIds = make([]int64, 0)
		for _, el := range li {
			var attrMap = make(map[string]string)
			for _, attr := range el.Node.Attr {
				attrMap[attr.Key] = attr.Val
			}

			var pageID = attrMap["data-page-id"]
			if pageID == "" {
				return errors.New("page ID not found")
			}

			var id, err = strconv.Atoi(pageID)
			if err != nil {
				return errors.Wrap(
					err, "failed to convert page id to int",
				)
			}

			pageIds = append(pageIds, int64(id))
		}

		var pageList, err = pages.QuerySet().GetNodesByIDs(
			context.Background(), pageIds,
		)
		if err != nil {
			return errors.Wrap(
				err, "failed to get pages by ids",
			)
		}

		var idMap = make(map[int64]*page_models.PageNode)
		for _, page := range pageList {
			idMap[page.ID()] = &page
		}

		for _, el := range li {
			var attrMap = make(map[string]string)
			for _, attr := range el.Node.Attr {
				attrMap[attr.Key] = attr.Val
			}

			var pageID = attrMap["data-page-id"]
			var id, err = strconv.Atoi(pageID)
			if err != nil {
				return errors.Wrap(
					err, "failed to convert page id to int",
				)
			}

			var page = idMap[int64(id)]
			el.Node.Attr = []html.Attribute{
				{
					Key: "class",
					Val: "page-link",
				},
				{
					Key: "data-page-id",
					Val: pageID,
				},
				{
					Key: "href",
					Val: pages.URLPath(page),
				},
			}
		}

		return nil
	},
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
