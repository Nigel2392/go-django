//go:build test
// +build test

package public_test

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/shop"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/products/supplier"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type CartTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
}

func (u *CartTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions[*CartTestUser](ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("cart_test_users")
}

func (u *CartTestUser) IsAuthenticated() bool {
	return u.LoggedIn
}

func (u *CartTestUser) IsAdmin() bool {
	return u.IsAdministrator
}

type testCartSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *testCartSupplierProduct) ProductPrice(ctx context.Context) supplier.Price {
	return supplier.Price{
		Unit:   currency.EUR,
		Amount: p.Price,
	}
}

func (p *testCartSupplierProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*testCartSupplierProduct, any](ctx, p,
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
		fattrs.Field(p, "SKU", &p.SKU, func() fattrs.PtrFieldConfig[*testCartSupplierProduct, string] {
			return fattrs.PtrFieldConfig[*testCartSupplierProduct, string]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
	)
}

func createTestProduct(t *testing.T, cat *models.Category, title, slug, price, tax, discount string) *products.Product {
	t.Helper()

	spec := &testCartSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      supplier.SupplierValue("cart_test_supplier"),
			Price:         decimal.RequireFromString(price),
			TaxRate:       decimal.RequireFromString(tax),
			DiscountPrice: decimal.RequireFromString(discount),
		},
		SKU: "SKU-" + slug,
	}
	spec.Product.Instance = spec

	if _, err := queries.GetQuerySet(&testCartSupplierProduct{}).Create(spec); err != nil {
		t.Fatalf("failed to create specific product: %v", err)
	}
	if _, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	return spec.Product
}

