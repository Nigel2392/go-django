//go:build test
// +build test

package carts_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	_ "github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type CartQueryTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
}

func (u *CartQueryTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("cart_query_test_users")
}

func (u *CartQueryTestUser) IsAuthenticated() bool {
	return u != nil && u.LoggedIn
}

func (u *CartQueryTestUser) IsAdmin() bool {
	return u != nil && u.IsAdministrator
}

type Order struct {
	ID drivers.ULID
}

func (o *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, o,
		fattrs.Field(o, "ID", &o.ID, func() fattrs.PtrFieldConfig[*Order, drivers.ULID] {
			return fattrs.PtrFieldConfig[*Order, drivers.ULID]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
	)
}

type testCartSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *testCartSupplierProduct) ProductPrice(ctx context.Context) products.Price {
	return products.Price{
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
				"AdditionalFee",
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

func setupCartQueryTestEnv(t *testing.T) {
	t.Helper()

	_, db := testdb.Open()

	mediafiles.RegisterBackend("memory", memory.NewBackend(5))
	mediafiles.SetDefault("memory")

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &CartQueryTestUser{},
		GetObject:      func() any { return &CartQueryTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "Cart Query Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "Cart Query Test Users" },
	})

	orders.RegisterSupplier(&orders.Supplier{
		Order: 1,
		Name:  "cart_query_test_supplier",
		TextLabel: func(ctx context.Context) string {
			return "Cart Query Test Supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &testCartSupplierProduct{}
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
					&CartQueryTestUser{},
					&models.Category{},
					&orders.SupplierModel{},
					&testCartSupplierProduct{},
					&products.Product{},
					&carts.Cart{},
					&carts.CartItem{},
					&customers.Customer{},
					&Order{},
					&coupons.Coupon{},
					&coupons.CouponProductThrough{},
					&coupons.Redemption{},
				}
				return cfg
			}),
		)
		if err := app.Initialize(); err != nil {
			t.Fatalf("failed to initialize django app: %v", err)
		}
	}

	tables := quest.Table(t,
		&CartQueryTestUser{},
		&models.Category{},
		&orders.SupplierModel{},
		&testCartSupplierProduct{},
		&products.Product{},
		&carts.Cart{},
		&carts.CartItem{},
		&customers.Customer{},
		&Order{},
		&coupons.Coupon{},
		&coupons.CouponProductThrough{},
		&coupons.Redemption{},
	)
	tables.IfNotExists = true
	tables.Create()

	if err := orders.SuppliersQuery().Sync(); err != nil {
		t.Fatalf("failed to sync suppliers: %v", err)
	}

	t.Cleanup(func() {
		clearCartQueryTables(t)
		_, _ = queries.GetQuerySet(&products.Product{}).Delete()
		_ = db.Close()
	})
}

func clearCartQueryTables(t *testing.T) {
	t.Helper()
	modelsToClear := []attrs.Definer{
		&coupons.Redemption{},
		&coupons.CouponProductThrough{},
		&coupons.Coupon{},
		&carts.CartItem{},
		&carts.Cart{},
	}
	for _, m := range modelsToClear {
		if _, err := queries.GetQuerySet(m).Delete(); err != nil {
			t.Fatalf("failed to clear %T table: %v", m, err)
		}
	}
}

func d(v string) decimal.Decimal {
	return decimal.RequireFromString(v)
}

func seedUser(t *testing.T, id int, name string) *CartQueryTestUser {
	t.Helper()
	u, err := queries.GetQuerySet(&CartQueryTestUser{}).ExplicitSave().Create(&CartQueryTestUser{
		ID:       id,
		Name:     name,
		LoggedIn: true,
	})
	if err != nil {
		t.Fatalf("failed to create user %q: %v", name, err)
	}
	return u
}

func seedCategory(t *testing.T, name, slug string) *models.Category {
	t.Helper()
	c, err := queries.GetQuerySet(&models.Category{}).Create(&models.Category{
		Name: name,
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("failed to create category %q: %v", name, err)
	}
	return c
}

func seedCustomer(t *testing.T, email, first, last string) *customers.Customer {
	t.Helper()
	c, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
		Email:     drivers.MustParseEmail(email),
		FirstName: first,
		LastName:  last,
	})
	if err != nil {
		t.Fatalf("failed to create customer %q: %v", email, err)
	}
	return c
}

func seedProduct(t *testing.T, cat *models.Category, title, slug, sku, priceStr, feeStr, taxStr string) *products.Product {
	t.Helper()
	spec := &testCartSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      products.SupplierValue("cart_query_test_supplier"),
			Price:         d(priceStr),
			AdditionalFee: d(feeStr),
			TaxRate:       d(taxStr),
			DiscountPrice: decimal.Zero,
			Ranking:       1,
		},
		SKU: sku,
	}
	spec.Product.Instance = spec

	if _, err := queries.GetQuerySet(&testCartSupplierProduct{}).Create(spec); err != nil {
		t.Fatalf("failed to create specific product %q: %v", title, err)
	}
	prod, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product)
	if err != nil {
		t.Fatalf("failed to create product %q: %v", title, err)
	}
	return prod
}

func seedCart(t *testing.T, sessionToken string, user *CartQueryTestUser, active bool, createdAt time.Time) *carts.Cart {
	t.Helper()
	c := &carts.Cart{
		Session:   sessionToken,
		Active:    active,
		CreatedAt: createdAt,
	}
	if user != nil {
		c.User = user
	}
	created, err := queries.GetQuerySet(&carts.Cart{}).Create(c)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}
	if !createdAt.IsZero() {
		if _, err := queries.GetQuerySet(&carts.Cart{}).
			Filter("ID", created.ID).
			BulkUpdate(expr.As("CreatedAt", expr.Value(createdAt))); err != nil {
			t.Fatalf("failed to set cart CreatedAt: %v", err)
		}
		created.CreatedAt = createdAt
	}
	return created
}

