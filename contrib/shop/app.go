package shop

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/admin/icons"
	"github.com/Nigel2392/go-django/contrib/admin/searches"
	"github.com/Nigel2392/go-django/contrib/shop/adminsite"
	"github.com/Nigel2392/go-django/contrib/shop/core/handlers"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/contrib/shop/views/adminviews"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux/v2"
	"github.com/justinas/nosurf"
	"golang.org/x/text/currency"
)

//go:embed assets/**
var assetFilesystem embed.FS

//go:embed translations.yml
var translationsFS embed.FS

const APPVAR_SHOP_CURRENCY = "SHOP_CURRENCY"

const (
	APPVAR_SHOP_CHECKOUT_ORIGIN      = "SHOP_CHECKOUT_ORIGIN"
	APPVAR_SHOP_CHECKOUT_RETURN_PATH = "SHOP_CHECKOUT_RETURN_PATH"
)

type ShopAppConfig struct {
	*apps.DBRequiredAppConfig
	ADMIN_ROUTE *mux.Route
	ADMIN_MENU  func(r *http.Request) []menu.MenuItem
}

func (a *ShopAppConfig) BuildRouting(m mux.Multiplexer) {
	if a.ADMIN_ROUTE == nil {
		panic(fmt.Sprintf("admin route not set for %T", a))
	}

	a.Routing(a.ADMIN_ROUTE)
}

func (a *ShopAppConfig) OnReady() error {
	if err := a.DBRequiredAppConfig.OnReady(); err != nil {
		return err
	}

	return nil
}

func (a *ShopAppConfig) TranslationsFS(settings django.Settings) []fs.FS {
	return []fs.FS{translationsFS}
}

var SHOP = &ShopAppConfig{
	DBRequiredAppConfig: apps.NewDBAppConfig("shop"),
	ADMIN_ROUTE: admin.AdminSite.Route.Any(
		"shop/", nil, "shop",
	),
}

