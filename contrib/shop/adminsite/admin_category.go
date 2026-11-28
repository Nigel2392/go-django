package adminsite

import (
	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/searches"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var CategoryAdmin = admin.ModelOptions{
	MenuOrder:           100,
	Model:               &categories.Category{},
	MenuLabel:           trans.S("Category"),
	RegisterToAdminMenu: true,
	ListView: admin.ListViewOptions{
		Fields: []string{"Name", "ParentCategory"},
		Search: &searches.SearchOptions{
			Fields: []searches.SearchField{
				{Name: "Name", Lookup: expr.LOOKUP_ICONTANS},
			},
		},
	},
	AddView: admin.FormViewOptions{
		Panels: []admin.Panel{
			admin.FieldPanel("Name"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("ParentCategory"),
			&admin.ModelFormPanel[*categories.Category, modelforms.ModelForm[*categories.Category]]{
				TargetType: &categories.Category{},
				FieldName:  "Children",
				Classname:  "collapsible",
				// SubClassname: "collapsed",
				MinNum: 1,
				// DisallowAdd:    true,
				// DisallowRemove: true,
				Panels: []admin.Panel{
					admin.FieldPanel("Name"),
					admin.FieldPanel("Slug"),
				},
			},
		},
	},
	EditView: admin.FormViewOptions{
		Panels: []admin.Panel{
			admin.FieldPanel("Name"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("ParentCategory"),
			&admin.ModelFormPanel[*categories.Category, modelforms.ModelForm[*categories.Category]]{
				TargetType: &categories.Category{},
				FieldName:  "Children",
				Classname:  "collapsible",
				// SubClassname: "collapsed",
				MinNum: 1,
				// DisallowAdd:    true,
				// DisallowRemove: true,
				Panels: []admin.Panel{
					admin.FieldPanel("Name"),
					admin.FieldPanel("Slug"),
				},
			},
		},
	},
}