func seedCartItem(t *testing.T, cart *carts.Cart, prod *products.Product, amount uint64) *carts.CartItem {
	t.Helper()
	item, err := queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
		Cart:    cart,
		Product: prod,
		Amount:  amount,
	})
	if err != nil {
		t.Fatalf("failed to create cart item: %v", err)
	}
	return item
}

func seedCoupon(t *testing.T, code, title string, isPct bool, maxDiscount string, targetProducts ...*products.Product) *coupons.Coupon {
	t.Helper()
	now := time.Now()
	c, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Title:        title,
		Code:         coupons.CouponCode(code),
		IsPercentage: isPct,
		MaxDiscount:  d(maxDiscount),
		ValidFrom:    now.Add(-time.Hour),
		ValidUntil:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create coupon %q: %v", code, err)
	}
	if len(targetProducts) > 0 {
		_ = attrs.Define(t.Context(), c).Get("Products")
		if _, err := c.Products.Objects().AddTargets(targetProducts...); err != nil {
			t.Fatalf("failed to attach products to coupon %q: %v", code, err)
		}
	}
	return c
}

func seedRedemption(t *testing.T, cart *carts.Cart, coupon *coupons.Coupon, cust *customers.Customer, invalidated bool) *coupons.Redemption {
	t.Helper()
	r, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
		Cart:        cart,
		Coupon:      coupon,
		Customer:    cust,
		Invalidated: invalidated,
		RedeemedAt:  time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to create redemption: %v", err)
	}
	return r
}

func newSessionStore() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	sm.Lifetime = time.Hour
	return sm
}

// issueSessionToken runs scs middleware once and returns both the token and session cookie.
func issueSessionToken(t *testing.T, sm *scs.SessionManager) (string, *http.Cookie) {
	t.Helper()
	var token string
	mw := sessions.SessionMiddleware(sm)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := sessions.Retrieve(r)
		s.Set("init", true)
		_ = s.RenewToken()
		token = s.Token()
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session middleware to emit a cookie")
	}
	if token == "" {
		// Re-capture token on follow-up request so scs loads persisted token
		followReq := httptest.NewRequest(http.MethodGet, "/", nil)
		followReq.AddCookie(cookies[0])
		followRec := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token = sessions.Retrieve(r).Token()
		})).ServeHTTP(followRec, followReq)
	}
	return token, cookies[0]
}

func buildRequestWithMiddleware(t *testing.T, sm *scs.SessionManager, cookie *http.Cookie, user authentication.User) *http.Request {
	t.Helper()
	baseReq := httptest.NewRequest(http.MethodGet, "/cart", nil)
	if cookie != nil {
		baseReq.AddCookie(cookie)
	}
	if user != nil {
		baseReq = baseReq.WithContext(authentication.ContextWithUser(baseReq.Context(), user))
	}
	if sm == nil {
		return baseReq
	}

	var captured *http.Request
	handler := sessions.SessionMiddleware(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
	}))
	handler.ServeHTTP(httptest.NewRecorder(), baseReq)
	return captured
}

func collectCalc(t *testing.T, qs *carts.CartQuery) (int, []*carts.CartRow) {
	t.Helper()
	n, seq, err := qs.Calc()
	if err != nil {
		t.Fatalf("Calc() failed: %v", err)
	}
	var out []*carts.CartRow
	for row, err := range seq {
		if err != nil {
			t.Fatalf("Calc() iterator error: %v", err)
		}
		out = append(out, row)
	}
	return n, out
}

func assertDecimalEqual(t *testing.T, label string, want, got decimal.Decimal) {
	t.Helper()
	if !want.Equal(got) {
		t.Fatalf("%s: want %s, got %s", label, want, got)
	}
}

func assertTotalsEqual(t *testing.T, label string, want, got price.Totals) {
	t.Helper()
	assertDecimalEqual(t, label+".TotalPrice", want.TotalPrice, got.TotalPrice)
	assertDecimalEqual(t, label+".TotalFee", want.TotalFee, got.TotalFee)
	assertDecimalEqual(t, label+".TotalTax", want.TotalTax, got.TotalTax)
	assertDecimalEqual(t, label+".TotalDiscount", want.TotalDiscount, got.TotalDiscount)
	assertDecimalEqual(t, label+".LeftOverDiscount", want.LeftOverDiscount, got.LeftOverDiscount)
}

