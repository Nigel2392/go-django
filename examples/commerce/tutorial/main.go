package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/settings"
	"github.com/Nigel2392/go-django/contrib/shop"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/examples/commerce/src/suppliers/dbsupplier"
	"github.com/Nigel2392/go-django/examples/commerce/tutorial/paymentproviders"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var shopCurrency = currency.EUR

var _ orders.SupplierBackend = (*dbsupplier.DBSupplierAppConfig)(nil)
var _ products.ProductModel = (*dbsupplier.Product)(nil)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	drivers.LOG_SQL_QUERIES = false
	db, err := drivers.Open(ctx, "sqlite3", "file:commerce_tutorial?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		return err
	}
	defer db.Close()

	mediafiles.RegisterBackend("memory", memory.NewBackend(1))
	mediafiles.SetDefault("memory")
	provider := paymentproviders.RegisterDemo()

	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE:    db,
			shop.APPVAR_SHOP_CURRENCY: shopCurrency,
		}),
		django.Flag(django.FlagSkipChecks, django.FlagSkipCmds),
		django.Apps(
			session.NewAppConfig, messages.NewAppConfig, auth.NewAppConfig,
			admin.NewAppConfig, reports.NewAppConfig, auditlogs.NewAppConfig,
			editor.NewAppConfig, blocks.NewAppConfig, revisions.NewAppConfig,
			settings.NewAppConfig, dbsupplier.NewAppConfig, shop.NewAppConfig,
			tutorialSchema,
		),
	)
	if err := app.Initialize(); err != nil {
		return err
	}

	opts, err := seedCheckout(ctx)
	if err != nil {
		return err
	}
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
	result, err := orders.CheckoutCart(req, opts)
	if err != nil {
		return err
	}
	fmt.Printf("Prepared order %s with payment %s for %s %s\n",
		result.Order.ID, result.Payment.ID, result.Payment.Currency, result.Payment.Amount.StringFixed(2))
	fmt.Printf("Opened checkout in %s at %s\n", result.Checkout.State, result.Checkout.URL)

	if err := provider.MarkPaid(result.Checkout.ProviderRef); err != nil {
		return err
	}
	payment, err := payments.Refresh(ctx, result.Payment.ID)
	if err != nil {
		return err
	}
	order, err := orders.Query().WithContext(ctx).Filter("ID", result.Order.ID).Get()
	if err != nil {
		return err
	}
	fmt.Printf("Reconciled payment in %s. Order is %s.\n", payment.State, order.Object.State)
	return nil
}

func tutorialSchema() *apps.DBRequiredAppConfig {
	config := apps.NewDBAppConfig("commerce_tutorial")
	config.Init = func(_ django.Settings, db drivers.Database) error {
		schema := sqlite.NewSQLiteSchemaEditor(db)
		for _, model := range []attrs.Definer{
			&auth.User{}, &auditlogs.Entry{}, &models.Category{}, &orders.SupplierModel{},
			&products.Product{}, &dbsupplier.Product{}, &customers.Customer{},
			&carts.Cart{}, &carts.CartItem{}, &orders.Order{}, &orders.OrderItem{},
			&coupons.Coupon{}, &coupons.CouponProductThrough{}, &coupons.Redemption{},
			&payments.ProviderAccount{}, &payments.Payment{},
		} {
			if err := schema.CreateTable(context.Background(), migrator.NewModelTable(model), true); err != nil {
				return fmt.Errorf("create %T table: %w", model, err)
			}
		}
		return nil
	}
	return config
}

func seedCheckout(ctx context.Context) (orders.CheckoutOptions, error) {
	var opts orders.CheckoutOptions
	category, err := queries.GetQuerySet(&models.Category{}).WithContext(ctx).Create(&models.Category{
		Name: "Books",
		Slug: "books",
	})
	if err != nil {
		return opts, err
	}
	product := &dbsupplier.Product{
		Product: &products.Product{
			Title: "Notebook", Slug: "notebook", Category: category,
			Supplier: products.SupplierValue(dbsupplier.CONFIG.InternalName()),
			Image:    &mediafiles.SimpleStoredObject{Filepath: "notebook.png"},
			Editor:   &editor.EditorJSBlockData{},
			Price:    decimal.NewFromInt(10),
			TaxRate:  decimal.NewFromInt(20),
		},
		SupplierID: "NOTEBOOK",
	}
	product.Instance = product
	if _, err := queries.GetQuerySet(&dbsupplier.Product{}).WithContext(ctx).Create(product); err != nil {
		return opts, err
	}
	if _, err := queries.GetQuerySet(&products.Product{}).WithContext(ctx).Create(product.Product); err != nil {
		return opts, err
	}

	cart, err := queries.GetQuerySet(&carts.Cart{}).WithContext(ctx).Create(&carts.Cart{
		Session: "tutorial", Active: true,
	})
	if err != nil {
		return opts, err
	}
	if _, err := queries.GetQuerySet(&carts.CartItem{}).WithContext(ctx).Create(&carts.CartItem{
		Cart: cart, Product: product.Product, Amount: 2,
	}); err != nil {
		return opts, err
	}
	if _, err := carts.SetCustomer(ctx, cart, carts.CustomerDetails{
		FirstName: "Sam",
		LastName:  "Example",
		Email:     "sam@example.test",
	}); err != nil {
		return opts, err
	}

	if err := payments.ProvidersQuery().WithContext(ctx).Sync(); err != nil {
		return opts, err
	}
	account, err := queries.GetQuerySet(&payments.ProviderAccount{}).
		WithContext(ctx).
		Filter("InternalName", paymentproviders.DemoDriver).
		Get()
	if err != nil {
		return opts, err
	}
	const origin = "https://merchant.example.test"
	return orders.CheckoutOptions{
		Cart:      cart,
		AccountID: account.Object.ID,
		Currency:  shopCurrency,
		ReturnURL: func(paymentID, _ drivers.ULID) string {
			return origin + "/checkout/return?payment_id=" + paymentID.String()
		},
		WebhookURL: func(a *payments.ProviderAccount) string {
			return origin + django.Reverse("api:payments:webhook", a.ID)
		},
	}, nil
}
