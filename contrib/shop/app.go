package shop

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
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

var NewHandler = app.ShopToHttpHandlerBuilder(SHOP)

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
		"session", "auth", "admin", "editor",
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

	SHOP.ModelObjects = []attrs.Definer{
		// Cart
		&models.CartItem{},
		&models.Cart{},

		// Orders
		&models.OrderItem{},
		&models.Order{},

		// Payments
		&models.Payment{},

		// Products
		&models.ProductSku{},
		&models.Product{},
	}

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
				BaseItem: menu.BaseItem{
					ItemName: "index",
					Label:    trans.T(r.Context(), "Home"),
				},
				Link: func() string {
					return django.Reverse("admin:shop:home")
				},
			},
			&menu.Item{
				BaseItem: menu.BaseItem{
					ItemName: "products",
					Label:    trans.T(r.Context(), "Products"),
				},
				Link: func() string {
					return django.Reverse("admin:shop:products")
				},
			},
		}
	}

	SHOP.Routing = func(m mux.Multiplexer) {
		m.Get("home", NewHandler(adminviews.Home), "home")

		products := m.Get("products/", NewHandler(adminviews.ViewProductList.ServeHTTP), "products")
		products.Get("add/", NewHandler(adminviews.ViewAddProduct), "add")
		products.Post("add/", NewHandler(adminviews.ViewAddProduct))
		products.Get("edit/<<product_id>>/", NewHandler(adminviews.ViewEditProduct), "edit")
		products.Post("edit/<<product_id>>/", NewHandler(adminviews.ViewEditProduct))

	}

	SHOP.Init = func(settings django.Settings, db drivers.Database) error {

		// admin.RegisterSearchHook

		// always run the sync on startup
		// /mostly/ handled in DB
		return models.Products().SyncProducts()
	}

	return SHOP, nil
}