func TestCartQuery(t *testing.T) {
	setupCartQueryTestEnv(t)

	cat := seedCategory(t, "Hardware", "hardware")
	customer := seedCustomer(t, "buyer@example.com", "Ada", "Lovelace")
	user1 := seedUser(t, 10, "alice")
	user2 := seedUser(t, 20, "bob")

	// Distinct prices so OrderBy("Product.Price") ascending is deterministic across items
	prodCheap := seedProduct(t, cat, "Cable", "cable", "SKU-CBL", "20.00", "2.00", "21.00")
	prodMid := seedProduct(t, cat, "Mouse", "mouse", "SKU-MS", "50.00", "5.00", "21.00")
	prodExpensive := seedProduct(t, cat, "Monitor", "monitor", "SKU-MON", "100.00", "10.00", "21.00")
	prodForty := seedProduct(t, cat, "Keycaps", "keycaps", "SKU-KC", "40.00", "0.00", "20.00")
	prodFree := seedProduct(t, cat, "Free Manual", "free-manual", "SKU-FREE", "0.00", "0.00", "21.00")
	prodFreeWithFee := seedProduct(t, cat, "Free Sample With Shipping", "free-sample-fee", "SKU-FREE-FEE", "0.00", "8.00", "10.00")
	prodOther := seedProduct(t, cat, "Webcam", "webcam", "SKU-WC", "80.00", "0.00", "21.00")

	t.Run("Query and CloneQuerySet isolate query mutations", func(t *testing.T) {
		clearCartQueryTables(t)

		base := carts.Query()
		filtered := base.Filter("Active", true)
		cloned := base.Clone()
		if base == filtered || base == cloned {
			t.Fatal("expected Filter and Clone to return new *CartQuery instances")
		}

		c1 := seedCart(t, "s-clone-1", nil, true, time.Now())
		_ = seedCart(t, "s-clone-2", nil, false, time.Now())

		allRows, err := base.All()
		if err != nil {
			t.Fatalf("base.All() failed: %v", err)
		}
		activeRows, err := filtered.All()
		if err != nil {
			t.Fatalf("filtered.All() failed: %v", err)
		}
		if len(allRows) != 2 {
			t.Fatalf("expected base Query() to return both carts, got %d", len(allRows))
		}
		if len(activeRows) != 1 || activeRows[0].Object.ID != c1.ID {
			t.Fatalf("expected filtered Query() to return only active cart %s, got %d rows", c1.ID, len(activeRows))
		}
	})

	t.Run("CartForRequest filters by Session, User, Active=true, and orders by CreatedAt", func(t *testing.T) {
		sm := newSessionStore()
		token1, cookie1 := issueSessionToken(t, sm)
		token2, cookie2 := issueSessionToken(t, sm)

		t.Run("Anonymous session matches active carts in CreatedAt order and skips inactive/foreign carts", func(t *testing.T) {
			clearCartQueryTables(t)

			t0 := time.Now().Add(-3 * time.Hour)
			laterActive := seedCart(t, token1, nil, true, t0.Add(2*time.Hour))
			earlierActive := seedCart(t, token1, nil, true, t0)
			_ = seedCart(t, token1, nil, false, t0.Add(time.Minute)) // inactive
			_ = seedCart(t, token2, nil, true, t0.Add(time.Minute))  // other session

			req := buildRequestWithMiddleware(t, sm, cookie1, nil)
			rows, err := carts.Query().CartForRequest(req).All()
			if err != nil {
				t.Fatalf("CartForRequest anonymous failed: %v", err)
			}
			if len(rows) != 2 {
				t.Fatalf("expected 2 active carts for token1, got %d", len(rows))
			}
			if rows[0].Object.ID != earlierActive.ID || rows[1].Object.ID != laterActive.ID {
				t.Fatalf("expected CreatedAt ascending order (%s, %s), got (%s, %s)",
					earlierActive.ID, laterActive.ID, rows[0].Object.ID, rows[1].Object.ID)
			}
		})

		t.Run("Authenticated user matches carts by User OR Session", func(t *testing.T) {
			clearCartQueryTables(t)

			t0 := time.Now().Add(-2 * time.Hour)
			userCartOldSession := seedCart(t, "stale-token", user1, true, t0)
			currentSessionCart := seedCart(t, token2, nil, true, t0.Add(time.Hour))
			_ = seedCart(t, token1, user2, true, t0)         // different user & session
			_ = seedCart(t, "stale-token", user1, false, t0) // inactive user cart

			req := buildRequestWithMiddleware(t, sm, cookie2, user1)
			rows, err := carts.Query().CartForRequest(req).All()
			if err != nil {
				t.Fatalf("CartForRequest authenticated failed: %v", err)
			}
			if len(rows) != 2 {
				t.Fatalf("expected 2 carts matching user1 OR token2, got %d", len(rows))
			}
			if rows[0].Object.ID != userCartOldSession.ID || rows[1].Object.ID != currentSessionCart.ID {
				t.Fatalf("unexpected carts returned: (%s, %s)", rows[0].Object.ID, rows[1].Object.ID)
			}
		})

		t.Run("Missing middleware / nil request / unauthenticated user without session do not panic", func(t *testing.T) {
			clearCartQueryTables(t)

			_ = seedCart(t, "", nil, true, time.Now())
			userCart := seedCart(t, "some-sess", user1, true, time.Now())

			// 1. nil request -> no panic, returns 0 rows
			nilReqRows, err := carts.Query().CartForRequest(nil).All()
			if err != nil {
				t.Fatalf("CartForRequest(nil) failed: %v", err)
			}
			if len(nilReqRows) != 0 {
				t.Fatalf("expected 0 carts for nil request, got %d", len(nilReqRows))
			}

			// 2. Request with no session middleware and no user -> no panic, returns 0 rows
			bareReq := httptest.NewRequest(http.MethodGet, "/cart", nil)
			bareRows, err := carts.Query().CartForRequest(bareReq).All()
			if err != nil {
				t.Fatalf("CartForRequest(bareReq) failed: %v", err)
			}
			if len(bareRows) != 0 {
				t.Fatalf("expected 0 carts when neither session nor authenticated user present, got %d", len(bareRows))
			}

			// 3. Authenticated user with no session middleware -> matches user's active cart
			authNoSessionReq := buildRequestWithMiddleware(t, nil, nil, user1)
			authRows, err := carts.Query().CartForRequest(authNoSessionReq).All()
			if err != nil {
				t.Fatalf("CartForRequest(authNoSessionReq) failed: %v", err)
			}
			if len(authRows) != 1 || authRows[0].Object.ID != userCart.ID {
				t.Fatalf("expected user1 cart %s, got %d rows", userCart.ID, len(authRows))
			}

			// 4. Logged-out / unauthenticated user struct with nil session
			unauthUser := &CartQueryTestUser{ID: user1.ID, Name: user1.Name, LoggedIn: false}
			unauthReq := buildRequestWithMiddleware(t, nil, nil, unauthUser)
			unauthRows, err := carts.Query().CartForRequest(unauthReq).All()
			if err != nil {
				t.Fatalf("CartForRequest(unauthReq) failed: %v", err)
			}
			if len(unauthRows) != 0 {
				t.Fatalf("expected 0 carts for unauthenticated user without session, got %d", len(unauthRows))
			}
		})
	})

	t.Run("PreloadProducts loads all or selected Product fields", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-preload", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodMid, 3)

		t.Run("No arguments preloads Product.*", func(t *testing.T) {
			row, err := carts.Query().Filter("ID", cart.ID).PreloadProducts().Get()
			if err != nil {
				t.Fatalf("PreloadProducts() failed: %v", err)
			}
			items := row.Object.CartItems.AsList()
			if len(items) != 1 {
				t.Fatalf("expected 1 cart item, got %d", len(items))
			}
			gotProd := items[0].Product
			if gotProd == nil || gotProd.ID != prodMid.ID {
				t.Fatalf("expected product %s, got %+v", prodMid.ID, gotProd)
			}
			if gotProd.Title != "Mouse" || gotProd.Slug != "mouse" {
				t.Fatalf("expected full Product fields loaded, got title=%q slug=%q", gotProd.Title, gotProd.Slug)
			}
			assertDecimalEqual(t, "Price", d("50.00"), gotProd.Price)
			assertDecimalEqual(t, "AdditionalFee", d("5.00"), gotProd.AdditionalFee)
			assertDecimalEqual(t, "TaxRate", d("21.00"), gotProd.TaxRate)
		})

		t.Run("Explicit productFields selects only requested Product fields", func(t *testing.T) {
			row, err := carts.Query().Filter("ID", cart.ID).PreloadProducts("ID", "Title", "Price").Get()
			if err != nil {
				t.Fatalf("PreloadProducts(fields) failed: %v", err)
			}
			items := row.Object.CartItems.AsList()
			if len(items) != 1 {
				t.Fatalf("expected 1 cart item, got %d", len(items))
			}
			gotProd := items[0].Product
			if gotProd == nil || gotProd.ID != prodMid.ID || gotProd.Title != "Mouse" {
				t.Fatalf("unexpected preloaded product: %+v", gotProd)
			}
			assertDecimalEqual(t, "Price", d("50.00"), gotProd.Price)
			if gotProd.Slug != "" {
				t.Fatalf("expected unselected field Slug to remain empty, got %q", gotProd.Slug)
			}
		})
	})

	t.Run("Calc: Empty queryset and empty cart", func(t *testing.T) {
		clearCartQueryTables(t)

		// Empty queryset -> 0 rows
		n, rows := collectCalc(t, carts.Query().Filter("Active", true))
		if n != 0 || len(rows) != 0 {
			t.Fatalf("expected 0 rows for empty queryset, got n=%d len=%d", n, len(rows))
		}

		// Empty cart -> 1 row with empty items and zero totals
		cart := seedCart(t, "sess-empty", nil, true, time.Now())
		n, rows = collectCalc(t, carts.Query().Filter("ID", cart.ID))
		if n != 1 || len(rows) != 1 {
			t.Fatalf("expected 1 CartRow, got n=%d len=%d", n, len(rows))
		}
		if len(rows[0].Items) != 0 {
			t.Fatalf("expected 0 items in empty cart, got %d", len(rows[0].Items))
		}
		if len(rows[0].Coupons) != 0 {
			t.Fatalf("expected 0 coupons in empty cart, got %d", len(rows[0].Coupons))
		}
		assertTotalsEqual(t, "emptyCart.Totals", price.Totals{
			TotalPrice:       d("0"),
			TotalFee:         d("0"),
			TotalTax:         d("0"),
			TotalDiscount:    d("0"),
			LeftOverDiscount: d("0"),
		}, rows[0].Totals)
	})

	t.Run("Calc: No coupons calculates item prices, AdditionalFee, TaxRate, and sorts items by Product.Price asc", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-no-coupons", nil, true, time.Now())
		// Insert out of price order (100 then 20 then 50)
		_ = seedCartItem(t, cart, prodExpensive, 2) // Price 2*100=200, Fee 2*10=20, Tax 220*0.21=46.20
		_ = seedCartItem(t, cart, prodCheap, 1)     // Price 20, Fee 2, Tax 22*0.21=4.62
		_ = seedCartItem(t, cart, prodMid, 3)       // Price 150, Fee 15, Tax 165*0.21=34.65

		n, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		if n != 1 || len(rows) != 1 {
			t.Fatalf("expected 1 CartRow, got n=%d len=%d", n, len(rows))
		}
		cr := rows[0]
		if len(cr.Items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(cr.Items))
		}
		// Must be sorted ascending by Product.Price: Cheap(20), Mid(50), Expensive(100)
		if cr.Items[0].CartItem.Product.ID != prodCheap.ID ||
			cr.Items[1].CartItem.Product.ID != prodMid.ID ||
			cr.Items[2].CartItem.Product.ID != prodExpensive.ID {
			t.Fatalf("expected items sorted by Product.Price asc, got (%s, %s, %s)",
				cr.Items[0].CartItem.Product.ID,
				cr.Items[1].CartItem.Product.ID,
				cr.Items[2].CartItem.Product.ID,
			)
		}

		assertTotalsEqual(t, "item[0]", price.Totals{
			TotalPrice: d("20"), TotalFee: d("2"), TotalTax: d("4.62"), TotalDiscount: d("0"), LeftOverDiscount: d("0"),
		}, cr.Items[0].CalculatedPrice)
		assertTotalsEqual(t, "item[1]", price.Totals{
			TotalPrice: d("150"), TotalFee: d("15"), TotalTax: d("34.65"), TotalDiscount: d("0"), LeftOverDiscount: d("0"),
		}, cr.Items[1].CalculatedPrice)
		assertTotalsEqual(t, "item[2]", price.Totals{
			TotalPrice: d("200"), TotalFee: d("20"), TotalTax: d("46.20"), TotalDiscount: d("0"), LeftOverDiscount: d("0"),
		}, cr.Items[2].CalculatedPrice)

		assertTotalsEqual(t, "cart.Totals", price.Totals{
			TotalPrice:       d("370"),
			TotalFee:         d("37"),
			TotalTax:         d("85.47"),
			TotalDiscount:    d("0"),
			LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Cart-wide fixed coupon cascades across items from cheapest to most expensive", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-cart-fixed-cascade", nil, true, time.Now())
		// Cheap: price 20 + fee 2 = 22 total absorbable fixed cost
		_ = seedCartItem(t, cart, prodCheap, 1)
		// Mid: price 50 + fee 5 = 55 total absorbable fixed cost
		_ = seedCartItem(t, cart, prodMid, 1)

		// Two cart-wide fixed coupons: 15 + 25 = 40 total cart-wide pool
		// Item 0 (Cheap): absorbs 22 (20 price + 2 fee), leaves 18 in pool -> tax 0
		// Item 1 (Mid): receives remaining 18 from pool -> price 32, fee 5, tax (37)*0.21 = 7.77
		cp1 := seedCoupon(t, "FIXED15", "Fixed 15", false, "15.00")
		cp2 := seedCoupon(t, "FIXED25", "Fixed 25", false, "25.00")
		_ = seedRedemption(t, cart, cp1, customer, false)
		_ = seedRedemption(t, cart, cp2, customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		if len(rows) != 1 {
			t.Fatalf("expected 1 CartRow, got %d", len(rows))
		}
		cr := rows[0]
		if len(cr.Coupons) != 2 {
			t.Fatalf("expected 2 cart coupons, got %d", len(cr.Coupons))
		}
		// Cart-wide coupons must NOT be listed under CartItemRow.Redemptions (product-specific only)
		if len(cr.Items[0].Redemptions) != 0 || len(cr.Items[1].Redemptions) != 0 {
			t.Fatalf("expected 0 item-specific redemptions, got (%d, %d)", len(cr.Items[0].Redemptions), len(cr.Items[1].Redemptions))
		}

		assertTotalsEqual(t, "item[0]", price.Totals{
			TotalPrice: d("0"), TotalFee: d("0"), TotalTax: d("0"), TotalDiscount: d("22"), LeftOverDiscount: d("18"),
		}, cr.Items[0].CalculatedPrice)
		assertTotalsEqual(t, "item[1]", price.Totals{
			TotalPrice: d("32"), TotalFee: d("5"), TotalTax: d("7.77"), TotalDiscount: d("18"), LeftOverDiscount: d("0"),
		}, cr.Items[1].CalculatedPrice)
		assertTotalsEqual(t, "cart.Totals", price.Totals{
			TotalPrice: d("32"), TotalFee: d("5"), TotalTax: d("7.77"), TotalDiscount: d("40"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Cart-wide percentage coupons stack and cap at 100%", func(t *testing.T) {
		clearCartQueryTables(t)

		t.Run("Stacking 10% + 20% = 30% preserves AdditionalFee", func(t *testing.T) {
			clearCartQueryTables(t)
			cart := seedCart(t, "sess-pct-stack", nil, true, time.Now())
			_ = seedCartItem(t, cart, prodMid, 2) // 100 price, 10 fee

			cp1 := seedCoupon(t, "PCT100", "10% off", true, "10.00")
			cp2 := seedCoupon(t, "PCT200", "20% off", true, "20.00")
			_ = seedRedemption(t, cart, cp1, customer, false)
			_ = seedRedemption(t, cart, cp2, customer, false)

			_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
			cr := rows[0]
			// 30% of 100 = 30 discount -> TotalPrice 70, TotalFee 10, Tax (80*0.21)=16.80
			assertTotalsEqual(t, "cart.Totals", price.Totals{
				TotalPrice: d("70"), TotalFee: d("10"), TotalTax: d("16.80"), TotalDiscount: d("30"), LeftOverDiscount: d("0"),
			}, cr.Totals)
		})

		t.Run("Over 100% combined percentage caps at 100%", func(t *testing.T) {
			clearCartQueryTables(t)
			cart := seedCart(t, "sess-pct-cap", nil, true, time.Now())
			_ = seedCartItem(t, cart, prodMid, 1) // 50 price, 5 fee

			cp1 := seedCoupon(t, "PCT700", "70% off", true, "70.00")
			cp2 := seedCoupon(t, "PCT600", "60% off", true, "60.00")
			_ = seedRedemption(t, cart, cp1, customer, false)
			_ = seedRedemption(t, cart, cp2, customer, false)

			_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
			cr := rows[0]
			// Capped at 100% of 50 -> TotalPrice 0, TotalFee 5 intact, Tax 5*0.21 = 1.05
			assertTotalsEqual(t, "cart.Totals", price.Totals{
				TotalPrice: d("0"), TotalFee: d("5"), TotalTax: d("1.05"), TotalDiscount: d("50"), LeftOverDiscount: d("0"),
			}, cr.Totals)
		})
	})

	t.Run("Calc: Product-specific coupons isolate discounts and leftover item discount does not spill over", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-item-specific", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodCheap, 1)     // 20 price, 2 fee = 22 cost
		_ = seedCartItem(t, cart, prodExpensive, 1) // 100 price, 10 fee

		// Specific fixed coupon 35 ONLY for prodCheap -> absorbs 22, leftover 13 must NOT spill to prodExpensive
		// Specific percentage coupon 10% ONLY for prodExpensive -> discounts 10 from 100 = 90
		cpCheapFixed := seedCoupon(t, "CHPFXD", "Cheap Fixed 35", false, "35.00", prodCheap)
		cpExpPct := seedCoupon(t, "EXPPCT", "Expensive 10%", true, "10.00", prodExpensive)
		r1 := seedRedemption(t, cart, cpCheapFixed, customer, false)
		r2 := seedRedemption(t, cart, cpExpPct, customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		if len(cr.Coupons) != 2 {
			t.Fatalf("expected 2 coupons on cartRow, got %d", len(cr.Coupons))
		}
		if len(cr.Items[0].Redemptions) != 1 || cr.Items[0].Redemptions[0].ID != r1.ID {
			t.Fatalf("expected item[0] redemption %d, got %+v", r1.ID, cr.Items[0].Redemptions)
		}
		if len(cr.Items[1].Redemptions) != 1 || cr.Items[1].Redemptions[0].ID != r2.ID {
			t.Fatalf("expected item[1] redemption %d, got %+v", r2.ID, cr.Items[1].Redemptions)
		}

		assertTotalsEqual(t, "cheap item", price.Totals{
			TotalPrice: d("0"), TotalFee: d("0"), TotalTax: d("0"), TotalDiscount: d("22"), LeftOverDiscount: d("13"),
		}, cr.Items[0].CalculatedPrice)
		assertTotalsEqual(t, "expensive item", price.Totals{
			TotalPrice: d("90"), TotalFee: d("10"), TotalTax: d("21.00"), TotalDiscount: d("10"), LeftOverDiscount: d("0"),
		}, cr.Items[1].CalculatedPrice)
		assertTotalsEqual(t, "cart.Totals", price.Totals{
			TotalPrice: d("90"), TotalFee: d("10"), TotalTax: d("21.00"), TotalDiscount: d("32"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Product-specific fixed + Cart-wide fixed pool interactions", func(t *testing.T) {
		t.Run("Item-specific fixed covers full item leaving cart-wide pool intact for next item", func(t *testing.T) {
			clearCartQueryTables(t)

			cart := seedCart(t, "sess-pool-intact", nil, true, time.Now())
			_ = seedCartItem(t, cart, prodCheap, 1) // 20 price + 2 fee = 22
			_ = seedCartItem(t, cart, prodMid, 1)   // 50 price + 5 fee

			cpItem := seedCoupon(t, "ITEM30", "Item 30", false, "30.00", prodCheap)
			cpCart := seedCoupon(t, "CART15", "Cart 15", false, "15.00")
			_ = seedRedemption(t, cart, cpItem, customer, false)
			_ = seedRedemption(t, cart, cpCart, customer, false)

			_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
			cr := rows[0]

			// Item 0 absorbs 22 using ITEM30 (leftover 8 item-specific + 15 cart-wide = 23 >= 15, so CART15 untouched!)
			assertTotalsEqual(t, "cheap", price.Totals{
				TotalPrice: d("0"), TotalFee: d("0"), TotalTax: d("0"), TotalDiscount: d("22"), LeftOverDiscount: d("23"),
			}, cr.Items[0].CalculatedPrice)
			// Item 1 receives full 15 from CART15 -> 35 price, 5 fee, (40*0.21)=8.40 tax
			assertTotalsEqual(t, "mid", price.Totals{
				TotalPrice: d("35"), TotalFee: d("5"), TotalTax: d("8.40"), TotalDiscount: d("15"), LeftOverDiscount: d("0"),
			}, cr.Items[1].CalculatedPrice)
			assertTotalsEqual(t, "cart", price.Totals{
				TotalPrice: d("35"), TotalFee: d("5"), TotalTax: d("8.40"), TotalDiscount: d("37"), LeftOverDiscount: d("0"),
			}, cr.Totals)
		})

		t.Run("Item-specific fixed partially covers item and draws remainder from cart-wide pool", func(t *testing.T) {
			clearCartQueryTables(t)

			cart := seedCart(t, "sess-pool-partial", nil, true, time.Now())
			_ = seedCartItem(t, cart, prodCheap, 1) // 20 + 2 = 22
			_ = seedCartItem(t, cart, prodMid, 1)   // 50 + 5

			// Item-specific 12 covers 12 of Cheap; remaining 10 comes from Cart-wide 15 -> 5 left for Mid
			cpItem := seedCoupon(t, "ITEM12", "Item 12", false, "12.00", prodCheap)
			cpCart := seedCoupon(t, "CART15", "Cart 15", false, "15.00")
			_ = seedRedemption(t, cart, cpItem, customer, false)
			_ = seedRedemption(t, cart, cpCart, customer, false)

			_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
			cr := rows[0]

			assertTotalsEqual(t, "cheap", price.Totals{
				TotalPrice: d("0"), TotalFee: d("0"), TotalTax: d("0"), TotalDiscount: d("22"), LeftOverDiscount: d("5"),
			}, cr.Items[0].CalculatedPrice)
			assertTotalsEqual(t, "mid", price.Totals{
				TotalPrice: d("45"), TotalFee: d("5"), TotalTax: d("10.50"), TotalDiscount: d("5"), LeftOverDiscount: d("0"),
			}, cr.Items[1].CalculatedPrice)
			assertTotalsEqual(t, "cart", price.Totals{
				TotalPrice: d("45"), TotalFee: d("5"), TotalTax: d("10.50"), TotalDiscount: d("27"), LeftOverDiscount: d("0"),
			}, cr.Totals)
		})
	})

	t.Run("Calc: Multi-product coupon does not duplicate discount when multiple matching products are in cart", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-m2m-dedupe", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodCheap, 1) // 20 + 2
		_ = seedCartItem(t, cart, prodMid, 1)   // 50 + 5

		cpBoth := seedCoupon(t, "BOTH10", "10 off both", false, "10.00", prodCheap, prodMid)
		_ = seedRedemption(t, cart, cpBoth, customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		if len(cr.Coupons) != 1 {
			t.Fatalf("expected 1 deduplicated coupon on CartRow, got %d", len(cr.Coupons))
		}
		if len(cr.Items[0].Redemptions) != 1 || len(cr.Items[1].Redemptions) != 1 {
			t.Fatalf("expected 1 redemption per matching item, got (%d, %d)", len(cr.Items[0].Redemptions), len(cr.Items[1].Redemptions))
		}

		// Cheap: 20-10 = 10, fee 2, tax 12*0.21 = 2.52
		// Mid: 50-10 = 40, fee 5, tax 45*0.21 = 9.45
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("50"), TotalFee: d("7"), TotalTax: d("11.97"), TotalDiscount: d("20"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Combined Fixed + Percentage applied sequentially with exact non-divisible quantity math", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-fixed-plus-pct", nil, true, time.Now())
		// 3 * 40.00 = 120.00; fixed discount 20 -> remaining 100.00 (not divisible by 3!); 10% of 100 = 10 -> final 90.00
		// TaxRate 20% -> 90.00 * 0.20 = 18.00; totalDiscount = 20 + 10 = 30.00
		_ = seedCartItem(t, cart, prodForty, 3)

		cpFixed := seedCoupon(t, "FXD200", "Fixed 20", false, "20.00")
		cpPct := seedCoupon(t, "PCT10A", "Pct 10", true, "10.00")
		_ = seedRedemption(t, cart, cpFixed, customer, false)
		_ = seedRedemption(t, cart, cpPct, customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("90"), TotalFee: d("0"), TotalTax: d("18"), TotalDiscount: d("30"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Invalidated and non-matching product redemptions are ignored", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-invalidated", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodMid, 1) // 50 price, 5 fee, tax 21% -> 11.55

		cpInvalidated := seedCoupon(t, "INVLD1", "Invalidated 25", false, "25.00")
		_ = seedRedemption(t, cart, cpInvalidated, customer, true) // Invalidated = true

		cpUnmatchedProd := seedCoupon(t, "OTHER1", "Other Product Only", false, "25.00", prodOther)
		_ = seedRedemption(t, cart, cpUnmatchedProd, customer, false) // prodOther not in cart

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		if len(cr.Coupons) != 0 {
			t.Fatalf("expected 0 active matching coupons, got %d", len(cr.Coupons))
		}
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("50"), TotalFee: d("5"), TotalTax: d("11.55"), TotalDiscount: d("0"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Multiple carts in one query keep items and redemptions strictly isolated", func(t *testing.T) {
		clearCartQueryTables(t)

		t0 := time.Now().Add(-time.Hour)
		cartA := seedCart(t, "sess-multi-a", nil, true, t0)
		cartB := seedCart(t, "sess-multi-b", nil, true, t0.Add(time.Minute))

		_ = seedCartItem(t, cartA, prodMid, 1)       // 50 + 5
		_ = seedCartItem(t, cartB, prodExpensive, 1) // 100 + 10

		cpFixedA := seedCoupon(t, "CARTA1", "Cart A 10", false, "10.00")
		cpPctB := seedCoupon(t, "CARTB1", "Cart B 25%", true, "25.00")
		rA := seedRedemption(t, cartA, cpFixedA, customer, false)
		rB := seedRedemption(t, cartB, cpPctB, customer, false)

		n, rows := collectCalc(t, carts.Query().Filter("Active", true).OrderBy("CreatedAt"))
		if n != 2 || len(rows) != 2 {
			t.Fatalf("expected 2 CartRows, got n=%d len=%d", n, len(rows))
		}

		// Cart A: 50-10 = 40, fee 5, tax 9.45, discount 10
		if rows[0].Object.ID != cartA.ID || len(rows[0].Coupons) != 1 || rows[0].Coupons[0].ID != rA.ID {
			t.Fatalf("unexpected Cart A row/coupons: id=%s coupons=%+v", rows[0].Object.ID, rows[0].Coupons)
		}
		assertTotalsEqual(t, "cartA", price.Totals{
			TotalPrice: d("40"), TotalFee: d("5"), TotalTax: d("9.45"), TotalDiscount: d("10"), LeftOverDiscount: d("0"),
		}, rows[0].Totals)

		// Cart B: 100 * 0.75 = 75, fee 10, tax 85*0.21 = 17.85, discount 25
		if rows[1].Object.ID != cartB.ID || len(rows[1].Coupons) != 1 || rows[1].Coupons[0].ID != rB.ID {
			t.Fatalf("unexpected Cart B row/coupons: id=%s coupons=%+v", rows[1].Object.ID, rows[1].Coupons)
		}
		assertTotalsEqual(t, "cartB", price.Totals{
			TotalPrice: d("75"), TotalFee: d("10"), TotalTax: d("17.85"), TotalDiscount: d("25"), LeftOverDiscount: d("0"),
		}, rows[1].Totals)
	})

	t.Run("Calc: Zero amount with percentage coupon and zero-price items do not panic", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-zero-edge", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodFree, 2)
		_ = seedCartItem(t, cart, prodFreeWithFee, 1) // 0 price, 8 fee, 10% tax
		_ = seedCartItem(t, cart, prodMid, 0)         // amount = 0

		cpFixed := seedCoupon(t, "ZEROFIX", "Zero Fixed 5", false, "5.00")
		cpPct := seedCoupon(t, "ZEROPCT", "Zero Pct 20", true, "20.00")
		_ = seedRedemption(t, cart, cpFixed, customer, false)
		_ = seedRedemption(t, cart, cpPct, customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		if len(cr.Items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(cr.Items))
		}
		// All three items have TotalPrice 0 before fixed coupon -> fixed 5 cascades onto prodFreeWithFee's 8 fee -> remaining 3 fee, tax 0.30
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("0"), TotalFee: d("3"), TotalTax: d("0.30"), TotalDiscount: d("5"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Works inside caller transaction and supports early iterator break", func(t *testing.T) {
		clearCartQueryTables(t)

		t0 := time.Now().Add(-time.Hour)
		c1 := seedCart(t, "sess-tx-1", nil, true, t0)
		c2 := seedCart(t, "sess-tx-2", nil, true, t0.Add(time.Minute))
		_ = seedCartItem(t, c1, prodMid, 1)
		_ = seedCartItem(t, c2, prodExpensive, 1)

		txCtx, tx, err := queries.StartTransaction(t.Context())
		if err != nil {
			t.Fatalf("failed to start outer tx: %v", err)
		}
		defer tx.Rollback(txCtx)

		n, seq, err := carts.Query().WithContext(txCtx).Filter("Active", true).OrderBy("CreatedAt").Calc()
		if err != nil {
			t.Fatalf("Calc inside outer tx failed: %v", err)
		}
		if n != 2 {
			t.Fatalf("expected n=2, got %d", n)
		}
		var count int
		for row, iterErr := range seq {
			if iterErr != nil {
				t.Fatalf("unexpected iterErr: %v", iterErr)
			}
			count++
			if !strings.HasPrefix(row.Object.Session, "sess-tx-") {
				t.Fatalf("unexpected session %q", row.Object.Session)
			}
			break // verify early break
		}
		if count != 1 {
			t.Fatalf("expected 1 yielded row before break, got %d", count)
		}
	})

	t.Run("Calc: Stacked item-specific and cart-wide fixed and percentage coupons combine accurately", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-full-stack", nil, true, time.Now())
		// Cheap: price 20 + fee 2; Item fixed 5 + 3 = 8 + cart fixed 12 = 20 -> price 0, fee 2, leftover 0
		// Pool consumed completely on Cheap!
		// Expensive: price 100 + fee 10; Item fixed 15 + 5 = 20 -> price 80, fee 10;
		// Percentage: item 10% + 5% + cart 25% = 40% of 80 = 32 discount -> price 48, fee 10, tax (58*0.21)=12.18
		_ = seedCartItem(t, cart, prodCheap, 1)
		_ = seedCartItem(t, cart, prodExpensive, 1)

		_ = seedRedemption(t, cart, seedCoupon(t, "CHPFI1", "Cheap F 3", false, "3.00", prodCheap), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "CHPFI2", "Cheap F 5", false, "5.00", prodCheap), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "CARTF1", "Cart F 12", false, "12.00"), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "EXPFI1", "Exp F 5", false, "5.00", prodExpensive), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "EXPFI2", "Exp F 15", false, "15.00", prodExpensive), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "EXPPC1", "Exp P 10", true, "10.00", prodExpensive), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "EXPPC2", "Exp P 5", true, "5.00", prodExpensive), customer, false)
		_ = seedRedemption(t, cart, seedCoupon(t, "CARTP1", "Cart P 25", true, "25.00"), customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		cr := rows[0]
		if len(cr.Coupons) != 8 {
			t.Fatalf("expected 8 coupons attached to cart, got %d", len(cr.Coupons))
		}
		// Cheap: price 0, fee 2, tax 0.42, fixed discount 20 (since 0% = 0 extra)
		assertTotalsEqual(t, "cheap", price.Totals{
			TotalPrice: d("0"), TotalFee: d("2"), TotalTax: d("0.42"), TotalDiscount: d("20"), LeftOverDiscount: d("0"),
		}, cr.Items[0].CalculatedPrice)
		// Expensive: fixed 20 + pct 32 = 52 discount
		assertTotalsEqual(t, "expensive", price.Totals{
			TotalPrice: d("48"), TotalFee: d("10"), TotalTax: d("12.18"), TotalDiscount: d("52"), LeftOverDiscount: d("0"),
		}, cr.Items[1].CalculatedPrice)
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("48"), TotalFee: d("12"), TotalTax: d("12.60"), TotalDiscount: d("72"), LeftOverDiscount: d("0"),
		}, cr.Totals)
	})

	t.Run("Calc: Composes cleanly with CartForRequest and PreloadProducts without duplicate rows", func(t *testing.T) {
		clearCartQueryTables(t)

		sm := newSessionStore()
		token, cookie := issueSessionToken(t, sm)
		// Cart matches BOTH Session and User simultaneously
		cart := seedCart(t, token, user1, true, time.Now())
		_ = seedCartItem(t, cart, prodMid, 2) // 100 price, 10 fee
		_ = seedRedemption(t, cart, seedCoupon(t, "COMBO1", "Combo 15", false, "15.00"), customer, false)

		req := buildRequestWithMiddleware(t, sm, cookie, user1)
		basePreloaded := carts.Query().CartForRequest(req).PreloadProducts("Title", "Slug")
		n, rows := collectCalc(t, basePreloaded)
		if n != 1 || len(rows) != 1 {
			t.Fatalf("expected exactly 1 deduplicated CartRow, got n=%d len=%d", n, len(rows))
		}
		if rows[0].Items[0].CartItem.Product.Title != "Mouse" || rows[0].Items[0].CartItem.Product.Slug != "mouse" {
			t.Fatalf("expected PreloadProducts fields Title/Slug preserved, got title=%q slug=%q",
				rows[0].Items[0].CartItem.Product.Title,
				rows[0].Items[0].CartItem.Product.Slug,
			)
		}
		// 100 - 15 = 85 price, fee 10, tax (95*0.21)=19.95
		assertTotalsEqual(t, "cart", price.Totals{
			TotalPrice: d("85"), TotalFee: d("10"), TotalTax: d("19.95"), TotalDiscount: d("15"), LeftOverDiscount: d("0"),
		}, rows[0].Totals)

		// Verify PreloadProducts() (Product.*) + Calc() also works cleanly
		nAll, rowsAll := collectCalc(t, carts.Query().CartForRequest(req).PreloadProducts())
		if nAll != 1 || len(rowsAll) != 1 || rowsAll[0].Items[0].CartItem.Product.Title != "Mouse" {
			t.Fatalf("expected PreloadProducts().Calc() to preserve all Product fields and totals, got n=%d len=%d", nAll, len(rowsAll))
		}
		assertTotalsEqual(t, "cartAll", price.Totals{
			TotalPrice: d("85"), TotalFee: d("10"), TotalTax: d("19.95"), TotalDiscount: d("15"), LeftOverDiscount: d("0"),
		}, rowsAll[0].Totals)
	})

	t.Run("Calc: Cancelled context returns wrapped error and zero coupon leaves prices unchanged", func(t *testing.T) {
		clearCartQueryTables(t)

		cart := seedCart(t, "sess-zero-coupon", nil, true, time.Now())
		_ = seedCartItem(t, cart, prodMid, 1) // 50 + 5, 21% tax
		_ = seedRedemption(t, cart, seedCoupon(t, "ZROCPN", "Zero Coupon", false, "0.00"), customer, false)

		_, rows := collectCalc(t, carts.Query().Filter("ID", cart.ID))
		assertTotalsEqual(t, "zero-coupon cart", price.Totals{
			TotalPrice: d("50"), TotalFee: d("5"), TotalTax: d("11.55"), TotalDiscount: d("0"), LeftOverDiscount: d("0"),
		}, rows[0].Totals)

		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()

		n, seq, err := carts.Query().WithContext(cancelledCtx).Filter("ID", cart.ID).Calc()
		if err == nil {
			t.Fatal("expected error for cancelled context in Calc()")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected error wrapping context.Canceled, got: %v", err)
		}
		if n != 0 || seq != nil {
			t.Fatalf("expected n=0 and seq=nil on error, got n=%d seq=%v", n, seq)
		}
	})
}
