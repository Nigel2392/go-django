package pages

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/columns"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views/list"
)

const (
	AdminPagesAppName   = "pages"
	AdminPagesModelPath = "PageNode"
)

var pageAdminAppOptions = admin.AppOptions{
	AppLabel:       trans.S("Pages"),
	AppDescription: trans.S("Manage pages in a hierarchical structure."),
	MediaFn: func() media.Media {
		var m = media.NewMedia()
		m.AddCSS(
			media.CSS(django.Static("pages/admin/css/index.css")),
		)
		m.AddJS(
			media.JS(django.Static("pages/admin/js/index.js")),
		)
		return m
	},
}

var pageAdminModelOptions = admin.ModelOptions{
	Name:           AdminPagesModelPath,
	Model:          &PageNode{},
	DisallowList:   true,
	DisallowCreate: true,
	DisallowEdit:   true,
	DisallowDelete: true,
	Labels: map[string]func(ctx context.Context) string{
		"ID":          trans.S("ID"),
		"Title":       trans.S("Title"),
		"Path":        trans.S("Tree Path"),
		"Depth":       trans.S("Tree Depth"),
		"Numchild":    trans.S("Number of Children"),
		"UrlPath":     trans.S("URL Path"),
		"Slug":        trans.S("Slug"),
		"StatusFlags": trans.S("Status Flags"),
		"PageID":      trans.S("Page ID"),
		"ContentType": trans.S("Content Type"),
		"CreatedAt":   trans.S("Created At"),
		"UpdatedAt":   trans.S("Updated At"),
	},
	ListView: admin.ListViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{
				"Title",
				"Slug",
				"ContentType",
				"CreatedAt",
				"UpdatedAt",
				"Children",
			},
		},
		Search: &admin.SearchOptions{
			GetEditLink: func(req *http.Request, id any) string {
				return django.Reverse("admin:pages:edit", id)
			},
			ListFields: []string{
				"Title",
				"Slug",
				"ContentType",
				"CreatedAt",
				"UpdatedAt",
			},
			Fields: []admin.SearchField{
				{
					Name:   "Title",
					Lookup: expr.LOOKUP_ICONTANS,
				},
				{
					Name:   "Slug",
					Lookup: expr.LOOKUP_ICONTANS,
				},
				{
					Name:   "UrlPath",
					Lookup: expr.LOOKUP_ICONTANS,
				},
				{
					Name:   "ContentType",
					Lookup: expr.LOOKUP_ICONTANS,
				},
			},
		},
		Columns: map[string]list.ListColumn[attrs.Definer]{
			"Children": list.ProcessableFieldColumn(
				func(ctx context.Context) string { return "" },
				"Numchild",
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer, children int64) any {
					var node = row.(*PageNode)
					if node.Numchild > 0 {
						var url = django.Reverse(
							"admin:pages:list",
							node.PK,
						)

						return template.HTML(fmt.Sprintf(
							`<a href="%s" class="button secondary hollow">%s</a>`,
							url, trans.P(
								r.Context(), "%d child", "%d children",
								int(node.Numchild), node.Numchild,
							),
						))
					}
					return template.HTML("")
				},
			),
			//	"ChooserChildren": list.ProcessableFieldColumn(
			//		func(ctx context.Context) string { return "" },
			//		"Numchild",
			//		func(r *http.Request, _ attrs.Definitions, row attrs.Definer, children int64) any {
			//			var node = row.(*PageNode)
			//			if node.Numchild > 0 {
			//				var chooserURL = django.Reverse(
			//					"admin:apps:model:chooser:list",
			//					AdminPagesAppName, AdminPagesModelPath, chooser.DEFAULT_KEY,
			//				)
			//
			//				var q = make(url.Values)
			//				q.Set("parent", fmt.Sprintf("%d", node.PK))
			//
			//				return template.HTML(fmt.Sprintf(
			//					`<a href="%s" class="button secondary hollow chooser-list-link">%s</a>`,
			//					fmt.Sprintf("%s?%s", chooserURL, q.Encode()), trans.P(
			//						r.Context(), "%d child", "%d children",
			//						int(node.Numchild), node.Numchild,
			//					),
			//				))
			//			}
			//			return template.HTML("")
			//		},
			//	),
			"Live": list.BooleanColumnFunc(
				trans.S("Live"),
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) bool {
					var node = row.(*PageNode)
					return node.IsPublished()
				},
			),
			"UpdatedAt": columns.TimeSinceColumn[attrs.Definer](
				trans.S("Time since last update"),
				"UpdatedAt",
			),
			"ContentType": list.ProcessableFieldColumn(
				trans.S("Content Type"),
				"ContentType",
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer, cType string) any {
					var ctype = DefinitionForType(cType)
					var typStr = trans.T(r.Context(), "Unknown")
					if ctype != nil {
						typStr = ctype.Label(r.Context())
					}

					return template.HTML(fmt.Sprintf(
						`<span class="badge">%s</span>`,
						typStr,
					))
				},
			),
			"UrlPath": list.ProcessableFieldColumn[attrs.Definer, string](
				trans.S("URL Path"),
				"UrlPath",
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer, urlPath string) any {
					var node = row.(*PageNode)

					if !node.IsPublished() {
						return template.HTML(fmt.Sprintf(
							`<span class="badge warning">%s</span>`,
							node.UrlPath,
						))
					}

					var _, site, err = SiteForRequest(r)
					if err != nil {
						return template.HTML(fmt.Sprintf(
							`<span class="badge warning">%s</span>`,
							node.UrlPath,
						))
					}

					if site.Root == nil || !strings.HasPrefix(node.Path, site.Root.Path) {
						return template.HTML(fmt.Sprintf(
							`<span class="badge warning">%s</span>`,
							node.UrlPath,
						))
					}

					return template.HTML(fmt.Sprintf(
						`<a href="%s">%s</a>`,
						URLPath(node), node.UrlPath,
					))
				},
			),
		},
		PerPage: 16,
	},
	AddView: admin.FormViewOptions{
		SaveInstance: saveInstanceFunc,
		Panels: []admin.Panel{
			admin.TitlePanel(
				admin.FieldPanel("Title"),
			),
			admin.RowPanel(
				admin.FieldPanel("Path"),
				admin.FieldPanel("Depth"),
			),
			admin.FieldPanel("PageID"),
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("StatusFlags"),
		},
		// Change form interaction
	},
	EditView: admin.FormViewOptions{
		SaveInstance: saveInstanceFunc,
		// Change form interaction
		Panels: []admin.Panel{
			admin.TitlePanel(
				admin.FieldPanel("Title"),
			),
			admin.RowPanel(
				admin.FieldPanel("Path"),
				admin.FieldPanel("Depth"),
			),
			admin.FieldPanel("PageID"),
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("StatusFlags"),
			admin.FieldPanel("CreatedAt"),
			admin.FieldPanel("UpdatedAt"),
		},
	},
}

