package shop

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/admin/icons"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/internal/logging"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/supplier"
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/changes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var assetFilesystem embed.FS

type ShopAppConfig = app.ShopAppConfig

var SHOP = &ShopAppConfig{
	DBRequiredAppConfig: apps.NewDBAppConfig("shop"),
	ADMIN_ROUTE: admin.AdminSite.Route.Any(
		"shop/", nil, "shop",
	),
	SIGNALS: signals.NewManager(),
}

func NewAppConfig() (django.AppConfig, error) {
	if SHOP == nil || SHOP.IsReady() {
		return SHOP, nil
	}

	tplFS, err := fs.Sub(assetFilesystem, "assets/templates")
	if err != nil {
		return SHOP, err
	}

	staticFS, err := fs.Sub(assetFilesystem, "assets/static")
	if err != nil {
		return SHOP, err
	}

	SHOP.Deps = []string{
		"session", "auth", "admin", "editor", "blocks", "revisions",
	}

	staticfiles.AddFS(staticFS, filesystem.MatchAnd(
		filesystem.MatchPrefix("shop/"),
		filesystem.MatchOr(
			filesystem.MatchExt(".css"),
			filesystem.MatchExt(".js"),
			filesystem.MatchExt(".png"),
			filesystem.MatchExt(".jpg"),
			filesystem.MatchExt(".jpeg"),
			filesystem.MatchExt(".svg"),
			filesystem.MatchExt(".gif"),
			filesystem.MatchExt(".ico"),
		),
	))

	icons.Register(staticFS,
		"shop/admin/icons/funnel.svg",
	)

	SHOP.ModelObjects = []attrs.Definer{
		// Cart
		&models.CartItem{},
		&models.Cart{},

		// Orders
		&models.OrderState{},
		&models.OrderItem{},
		&models.Order{},

		// Payments
		&models.Payment{},

		// Products
		&models.Product{},

		// Categories
		&models.Category{},

		// Suppliers
		&supplier.SupplierModel{},
	}

	//	migrator.RegisterMigrateFunc(&models.Product{}, "0003_exec_test.mig", func(ctx context.Context, me *migrator.MigrationEngine, mt *migrator.ModelTable) error {
	//		var _, err = queries.GetQuerySet(&models.Product{}).Select("*").All()
	//		if err != nil {
	//			return err
	//		}
	//
	//		return fmt.Errorf("Enough testing now.")
	//	})

	SHOP.TemplateConfig = tpl.MergeConfig(
		&tpl.Config{
			AppName: "shop",
			FS:      tplFS,
			Bases: []string{
				"shop/admin/shop_base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("shop/"),
				filesystem.MatchOr(
					filesystem.MatchSuffix(".tmpl"),
				),
			),
		},
		admin.AdminSite.TemplateConfig,
	)

	SHOP.ADMIN_MENU = func(r *http.Request) []menu.MenuItem {
		return []menu.MenuItem{
			&menu.Item{
				ItemName: "index",
				Label:    trans.T(r.Context(), "Home"),
				Link: func() string {
					return django.Reverse("admin:apps", "shop")
				},
			},
			&menu.Item{
				ItemName: "products",
				Label:    trans.T(r.Context(), "Products"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "products")
				},
			},
			&menu.Item{
				ItemName: "orders",
				Label:    trans.T(r.Context(), "Orders"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "orders")
				},
			},
		}
	}

	SHOP.Routing = func(m mux.Multiplexer) {

	}

	SHOP.Init = func(settings django.Settings, db drivers.Database) error {

		goldcrest.Register(admin.RegisterRouteHook, 0, admin.RegisterRouteHookFunc(func(adminSite *admin.AdminApplication, route mux.Multiplexer) {
			sub := route.Any("commerce/", nil, "commerce")
			sub.Handle(
				mux.ANY, "products/add/<<supplier>>",
				views.Serve(app.ShopHandler{
					Shop:   SHOP,
					Handle: adminviews.ViewAddProduct,
				}),
				"new_with_supplier",
			)
		}))

		admin.RegisterApp(
			"shop",
			admin.AppOptions{
				AppLabel:            trans.S("Shop"),
				MenuLabel:           trans.S("Commerce"),
				FullAdminMenu:       true,
				RegisterToAdminMenu: true,
				MenuIcon: func() string {
					return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-cart4" viewBox="0 0 16 16">
  	<path d="M0 2.5A.5.5 0 0 1 .5 2H2a.5.5 0 0 1 .485.379L2.89 4H14.5a.5.5 0 0 1 .485.621l-1.5 6A.5.5 0 0 1 13 11H4a.5.5 0 0 1-.485-.379L1.61 3H.5a.5.5 0 0 1-.5-.5M3.14 5l.5 2H5V5zM6 5v2h2V5zm3 0v2h2V5zm3 0v2h1.36l.5-2zm1.11 3H12v2h.61zM11 8H9v2h2zM8 8H6v2h2zM5 8H3.89l.5 2H5zm0 5a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0m9-1a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0"/>
</svg>`
				},
			},
			admin.ModelOptions{
				Model:               &models.Category{},
				MenuLabel:           trans.S("Category"),
				RegisterToAdminMenu: true,
			},
			admin.ModelOptions{
				Name:                "orders",
				Model:               &models.Order{},
				MenuLabel:           trans.S("Orders"),
				RegisterToAdminMenu: true,
				ListView: admin.ListViewOptions{
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition) views.View {
						return adminviews.ViewOrderList
					},
				},
				AddView: admin.FormViewOptions{
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
						return app.ShopHandler{
							Shop:   SHOP,
							Handle: adminviews.ViewAddOrder,
						}
					},
				},
				EditView: admin.FormViewOptions{
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
						return app.ShopHandler{
							Shop:   SHOP,
							Handle: adminviews.ViewEditOrder,
						}
					},
				},
			},
			admin.ModelOptions{
				Name:                "products",
				Model:               &models.Product{},
				MenuLabel:           trans.S("Products"),
				RegisterToAdminMenu: true,
				ListView: admin.ListViewOptions{
					//	Prefetch: admin.Prefetch{
					//		SelectRelated: []string{"Category"},
					//	},
					//	ViewOptions: admin.ViewOptions{
					//		Fields: []string{"Title", "ID", "Price", "DiscountPrice", "Category", "Supplier"},
					//	},
					//	Columns: map[string]list.ListColumn[attrs.Definer]{
					//		"Title": list.FieldColumn[attrs.Definer](
					//			trans.S("Title"),
					//			"Title",
					//		),
					//		"ID": list.FieldColumn[attrs.Definer](
					//			trans.S("Product ID"),
					//			"ID",
					//		),
					//		"Price": list.FieldColumn[attrs.Definer](
					//			trans.S("Price"),
					//			"Price",
					//		),
					//		"DiscountPrice": list.FieldColumn[attrs.Definer](
					//			trans.S("Discount"),
					//			"DiscountPrice",
					//		),
					//		"Category": list.FieldColumn[attrs.Definer](
					//			trans.S("Category"),
					//			"Category",
					//		),
					//		"Supplier": list.FieldColumn[attrs.Definer](
					//			trans.S("Supplier"),
					//			"Supplier",
					//		),
					//	},
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition) views.View {
						return adminviews.ViewProductList
					},
				},
				AddView: admin.FormViewOptions{
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
						return app.ShopHandler{
							Shop:   SHOP,
							Handle: adminviews.ViewAddNewProductSupplier,
						}
					},
				},
				EditView: admin.FormViewOptions{
					GetHandler: func(adminSite *admin.AdminApplication, adminApp *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
						return app.ShopInstanceHandler[*models.Product]{
							Shop:     SHOP,
							Instance: instance.(*models.Product),
							Handle:   adminviews.ViewEditProduct,
						}
					},
				},
			},
		)
		// admin.RegisterSearchHook

		return nil
	}

	SHOP.Ready = func() error {

		// always run the sync on startup
		// /mostly/ handled in DB
		return errors.Join(
			supplier.Suppliers().Sync(),
		)
	}

	auditlogs.RegisterDefinition("shop:product:add", logging.NewShopLogDefinition())
	auditlogs.RegisterDefinition("shop:product:edit", logging.NewShopLogDefinition())

	SHOP.SIGNALS.Products.Created.Listen(context.Background(), func(ctx context.Context, s signals.Signal[*signals.ProductSignalData], psd *signals.ProductSignalData) error {
		_, err := auditlogs.Log(psd.Context, "shop:product:add", logger.INF, psd.Product.New, map[string]any{
			"title": psd.Product.New.Title,
		})
		return err
	})

	SHOP.SIGNALS.Products.Updated.Listen(context.Background(), func(ctx context.Context, s signals.Signal[*signals.ProductSignalData], psd *signals.ProductSignalData) error {
		changes, err := changes.Changes(ctx, psd.Product.Old, psd.Product.New)
		if err != nil {
			return err
		}

		_, err = auditlogs.Log(psd.Context, "shop:product:edit", logger.INF, psd.Product.New, map[string]any{
			"title":   psd.Product.New.Title,
			"changes": changes,
		})
		return err
	})

	return SHOP, nil
}
