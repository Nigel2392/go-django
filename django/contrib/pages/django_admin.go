package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

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
		"pages",
		admin.AppOptions{
			AppLabel:       fields.S("Pages"),
			AppDescription: fields.S("Manage pages in a hierarchical structure."),
		},
		admin.ModelOptions{
			Name:  "Page",
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
				var ctx = context.Background()
				var n, err = pageApp.QuerySet().GetNodeByID(ctx, identifier.(int64))
				return &n, err
			},
			GetList: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
				var ctx = context.Background()
				var nodes, err = pageApp.QuerySet().AllNodes(ctx, int32(offset), int32(amount))
				var items = make([]attrs.Definer, 0)
				for _, n := range nodes {
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
				// Change form interaction
			},
			EditView: admin.FormViewOptions{
				// Change form interaction
			},
		},
	)
}
