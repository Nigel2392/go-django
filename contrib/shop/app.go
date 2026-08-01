package shop

import (
	"embed"
	"io/fs"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var assetFilesystem embed.FS

var SHOP = &ShopAppConfig{
	DBRequiredAppConfig: apps.NewDBAppConfig("shop"),
	ADMIN_ROUTE: admin.AdminSite.Route.Any(
		"products/", nil, "products",
	),
	SIGNALS: newSignalManager(),
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
		"session", "auth", "admin",
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
		&models.Product{},
		&models.ProductSku{},
		&models.Cart{},
		&models.CartItem{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
	}

	SHOP.TemplateConfig = tpl.MergeConfig(
		&tpl.Config{
			AppName: "shop",
			FS:      tplFS,
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("shop/"),
				filesystem.MatchOr(
					filesystem.MatchSuffix(".tmpl"),
				),
			),
		},
		admin.AdminSite.TemplateConfig,
	)

	SHOP.Routing = func(m mux.Multiplexer) {

	}

	SHOP.Init = func(settings django.Settings, db drivers.Database) error {
		return nil
	}

	return SHOP, nil
}
