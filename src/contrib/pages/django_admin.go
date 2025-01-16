package pages

import (
	"context"
	"fmt"
	"html/template"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
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
	Model: &models.PageNode{},
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
				func(defs attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*models.PageNode)
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
				func(defs attrs.Definitions, row attrs.Definer) interface{} {
					var node = row.(*models.PageNode)
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
	var n = d.(*models.PageNode)
	var err error
	if n.PK == 0 {
		_, err = pageApp.QuerySet().InsertNode(
			ctx,
			n.Title,
			n.Path,
			n.Depth,
			n.Numchild,
			n.UrlPath,
			n.Slug,
			int64(n.StatusFlags),
			n.PageID,
			n.ContentType,
			n.LatestRevisionID,
		)
	} else {
		err = pageApp.QuerySet().UpdateNode(
			ctx,
			n.Title,
			n.Path,
			n.Depth,
			n.Numchild,
			n.UrlPath,
			n.Slug,
			int64(n.StatusFlags),
			n.PageID,
			n.ContentType,
			n.LatestRevisionID,
			n.PK,
		)
	}

	if err != nil {
		return err
	}

	django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
		return FixTree(pageApp.QuerySet(), ctx)
	})

	return nil
}

func FixTree(querySet models.DBQuerier, ctx context.Context) error {
	var tx, err = querySet.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var qs = QuerySet().WithTx(tx)
	allNodesCount, err := qs.CountNodes(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to count nodes")
	}

	allNodes, err := qs.AllNodes(ctx, int32(allNodesCount), 0)
	if err != nil {
		return errors.Wrap(err, "failed to get all nodes")
	}

	var nodeRefs = make([]*models.PageNode, len(allNodes))
	for i := 0; i < len(allNodes); i++ {
		nodeRefs[i] = &allNodes[i]
	}

	var tree = NewNodeTree(nodeRefs)

	tree.FixTree()

	err = qs.UpdateNodes(ctx, nodeRefs)
	if err != nil {
		return errors.Wrap(err, "failed to update nodes")
	}

	return tx.Commit()
}
