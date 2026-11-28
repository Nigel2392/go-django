package adminsite

import (
	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
)

var ProductAdmin = admin.ModelOptions{
	Name:                "products",
	MenuOrder:           200,
	Model:               &products.Product{},
	MenuLabel:           trans.S("Products"),
	RegisterToAdminMenu: true,
	ListView: admin.ListViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition) views.View {
			return adminviews.ViewProductList
		},
	},
	AddView: admin.FormViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
			return ShopHandler{
				Handle: adminviews.ViewAddNewProductSupplier,
			}
		},
	},
	EditView: admin.FormViewOptions{
		GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
			return ShopInstanceHandler[*products.Product]{
				Instance: instance.(*products.Product),
				Handle:   adminviews.ViewEditProduct,
			}
		},
	},
}
