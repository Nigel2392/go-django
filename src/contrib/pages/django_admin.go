package pages

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/pkg/errors"
)

const (
	AdminPagesAppName   = "pages"
	AdminPagesModelPath = "Page"
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
	Name:  AdminPagesModelPath,
	Model: &PageNode{},
	Labels: map[string]func() string{
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
		Columns: map[string]list.ListColumn[attrs.Definer]{
			"Children": list.HTMLColumn[attrs.Definer](
				trans.S(""),
				func(_ *http.Request, _ attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*PageNode)
					if node.Numchild > 0 {
						var url = django.Reverse(
							"admin:pages:list",
							node.PK,
						)
						var childText = "child"
						if node.Numchild > 1 || node.Numchild == 0 {
							childText = "children"
						}
						return template.HTML(fmt.Sprintf(
							`<a href="%s" class="button primary hollow">%d %s</a>`,
							url, node.Numchild, childText,
						))
					}
					return template.HTML("")
				},
			),
			"ContentType": list.FuncColumn(
				trans.S("Content Type"),
				func(_ *http.Request, _ attrs.Definitions, row attrs.Definer) interface{} {
					var node = row.(*PageNode)
					var ctype = DefinitionForType(node.ContentType)
					return ctype.Label()
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
			admin.MultiPanel(
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
			admin.MultiPanel(
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
	if n.PK == 0 {
		_, err = insertNode(ctx, n)
	} else {
		err = UpdateNode(ctx, n)
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
	defer transaction.Rollback()

	allNodesCount, err := CountNodes(ctx, StatusFlagNone)
	if err != nil {
		return errors.Wrap(err, "failed to count nodes")
	}

	allNodes, err := AllNodes(ctx, StatusFlagNone, 0, int32(allNodesCount))
	if err != nil {
		return errors.Wrap(err, "failed to get all nodes")
	}

	var tree = NewNodeTree(allNodes)

	tree.FixTree()

	err = updateNodes(ctx, allNodes)
	if err != nil {
		return errors.Wrap(err, "failed to update nodes")
	}

	return transaction.Commit()
}
