package adminsite

import (
	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/searches"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var CustomersAdmin = admin.ModelOptions{
	Name:                "customers",
	Model:               &customers.Customer{},
	MenuOrder:           500,
	MenuLabel:           trans.S("Customers"),
	RegisterToAdminMenu: true,
	ListView: admin.ListViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{"Email", "FirstName", "LastName", "FirstSeen"},
		},
		Ordering: []string{"-FirstSeen"},
		Search: &searches.SearchOptions{
			Fields: []searches.SearchField{
				{Name: "Email", Lookup: expr.LOOKUP_ICONTANS},
				{Name: "FirstName", Lookup: expr.LOOKUP_ICONTANS},
				{Name: "LastName", Lookup: expr.LOOKUP_ICONTANS},
			},
		},
	},
	AddView: admin.FormViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{"Email", "FirstName", "LastName"},
		},
		Panels: []admin.Panel{
			admin.FieldPanel("Email"),
			admin.RowPanel(admin.FieldPanel("FirstName"), admin.FieldPanel("LastName")),
		},
	},
	EditView: admin.FormViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{"Email", "FirstName", "LastName"},
		},
		Panels: []admin.Panel{
			admin.FieldPanel("Email"),
			admin.RowPanel(admin.FieldPanel("FirstName"), admin.FieldPanel("LastName")),
		},
	},
}
