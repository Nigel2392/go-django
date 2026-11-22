//go:build test
// +build test

package orders_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/products/supplier"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	qerrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/fields"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	gosignals "github.com/Nigel2392/go-signals"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type OrderTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
}

func (u *OrderTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("order_test_users")
}

func (u *OrderTestUser) IsAuthenticated() bool {
	return u.LoggedIn
}

func (u *OrderTestUser) IsAdmin() bool {
	return u.IsAdministrator
}

type testOrderSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *testOrderSupplierProduct) ProductPrice(ctx context.Context) supplier.Price {
	return supplier.Price{
		Unit:   currency.EUR,
		Amount: p.Price,
	}
}

func (p *testOrderSupplierProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*testOrderSupplierProduct, any](ctx, p,
		fields.Embed(ctx, &p.Product, fields.EmbedOptions{
			EmbedFields: []any{
				"ID",
				"Image",
				"Title",
				"Slug",
				"Editor",
				"Category",
				"Supplier",
				"Price",
				"TaxRate",
				"DiscountPrice",
				"Ranking",
			},
		}),
		fattrs.Field(p, "SKU", &p.SKU, func() fattrs.PtrFieldConfig[*testOrderSupplierProduct, string] {
			return fattrs.PtrFieldConfig[*testOrderSupplierProduct, string]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
	)
}

func setupOrderTestEnv(t *testing.T) {
	t.Helper()

	_, db := testdb.Open()

	mediafiles.RegisterBackend("memory", memory.NewBackend(5))
	mediafiles.SetDefault("memory")

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &OrderTestUser{},
		GetObject:      func() any { return &OrderTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "Order Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "Order Test Users" },
	})

	supplier.Register(&supplier.Supplier{
		Order: 1,
		Name:  "order_test_supplier",
		TextLabel: func(ctx context.Context) string {
			return "Order Test Supplier"
		},
		ProductModel: func(ctx context.Context) supplier.ProductModel {
			return &testOrderSupplierProduct{}
		},
	})

	if django.Global == nil {
		app := django.App(
			django.Configure(map[string]any{
				django.APPVAR_DATABASE: db,
			}),
			django.Flag(
				django.FlagSkipDepsCheck,
				django.FlagSkipChecks,
				django.FlagSkipCmds,
			),
			django.Apps(func() django.AppConfig {
				cfg := apps.NewDBAppConfig("shop")
				cfg.ModelObjects = []attrs.Definer{
					&OrderTestUser{},
					&models.Category{},
					&supplier.SupplierModel{},
					&testOrderSupplierProduct{},
					&products.Product{},
					&carts.Cart{},
					&carts.CartItem{},
					&payments.ProviderAccount{},
					&orders.OrderState{},
					&orders.Order{},
					&payments.Payment{},
					&orders.OrderItem{},
				}
				return cfg
			}),
		)
		if err := app.Initialize(); err != nil {
			t.Fatalf("failed to initialize django app: %v", err)
		}
	}

	tables := quest.Table(t,
		&OrderTestUser{},
		&models.Category{},
		&supplier.SupplierModel{},
		&testOrderSupplierProduct{},
		&products.Product{},
		&carts.Cart{},
		&carts.CartItem{},
		&payments.ProviderAccount{},
		&customers.Customer{},
		&orders.OrderState{},
		&orders.Order{},
		&payments.Payment{},
		&orders.OrderItem{},
	)
	tables.IfNotExists = true
	tables.Create()

}

func seedTestCategory(t *testing.T, name, slug string) *models.Category {
	t.Helper()
	cat, err := queries.GetQuerySet(&models.Category{}).Create(&models.Category{
		Name: name,
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("failed to create category %q: %v", name, err)
	}
	return cat
}

func seedTestProduct(t *testing.T, cat *models.Category, title, slug, sku, price, tax, discount string) *products.Product {
	t.Helper()
	spec := &testOrderSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      supplier.SupplierValue("order_test_supplier"),
			Price:         decimal.RequireFromString(price),
			TaxRate:       decimal.RequireFromString(tax),
			DiscountPrice: decimal.RequireFromString(discount),
			Ranking:       1,
		},
		SKU: sku,
	}
	spec.Product.Instance = spec

	if _, err := queries.GetQuerySet(&testOrderSupplierProduct{}).Create(spec); err != nil {
		t.Fatalf("failed to create specific product %q: %v", title, err)
	}
	prod, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product)
	if err != nil {
		t.Fatalf("failed to create base product %q: %v", title, err)
	}
	return prod
}

func seedTestCart(t *testing.T, sessionToken string) *carts.Cart {
	t.Helper()
	created, err := queries.GetQuerySet(&carts.Cart{}).Create(&carts.Cart{
		Session: sessionToken,
		Active:  true,
	})
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}
	return created
}

func seedTestCartItem(t *testing.T, cart *carts.Cart, prod *products.Product, amount uint64) *carts.CartItem {
	t.Helper()
	item, err := queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
		Cart:    cart,
		Product: prod,
		Amount:  amount,
	})
	if err != nil {
		t.Fatalf("failed to create cart item: %v", err)
	}
	item.Product = prod
	return item
}

func clearOrderAndCartTables(t *testing.T) {
	t.Helper()
	if _, err := queries.GetQuerySet(&orders.OrderItem{}).Delete(); err != nil {
		t.Fatalf("failed to clear OrderItem table: %v", err)
	}
	if _, err := queries.GetQuerySet(&orders.Order{}).Delete(); err != nil {
		t.Fatalf("failed to clear Order table: %v", err)
	}
	if _, err := queries.GetQuerySet(&carts.CartItem{}).Delete(); err != nil {
		t.Fatalf("failed to clear CartItem table: %v", err)
	}
	if _, err := queries.GetQuerySet(&carts.Cart{}).Delete(); err != nil {
		t.Fatalf("failed to clear Cart table: %v", err)
	}
}