func TestCartsAPI(t *testing.T) {
	var _, db = testdb.Open()
	defer db.Close()

	mediafiles.RegisterBackend("memory", memory.NewBackend(5))
	mediafiles.SetDefault("memory")

	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &CartTestUser{},
		GetObject:      func() any { return &CartTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "Cart Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "Cart Test Users" },
	})

	supplier.Register(&supplier.Supplier{
		Order: 1,
		Name:  "cart_test_supplier",
		HTMLIcon: func(ctx context.Context) template.HTML {
			return "<svg></svg>"
		},
		TextLabel: func(ctx context.Context) string {
			return "Cart Test Supplier"
		},
		TextDescription: func(ctx context.Context) string {
			return "Supplier for Cart API tests"
		},
		ProductModel: func(ctx context.Context) supplier.ProductModel {
			return &testCartSupplierProduct{}
		},
	})

	setupApp := func() django.AppConfig {
		cfg := apps.NewDBAppConfig("cart_test_supplier")
		cfg.ModelObjects = []attrs.Definer{
			&CartTestUser{},
			&testCartSupplierProduct{},
		}
		cfg.Init = func(settings django.Settings, db drivers.Database) error {
			if ed, err := migrator.GetSchemaEditor(db.Driver()); err == nil {
				_ = ed.DropTable(context.Background(), migrator.NewModelTable(&auditlogs.Entry{}), true)
			}
			tables := quest.Table(t,
				&CartTestUser{},
				&auditlogs.Entry{},
				&models.Category{},
				&supplier.SupplierModel{},
				&testCartSupplierProduct{},
				&products.Product{},
				&carts.Cart{},
				&carts.CartItem{},
				&customers.Customer{},
				&payments.Payment{},
				&orders.Order{},
				&coupons.Coupon{},
				&coupons.Redemption{},
			)
			tables.IfNotExists = true
			tables.Create()
			_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
			_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
			_, _ = queries.GetQuerySet(&products.Product{}).Delete()
			return nil
		}
		return cfg
	}

	var prod1, prod2, prodNotInCart *products.Product

	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
			django.APPVAR_DISABLE_NOSURF: true,
			django.APPVAR_DEBUG:          true,
			django.APPVAR_DATABASE:       db,
		},
		Flags: []django.AppFlag{
			django.FlagSkipCmds,
			django.FlagSkipChecks,
			django.FlagSkipDepsCheck,
		},
		Apps: []djester.AppInitFuncOrAppConfig{
			session.NewAppConfig,
			messages.NewAppConfig,
			admin.NewAppConfig,
			setupApp,
			reports.NewAppConfig,
			auditlogs.NewAppConfig,
			editor.NewAppConfig,
			blocks.NewAppConfig,
			revisions.NewAppConfig,
			shop.NewAppConfig,
		},
		Auth: &djester.TesterAuth{
			UnauthenticatedUser: func() authentication.User {
				return &CartTestUser{LoggedIn: false, IsAdministrator: false}
			},
			Users: map[string]authentication.User{
				"shop_user": &CartTestUser{
					ID:              1,
					Name:            "shop_user",
					LoggedIn:        true,
					IsAdministrator: false,
				},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Seed products and test user",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := queries.GetQuerySet(&CartTestUser{}).ExplicitSave().Create(&CartTestUser{
						ID:              1,
						Name:            "shop_user",
						LoggedIn:        true,
						IsAdministrator: false,
					})
					if err != nil {
						t.Fatalf("failed to seed CartTestUser: %v", err)
					}

					cat := &models.Category{
						Name: "Peripherals",
						Slug: "peripherals",
					}
					if _, err := queries.GetQuerySet(&models.Category{}).Create(cat); err != nil {
						t.Fatalf("failed to seed Category: %v", err)
					}

					prod1 = createTestProduct(t, cat, "Wireless Mouse", "wireless-mouse", "29.99", "21.00", "24.99")
					prod2 = createTestProduct(t, cat, "Mechanical Keyboard", "mechanical-keyboard", "99.50", "21.00", "0.00")
					prodNotInCart = createTestProduct(t, cat, "HD Webcam", "hd-webcam", "59.00", "21.00", "0.00")
				},
			},
			&djester.BasicTest{
				Label: "GetOrCreateCart creates new active cart and reuses it on subsequent calls",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()

					var createdSignals int
					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()
					_, err := carts.Signals.Created.Listen(sigCtx, func(ctx context.Context, s signals.Signal[carts.CartAPISignal], sig carts.CartAPISignal) error {
						createdSignals++
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to Created signal: %v", err)

					// First GET /api/carts -> creates new cart
					var cartResp public.CartResponse
					resp, err := d.Get(django.Reverse("api:carts"), nil, nil)
					asserter.Assert(err == nil, "GET /api/carts failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.Assert(resp.JSON(&cartResp) == nil, "Failed to decode CartResponse JSON")
					resp.Body.Close()
					asserter.AssertNotNil(cartResp.Cart)
					asserter.Assert(!cartResp.Cart.ID.IsZero(), "Expected non-zero cart ID")
					asserter.Assert(cartResp.Cart.Active, "Expected cart to be active")
					asserter.AssertEqual(0, len(cartResp.Items))
					asserter.AssertEqual(1, createdSignals)

					firstCartID := cartResp.Cart.ID

					// Second GET /api/carts in same session -> returns existing cart without creating a new one
					var cartResp2 public.CartResponse
					resp2, err := d.Get(django.Reverse("api:carts"), nil, nil)
					asserter.Assert(err == nil, "Second GET /api/carts failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp2.StatusCode)
					asserter.Assert(resp2.JSON(&cartResp2) == nil, "Failed to decode second CartResponse JSON")
					resp2.Body.Close()
					asserter.AssertNotNil(cartResp2.Cart)
					asserter.AssertEqual(firstCartID, cartResp2.Cart.ID)
					asserter.AssertEqual(1, createdSignals)
				},
			},
			&djester.BasicTest{
				Label: "GetOrCreateCart works for authenticated users",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("shop_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					var cartResp public.CartResponse
					resp, err := d.Get(django.Reverse("api:carts"), nil, nil)
					asserter.Assert(err == nil, "GET /api/carts for authenticated user failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.Assert(resp.JSON(&cartResp) == nil, "Failed to decode CartResponse JSON")
					resp.Body.Close()
					asserter.AssertNotNil(cartResp.Cart)
					asserter.Assert(!cartResp.Cart.ID.IsZero(), "Expected valid cart ID for authenticated user")
					asserter.Assert(cartResp.Cart.Active, "Expected active cart for authenticated user")
				},
			},
			&djester.BasicTest{
				Label: "AddCartItem validation and product not found errors",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()
					addURL := django.Reverse("api:carts:add_item")

					// 1. Invalid JSON body -> 400 Bad Request
					var errResp api.ErrorResponse
					resp, err := d.PostJson(addURL, nil, nil, bytes.NewBufferString("{invalid-json"), &errResp)
					asserter.Assert(err == nil, "POST invalid JSON failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusBadRequest, resp.StatusCode)
					asserter.Assert(errResp.Message != "", "Expected non-empty error message")

					// 2. Zero and negative amounts -> 400 Bad Request
					for _, amt := range []int{0, -1, -5} {
						errResp = api.ErrorResponse{}
						resp, err = d.PostJson(addURL, nil, nil, public.CartItemRequest{
							ProductID: prod1.ID.String(),
							Amount:    amt,
						}, &errResp)
						asserter.Assert(err == nil, "POST amount=%d failed: %v", amt, err)
						resp.Body.Close()
						asserter.AssertEqual(http.StatusBadRequest, resp.StatusCode)
						asserter.AssertEqual(fmt.Sprintf("amount cannot be less than 0: %v", amt), errResp.Message)
					}

					// 3. Non-existent product -> 404 Not Found
					errResp = api.ErrorResponse{}
					resp, err = d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: drivers.NewULID().String(),
						Amount:    2,
					}, &errResp)
					asserter.Assert(err == nil, "POST missing product failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusNotFound, resp.StatusCode)
					asserter.Assert(strings.Contains(errResp.Message, "failed to retrieve product to add to cart"), "Unexpected error message: %q", errResp.Message)
				},
			},
			&djester.BasicTest{
				Label: "AddCartItem creates cart when none exists, increments existing item, and adds distinct items",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()
					_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
					_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
					addURL := django.Reverse("api:carts:add_item")

					var cartCreatedCount int
					var itemAddedCount int
					var lastAddedAmount uint64

					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()

					_, err := carts.Signals.Created.Listen(sigCtx, func(ctx context.Context, s signals.Signal[carts.CartAPISignal], sig carts.CartAPISignal) error {
						cartCreatedCount++
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to Created signal: %v", err)

					_, err = carts.Signals.Item.Add.Listen(sigCtx, func(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], sig carts.CartItemAPISignal) error {
						itemAddedCount++
						if sig.CartItem != nil {
							lastAddedAmount = sig.CartItem.Amount
						}
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to Item.Add signal: %v", err)

					// 1. Add prod1 (amount 2) when no cart exists yet -> creates Cart + CartItem
					var itemResp1 public.CartItemResponse
					resp, err := d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    2,
					}, &itemResp1)
					asserter.Assert(err == nil, "First AddCartItem failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					asserter.AssertNotNil(itemResp1.Cart)
					asserter.AssertNotNil(itemResp1.Item)
					asserter.AssertNotNil(itemResp1.Product)
					asserter.AssertEqual(uint64(2), itemResp1.Item.Amount)
					asserter.AssertEqual(prod1.ID, itemResp1.Product.ID)
					asserter.AssertEqual(1, cartCreatedCount)
					asserter.AssertEqual(1, itemAddedCount)
					asserter.AssertEqual(uint64(2), lastAddedAmount)

					cartID := itemResp1.Cart.ID

					// 2. Add prod1 again (amount 3) -> reuses Cart and increments CartItem.Amount to 5
					var itemResp2 public.CartItemResponse
					resp2, err := d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    3,
					}, &itemResp2)
					asserter.Assert(err == nil, "Second AddCartItem failed: %v", err)
					resp2.Body.Close()
					asserter.AssertEqual(http.StatusOK, resp2.StatusCode)
					asserter.AssertEqual(cartID, itemResp2.Cart.ID)
					asserter.AssertEqual(uint64(5), itemResp2.Item.Amount)
					asserter.AssertEqual(1, cartCreatedCount)
					asserter.AssertEqual(2, itemAddedCount)
					asserter.AssertEqual(uint64(5), lastAddedAmount)

					// Verify DB state directly
					dbItem, err := queries.GetQuerySet(&carts.CartItem{}).
						Filter("Cart", cartID).
						Filter("Product", prod1.ID).
						Get()
					asserter.Assert(err == nil, "Failed to query CartItem from DB: %v", err)
					asserter.AssertEqual(uint64(5), dbItem.Object.Amount)

					// 3. Add prod2 (amount 1) to existing cart -> creates second CartItem
					var itemResp3 public.CartItemResponse
					resp3, err := d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: prod2.ID.String(),
						Amount:    1,
					}, &itemResp3)
					asserter.Assert(err == nil, "Third AddCartItem failed: %v", err)
					resp3.Body.Close()
					asserter.AssertEqual(http.StatusOK, resp3.StatusCode)
					asserter.AssertEqual(cartID, itemResp3.Cart.ID)
					asserter.AssertEqual(uint64(1), itemResp3.Item.Amount)
					asserter.AssertEqual(prod2.ID, itemResp3.Product.ID)

					// 4. Verify GET /api/carts preloads both items and their product details
					var fullCart public.CartResponse
					getResp, err := d.Get(django.Reverse("api:carts"), nil, nil)
					asserter.Assert(err == nil, "GET /api/carts failed: %v", err)
					asserter.AssertEqual(http.StatusOK, getResp.StatusCode)
					asserter.Assert(getResp.JSON(&fullCart) == nil, "Failed to decode fullCart JSON")
					getResp.Body.Close()
					asserter.AssertEqual(2, len(fullCart.Items))
				},
			},
			&djester.BasicTest{
				Label: "RemoveCartItem validation and not-found errors",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()
					_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
					_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
					remURL := django.Reverse("api:carts:remove_item")

					// 1. Invalid JSON -> 400 Bad Request
					var errResp api.ErrorResponse
					resp, err := d.PostJson(remURL, nil, nil, bytes.NewBufferString("not-json"), &errResp)
					asserter.Assert(err == nil, "POST invalid JSON failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusBadRequest, resp.StatusCode)

					// 2. Zero or negative amount -> 400 Bad Request
					for _, amt := range []int{0, -3} {
						errResp = api.ErrorResponse{}
						resp, err = d.PostJson(remURL, nil, nil, public.CartItemRequest{
							ProductID: prod1.ID.String(),
							Amount:    amt,
						}, &errResp)
						asserter.Assert(err == nil, "POST amount=%d failed: %v", amt, err)
						resp.Body.Close()
						asserter.AssertEqual(http.StatusBadRequest, resp.StatusCode)
						asserter.AssertEqual(fmt.Sprintf("amount cannot be less than 0: %v", amt), errResp.Message)
					}

					// 3. Non-existent product -> 404 Not Found
					errResp = api.ErrorResponse{}
					resp, err = d.PostJson(remURL, nil, nil, public.CartItemRequest{
						ProductID: drivers.NewULID().String(),
						Amount:    1,
					}, &errResp)
					asserter.Assert(err == nil, "POST missing product failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusNotFound, resp.StatusCode)
					asserter.Assert(strings.Contains(errResp.Message, "failed to retrieve product to remove from cart"), "Unexpected error message: %q", errResp.Message)

					// 4. Product exists, but session has no active cart -> 404 Not Found ("cart does not exist, cannot remove items")
					errResp = api.ErrorResponse{}
					resp, err = d.PostJson(remURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    1,
					}, &errResp)
					asserter.Assert(err == nil, "POST with no active cart failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusNotFound, resp.StatusCode)
					asserter.Assert(strings.Contains(errResp.Message, "cart does not exist, cannot remove items"), "Unexpected error message: %q", errResp.Message)

					// 5. Create active cart via GET /api/carts, then try removing an item not in the cart -> 404 Not Found ("cart item doesn't exist, cannot remove item")
					getResp, err := d.Get(django.Reverse("api:carts"), nil, nil)
					asserter.Assert(err == nil, "GET /api/carts failed: %v", err)
					getResp.Body.Close()

					errResp = api.ErrorResponse{}
					resp, err = d.PostJson(remURL, nil, nil, public.CartItemRequest{
						ProductID: prodNotInCart.ID.String(),
						Amount:    1,
					}, &errResp)
					asserter.Assert(err == nil, "POST item not in cart failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusNotFound, resp.StatusCode)
					asserter.Assert(strings.Contains(errResp.Message, "cart item doesn't exist, cannot remove item"), "Unexpected error message: %q", errResp.Message)
				},
			},
			&djester.BasicTest{
				Label: "RemoveCartItem partial decrement and full removal",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()
					_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
					_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
					addURL := django.Reverse("api:carts:add_item")
					remURL := django.Reverse("api:carts:remove_item")

					// Add 5 units of prod1
					var addResp public.CartItemResponse
					resp, err := d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    5,
					}, &addResp)
					asserter.Assert(err == nil, "AddCartItem failed: %v", err)
					resp.Body.Close()
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)

					var removedSignals int
					var lastRemaining uint64
					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()
					_, err = carts.Signals.Item.Remove.Listen(sigCtx, func(ctx context.Context, s signals.Signal[carts.CartItemAPISignal], sig carts.CartItemAPISignal) error {
						removedSignals++
						if sig.CartItem != nil {
							lastRemaining = sig.CartItem.Amount
						}
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to Item.Remove signal: %v", err)

					// 1. Partial removal: remove 2 -> 3 remaining
					var remResp1 public.CartItemResponse
					r1, err := d.PostJson(remURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    2,
					}, &remResp1)
					asserter.Assert(err == nil, "Partial RemoveCartItem failed: %v", err)
					r1.Body.Close()
					asserter.AssertEqual(http.StatusOK, r1.StatusCode)
					asserter.AssertNotNil(remResp1.Item)
					asserter.AssertEqual(uint64(3), remResp1.Item.Amount)
					asserter.AssertEqual(1, removedSignals)
					asserter.AssertEqual(uint64(3), lastRemaining)

					// Verify DB still has the CartItem with Amount == 3
					dbItem, err := queries.GetQuerySet(&carts.CartItem{}).
						Filter("Cart", addResp.Cart.ID).
						Filter("Product", prod1.ID).
						Get()
					asserter.Assert(err == nil, "Expected CartItem to remain in DB after partial removal: %v", err)
					asserter.AssertEqual(uint64(3), dbItem.Object.Amount)

					// 2. Full removal: remove 5 (>= 3) -> deletes CartItem from DB and returns Amount == 0
					var remResp2 public.CartItemResponse
					r2, err := d.PostJson(remURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    5,
					}, &remResp2)
					asserter.Assert(err == nil, "Full RemoveCartItem failed: %v", err)
					r2.Body.Close()
					asserter.AssertEqual(http.StatusOK, r2.StatusCode)
					asserter.AssertNotNil(remResp2.Item)
					asserter.AssertEqual(uint64(0), remResp2.Item.Amount)
					asserter.AssertEqual(2, removedSignals)
					asserter.AssertEqual(uint64(0), lastRemaining)

					// Verify CartItem row was deleted from DB
					exists, err := queries.GetQuerySet(&carts.CartItem{}).
						Filter("Cart", addResp.Cart.ID).
						Filter("Product", prod1.ID).
						Exists()
					asserter.Assert(err == nil, "Failed checking CartItem existence: %v", err)
					asserter.Assert(!exists, "Expected CartItem row to be deleted from DB when amount reaches 0")
				},
			},
			&djester.BasicTest{
				Label: "ClearCart deactivates current cart, emits signal, and returns a new active cart",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					_, _ = d.Logout()
					_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
					_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
					addURL := django.Reverse("api:carts:add_item")
					clearURL := django.Reverse("api:carts:clear")

					// 1. Add item to create an active cart with items
					var addResp public.CartItemResponse
					r1, err := d.PostJson(addURL, nil, nil, public.CartItemRequest{
						ProductID: prod1.ID.String(),
						Amount:    2,
					}, &addResp)
					asserter.Assert(err == nil, "AddCartItem failed: %v", err)
					r1.Body.Close()
					oldCartID := addResp.Cart.ID

					var clearedCarts []drivers.ULID
					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()
					_, err = carts.Signals.Cleared.Listen(sigCtx, func(ctx context.Context, s signals.Signal[carts.CartAPISignal], sig carts.CartAPISignal) error {
						if sig.Cart != nil {
							clearedCarts = append(clearedCarts, sig.Cart.ID)
						}
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to Cleared signal: %v", err)

					// 2. Call POST /api/carts/clear
					var clearResp public.CartResponse
					r2, err := d.PostJson(clearURL, nil, nil, map[string]any{}, &clearResp)
					asserter.Assert(err == nil, "ClearCart failed: %v", err)
					r2.Body.Close()
					asserter.AssertEqual(http.StatusOK, r2.StatusCode)
					asserter.AssertNotNil(clearResp.Cart)
					asserter.Assert(!clearResp.Cart.ID.IsZero(), "Expected new cart to have non-zero ID")
					asserter.AssertNotEqual(oldCartID, clearResp.Cart.ID)
					asserter.Assert(clearResp.Cart.Active, "Expected newly returned cart to be active")
					asserter.AssertEqual(0, len(clearResp.Items))
					asserter.AssertEqual(1, len(clearedCarts))
					asserter.AssertEqual(oldCartID, clearedCarts[0])

					// Verify old cart is marked Active = false in DB
					oldCartRow, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", oldCartID).Get()
					asserter.Assert(err == nil, "Failed to query old cart from DB: %v", err)
					asserter.Assert(!oldCartRow.Object.Active, "Expected old cart to be marked Active=false in DB")

					// 3. Calling ClearCart when no active cart exists for the session still returns a new active cart
					_, _ = d.Logout()
					_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
					clearedCarts = nil

					var clearResp2 public.CartResponse
					r3, err := d.PostJson(clearURL, nil, nil, map[string]any{}, &clearResp2)
					asserter.Assert(err == nil, "ClearCart without prior cart failed: %v", err)
					r3.Body.Close()
					asserter.AssertEqual(http.StatusOK, r3.StatusCode)
					asserter.AssertNotNil(clearResp2.Cart)
					asserter.Assert(clearResp2.Cart.Active, "Expected new active cart")
					asserter.AssertEqual(0, len(clearedCarts))
				},
			},
		},
	}

	d.Test(t)
}
