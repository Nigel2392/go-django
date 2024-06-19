package pages

import (
	"context"
	"strconv"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/goldcrest"
	"github.com/pkg/errors"
)

const (
	AdminPagesAppName   = "pages"
	AdminPagesModelPath = "Page"
)

var pageAdminAppOptions = admin.AppOptions{
	AppLabel:       fields.S("Pages"),
	AppDescription: fields.S("Manage pages in a hierarchical structure."),
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
		"ID":          fields.S("ID"),
		"Title":       fields.S("Title"),
		"Path":        fields.S("Tree Path"),
		"Depth":       fields.S("Tree Depth"),
		"Numchild":    fields.S("Number of Children"),
		"UrlPath":     fields.S("URL Path"),
		"StatusFlags": fields.S("Status Flags"),
		"PageID":      fields.S("Page ID"),
		"ContentType": fields.S("Content Type"),
		"CreatedAt":   fields.S("Created At"),
		"UpdatedAt":   fields.S("Updated At"),
	},
	GetForID: func(identifier any) (attrs.Definer, error) {
		var id int64
		switch v := identifier.(type) {
		case int:
			id = int64(v)
		case int64:
			id = v
		case string:
			var err error
			id, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errs.ErrInvalidType
		}
		var ctx = context.Background()
		var node, err = pageApp.QuerySet().GetNodeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &node, nil
	},
	GetList: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
		var ctx = context.Background()
		var nodes, err = pageApp.QuerySet().AllNodes(ctx, int32(amount), int32(offset))
		var items = make([]attrs.Definer, 0)
		for _, n := range nodes {
			n := n
			items = append(items, &n)
		}
		return items, err
	},
	ListView: admin.ListViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{
				"Title",
				"CreatedAt",
				"UpdatedAt",
			},
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
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
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
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
			admin.FieldPanel("StatusFlags"),
			admin.FieldPanel("CreatedAt"),
			admin.FieldPanel("UpdatedAt"),
		},
	},
}

func init() {
	var hookFn = func(site *admin.AdminApplication, items menu.Items) {
		items.Append(&PagesMenuItem{
			BaseItem: menu.BaseItem{
				Label:    fields.S("Pages"),
				ItemName: "pages",
				Ordering: -1,
			},
		})
	}

	goldcrest.Register(admin.RegisterMenuItemHook, 0, hookFn)

	admin.RegisterApp(
		AdminPagesAppName,
		pageAdminAppOptions,
		pageAdminModelOptions,
	)
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
			int64(n.StatusFlags),
			n.PageID,
			n.ContentType,
		)
	} else {
		err = pageApp.QuerySet().UpdateNode(
			ctx,
			n.Title,
			n.Path,
			n.Depth,
			n.Numchild,
			n.UrlPath,
			int64(n.StatusFlags),
			n.PageID,
			n.ContentType,
			n.PK,
		)
	}

	if err != nil {
		return err
	}

	django.Task("Fixing tree structure upon manual page node save", func(app *django.Application) error {
		allNodesCount, err := pageApp.QuerySet().CountNodes(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to count nodes")
		}

		allNodes, err := pageApp.QuerySet().AllNodes(ctx, int32(allNodesCount), 0)
		if err != nil {
			return errors.Wrap(err, "failed to get all nodes")
		}

		var nodeRefs = make([]*models.PageNode, len(allNodes))
		for i := 0; i < len(allNodes); i++ {
			nodeRefs[i] = &allNodes[i]
		}

		var tree = NewNodeTree(nodeRefs)

		tree.FixTree()

		err = pageApp.QuerySet().UpdateNodes(ctx, nodeRefs)
		if err != nil {
			return errors.Wrap(err, "failed to update nodes")
		}

		return nil
	})

	return nil
}
