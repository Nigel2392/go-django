//go:build test
// +build test

package adminviews_test

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/auth/users"
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
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
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
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/PuerkitoBio/goquery"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

type AdminProductTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
	Perms           map[string]bool `attrs:"-"`
}

func (u *AdminProductTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions[*AdminProductTestUser](ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("admin_product_test_users")
}

func (u *AdminProductTestUser) IsAuthenticated() bool {
	return u.LoggedIn
}

func (u *AdminProductTestUser) IsAdmin() bool {
	return u.IsAdministrator
}

func (u *AdminProductTestUser) HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool {
	if u.IsAdministrator {
		return true
	}
	for _, p := range perms {
		if u.Perms == nil || !u.Perms[p] {
			return false
		}
	}
	return true
}

type testAdminSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *testAdminSupplierProduct) ProductPrice(ctx context.Context) price.Money {
	return price.Money{
		Currency: currency.EUR,
		Amount:   p.Price,
	}
}

func (p *testAdminSupplierProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{
			Instance: p,
			Image:    &mediafiles.SimpleStoredObject{Filepath: "default-product.png"},
			Editor:   &editor.EditorJSBlockData{},
		}
	}
	return attrs.Make[*testAdminSupplierProduct, any](ctx, p,
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
		fattrs.Field(p, "SKU", &p.SKU, func() fattrs.PtrFieldConfig[*testAdminSupplierProduct, string] {
			return fattrs.PtrFieldConfig[*testAdminSupplierProduct, string]{
				Config: attrs.Config{
					Blank: false,
				},
			}
		}),
	)
}

func testProductPanels(ctx context.Context) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Slug"),
		admin.FieldPanel("Category"),
		admin.FieldPanel("Price"),
		admin.FieldPanel("TaxRate"),
		admin.FieldPanel("DiscountPrice"),
		admin.FieldPanel("Ranking"),
		admin.FieldPanel("SKU"),
	}
}

func seedAdminProduct(t *testing.T, cat *models.Category, sup string, title, slug, sku, price, discount string) (*testAdminSupplierProduct, *products.Product) {
	t.Helper()

	spec := &testAdminSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      products.SupplierValue(sup),
			Price:         decimal.RequireFromString(price),
			TaxRate:       decimal.RequireFromString("21.00"),
			DiscountPrice: decimal.RequireFromString(discount),
			Ranking:       1,
		},
		SKU: sku,
	}
	spec.Product.Instance = spec

	if _, err := queries.GetQuerySet(&testAdminSupplierProduct{}).Create(spec); err != nil {
		t.Fatalf("failed to create specific product: %v", err)
	}
	if _, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	return spec, spec.Product
}