func NewAppConfig() (django.AppConfig, error) {
	if SHOP == nil || SHOP.IsReady() {
		return SHOP, nil
	}

	supplierType := contenttypes.NewContentType(&orders.SupplierModel{}).TypeName()
	contenttypes.RegisterAlias("github.com/Nigel2392/go-django/contrib/shop/supplier.SupplierModel", supplierType)
	contenttypes.RegisterAlias("github.com/Nigel2392/go-django/contrib/shop/models/products/supplier.SupplierModel", supplierType)

	tplFS, err := fs.Sub(assetFilesystem, "assets/templates")
	if err != nil {
		return SHOP, err
	}

	staticFS, err := fs.Sub(assetFilesystem, "assets/static")
	if err != nil {
		return SHOP, err
	}

	SHOP.Deps = []string{
		"session", "admin", "editor",
		"blocks", "revisions", "settings",
	}

	SHOP.Cmd = []command.Command{
		syncPSPCmd,
		syncSuppliersCmd,
	}

	static.AddFS(staticFS, filesystem.MatchAnd(
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

	handlers.RegisterAuditLogs()

	SHOP.ModelObjects = []attrs.Definer{
		// Cart
		&carts.CartItem{},
		&carts.Cart{},

		// Orders
		&orders.OrderState{},
		&orders.OrderItem{},
		&orders.Order{},

		// Payments
		&payments.ProviderAccount{},
		&payments.Payment{},

		// Products
		&products.Product{},

		// Categories
		&categories.Category{},

		// Suppliers
		&orders.SupplierModel{},

		// Coupons
		&coupons.Coupon{},
		&coupons.CouponProductThrough{},
		&coupons.Redemption{},

		// Customers
		&customers.Customer{},
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
			&menu.Item{
				ItemName: "coupons",
				Label:    trans.T(r.Context(), "Coupons"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "coupons")
				},
			},
			&menu.Item{
				ItemName: "customers",
				Label:    trans.T(r.Context(), "Customers"),
				Hidden:   !permissions.HasObjectPermission(r, &customers.Customer{}, "admin:list"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "customers")
				},
			},
			&menu.Item{
				ItemName: "payments",
				Label:    trans.T(r.Context(), "Payments"),
				Hidden:   !permissions.HasObjectPermission(r, &payments.Payment{}, "admin:list"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "payments")
				},
			},
			&menu.Item{
				ItemName: "carts",
				Label:    trans.T(r.Context(), "Carts"),
				Hidden:   !permissions.HasObjectPermission(r, &carts.Cart{}, "admin:list"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "carts")
				},
			},
			&menu.Item{
				ItemName: "redemptions",
				Label:    trans.T(r.Context(), "Coupon redemptions"),
				Hidden:   !permissions.HasObjectPermission(r, &coupons.Redemption{}, "admin:list"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "redemptions")
				},
			},
			&menu.Item{
				ItemName: "order_states",
				Label:    trans.T(r.Context(), "Order states"),
				Hidden:   !permissions.HasObjectPermission(r, &orders.OrderState{}, "admin:list"),
				Link: func() string {
					return django.Reverse("admin:apps:model", "shop", "order_states")
				},
			},
		}
	}

	SHOP.Routing = func(m mux.Multiplexer) {

	}

	chooser.Register(&chooser.ChooserDefinition[*products.Product]{
		Model: &products.Product{},
		Title: trans.S("Product Chooser"),
		PreviewString: func(ctx context.Context, instance *products.Product) string {
			return instance.Title
		},
		ListPage: &chooser.ChooserListPage[*products.Product]{
			Fields: []string{"Title", "Supplier", "Price", "TaxRate", "DiscountPrice"},
			QuerySet: func(r *http.Request, model *products.Product) (*queries.QuerySet[*products.Product], error) {
				return queries.GetQuerySet(model).OrderBy("Title", "-Price"), nil
			},
			SearchFields: []searches.SearchField{
				{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
			},
		},
	})

	SHOP.Init = func(djangoSettings django.Settings, db drivers.Database) error {

		goldcrest.Register(admin.RegisterRouteHook, 0, admin.RegisterRouteHookFunc(func(adminSite *admin.AdminApplication, route mux.Multiplexer) {
			sub := route.Any("commerce/", nil, "commerce")
			sub.Handle(
				mux.ANY, "products/add/<<supplier>>",
				views.Serve(ShopHandler{
					Handle: adminviews.ViewAddProduct,
				}),
				"new_with_supplier",
			)
		}))

		goldcrest.Register(admin.RegisterHomePageDisplayPanelHook, 2, admin.RegisterHomePageDisplayPanelHookFunc(func(*http.Request, *admin.AdminApplication) []admin.DisplayPanel {
			return []admin.DisplayPanel{{
				IconName: "icon-file-document",
				Title: func(ctx context.Context, count int64) string {
					return trans.P(ctx, "Product", "Products", count)
				},
				QuerySet: func(r *http.Request) *queries.QuerySet[attrs.Definer] {
					return queries.GetQuerySet[attrs.Definer](&products.Product{})
				},
				URL: func(r *http.Request) string {
					return django.Reverse("admin:apps:model", "documents", "document")
				},
			}}
		}))

		app := admin.RegisterApp(
			"shop",
			admin.AppOptions{
				AppLabel:            trans.S("Shop"),
				MenuLabel:           trans.S("Commerce"),
				FullAdminMenu:       true,
				RegisterToAdminMenu: true,
				EnableIndexView:     true,
				AppDescription:      trans.S("The commerce module is intended to manage your webshop, such as product stock, orders, refunds, etc."),
				MenuIcon: func() string {
					return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-cart4" viewBox="0 0 16 16">
  	<path d="M0 2.5A.5.5 0 0 1 .5 2H2a.5.5 0 0 1 .485.379L2.89 4H14.5a.5.5 0 0 1 .485.621l-1.5 6A.5.5 0 0 1 13 11H4a.5.5 0 0 1-.485-.379L1.61 3H.5a.5.5 0 0 1-.5-.5M3.14 5l.5 2H5V5zM6 5v2h2V5zm3 0v2h2V5zm3 0v2h1.36l.5-2zm1.11 3H12v2h.61zM11 8H9v2h2zM8 8H6v2h2zM5 8H3.89l.5 2H5zm0 5a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0m9-1a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0"/>
</svg>`
				},
			},
			adminsite.CategoryAdmin,
			adminsite.ProductAdmin,
			adminsite.OrdersAdmin,
			adminsite.CouponsAdmin,
			adminsite.CustomersAdmin,
		)

		// app.Register(admin.ModelOptions{
		// 	Name:                "payments",
		// 	Model:               &payments.Payment{},
		// 	MenuOrder:           700,
		// 	MenuLabel:           trans.S("Payments"),
		// 	RegisterToAdminMenu: true,
		// 	DisallowCreate:      true,
		// 	DisallowEdit:        true,
		// 	DisallowDelete:      true,
		// 	ListView: admin.ListViewOptions{
		// 		ViewOptions: admin.ViewOptions{
		// 			Fields: []string{"ID", "Order", "Account", "Provider", "TransactionID", "Amount", "Fee", "Currency", "State", "CreatedAt"},
		// 		},
		// 		Ordering: []string{"-CreatedAt"},
		// 		Search: &searches.SearchOptions{
		// 			Fields: []searches.SearchField{
		// 				{Name: "ID", Lookup: expr.LOOKUP_ICONTANS},
		// 				{Name: "TransactionID", Lookup: expr.LOOKUP_ICONTANS},
		// 				{Name: "Provider", Lookup: expr.LOOKUP_ICONTANS},
		// 				{Name: "State", Lookup: expr.LOOKUP_ICONTANS},
		// 			},
		// 		},
		// 	},
		// })

		// app.Register(admin.ModelOptions{
		// 	Name:                "carts",
		// 	Model:               &carts.Cart{},
		// 	MenuOrder:           800,
		// 	MenuLabel:           trans.S("Carts"),
		// 	RegisterToAdminMenu: true,
		// 	DisallowCreate:      true,
		// 	DisallowEdit:        true,
		// 	DisallowDelete:      true,
		// 	ListView: admin.ListViewOptions{
		// 		ViewOptions: admin.ViewOptions{Fields: []string{"ID", "Active", "CreatedAt"}},
		// 		Ordering:    []string{"-CreatedAt"},
		// 	},
		// })

		// app.Register(admin.ModelOptions{
		// 	Name:                "redemptions",
		// 	Model:               &coupons.Redemption{},
		// 	MenuOrder:           900,
		// 	MenuLabel:           trans.S("Coupon redemptions"),
		// 	RegisterToAdminMenu: true,
		// 	DisallowCreate:      true,
		// 	DisallowEdit:        true,
		// 	DisallowDelete:      true,
		// 	ListView: admin.ListViewOptions{
		// 		ViewOptions: admin.ViewOptions{
		// 			Fields: []string{"ID", "Coupon", "DiscountAmount", "Invalidated", "RedeemedAt"},
		// 		},
		// 		Ordering: []string{"-RedeemedAt"},
		// 	},
		// })

		// app.Register(admin.ModelOptions{
		// 	Name:                "order_states",
		// 	Model:               &orders.OrderState{},
		// 	MenuOrder:           1000,
		// 	MenuLabel:           trans.S("Order states"),
		// 	RegisterToAdminMenu: true,
		// 	ListView: admin.ListViewOptions{
		// 		ViewOptions: admin.ViewOptions{Fields: []string{"ID", "Label", "Action", "WorkflowID"}},
		// 	},
		// 	AddView: admin.FormViewOptions{
		// 		ViewOptions:  admin.ViewOptions{Fields: []string{"ID", "Label", "Action", "WorkflowID"}},
		// 		SaveInstance: createModel,
		// 		Panels: []admin.Panel{
		// 			admin.FieldPanel("ID"),
		// 			admin.FieldPanel("Label"),
		// 			admin.FieldPanel("Action"),
		// 			admin.FieldPanel("WorkflowID"),
		// 		},
		// 	},
		// 	EditView: admin.FormViewOptions{
		// 		ViewOptions:  admin.ViewOptions{Fields: []string{"Label", "Action", "WorkflowID"}},
		// 		SaveInstance: updateModel,
		// 		Panels: []admin.Panel{
		// 			admin.FieldPanel("Label"),
		// 			admin.FieldPanel("Action"),
		// 			admin.FieldPanel("WorkflowID"),
		// 		},
		// 	},
		// 	DeleteView: admin.DeleteViewOptions{DeleteInstances: deleteModels},
		// })

		app.Routing = func(m mux.Multiplexer) {
			// admin:ext:shop
			shopAdminApi := m.Any("/shop", nil, "shop")

			// admin:ext:shop:generate_coupon
			shopAdminApi.Post("/coupon/generate", http.HandlerFunc(adminviews.GenerateCouponCode), "generate_coupon")
		}

		return nil
	}

	SHOP.Ready = func() error {
		baseApiRoute := django.Global.Mux.Get("/api", nil, "api")

		/*
			api:carts			  GET  								   -> [carts.CartRow]
			api:carts:clear		  POST 								   -> [carts.CartRow]
			api:carts:customer	  POST -> [public.CartCustomerRequest] -> [carts.CartRow]
			api:carts:add_item	  POST -> [public.CartItemRequest]     -> [public.CartItemResponse]
			api:carts:remove_item POST -> [public.CartItemRequest]     -> [public.CartItemResponse]
		*/
		c := public.CartAPI{}
		cartBase := baseApiRoute.Get("/carts", http.HandlerFunc(c.GetOrCreateCart), "carts")
		cartBase.Post("/clear", http.HandlerFunc(c.ClearCart), "clear")
		cartBase.Post("/customer", http.HandlerFunc(c.SetCustomer), "customer")
		cartBase.Post("/items/add", http.HandlerFunc(c.AddCartItem), "add_item")
		cartBase.Post("/items/remove", http.HandlerFunc(c.RemoveCartItem), "remove_item")

		/*
			api:coupons:redeem POST -> [RedemptionRequest] -> [carts.CartRow]
		*/
		redemption := public.RedemptionAPI{
			ErrInvalid:            trans.S("Invalid coupon code"),
			ErrNoMatchingProducts: trans.S("No products in your cart match this coupon"),
			ErrRedeemed:           trans.S("This coupon has reached its usage limit"),
		}

		coupons := baseApiRoute.Get("/coupons", nil, "coupons")
		coupons.Post("/redeem", http.HandlerFunc(redemption.RedeemCode), "redeem")

		/*
			api:payments:options	   GET   	   -> [][public.PaymentOption]
			api:payments:checkout      POST         -> [public.PaymentCheckoutRequest] -> provider redirect
			api:payments:webhook, <ID> POST -> ??? -> [payments.Provider.HandleWebhook]
		*/
		var paymentsHandler = func(w http.ResponseWriter, r *http.Request) {
			payments.ServeWebhook(w, r, mux.Vars(r).Get("account"))
		}

		unit, getCurrency := resolveShopCurrency()
		p := public.PaymentsAPI{
			Currency:    unit,
			GetCurrency: getCurrency,
		}
		checkout := public.CheckoutAPI{
			Currency:    unit,
			GetCurrency: getCurrency,
			Origin:      djangoapps.CONFIG.Get(APPVAR_SHOP_CHECKOUT_ORIGIN, ""),
			ReturnPath:  djangoapps.CONFIG.Get(APPVAR_SHOP_CHECKOUT_RETURN_PATH, "/"),
		}
		pay := baseApiRoute.Get("/payments", nil, "payments")
		pay.Get("/options", http.HandlerFunc(p.PaymentOptions), "options")
		pay.Post("/checkout", http.HandlerFunc(checkout.Checkout), "checkout")
		pay.Post("/webhook/<<account>>", http.HandlerFunc(paymentsHandler), "webhook")

		goldcrest.Register(django.HOOK_SETUP_NOSURF, 0, django.NosurfSetupHook(func(_ *django.Application, h *nosurf.CSRFHandler) {
			h.ExemptRegexp(`^/api/payments/webhook/[0-9A-HJKMNP-TV-Z]{26}/?$`)
		}))

		return nil
	}

	return SHOP, nil
}

func resolveShopCurrency() (currency.Unit, func(context.Context) currency.Unit) {
	if djangoapps.CONFIG.ApplicationSettings == nil {
		return currency.Unit{}, nil
	}
	raw, ok := djangoapps.CONFIG.ApplicationSettings.Get(APPVAR_SHOP_CURRENCY)
	if !ok || raw == nil {
		return currency.Unit{}, nil
	}
	switch v := raw.(type) {
	case currency.Unit:
		return v, nil
	case string:
		unit, err := currency.ParseISO(v)
		if err != nil || unit.String() != v {
			return currency.Unit{}, nil
		}
		return unit, nil
	case func(context.Context) currency.Unit:
		return currency.Unit{}, v
	default:
		return currency.Unit{}, nil
	}
}