func saveInstanceFunc(ctx context.Context, d attrs.Definer) error {
	var n = d.(*PageNode)
	var err error
	var qs = NewPageQuerySet().WithContext(ctx)
	if n.PK == 0 {
		_, err = qs.ExplicitSave().Create(n)
	} else {
		err = qs.UpdateNode(n)
	}

	if err != nil {
		return err
	}

	//django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
	//	return FixTree(pageApp.QuerySet(), ctx)
	//})

	return nil
}

// Fixtree fixes the tree structure of the page nodes.
//
// It scans for errors in the tree structure in the database and fixes them.
func FixTree(ctx context.Context) error {
	var querySet = queries.GetQuerySet(&PageNode{}).WithContext(ctx)
	var transaction, err = querySet.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(ctx)

	var qs = NewPageQuerySet().WithContext(ctx)
	allNodesCount, err := qs.Count()
	if err != nil {
		return errors.Wrap(err, "failed to count nodes")
	}

	allNodes, err := qs.Offset(0).Limit(int(allNodesCount)).AllNodes()
	if err != nil {
		return errors.Wrap(err, "failed to get all nodes")
	}

	var tree = NewNodeTree(allNodes)

	var changed = tree.FixTree()
	if len(changed) == 0 {
		return errors.NoChanges.Wrapf(
			"no changes found in tree structure for %d nodes",
			allNodesCount,
		)
	}

	_, err = qs.
		Base().
		ExplicitSave().
		Select("Path", "Depth", "Numchild", "UrlPath").
		BulkUpdate(allNodes)
	if err != nil {
		return errors.Wrap(err, "failed to update nodes")
	}

	return transaction.Commit(ctx)
}
