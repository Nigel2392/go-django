package adminsite

import (
	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
)

var OrdersAdmin = admin.ModelOptions{
	Name:                "orders",
	MenuOrder:           300,
	Model:               &orders.Order{},
	MenuLabel:           trans.S("Orders"),
	RegisterToAdminMenu: true,
	ListView: admin.ListViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition) views.View {
			return adminviews.ViewOrderList
		},
	},
	AddView: admin.FormViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
			return ShopHandler{
				Handle: adminviews.ViewAddOrder,
			}
		},
	},
	EditView: admin.FormViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
			return ShopHandler{
				Handle: adminviews.ViewEditOrder,
			}
		},
	},
}