func listenOrderCreated(
	t *testing.T,
	fn func(context.Context, signals.Signal[orders.OrderCreatedSignal], orders.OrderCreatedSignal) error,
) {
	t.Helper()
	sigCtx, cancelSig := context.WithCancel(t.Context())
	recv, err := orders.Signals.Created.Listen(sigCtx, fn)
	if err != nil {
		cancelSig()
		t.Fatalf("failed to register Orders.Created listener: %v", err)
	}
	t.Cleanup(func() {
		if err := orders.Signals.Created.Disconnect(context.Background(), recv); err != nil {
			t.Fatalf("failed to disconnect Orders.Created listener: %v", err)
		}
		cancelSig()
	})
}

func TestCreateOrder(t *testing.T) {
	setupOrderTestEnv(t)

	cat := seedTestCategory(t, "Peripherals", "peripherals")
	prodKeyboard := seedTestProduct(t, cat, "Mechanical Keyboard", "mechanical-keyboard", "SKU-KB-01", "129.99", "21.00", "109.99")
	prodMouse := seedTestProduct(t, cat, "Wireless Mouse", "wireless-mouse", "SKU-MS-02", "59.50", "21.00", "0.00")
	prodMat := seedTestProduct(t, cat, "Desk Mat XL", "desk-mat-xl", "SKU-DM-03", "24.95", "21.00", "19.95")

	customer, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
		Email:     drivers.MustParseEmail("admin@localhost"),
		FirstName: "John",
		LastName:  "Doe",
	})

	if err != nil {
		t.Fatalf("error creating customer: %v", err)
	}

	t.Run("Explicit variadic items creates Order and OrderItems without querying CartItems", func(t *testing.T) {
		clearOrderAndCartTables(t)

		// Create an empty cart in DB (no CartItems stored in DB for this cart)
		cart := seedTestCart(t, "sess-explicit-items")

		// Pass explicit CartItems directly
		explicitItem1 := &carts.CartItem{
			ID:      drivers.NewULID(),
			Cart:    cart,
			Product: prodKeyboard,
			Amount:  2,
		}
		explicitItem2 := &carts.CartItem{
			ID:      drivers.NewULID(),
			Cart:    cart,
			Product: prodMouse,
			Amount:  3,
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart, explicitItem1, explicitItem2)
		if err != nil {
			t.Fatalf("CreateOrder with explicit items failed: %v", err)
		}

		// Verify returned Order
		if order == nil {
			t.Fatal("expected non-nil Order")
		}
		if order.ID.IsZero() {
			t.Fatal("expected non-zero Order.ID")
		}
		if string(order.State) != "order_pending" {
			t.Fatalf("expected Order.State 'order_pending', got %q", order.State)
		}
		if gotLabel := order.State.Label(t.Context()); gotLabel != "Order Pending" {
			t.Fatalf("expected Order.State.Label 'Order Pending', got %q", gotLabel)
		}
		if order.Completed {
			t.Fatal("expected Order.Completed to be false")
		}
		if order.Processed {
			t.Fatal("expected Order.Processed to be false")
		}
		if len(order.Payments.AsList()) != 0 {
			t.Fatalf("expected Order.Payments to be empty, got %+v", order.Payments.AsList())
		}

		// Verify returned OrderItems
		if len(orderItems) != 2 {
			t.Fatalf("expected 2 OrderItems, got %d", len(orderItems))
		}
		if orderItems[0].ID.IsZero() || orderItems[1].ID.IsZero() {
			t.Fatal("expected non-zero IDs on returned OrderItems")
		}
		if orderItems[0].ID == orderItems[1].ID {
			t.Fatal("expected distinct IDs for each OrderItem")
		}

		// Item 1 checks
		if orderItems[0].Order == nil || orderItems[0].Order.ID != order.ID {
			t.Fatalf("expected orderItems[0].Order.ID == %s", order.ID)
		}
		if !orderItems[0].ItemPrice.Equal(decimal.RequireFromString("129.99")) {
			t.Fatalf("expected orderItems[0].ItemPrice 129.99, got %s", orderItems[0].ItemPrice)
		}
		if orderItems[0].Amount != 2 {
			t.Fatalf("expected orderItems[0].Amount 2, got %d", orderItems[0].Amount)
		}
		if orderItems[0].Product.Data.ID != prodKeyboard.ID {
			t.Fatalf("expected orderItems[0].Product.Data.ID %s, got %s", prodKeyboard.ID, orderItems[0].Product.Data.ID)
		}
		if orderItems[0].Product.Data.Title != "Mechanical Keyboard" {
			t.Fatalf("expected orderItems[0].Product.Data.Title 'Mechanical Keyboard', got %q", orderItems[0].Product.Data.Title)
		}

		// Item 2 checks
		if orderItems[1].Order == nil || orderItems[1].Order.ID != order.ID {
			t.Fatalf("expected orderItems[1].Order.ID == %s", order.ID)
		}
		if !orderItems[1].ItemPrice.Equal(decimal.RequireFromString("59.50")) {
			t.Fatalf("expected orderItems[1].ItemPrice 59.50, got %s", orderItems[1].ItemPrice)
		}
		if orderItems[1].Amount != 3 {
			t.Fatalf("expected orderItems[1].Amount 3, got %d", orderItems[1].Amount)
		}
		if orderItems[1].Product.Data.ID != prodMouse.ID {
			t.Fatalf("expected orderItems[1].Product.Data.ID %s, got %s", prodMouse.ID, orderItems[1].Product.Data.ID)
		}

		// Verify persistence in DB via Orders() queryset including Preload("OrderItems")
		dbOrderRow, err := orders.Query().Preload("OrderItems").Filter("ID", order.ID).Get()
		if err != nil {
			t.Fatalf("failed to fetch Order from DB: %v", err)
		}
		if dbOrderRow.Object.ID != order.ID {
			t.Fatalf("expected persisted Order ID %s, got %s", order.ID, dbOrderRow.Object.ID)
		}
		if string(dbOrderRow.Object.State) != "order_pending" {
			t.Fatalf("expected persisted Order State 'order_pending', got %q", dbOrderRow.Object.State)
		}
		if dbOrderRow.Object.Completed {
			t.Fatal("expected persisted Order Completed to be false")
		}
		if dbOrderRow.Object.Processed {
			t.Fatal("expected persisted Order Processed to be false")
		}
		if len(dbOrderRow.Object.Payments.AsList()) != 0 {
			t.Fatalf("expected persisted Order.Payments to be empty, got %+v", dbOrderRow.Object.Payments.AsList())
		}
		preloadedOrderItems := dbOrderRow.Object.OrderItems.AsList()
		if len(preloadedOrderItems) != 2 {
			t.Fatalf("expected 2 preloaded OrderItems on persisted Order, got %d", len(preloadedOrderItems))
		}

		// Verify persisted OrderItems in DB including Amount, ItemPrice, and Product snapshot
		dbItemRows, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).All()
		if err != nil {
			t.Fatalf("failed to query OrderItems from DB: %v", err)
		}
		if len(dbItemRows) != 2 {
			t.Fatalf("expected 2 persisted OrderItems in DB, got %d", len(dbItemRows))
		}
		dbAmountsByProd := make(map[string]int, len(dbItemRows))
		dbPricesByProd := make(map[string]decimal.Decimal, len(dbItemRows))
		for _, row := range dbItemRows {
			dbAmountsByProd[row.Object.Product.Data.ID.String()] = row.Object.Amount
			dbPricesByProd[row.Object.Product.Data.ID.String()] = row.Object.ItemPrice
		}
		if got := dbAmountsByProd[prodKeyboard.ID.String()]; got != 2 {
			t.Fatalf("expected persisted prodKeyboard Amount 2, got %d", got)
		}
		if got := dbAmountsByProd[prodMouse.ID.String()]; got != 3 {
			t.Fatalf("expected persisted prodMouse Amount 3, got %d", got)
		}
		if got := dbPricesByProd[prodKeyboard.ID.String()]; !got.Equal(decimal.RequireFromString("129.99")) {
			t.Fatalf("expected persisted prodKeyboard ItemPrice 129.99, got %s", got)
		}
		if got := dbPricesByProd[prodMouse.ID.String()]; !got.Equal(decimal.RequireFromString("59.50")) {
			t.Fatalf("expected persisted prodMouse ItemPrice 59.50, got %s", got)
		}
	})

	t.Run("Explicit variadic items take precedence over CartItems stored in DB", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-precedence")
		// Seed 2 items in DB for this cart
		_ = seedTestCartItem(t, cart, prodKeyboard, 1)
		_ = seedTestCartItem(t, cart, prodMouse, 2)

		// Pass only a single explicit item (prodMat) to CreateOrder
		overrideItem := &carts.CartItem{
			ID:      drivers.NewULID(),
			Cart:    cart,
			Product: prodMat,
			Amount:  5,
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart, overrideItem)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}

		if len(orderItems) != 1 {
			t.Fatalf("expected only the 1 explicit item, got %d", len(orderItems))
		}
		if orderItems[0].Product.Data.ID != prodMat.ID {
			t.Fatalf("expected explicit product %s, got %s", prodMat.ID, orderItems[0].Product.Data.ID)
		}
		if orderItems[0].Amount != 5 {
			t.Fatalf("expected explicit amount 5, got %d", orderItems[0].Amount)
		}

		persistedRows, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).All()
		if err != nil {
			t.Fatalf("failed to query order items: %v", err)
		}
		if len(persistedRows) != 1 {
			t.Fatalf("expected 1 persisted OrderItem in DB, got %d", len(persistedRows))
		}
	})

	t.Run("Preloaded Cart.CartItems in memory (AsList non-empty) are used when no variadic items passed", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-preloaded")
		_ = seedTestCartItem(t, cart, prodKeyboard, 1)
		_ = seedTestCartItem(t, cart, prodMat, 4)

		// Fetch cart with Preload("CartItems") + Select("*", "Product.*") so cart.CartItems.AsList() is populated
		preloadedCartRow, err := queries.GetQuerySet(&carts.Cart{}).
			Preload(queries.Preload{
				Path:    "CartItems",
				NoJoins: true,
				QuerySet: queries.GetQuerySet[attrs.Definer](&carts.CartItem{}).
					Select("*", "Product.*"),
			}).
			Filter("ID", cart.ID).
			Get()
		if err != nil {
			t.Fatalf("failed to preload cart: %v", err)
		}
		preloadedCart := preloadedCartRow.Object
		if len(preloadedCart.CartItems.AsList()) != 2 {
			t.Fatalf("expected 2 preloaded items in AsList(), got %d", len(preloadedCart.CartItems.AsList()))
		}

		// Delete CartItems from DB to prove CreateOrder uses the in-memory AsList() slice
		// without hitting cart.CartItems.Objects().IterAll()
		if _, err := queries.GetQuerySet(&carts.CartItem{}).Filter("Cart", cart.ID).Delete(); err != nil {
			t.Fatalf("failed to delete cart items from DB: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, preloadedCart)
		if err != nil {
			t.Fatalf("CreateOrder from preloaded cart failed: %v", err)
		}

		if order == nil || order.ID.IsZero() {
			t.Fatal("expected valid Order")
		}
		if len(orderItems) != 2 {
			t.Fatalf("expected 2 OrderItems from preloaded cart, got %d", len(orderItems))
		}

		titles := map[string]decimal.Decimal{}
		for _, item := range orderItems {
			titles[item.Product.Data.Title] = item.ItemPrice
		}
		if price, ok := titles["Mechanical Keyboard"]; !ok || !price.Equal(decimal.RequireFromString("129.99")) {
			t.Fatalf("unexpected Mechanical Keyboard item in orderItems: %v", titles)
		}
		if price, ok := titles["Desk Mat XL"]; !ok || !price.Equal(decimal.RequireFromString("24.95")) {
			t.Fatalf("unexpected Desk Mat XL item in orderItems: %v", titles)
		}
	})

	t.Run("Database fallback via Cart.CartItems.Objects().IterAll() when AsList() is empty", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-db-iterall")
		_ = seedTestCartItem(t, cart, prodMouse, 2)
		_ = seedTestCartItem(t, cart, prodMat, 1)

		// Fetch cart WITHOUT Preload so cart.CartItems.AsList() is empty
		unloadedCartRow, err := queries.GetQuerySet(&carts.Cart{}).
			Filter("ID", cart.ID).
			Get()
		if err != nil {
			t.Fatalf("failed to get cart from DB: %v", err)
		}
		unloadedCart := unloadedCartRow.Object
		if len(unloadedCart.CartItems.AsList()) != 0 {
			t.Fatalf("expected empty AsList() on non-preloaded cart, got %d", len(unloadedCart.CartItems.AsList()))
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, unloadedCart)
		if err != nil {
			t.Fatalf("CreateOrder via DB IterAll fallback failed: %v", err)
		}

		if order == nil || order.ID.IsZero() {
			t.Fatal("expected valid Order")
		}
		if len(orderItems) != 2 {
			t.Fatalf("expected 2 OrderItems loaded from DB, got %d", len(orderItems))
		}

		// Verify persisted OrderItems in DB
		persistedRows, err := queries.GetQuerySet(&orders.OrderItem{}).
			Filter("Order", order.ID).
			All()
		if err != nil {
			t.Fatalf("failed to query persisted OrderItems: %v", err)
		}
		if len(persistedRows) != 2 {
			t.Fatalf("expected 2 persisted OrderItems in DB, got %d", len(persistedRows))
		}

		foundMouse := false
		foundMat := false
		for _, row := range persistedRows {
			switch row.Object.Product.Data.Slug {
			case "wireless-mouse":
				foundMouse = true
				if !row.Object.ItemPrice.Equal(decimal.RequireFromString("59.50")) {
					t.Fatalf("expected persisted wireless-mouse ItemPrice 59.50, got %s", row.Object.ItemPrice)
				}
			case "desk-mat-xl":
				foundMat = true
				if !row.Object.ItemPrice.Equal(decimal.RequireFromString("24.95")) {
					t.Fatalf("expected persisted desk-mat-xl ItemPrice 24.95, got %s", row.Object.ItemPrice)
				}
			}
		}
		if !foundMouse || !foundMat {
			t.Fatalf("expected both wireless-mouse and desk-mat-xl in persisted OrderItems, got mouse=%v mat=%v", foundMouse, foundMat)
		}
	})

	t.Run("OrderItem Product JSON snapshot remains immutable when Product is later modified in DB", func(t *testing.T) {
		clearOrderAndCartTables(t)

		tempProd := seedTestProduct(t, cat, "Limited Edition Keycaps", "limited-keycaps", "SKU-KC-01", "75.00", "21.00", "0.00")
		cart := seedTestCart(t, "sess-snapshot")
		_ = seedTestCartItem(t, cart, tempProd, 1)

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, _, err := orders.CreateOrder(req, customer, cart)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}

		// Modify the Product in the database after the order has been created
		tempProd.Title = "Modified Keycaps V2"
		tempProd.Price = decimal.RequireFromString("150.00")
		if _, err := queries.GetQuerySet(&products.Product{}).
			Select("Title", "Price").
			Filter("ID", tempProd.ID).
			Update(tempProd); err != nil {
			t.Fatalf("failed to update product in DB: %v", err)
		}

		// Reload OrderItem from the database and verify its Product JSON snapshot is unchanged
		itemRow, err := queries.GetQuerySet(&orders.OrderItem{}).
			Filter("Order", order.ID).
			Get()
		if err != nil {
			t.Fatalf("failed to reload OrderItem from DB: %v", err)
		}
		if itemRow.Object.Product.Data.Title != "Limited Edition Keycaps" {
			t.Fatalf("expected snapshot Title 'Limited Edition Keycaps', got %q", itemRow.Object.Product.Data.Title)
		}
		if !itemRow.Object.Product.Data.Price.Equal(decimal.RequireFromString("75.00")) {
			t.Fatalf("expected snapshot Product.Data.Price 75.00, got %s", itemRow.Object.Product.Data.Price)
		}
		if !itemRow.Object.ItemPrice.Equal(decimal.RequireFromString("75.00")) {
			t.Fatalf("expected OrderItem.ItemPrice 75.00, got %s", itemRow.Object.ItemPrice)
		}
	})

	t.Run("Signals.Created is emitted with transactional context, Order, and Lines", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-signal-success")
		_ = seedTestCartItem(t, cart, prodKeyboard, 2)
		_ = seedTestCartItem(t, cart, prodMouse, 1)

		var receivedSignals []orders.OrderCreatedSignal
		var txCountDuringSignal int64
		listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
			receivedSignals = append(receivedSignals, sig)

			// Verify that inside the signal handler, the transaction context already sees the created OrderItems
			cnt, countErr := queries.GetQuerySet(&orders.OrderItem{}).
				WithContext(sig.Request.Context()).
				Filter("Order", sig.Order.ID).
				Count()
			if countErr != nil {
				return countErr
			}
			txCountDuringSignal = cnt
			return nil
		})

		req := httptest.NewRequest(http.MethodPost, "/api/checkout", nil)
		req.Header.Set("X-Test-Checkout", "signal-check")

		order, orderItems, err := orders.CreateOrder(req, customer, cart)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}

		if len(receivedSignals) != 1 {
			t.Fatalf("expected 1 OrderCreatedSignal, got %d", len(receivedSignals))
		}
		sig := receivedSignals[0]
		if sig.Order == nil || sig.Order.ID != order.ID {
			t.Fatalf("expected signal Order ID %s, got %+v", order.ID, sig.Order)
		}
		if len(sig.Lines) != len(orderItems) {
			t.Fatalf("expected %d signal Lines, got %d", len(orderItems), len(sig.Lines))
		}
		for i := range orderItems {
			if sig.Lines[i].ID != orderItems[i].ID {
				t.Fatalf("expected sig.Lines[%d].ID == %s, got %s", i, orderItems[i].ID, sig.Lines[i].ID)
			}
		}
		if sig.Request == nil {
			t.Fatal("expected non-nil Request on OrderCreatedSignal")
		}
		if sig.Request.Header.Get("X-Test-Checkout") != "signal-check" {
			t.Fatalf("expected Request header preserved on signal, got %q", sig.Request.Header.Get("X-Test-Checkout"))
		}
		if txCountDuringSignal != 2 {
			t.Fatalf("expected 2 OrderItems visible in transaction during signal, got %d", txCountDuringSignal)
		}
	})

	t.Run("Error: Empty cart returns errors.NoRows, emits no signal, and persists nothing", func(t *testing.T) {
		clearOrderAndCartTables(t)

		emptyCart := seedTestCart(t, "sess-empty-cart")

		signalFired := false
		listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
			signalFired = true
			return nil
		})

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, emptyCart)
		if err == nil {
			t.Fatal("expected error for empty cart, got nil")
		}
		if !qerrors.Is(err, qerrors.NoRows) {
			t.Fatalf("expected errors.NoRows, got: %v", err)
		}
		if !strings.Contains(err.Error(), emptyCart.ID.String()) {
			t.Fatalf("expected error message to mention cart ID %s, got: %v", emptyCart.ID, err)
		}
		if order != nil || orderItems != nil {
			t.Fatalf("expected nil order and nil orderItems on error, got order=%v items=%v", order, orderItems)
		}
		if signalFired {
			t.Fatal("expected Signals.Created NOT to fire when cart is empty")
		}

		orderCount, _ := orders.Query().Count()
		itemCount, _ := queries.GetQuerySet(&orders.OrderItem{}).Count()
		if orderCount != 0 || itemCount != 0 {
			t.Fatalf("expected 0 orders and 0 order items in DB, got orders=%d items=%d", orderCount, itemCount)
		}
	})

	t.Run("Error: Nil Product in explicit CartItem returns errors.NilPointer and persists nothing", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-nil-product-explicit")
		validItem := &carts.CartItem{
			ID:      drivers.NewULID(),
			Cart:    cart,
			Product: prodKeyboard,
			Amount:  1,
		}
		nilProdItem := &carts.CartItem{
			ID:      drivers.NewULID(),
			Cart:    cart,
			Product: nil,
			Amount:  2,
		}

		signalFired := false
		listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
			signalFired = true
			return nil
		})

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart, validItem, nilProdItem)
		if err == nil {
			t.Fatal("expected error when CartItem.Product is nil, got nil")
		}
		if !qerrors.Is(err, qerrors.NilPointer) {
			t.Fatalf("expected errors.NilPointer, got: %v", err)
		}
		if !strings.Contains(err.Error(), nilProdItem.ID.String()) {
			t.Fatalf("expected error message to mention CartItem ID %s, got: %v", nilProdItem.ID, err)
		}
		if order != nil || orderItems != nil {
			t.Fatalf("expected nil order and nil orderItems on error, got order=%v items=%v", order, orderItems)
		}
		if signalFired {
			t.Fatal("expected Signals.Created NOT to fire when CartItem.Product is nil")
		}

		orderCount, _ := orders.Query().Count()
		itemCount, _ := queries.GetQuerySet(&orders.OrderItem{}).Count()
		if orderCount != 0 || itemCount != 0 {
			t.Fatalf("expected 0 orders and 0 order items in DB, got orders=%d items=%d", orderCount, itemCount)
		}
	})

	t.Run("Error: Nil Product in preloaded Cart.CartItems (Preload without Product) returns errors.NilPointer", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-nil-product-preloaded")
		seededItem := seedTestCartItem(t, cart, prodKeyboard, 1)

		// Preload only "CartItems" without ".Product", leaving CartItem.Product == nil
		cartRow, err := queries.GetQuerySet(&carts.Cart{}).
			Preload("CartItems").
			Filter("ID", cart.ID).
			Get()
		if err != nil {
			t.Fatalf("failed to load cart with Preload(CartItems): %v", err)
		}
		// Ensure Product on preloaded item is nil
		for _, ci := range cartRow.Object.CartItems.AsList() {
			ci.Product = nil
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cartRow.Object)
		if err == nil {
			t.Fatal("expected error when preloaded CartItem.Product is nil, got nil")
		}
		if !qerrors.Is(err, qerrors.NilPointer) {
			t.Fatalf("expected errors.NilPointer, got: %v", err)
		}
		if !strings.Contains(err.Error(), seededItem.ID.String()) {
			t.Fatalf("expected error message to mention CartItem ID %s, got: %v", seededItem.ID, err)
		}
		if order != nil || orderItems != nil {
			t.Fatalf("expected nil order and nil orderItems, got order=%v items=%v", order, orderItems)
		}
	})

	t.Run("Error: Signal listener failure rolls back Order and OrderItems transaction", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-signal-rollback")
		_ = seedTestCartItem(t, cart, prodKeyboard, 1)
		_ = seedTestCartItem(t, cart, prodMouse, 2)

		errDownstreamAbort := errors.New("payment gateway pre-check failed in signal listener")
		listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
			return errDownstreamAbort
		})

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart)
		if err == nil {
			t.Fatal("expected CreateOrder to fail when signal listener returns an error")
		}
		if !strings.Contains(err.Error(), "OrderQuery/CreateOrder/Signal") {
			t.Fatalf("expected wrapped signal error, got: %v", err)
		}
		sigErr, ok := gosignals.SignalError(err)
		if !ok {
			t.Fatalf("expected error to contain a SignalError, got: %v", err)
		}
		foundCause := false
		for _, rel := range sigErr.Related {
			if errors.Is(rel, errDownstreamAbort) {
				foundCause = true
				break
			}
		}
		if !foundCause {
			t.Fatalf("expected SignalError.Related to wrap errDownstreamAbort, got related=%v (err=%v)", sigErr.Related, err)
		}
		if order != nil || orderItems != nil {
			t.Fatalf("expected nil order and nil orderItems on signal failure, got order=%v items=%v", order, orderItems)
		}

		// Verify transaction rollback: neither Order nor OrderItems should exist in DB
		orderCount, err := orders.Query().Count()
		if err != nil {
			t.Fatalf("failed to count orders after rollback: %v", err)
		}
		if orderCount != 0 {
			t.Fatalf("expected 0 orders in DB after signal rollback, got %d", orderCount)
		}

		itemCount, err := queries.GetQuerySet(&orders.OrderItem{}).Count()
		if err != nil {
			t.Fatalf("failed to count order items after rollback: %v", err)
		}
		if itemCount != 0 {
			t.Fatalf("expected 0 order items in DB after signal rollback, got %d", itemCount)
		}
	})

	t.Run("Multiple sequential CreateOrder calls create isolated Orders and OrderItems", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart1 := seedTestCart(t, "sess-multi-1")
		_ = seedTestCartItem(t, cart1, prodKeyboard, 1)

		cart2 := seedTestCart(t, "sess-multi-2")
		_ = seedTestCartItem(t, cart2, prodMouse, 2)
		_ = seedTestCartItem(t, cart2, prodMat, 3)

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)

		order1, items1, err := orders.CreateOrder(req, customer, cart1)
		if err != nil {
			t.Fatalf("CreateOrder for cart1 failed: %v", err)
		}
		order2, items2, err := orders.CreateOrder(req, customer, cart2)
		if err != nil {
			t.Fatalf("CreateOrder for cart2 failed: %v", err)
		}

		if order1.ID == order2.ID {
			t.Fatalf("expected distinct Order IDs, both were %s", order1.ID)
		}
		if len(items1) != 1 || len(items2) != 2 {
			t.Fatalf("unexpected item counts: len(items1)=%d, len(items2)=%d", len(items1), len(items2))
		}

		dbItems1, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order1.ID).All()
		if err != nil {
			t.Fatalf("failed to query order1 items: %v", err)
		}
		dbItems2, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order2.ID).All()
		if err != nil {
			t.Fatalf("failed to query order2 items: %v", err)
		}
		if len(dbItems1) != 1 {
			t.Fatalf("expected 1 persisted item for order1, got %d", len(dbItems1))
		}
		if len(dbItems2) != 2 {
			t.Fatalf("expected 2 persisted items for order2, got %d", len(dbItems2))
		}
		if dbItems1[0].Object.Product.Data.ID != prodKeyboard.ID {
			t.Fatalf("expected order1 item to be prodKeyboard, got %s", dbItems1[0].Object.Product.Data.ID)
		}
	})

	assertNothingPersisted := func(t *testing.T) {
		t.Helper()
		orderCount, err := orders.Query().Count()
		if err != nil {
			t.Fatalf("failed to count orders: %v", err)
		}
		itemCount, err := queries.GetQuerySet(&orders.OrderItem{}).Count()
		if err != nil {
			t.Fatalf("failed to count order items: %v", err)
		}
		if orderCount != 0 || itemCount != 0 {
			t.Fatalf("expected 0 orders and 0 order items in DB, got orders=%d items=%d", orderCount, itemCount)
		}
	}

	t.Run("Edge case: Zero-price, zero-amount, and discounted product handling", func(t *testing.T) {
		clearOrderAndCartTables(t)

		prodFree := seedTestProduct(t, cat, "Free Promo Sticker", "free-promo-sticker", "SKU-FREE-01", "0.00", "0.00", "0.00")
		cart := seedTestCart(t, "sess-edge-pricing")
		_ = seedTestCartItem(t, cart, prodFree, 0)
		_ = seedTestCartItem(t, cart, prodKeyboard, 10)

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart)
		if err != nil {
			t.Fatalf("CreateOrder failed on zero-price / zero-amount / discounted products: %v", err)
		}
		if len(orderItems) != 2 {
			t.Fatalf("expected 2 OrderItems, got %d", len(orderItems))
		}

		itemsByProd := make(map[string]*orders.OrderItem, len(orderItems))
		for _, item := range orderItems {
			itemsByProd[item.Product.Data.ID.String()] = item
		}

		freeItem := itemsByProd[prodFree.ID.String()]
		if freeItem == nil {
			t.Fatal("missing freeItem from returned OrderItems")
		}
		if !freeItem.ItemPrice.Equal(decimal.Zero) {
			t.Fatalf("expected free product ItemPrice 0, got %s", freeItem.ItemPrice)
		}
		if freeItem.Amount != 0 {
			t.Fatalf("expected zero-amount OrderItem.Amount 0, got %d", freeItem.Amount)
		}

		discountedItem := itemsByProd[prodKeyboard.ID.String()]
		if discountedItem == nil {
			t.Fatal("missing prodKeyboard item from returned OrderItems")
		}
		if !discountedItem.ItemPrice.Equal(decimal.RequireFromString("129.99")) {
			t.Fatalf("expected ItemPrice to match Product.Price 129.99, got %s", discountedItem.ItemPrice)
		}
		if !discountedItem.Product.Data.DiscountPrice.Equal(decimal.RequireFromString("109.99")) {
			t.Fatalf("expected snapshot Product.DiscountPrice 109.99, got %s", discountedItem.Product.Data.DiscountPrice)
		}
		if discountedItem.Amount != 10 {
			t.Fatalf("expected Amount 10, got %d", discountedItem.Amount)
		}

		dbItems, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).All()
		if err != nil {
			t.Fatalf("failed to load persisted OrderItems: %v", err)
		}
		if len(dbItems) != 2 {
			t.Fatalf("expected 2 persisted OrderItems, got %d", len(dbItems))
		}
		for _, row := range dbItems {
			switch row.Object.Product.Data.ID {
			case prodFree.ID:
				if row.Object.Amount != 0 || !row.Object.ItemPrice.Equal(decimal.Zero) {
					t.Fatalf("unexpected persisted free item values: amount=%d price=%s", row.Object.Amount, row.Object.ItemPrice)
				}
			case prodKeyboard.ID:
				if row.Object.Amount != 10 || !row.Object.ItemPrice.Equal(decimal.RequireFromString("129.99")) {
					t.Fatalf("unexpected persisted keyboard item values: amount=%d price=%s", row.Object.Amount, row.Object.ItemPrice)
				}
			default:
				t.Fatalf("unexpected product ID in persisted OrderItems: %s", row.Object.Product.Data.ID)
			}
		}
	})

	t.Run("Edge case: Returned Order exposes OrderItems reverse relation and default workflow fields", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-order-rev-rel")
		_ = seedTestCartItem(t, cart, prodMouse, 4)

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, _, err := orders.CreateOrder(req, customer, cart)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}

		if order.Completed || order.Processed || len(order.Payments.AsList()) != 0 {
			t.Fatalf("expected default order flags (Completed=false, Processed=false, Payments empty), got Completed=%v Processed=%v Payments=%v",
				order.Completed, order.Processed, order.Payments.AsList())
		}

		orderField, ok := attrs.Define(t.Context(), order).Field("OrderItems")
		if !ok {
			t.Fatal("expected Order.FieldDefs to contain OrderItems field")
		}
		if orderField.GetValue() == nil {
			t.Fatal("failed to bind Order.OrderItems field")
		}
		if order.OrderItems == nil {
			t.Fatal("expected Order.OrderItems to be bound after GetValue()")
		}
		revRows, err := order.OrderItems.Objects().WithContext(t.Context()).All()
		if err != nil {
			t.Fatalf("failed to query Order.OrderItems.Objects(): %v", err)
		}
		if len(revRows) != 1 || revRows[0].Object.Amount != 4 {
			t.Fatalf("expected 1 reverse OrderItem with Amount=4, got len=%d", len(revRows))
		}
	})

	t.Run("Edge case: nil cart with explicit variadic items succeeds", func(t *testing.T) {
		clearOrderAndCartTables(t)

		explicitItem := &carts.CartItem{
			ID:      drivers.NewULID(),
			Product: prodMat,
			Amount:  3,
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, nil, explicitItem)
		if err != nil {
			t.Fatalf("expected CreateOrder(req, nil, explicitItem) to succeed, got: %v", err)
		}
		if order == nil || len(orderItems) != 1 || orderItems[0].Amount != 3 {
			t.Fatalf("unexpected result for nil cart with explicit item: order=%v items=%v", order, orderItems)
		}
	})

	t.Run("Edge case: Inactive cart with DB CartItems still creates Order via DB fallback", func(t *testing.T) {
		clearOrderAndCartTables(t)

		inactiveCart, err := queries.GetQuerySet(&carts.Cart{}).Create(&carts.Cart{
			Session: "sess-inactive-cart",
			Active:  false,
		})
		if err != nil {
			t.Fatalf("failed to seed inactive cart: %v", err)
		}
		_ = seedTestCartItem(t, inactiveCart, prodMouse, 2)

		dbInactiveCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", inactiveCart.ID).Get()
		if err != nil {
			t.Fatalf("failed to reload inactive cart: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, dbInactiveCart.Object)
		if err != nil {
			t.Fatalf("expected CreateOrder on inactive cart with DB items to succeed, got: %v", err)
		}
		if order == nil || len(orderItems) != 1 || orderItems[0].Product.Data.ID != prodMouse.ID {
			t.Fatalf("unexpected order result for inactive cart: order=%v items=%v", order, orderItems)
		}
	})

	t.Run("Edge case: Cart with initialized empty CartItems RelRevFK falls back to DB CartItems", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-empty-rel-fallback")
		_ = seedTestCartItem(t, cart, prodKeyboard, 2)

		dbCartRow, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", cart.ID).Get()
		if err != nil {
			t.Fatalf("failed to fetch cart: %v", err)
		}
		dbCart := dbCartRow.Object
		// Force lazy allocation of Cart.CartItems RelRevFK without preloading rows
		_ = attrs.Define(t.Context(), dbCart).Get("CartItems")
		if dbCart.CartItems == nil || len(dbCart.CartItems.AsList()) != 0 {
			t.Fatalf("expected non-nil empty CartItems RelRevFK")
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, dbCart)
		if err != nil {
			t.Fatalf("CreateOrder failed with non-nil empty RelRevFK: %v", err)
		}
		if order == nil || len(orderItems) != 1 || orderItems[0].Amount != 2 {
			t.Fatalf("unexpected order result: order=%v items=%v", order, orderItems)
		}
	})

	t.Run("Error guards: nil cart without items, nil request, empty bound CartItems, and cancelled context", func(t *testing.T) {
		testCases := []struct {
			name    string
			setup   func(t *testing.T) (*http.Request, *carts.Cart)
			wantErr func(t *testing.T, err error)
		}{
			{
				name: "nil cart without items returns errors.NilPointer",
				setup: func(t *testing.T) (*http.Request, *carts.Cart) {
					return httptest.NewRequest(http.MethodPost, "/checkout", nil), nil
				},
				wantErr: func(t *testing.T, err error) {
					if !qerrors.Is(err, qerrors.NilPointer) {
						t.Fatalf("expected errors.NilPointer, got: %v", err)
					}
				},
			},
			{
				name: "nil request returns errors.NilPointer",
				setup: func(t *testing.T) (*http.Request, *carts.Cart) {
					cart := seedTestCart(t, "sess-nil-req")
					_ = seedTestCartItem(t, cart, prodKeyboard, 1)
					return nil, cart
				},
				wantErr: func(t *testing.T, err error) {
					if !qerrors.Is(err, qerrors.NilPointer) {
						t.Fatalf("expected errors.NilPointer, got: %v", err)
					}
				},
			},
			{
				name: "empty cart with bound non-nil CartItems RelRevFK returns errors.NoRows",
				setup: func(t *testing.T) (*http.Request, *carts.Cart) {
					cart := seedTestCart(t, "sess-empty-bound-rel")
					dbCartRow, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", cart.ID).Get()
					if err != nil {
						t.Fatalf("failed to fetch empty cart: %v", err)
					}
					_ = attrs.Define(t.Context(), dbCartRow.Object).Get("CartItems")
					return httptest.NewRequest(http.MethodPost, "/checkout", nil), dbCartRow.Object
				},
				wantErr: func(t *testing.T, err error) {
					if !qerrors.Is(err, qerrors.NoRows) {
						t.Fatalf("expected errors.NoRows, got: %v", err)
					}
				},
			},
			{
				name: "cancelled request context aborts transaction before persistence",
				setup: func(t *testing.T) (*http.Request, *carts.Cart) {
					cart := seedTestCart(t, "sess-cancelled-ctx")
					_ = seedTestCartItem(t, cart, prodKeyboard, 1)
					ctx, cancel := context.WithCancel(t.Context())
					cancel()
					return httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx), cart
				},
				wantErr: func(t *testing.T, err error) {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("expected context.Canceled error, got: %v", err)
					}
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				clearOrderAndCartTables(t)

				signalFired := false
				listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
					signalFired = true
					return nil
				})

				req, cart := tc.setup(t)
				order, orderItems, err := orders.CreateOrder(req, customer, cart)
				if err == nil {
					t.Fatal("expected non-nil error, got nil")
				}
				tc.wantErr(t, err)
				if order != nil || orderItems != nil {
					t.Fatalf("expected nil order and nil orderItems on error, got order=%v items=%v", order, orderItems)
				}
				if signalFired {
					t.Fatal("expected Signals.Created NOT to fire on failure")
				}
				assertNothingPersisted(t)
			})
		}
	})

	t.Run("Signals.Created listener shares active transaction context before commit", func(t *testing.T) {
		clearOrderAndCartTables(t)

		cart := seedTestCart(t, "sess-tx-visibility")
		_ = seedTestCartItem(t, cart, prodKeyboard, 2)

		var seenInTxOrderID string
		var seenInTxItemCount int
		listenOrderCreated(t, func(ctx context.Context, s signals.Signal[orders.OrderCreatedSignal], sig orders.OrderCreatedSignal) error {
			inTxOrder, err := orders.Query().WithContext(sig.Request.Context()).Filter("ID", sig.Order.ID).Get()
			if err != nil {
				return err
			}
			seenInTxOrderID = inTxOrder.Object.ID.String()

			inTxItems, err := queries.GetQuerySet(&orders.OrderItem{}).WithContext(sig.Request.Context()).Filter("Order", sig.Order.ID).All()
			if err != nil {
				return err
			}
			seenInTxItemCount = len(inTxItems)
			return nil
		})

		req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
		order, orderItems, err := orders.CreateOrder(req, customer, cart)
		if err != nil {
			t.Fatalf("CreateOrder failed: %v", err)
		}
		if seenInTxOrderID != order.ID.String() {
			t.Fatalf("expected listener to see uncommitted Order %s inside sig.Request.Context(), got %q", order.ID, seenInTxOrderID)
		}
		if seenInTxItemCount != len(orderItems) || seenInTxItemCount != 1 {
			t.Fatalf("expected listener to see 1 uncommitted OrderItem inside tx, got %d", seenInTxItemCount)
		}
	})
}