func TestAdminProductViews(t *testing.T) {
	var _, db = testdb.Open()
	defer db.Close()

	mediafiles.RegisterBackend("memory", memory.NewBackend(5))
	mediafiles.SetDefault("memory")
	permissions.Tester = users.NewPermissionsBackend()

	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})
	autherrors.RegisterHook("index")

	attrs.RegisterModel(&AdminProductTestUser{})
	admin.RegisterApp(
		"admin_product_test",
		admin.AppOptions{
			EnableIndexView: true,
		},
		admin.ModelOptions{Name: "User", Model: &AdminProductTestUser{}},
	)

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &AdminProductTestUser{},
		GetObject:      func() any { return &AdminProductTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "Admin Product Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "Admin Product Test Users" },
	})

	orders.RegisterSupplier(&orders.Supplier{
		Order: 1,
		Name:  "alpha_supplier",
		HTMLIcon: func(ctx context.Context) template.HTML {
			return `<svg class="icon-alpha"></svg>`
		},
		TextLabel: func(ctx context.Context) string {
			return "Alpha Supplier"
		},
		TextDescription: func(ctx context.Context) string {
			return "Primary alpha product supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &testAdminSupplierProduct{}
		},
		AddPanel:  testProductPanels,
		EditPanel: testProductPanels,
	})

	orders.RegisterSupplier(&orders.Supplier{
		Order: 2,
		Name:  "beta_supplier",
		TextLabel: func(ctx context.Context) string {
			return "Beta Supplier"
		},
		TextDescription: func(ctx context.Context) string {
			return "Secondary beta product supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &testAdminSupplierProduct{}
		},
		AddPanel:  testProductPanels,
		EditPanel: testProductPanels,
	})

	setupApp := func() django.AppConfig {
		cfg := apps.NewDBAppConfig("admin_product_test")
		cfg.ModelObjects = []attrs.Definer{
			&AdminProductTestUser{},
			&testAdminSupplierProduct{},
		}
		cfg.Init = func(settings django.Settings, db drivers.Database) error {
			if ed, err := migrator.GetSchemaEditor(db.Driver()); err == nil {
				_ = ed.DropTable(context.Background(), migrator.NewModelTable(&auditlogs.Entry{}), true)
			}
			tables := quest.Table(t,
				&AdminProductTestUser{},
				&auditlogs.Entry{},
				&models.Category{},
				&orders.SupplierModel{},
				&testAdminSupplierProduct{},
				&products.Product{},
				&carts.Cart{},
				&carts.CartItem{},
				&coupons.Coupon{},
				&coupons.CouponProductThrough{},
				&customers.Customer{},
				&orders.Order{},
				&orders.OrderItem{},
				&payments.ProviderAccount{},
				&payments.Payment{},
				&coupons.Redemption{},
			)
			tables.IfNotExists = true
			tables.Create()
			_, _ = queries.GetQuerySet(&coupons.CouponProductThrough{}).Delete()
			_, _ = queries.GetQuerySet(&coupons.Redemption{}).Delete()
			_, _ = queries.GetQuerySet(&coupons.Coupon{}).Delete()
			_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
			_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
			_, _ = queries.GetQuerySet(&products.Product{}).Delete()
			if err := orders.SuppliersQuery().Sync(); err != nil {
				return err
			}
			return nil
		}
		return cfg
	}

	var (
		catHardware *models.Category
		catAudio    *models.Category
		seededProd1 *products.Product
	)

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
				return &AdminProductTestUser{LoggedIn: false, IsAdministrator: false}
			},
			Users: map[string]authentication.User{
				"admin_user": &AdminProductTestUser{
					ID:              1,
					Name:            "admin_user",
					LoggedIn:        true,
					IsAdministrator: true,
				},
				"staff_no_perms": &AdminProductTestUser{
					ID:              2,
					Name:            "staff_no_perms",
					LoggedIn:        true,
					IsAdministrator: false,
					Perms: map[string]bool{
						"admin:access_admin": true,
					},
				},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Legacy supplier content types resolve to the orders model",
				Function: func(d *djester.Tester, t *testing.T) {
					for _, name := range []string{
						"github.com/Nigel2392/go-django/contrib/shop/supplier.SupplierModel",
						"github.com/Nigel2392/go-django/contrib/shop/products/supplier.SupplierModel",
					} {
						definition := contenttypes.DefinitionForType(name)
						if definition == nil {
							t.Fatalf("content type %q did not resolve", name)
						}
						if _, ok := definition.Object().(*orders.SupplierModel); !ok {
							t.Fatalf("content type %q resolved to %T", name, definition.Object())
						}
					}
				},
			},
			&djester.BasicTest{
				Label: "Seed categories, users, and initial products",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := queries.GetQuerySet(&AdminProductTestUser{}).ExplicitSave().Create(&AdminProductTestUser{
						ID:              1,
						Name:            "admin_user",
						LoggedIn:        true,
						IsAdministrator: true,
					})
					if err != nil {
						t.Fatalf("failed to seed admin_user: %v", err)
					}

					catHardware = &models.Category{Name: "Computer Hardware", Slug: "computer-hardware"}
					if _, err := queries.GetQuerySet(&models.Category{}).Create(catHardware); err != nil {
						t.Fatalf("failed to seed catHardware: %v", err)
					}

					catAudio = &models.Category{Name: "Studio Audio", Slug: "studio-audio"}
					if _, err := queries.GetQuerySet(&models.Category{}).Create(catAudio); err != nil {
						t.Fatalf("failed to seed catAudio: %v", err)
					}

					_, seededProd1 = seedAdminProduct(t, catHardware, "alpha_supplier", "Ultrawide Monitor", "ultrawide-monitor", "SKU-MON-01", "499.99", "449.99")
					_, _ = seedAdminProduct(t, catAudio, "beta_supplier", "Condenser Microphone", "condenser-microphone", "SKU-MIC-02", "149.50", "0.00")
				},
			},
			&djester.BasicTest{
				Label: "Order edit displays redeemed coupons and payment history",
				Function: func(d *djester.Tester, t *testing.T) {
					customer, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
						Email:     drivers.MustParseEmail("order-admin@example.test"),
						FirstName: "Order",
						LastName:  "Customer",
					})
					require.NoError(t, err)
					req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(t.Context())
					login, err := d.Login("admin_user")
					require.NoError(t, err)
					login.Body.Close()
					defer d.Logout()
					getOrderEdit := func(editURL string) *goquery.Document {
						t.Helper()
						resp, err := d.Get(editURL, nil, nil)
						require.NoError(t, err)
						defer resp.Body.Close()
						require.Equal(t, http.StatusOK, resp.StatusCode)
						doc, err := resp.DOM()
						require.NoError(t, err)
						return doc
					}

					emptyOrder, _, err := orders.CreateOrder(req, customer, nil, &carts.CartItem{
						Product: seededProd1,
						Amount:  1,
					})
					require.NoError(t, err)
					emptyEditURL := django.Reverse("admin:apps:model:edit", "shop", "orders", emptyOrder.ID)
					emptyDoc := getOrderEdit(emptyEditURL)
					require.Contains(t, emptyDoc.Text(), "Redeemed Coupons")
					require.Contains(t, emptyDoc.Text(), "No coupons redeemed for this order.")

					cart, err := queries.GetQuerySet(&carts.Cart{}).Create(&carts.Cart{Active: true})
					require.NoError(t, err)
					_, err = queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
						Cart:    cart,
						Product: seededProd1,
						Amount:  2,
					})
					require.NoError(t, err)
					coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        "<SAVE-12>",
						Title:       "Order discount",
						MaxDiscount: decimal.RequireFromString("12.50"),
					})
					require.NoError(t, err)
					couponIDs := []uint64{coupon.ID}
					t.Cleanup(func() {
						_, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("Coupon__in", couponIDs).Delete()
						require.NoError(t, err)
						_, err = queries.GetQuerySet(&coupons.Coupon{}).Filter("ID__in", couponIDs).Delete()
						require.NoError(t, err)
					})
					redeemedAt := time.Date(2025, time.March, 4, 5, 6, 7, 0, time.UTC)
					redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Cart:       cart,
						Coupon:     coupon,
						RedeemedAt: redeemedAt,
					})
					require.NoError(t, err)
					priced, err := carts.
						Query().
						WithContext(t.Context()).
						Filter("ID", cart.ID).
						CalcOneWithProducts()
					require.NoError(t, err)
					order, _, err := orders.CreateOrderFromCart(req, customer, priced)
					require.NoError(t, err)
					legacyCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        "LEGACY",
						Title:       "Earlier discount",
						MaxDiscount: decimal.NewFromInt(5),
					})
					require.NoError(t, err)
					couponIDs = append(couponIDs, legacyCoupon.ID)
					_, err = queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Order:       order,
						Customer:    customer,
						Coupon:      legacyCoupon,
						Invalidated: true,
					})
					require.NoError(t, err)
					payment, err := queries.GetQuerySet(&payments.Payment{}).Create(&payments.Payment{
						Order: order, Provider: "<admin-test-payment>", State: paystate.STATUS_INIT,
						Amount: decimal.NewFromInt(25), Currency: "EUR",
						History: drivers.JSON[[]payments.PaymentEvent]{
							Data: []payments.PaymentEvent{
								{
									At:    time.Unix(1, 0).UTC(),
									Kind:  payments.EventCheckoutOpened,
									State: paystate.STATUS_PENDING,
								},
								{
									At:    time.Unix(2, 0).UTC(),
									Kind:  payments.EventStatusUpdated,
									State: paystate.STATUS_INIT,
								},
							},
						},
					})
					require.NoError(t, err)

					editURL := django.Reverse("admin:apps:model:edit", "shop", "orders", order.ID)
					doc := getOrderEdit(editURL)
					require.Contains(t, doc.Text(), "Payment "+payment.ID.String())
					require.Contains(t, doc.Text(), "<admin-test-payment>")
					require.Zero(t, doc.Find("admin-test-payment").Length())
					events := doc.Find(".panel-alert ol li")
					require.Equal(t, 2, events.Length())
					require.Contains(t, events.Eq(0).Text(), "Provider checkout opened")
					require.Contains(t, events.Eq(1).Text(), "Provider status updated")
					require.Contains(t, doc.Text(), "Redeemed Coupons")
					require.Contains(t, doc.Text(), "Coupon <SAVE-12>")
					require.Contains(t, doc.Text(), "Order discount")
					require.Contains(t, doc.Text(), "Reserved for this order")
					require.Equal(t, 1, doc.Find(`time[datetime="`+redeemedAt.Format(time.RFC3339)+`"]`).Length())
					require.Zero(t, doc.Find("save-12").Length())
					require.Contains(t, doc.Find(".panel-alert ul li").Text(), "Ultrawide Monitor · 12.5 discount")
					require.Contains(t, doc.Text(), "Coupon LEGACY")
					require.Contains(t, doc.Text(), "Discount amount not recorded")
					require.Contains(t, doc.Text(), "Reservation released")
					action, ok := doc.Find("form.admin-form").Attr("action")
					require.True(t, ok)
					require.Equal(t, editURL, action)

					// Submit the rendered inline form's fields along with Completed.
					data := extractAdminFormValues(doc)
					data["State"] = "order_pending"
					data["Completed"] = "on"
					saved, err := d.PostForm(editURL, nil, nil, data)
					require.NoError(t, err)
					require.Equal(t, http.StatusOK, saved.StatusCode)
					savedDoc, err := saved.DOM()
					require.NoError(t, err)
					saved.Body.Close()
					require.Empty(t, strings.TrimSpace(savedDoc.Find(".error-list").Text()))

					row, err := queries.GetQuerySet(&orders.Order{}).Filter("ID", order.ID).Get()
					require.NoError(t, err)
					require.True(t, row.Object.Completed)
					applied, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
					require.NoError(t, err)
					require.True(t, applied.Object.Counted)
					used, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.EqualValues(t, 1, used.Object.UsageCount)
					completedDoc := getOrderEdit(editURL)
					require.Contains(t, completedDoc.Text(), "Counted toward coupon usage")
				},
			},
			&djester.BasicTest{
				Label: "Permission checks redirect unauthorized staff user to login",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("staff_no_perms")
					asserter.Assert(err == nil, "Login as staff_no_perms failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					endpoints := []string{
						django.Reverse("admin:apps:model", "shop", "products"),
						django.Reverse("admin:apps:model:add", "shop", "products"),
						django.Reverse("admin:commerce:new_with_supplier", "alpha_supplier"),
						django.Reverse("admin:apps:model:edit", "shop", "products", seededProd1.ID.String()),
					}

					for _, ep := range endpoints {
						resp, err := d.Get(ep, nil, nil)
						asserter.Assert(err == nil, "GET %s failed: %v", ep, err)
						// Following redirect from admin.ReLogin lands on /admin/relogin/
						asserter.Assert(
							strings.Contains(resp.Request.URL.Path, "/admin/relogin"),
							"Expected unauthorized user on %s to be redirected to /admin/relogin, landed on %s",
							ep, resp.Request.URL.Path,
						)
						resp.Body.Close()
					}
				},
			},
			&djester.BasicTest{
				Label: "ViewProductList renders product columns and filters by search query",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("admin_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					listURL := django.Reverse("admin:apps:model", "shop", "products")

					// 1. Full list renders both products, their categories, prices, and edit links
					resp, err := d.Get(listURL, nil, nil)
					asserter.Assert(err == nil, "GET product list failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)

					doc, err := resp.DOM()
					asserter.Assert(err == nil, "Failed to parse HTML document: %v", err)
					htmlText := doc.Text()
					asserter.Assert(strings.Contains(htmlText, "Ultrawide Monitor"), "Expected Ultrawide Monitor in list")
					asserter.Assert(strings.Contains(htmlText, "Condenser Microphone"), "Expected Condenser Microphone in list")
					asserter.Assert(strings.Contains(htmlText, "Computer Hardware"), "Expected Computer Hardware category in list")
					asserter.Assert(strings.Contains(htmlText, "Studio Audio"), "Expected Studio Audio category in list")

					editHref1 := django.Reverse("admin:apps:model:edit", "shop", "products", seededProd1.ID.String())
					resp.Assert(t, true).AssertHTML(
						djester.HasElement(fmt.Sprintf(`a[href="%s"]`, editHref1)),
					)
					resp.Body.Close()

					// 2. Search by Title ("Ultrawide") -> matches only seededProd1
					resp, err = d.Get(listURL, nil, url.Values{"filters-search": {"Ultrawide"}})
					asserter.Assert(err == nil, "GET filters-search=Ultrawide failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					doc, err = resp.DOM()
					asserter.Assert(err == nil, "Failed to parse HTML: %v", err)
					htmlText = doc.Text()
					asserter.Assert(strings.Contains(htmlText, "Ultrawide Monitor"), "Expected Ultrawide Monitor for filters-search=Ultrawide")
					asserter.Assert(!strings.Contains(htmlText, "Condenser Microphone"), "Did not expect Condenser Microphone for filters-search=Ultrawide")
					resp.Body.Close()

					// 3. Search by Slug ("condenser-microphone") -> matches only seededProd2
					resp, err = d.Get(listURL, nil, url.Values{"filters-search": {"condenser-microphone"}})
					asserter.Assert(err == nil, "GET filters-search=condenser-microphone failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					doc, err = resp.DOM()
					asserter.Assert(err == nil, "Failed to parse HTML: %v", err)
					htmlText = doc.Text()
					asserter.Assert(!strings.Contains(htmlText, "Ultrawide Monitor"), "Did not expect Ultrawide Monitor for slug search")
					asserter.Assert(strings.Contains(htmlText, "Condenser Microphone"), "Expected Condenser Microphone for slug search")
					resp.Body.Close()

					// 4. Search by Category.Name ("Studio Audio") -> matches only seededProd2
					resp, err = d.Get(listURL, nil, url.Values{"filters-search": {"Studio Audio"}})
					asserter.Assert(err == nil, "GET filters-search=Studio Audio failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					doc, err = resp.DOM()
					asserter.Assert(err == nil, "Failed to parse HTML: %v", err)
					htmlText = doc.Text()
					asserter.Assert(!strings.Contains(htmlText, "Ultrawide Monitor"), "Did not expect Ultrawide Monitor for category search")
					asserter.Assert(strings.Contains(htmlText, "Condenser Microphone"), "Expected Condenser Microphone for category search")
					resp.Body.Close()

					// 5. Search by Supplier ("alpha_supplier") -> matches only seededProd1
					resp, err = d.Get(listURL, nil, url.Values{"filters-search": {"alpha_supplier"}})
					asserter.Assert(err == nil, "GET filters-search=alpha_supplier failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)
					doc, err = resp.DOM()
					asserter.Assert(err == nil, "Failed to parse HTML: %v", err)
					htmlText = doc.Text()
					asserter.Assert(strings.Contains(htmlText, "Ultrawide Monitor"), "Expected Ultrawide Monitor for supplier search")
					asserter.Assert(!strings.Contains(htmlText, "Condenser Microphone"), "Did not expect Condenser Microphone for supplier search")
					resp.Body.Close()
				},
			},
			&djester.BasicTest{
				Label: "ViewAddNewProductSupplier lists enabled, disabled, and missing suppliers in order",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("admin_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					// Mark beta_supplier as disabled in DB and insert an unregistered supplier row
					_, err = queries.GetQuerySet(&orders.SupplierModel{}).
						Select("Enabled").
						Filter("InternalName", "beta_supplier").
						Update(&orders.SupplierModel{Enabled: false})
					asserter.Assert(err == nil, "Failed to disable beta_supplier: %v", err)
					defer func() {
						_, _ = queries.GetQuerySet(&orders.SupplierModel{}).
							Select("Enabled").
							Filter("InternalName", "beta_supplier").
							Update(&orders.SupplierModel{Enabled: true})
					}()

					_, err = queries.GetQuerySet(&orders.SupplierModel{}).ExplicitSave().Create(&orders.SupplierModel{
						InternalName: "legacy_unregistered",
						Enabled:      false,
					})
					asserter.Assert(err == nil, "Failed to insert legacy_unregistered supplier: %v", err)
					defer func() {
						_, _ = queries.GetQuerySet(&orders.SupplierModel{}).Filter("InternalName", "legacy_unregistered").Delete()
					}()

					addSupplierURL := django.Reverse("admin:apps:model:add", "shop", "products")
					resp, err := d.Get(addSupplierURL, nil, nil)
					asserter.Assert(err == nil, "GET supplier selection view failed: %v", err)
					asserter.AssertEqual(http.StatusOK, resp.StatusCode)

					doc, err := resp.DOM()
					asserter.Assert(err == nil, "Failed to parse supplier HTML: %v", err)

					var hrefs []string
					doc.Find("div.admin-list a.admin-list-item").Each(func(i int, s *goquery.Selection) {
						if h, ok := s.Attr("href"); ok {
							hrefs = append(hrefs, h)
						}
					})

					alphaHref := django.Reverse("admin:commerce:new_with_supplier", "alpha_supplier")
					betaHref := django.Reverse("admin:commerce:new_with_supplier", "beta_supplier")
					legacyHref := django.Reverse("admin:commerce:new_with_supplier", "legacy_unregistered")

					asserter.AssertContains(hrefs, alphaHref)
					asserter.AssertContains(hrefs, betaHref)
					asserter.AssertContains(hrefs, legacyHref)

					// Verify enabled alpha_supplier comes before disabled beta_supplier and legacy_unregistered
					idxAlpha, idxBeta, idxLegacy := -1, -1, -1
					for i, h := range hrefs {
						switch h {
						case alphaHref:
							idxAlpha = i
						case betaHref:
							idxBeta = i
						case legacyHref:
							idxLegacy = i
						}
					}
					asserter.Assert(idxAlpha < idxLegacy, "Expected enabled alpha_supplier (%d) before disabled legacy_unregistered (%d)", idxAlpha, idxLegacy)
					asserter.Assert(idxLegacy < idxBeta, "Expected disabled legacy_unregistered (Order 0, idx %d) before disabled beta_supplier (Order 20, idx %d)", idxLegacy, idxBeta)

					htmlText := doc.Text()
					asserter.Assert(strings.Contains(htmlText, "Alpha Supplier"), "Expected Alpha Supplier label")
					asserter.Assert(strings.Contains(htmlText, "Beta Supplier (Disabled)"), "Expected disabled label for Beta Supplier")
					asserter.Assert(strings.Contains(htmlText, "legacy_unregistered (Disabled)"), "Expected fallback disabled label for unregistered supplier")
					resp.Body.Close()
				},
			},
			&djester.BasicTest{
				Label: "ViewAddProduct GET and POST create specific and base Product and emit Created signal",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("admin_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					// 1. Invalid supplier returns 404 Not Found
					invalidURL := django.Reverse("admin:commerce:new_with_supplier", "non_existent_supplier")
					invResp, err := d.Get(invalidURL, nil, nil)
					asserter.Assert(err == nil, "GET invalid supplier failed: %v", err)
					asserter.AssertEqual(http.StatusNotFound, invResp.StatusCode)
					invResp.Body.Close()

					// 2. GET with valid supplier renders the add form with both base and specific fields
					addURL := django.Reverse("admin:commerce:new_with_supplier", "alpha_supplier")
					getResp, err := d.Get(addURL, nil, nil)
					asserter.Assert(err == nil, "GET ViewAddProduct failed: %v", err)
					asserter.AssertEqual(http.StatusOK, getResp.StatusCode)
					getResp.Assert(t, true).AssertHTML(
						djester.HasElement(`input[name="Title"]`),
						djester.HasElement(`input[name="Slug"]`),
						djester.HasElement(`input[name="Price"]`),
						djester.HasElement(`input[name="SKU"]`),
					)
					getResp.Body.Close()

					// 3. POST creates specific and base Product, fires products.Signals.Created, and redirects
					var createdSignals []products.ProductCreatedSignal
					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()
					_, err = products.Signals.Created.Listen(sigCtx, func(ctx context.Context, s signals.Signal[products.ProductCreatedSignal], sig products.ProductCreatedSignal) error {
						createdSignals = append(createdSignals, sig)
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to products.Signals.Created: %v", err)

					postResp, err := d.PostForm(addURL, nil, nil, map[string]any{
						"Title":         "Ergonomic Trackball",
						"Slug":          "ergonomic-trackball",
						"Category":      fmt.Sprintf("%d", catHardware.ID),
						"Price":         "89.95",
						"TaxRate":       "21.00",
						"DiscountPrice": "79.95",
						"Ranking":       "5",
						"SKU":           "SKU-TRK-99",
					})
					asserter.Assert(err == nil, "POST ViewAddProduct failed: %v", err)
					asserter.AssertEqual(http.StatusOK, postResp.StatusCode)
					postResp.Body.Close()

					asserter.AssertEqual(1, len(createdSignals))
					createdProd := createdSignals[0].Product
					asserter.AssertNotNil(createdProd)
					asserter.AssertNotNil(createdSignals[0].Specific)
					asserter.AssertEqual("Ergonomic Trackball", createdProd.Title)
					asserter.AssertEqual("ergonomic-trackball", createdProd.Slug)
					asserter.AssertEqual(products.SupplierValue("alpha_supplier"), createdProd.Supplier)
					asserter.Assert(createdProd.Price.Equal(decimal.RequireFromString("89.95")), "Expected Price 89.95, got %s", createdProd.Price)

					// Verify base Product and specific ProductModel in DB
					dbProdRow, err := queries.GetQuerySet(&products.Product{}).
						Filter("Slug", "ergonomic-trackball").
						Get()
					asserter.Assert(err == nil, "Failed to query created Product from DB: %v", err)
					asserter.AssertEqual("Ergonomic Trackball", dbProdRow.Object.Title)
					asserter.AssertEqual(products.SupplierValue("alpha_supplier"), dbProdRow.Object.Supplier)

					specModel, err := dbProdRow.Object.Specific.Specific(t.Context(), true)
					asserter.Assert(err == nil, "Failed to load Specific model for created Product: %v", err)
					specTyped, ok := specModel.(*testAdminSupplierProduct)
					asserter.Assert(ok, "Expected *testAdminSupplierProduct, got %T", specModel)
					asserter.AssertEqual("SKU-TRK-99", specTyped.SKU)
				},
			},
			&djester.BasicTest{
				Label: "ViewEditProduct GET and POST update specific and base Product and emit Updated signal",
				Function: func(d *djester.Tester, t *testing.T) {
					asserter := djester.Asserter(t, true)
					loginResp, err := d.Login("admin_user")
					asserter.Assert(err == nil, "Login failed: %v", err)
					loginResp.Body.Close()
					defer d.Logout()

					editURL := django.Reverse("admin:apps:model:edit", "shop", "products", seededProd1.ID.String())

					// 1. GET renders edit form pre-populated with existing base and specific values
					getResp, err := d.Get(editURL, nil, nil)
					asserter.Assert(err == nil, "GET ViewEditProduct failed: %v", err)
					asserter.AssertEqual(http.StatusOK, getResp.StatusCode)
					getResp.Assert(t, true).AssertHTML(
						djester.HasElement(`input[name="Title"][value="Ultrawide Monitor"]`),
						djester.HasElement(`input[name="Slug"][value="ultrawide-monitor"]`),
						djester.HasElement(`input[name="SKU"][value="SKU-MON-01"]`),
					)
					getResp.Body.Close()

					// 2. POST updates both base Product and specific ProductModel and emits products.Signals.Updated
					var updatedSignals []products.ProductUpdatedSignal
					sigCtx, cancelSig := context.WithCancel(t.Context())
					defer cancelSig()
					_, err = products.Signals.Updated.Listen(sigCtx, func(ctx context.Context, s signals.Signal[products.ProductUpdatedSignal], sig products.ProductUpdatedSignal) error {
						updatedSignals = append(updatedSignals, sig)
						return nil
					})
					asserter.Assert(err == nil, "Failed to listen to products.Signals.Updated: %v", err)

					postResp, err := d.PostForm(editURL, nil, nil, map[string]any{
						"Title":         "Curved OLED Monitor",
						"Slug":          "curved-oled-monitor",
						"Category":      fmt.Sprintf("%d", catHardware.ID),
						"Price":         "649.00",
						"TaxRate":       "21.00",
						"DiscountPrice": "599.00",
						"Ranking":       "10",
						"SKU":           "SKU-MON-OLED-02",
					})
					asserter.Assert(err == nil, "POST ViewEditProduct failed: %v", err)
					asserter.AssertEqual(http.StatusOK, postResp.StatusCode)
					postResp.Body.Close()

					asserter.AssertEqual(1, len(updatedSignals))
					change := updatedSignals[0].Product
					asserter.AssertNotNil(change.Old)
					asserter.AssertNotNil(change.New)
					asserter.AssertEqual("Ultrawide Monitor", change.Old.Title)
					asserter.AssertEqual("Curved OLED Monitor", change.New.Title)
					asserter.Assert(change.Old.Price.Equal(decimal.RequireFromString("499.99")), "Expected Old.Price 499.99, got %s", change.Old.Price)
					asserter.Assert(change.New.Price.Equal(decimal.RequireFromString("649.00")), "Expected New.Price 649.00, got %s", change.New.Price)

					// Verify updated base Product and specific ProductModel in DB
					updatedRow, err := queries.GetQuerySet(&products.Product{}).
						Filter("ID", seededProd1.ID).
						Get()
					asserter.Assert(err == nil, "Failed to query updated Product from DB: %v", err)
					asserter.AssertEqual("Curved OLED Monitor", updatedRow.Object.Title)
					asserter.AssertEqual("curved-oled-monitor", updatedRow.Object.Slug)
					asserter.Assert(updatedRow.Object.Price.Equal(decimal.RequireFromString("649.00")), "Expected updated DB price 649.00, got %s", updatedRow.Object.Price)
					asserter.Assert(updatedRow.Object.DiscountPrice.Equal(decimal.RequireFromString("599.00")), "Expected updated DB discount price 599.00, got %s", updatedRow.Object.DiscountPrice)
					asserter.AssertEqual(uint64(1), updatedRow.Object.Ranking)

					specModel, err := updatedRow.Object.Specific.Specific(t.Context(), true)
					asserter.Assert(err == nil, "Failed to load updated Specific model: %v", err)
					specTyped, ok := specModel.(*testAdminSupplierProduct)
					asserter.Assert(ok, "Expected *testAdminSupplierProduct, got %T", specModel)
					asserter.AssertEqual("SKU-MON-OLED-02", specTyped.SKU)
				},
			},
		},
	}

	d.Tests = append(d.Tests, adminCouponUITests()...)

	defer func() {
		_, _ = queries.GetQuerySet(&coupons.CouponProductThrough{}).Delete()
		_, _ = queries.GetQuerySet(&coupons.Redemption{}).Delete()
		_, _ = queries.GetQuerySet(&coupons.Coupon{}).Delete()
		_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
		_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()
		_, _ = queries.GetQuerySet(&products.Product{}).Delete()
	}()
	d.Test(t)
}
