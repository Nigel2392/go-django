//go:build test

package shop_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	"github.com/Nigel2392/go-django/contrib/shop/handlers"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
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
	gosignals "github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

type e2eTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
}

func (u *e2eTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("e2e_test_users")
}

func (u *e2eTestUser) IsAuthenticated() bool {
	return u.LoggedIn
}

func (u *e2eTestUser) IsAdmin() bool {
	return u.IsAdministrator
}

type e2eSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *e2eSupplierProduct) ProductPrice(ctx context.Context) price.Money {
	return price.Money{
		Currency: currency.EUR,
		Amount:   p.Price,
	}
}

func (p *e2eSupplierProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*e2eSupplierProduct, any](ctx, p,
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
		fattrs.Field(p, "SKU", &p.SKU, func() fattrs.PtrFieldConfig[*e2eSupplierProduct, string] {
			return fattrs.PtrFieldConfig[*e2eSupplierProduct, string]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
	)
}

func signalErrorContains(err error, target error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, target) {
		return true
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil && signalErrorContains(unwrapped, target) {
		return true
	}
	sigErr, ok := gosignals.SignalError(err)
	if !ok {
		return false
	}
	if sigErr.Reason != nil && sigErr.Reason != err && signalErrorContains(sigErr.Reason, target) {
		return true
	}
	for _, rel := range sigErr.Related {
		if signalErrorContains(rel, target) {
			return true
		}
	}
	return false
}

func seedE2EProduct(t *testing.T, cat *models.Category, supplierName, title, slug, sku, price string) *products.Product {
	t.Helper()
	spec := &e2eSupplierProduct{
		Product: &products.Product{
			Image:         &mediafiles.SimpleStoredObject{Filepath: slug + ".png"},
			Title:         title,
			Slug:          slug,
			Category:      cat,
			Editor:        &editor.EditorJSBlockData{},
			Supplier:      products.SupplierValue(supplierName),
			Price:         decimal.RequireFromString(price),
			TaxRate:       decimal.RequireFromString("21.00"),
			DiscountPrice: decimal.Zero,
			Ranking:       1,
		},
		SKU: sku,
	}
	spec.Product.Instance = spec

	_, err := queries.GetQuerySet(&e2eSupplierProduct{}).Create(spec)
	require.NoError(t, err)

	created, err := queries.GetQuerySet(&products.Product{}).Create(spec.Product)
	require.NoError(t, err)
	return created
}

func auditEntriesForOrder(t *testing.T, orderID drivers.ULID) map[string][]*auditlogs.Entry {
	t.Helper()
	rows, err := queries.GetQuerySet(&auditlogs.Entry{}).
		OrderBy("Timestamp").
		All()
	require.NoError(t, err)

	byType := make(map[string][]*auditlogs.Entry)
	wantID := orderID.String()
	for _, row := range rows {
		entry := row.Object
		if attrs.ToString(entry.ObjectID()) == wantID {
			byType[entry.Type()] = append(byType[entry.Type()], entry)
		}
	}
	return byType
}

func requireSignalErrorContains(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	if !signalErrorContains(err, target) {
		t.Fatalf("expected error chain/SignalError to contain %v, got: %v", target, err)
	}
}

func TestSupplierWorkflowE2E(t *testing.T) {
	_, db := testdb.Open()
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
		ContentObject:  &e2eTestUser{},
		GetObject:      func() any { return &e2eTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "E2E Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "E2E Test Users" },
	})

	voucherBlock := blocks.NewStructBlock()
	voucherBlock.AddField("code", blocks.CharBlock())
	voucherBlock.AddField("serial", blocks.CharBlock())

	giftcardBlock := blocks.NewStructBlock()
	giftcardBlock.AddField("pin", blocks.CharBlock())

	var (
		voucherCalls  int
		giftcardCalls int
		failGiftcard  error
	)

	orders.RegisterSupplier(&orders.Supplier{
		Order: 1,
		Name:  "e2e_voucher_supplier",
		TextLabel: func(ctx context.Context) string {
			return "E2E Voucher Supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &e2eSupplierProduct{}
		},
		OrderBlockType: func(ctx context.Context, order *orders.Order, item *orders.OrderItem) (blocks.Block, error) {
			return voucherBlock, nil
		},
		OrderIsAccepted: func(ctx context.Context, order *orders.Order, item *orders.OrderItem) (*blocks.StructBlockValue, error) {
			voucherCalls++
			if order.Customer == nil || order.Customer.Email == nil || order.Customer.Email.Address == "" {
				return nil, errors.New("expected preloaded customer email on order")
			}
			return blocks.NewBlockValue(voucherBlock, map[string]any{
				"code":   fmt.Sprintf("VOUCHER-%s", item.ID.String()[:6]),
				"serial": "SER-1001",
			}), nil
		},
	})

	orders.RegisterSupplier(&orders.Supplier{
		Order: 2,
		Name:  "e2e_giftcard_supplier",
		TextLabel: func(ctx context.Context) string {
			return "E2E Giftcard Supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &e2eSupplierProduct{}
		},
		OrderBlockType: func(ctx context.Context, order *orders.Order, item *orders.OrderItem) (blocks.Block, error) {
			return giftcardBlock, nil
		},
		OrderIsAccepted: func(ctx context.Context, order *orders.Order, item *orders.OrderItem) (*blocks.StructBlockValue, error) {
			giftcardCalls++
			if failGiftcard != nil {
				return nil, failGiftcard
			}
			if order.Customer == nil || order.Customer.FirstName == "" {
				return nil, errors.New("expected preloaded customer first name on order")
			}
			return blocks.NewBlockValue(giftcardBlock, map[string]any{
				"pin": "PIN-9988",
			}), nil
		},
	})

	setupApp := func() django.AppConfig {
		cfg := apps.NewDBAppConfig("e2e_shop_test")
		cfg.ModelObjects = []attrs.Definer{
			&e2eTestUser{},
			&e2eSupplierProduct{},
		}
		cfg.Init = func(s django.Settings, db drivers.Database) error {
			if ed, err := migrator.GetSchemaEditor(db.Driver()); err == nil {
				_ = ed.DropTable(context.Background(), migrator.NewModelTable(&auditlogs.Entry{}), true)
			}
			tables := quest.Table(t,
				&e2eTestUser{},
				&auditlogs.Entry{},
				&models.Category{},
				&orders.SupplierModel{},
				&orders.OrderState{},
				&e2eSupplierProduct{},
				&products.Product{},
				&carts.Cart{},
				&carts.CartItem{},
				&customers.Customer{},
				&payments.ProviderAccount{},
				&orders.Order{},
				&orders.OrderItem{},
				&payments.Payment{},
				&coupons.Coupon{},
				&coupons.CouponProductThrough{},
				&coupons.Redemption{},
			)
			tables.IfNotExists = true
			tables.Create()
			return nil
		}
		return cfg
	}

	var (
		cat         *models.Category
		voucherProd *products.Product
		giftProd    *products.Product
		customer    *customers.Customer
	)

	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:           []string{"*"},
			django.APPVAR_DISABLE_NOSURF:          true,
			django.APPVAR_DEBUG:                   true,
			django.APPVAR_DATABASE:                db,
			shop.APPVAR_SHOP_CURRENCY:             "EUR",
			shop.APPVAR_SHOP_CHECKOUT_ORIGIN:      "https://merchant.example.test",
			shop.APPVAR_SHOP_CHECKOUT_RETURN_PATH: "/order/thanks",
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
			settings.NewAppConfig,
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
				return &e2eTestUser{LoggedIn: false}
			},
			Users: map[string]authentication.User{
				"e2e_user": &e2eTestUser{ID: 1, Name: "e2e_user", LoggedIn: true},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Seed shop stack and verify bootstrap",
				Function: func(_ *djester.Tester, t *testing.T) {
					handlers.RegisterAuditLogs()
					require.NoError(t, orders.SuppliersQuery().Sync())

					for _, name := range []string{"e2e_voucher_supplier", "e2e_giftcard_supplier"} {
						row, err := orders.SuppliersQuery().Filter("InternalName", name).Get()
						require.NoError(t, err)
						require.True(t, row.Object.Enabled, "supplier %s should be synced as enabled", name)
					}

					// Subsequent Sync() remains idempotent
					require.NoError(t, orders.SuppliersQuery().Sync())

					var err error
					cat, err = queries.GetQuerySet(&models.Category{}).Create(&models.Category{
						Name: "Digital Codes",
						Slug: "digital-codes",
					})
					require.NoError(t, err)

					voucherProd = seedE2EProduct(t, cat, "e2e_voucher_supplier", "Game Voucher 20", "game-voucher-20", "SKU-V20", "20.00")
					giftProd = seedE2EProduct(t, cat, "e2e_giftcard_supplier", "Gift Card 50", "gift-card-50", "SKU-G50", "50.00")

					customer, err = queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
						Email:     drivers.MustParseEmail("buyer@example.test"),
						FirstName: "Alice",
						LastName:  "Example",
					})
					require.NoError(t, err)
				},
			},
			&djester.BasicTest{
				Label: "Completed signal fulfills multi-supplier order, reconciles coupons, and records audit logs",
				Function: func(_ *djester.Tester, t *testing.T) {
					ctx := t.Context()
					voucherCalls = 0
					giftcardCalls = 0
					failGiftcard = nil

					now := time.Now()
					coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        coupons.CouponCode("E2ESAVE5"),
						Title:       "E2E Save 5",
						UsageLimit:  2,
						MaxDiscount: decimal.NewFromInt(5),
						ValidFrom:   now.Add(-time.Hour),
						ValidUntil:  now.Add(time.Hour),
					})
					require.NoError(t, err)

					req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
					order, createdItems, err := orders.CreateOrder(
						req,
						customer,
						nil,
						&carts.CartItem{Product: voucherProd, Amount: 1},
						&carts.CartItem{Product: giftProd, Amount: 1},
					)
					require.NoError(t, err)
					require.Len(t, createdItems, 2)

					redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Coupon:         coupon,
						Customer:       customer,
						Order:          order,
						DiscountAmount: decimal.NewFromInt(5),
						RedeemedAt:     now,
					})
					require.NoError(t, err)

					err = orders.Signals.Completed.Send(ctx, orders.OrderCompletedSignal{
						BaseSignal: signals.BaseSignal{Request: req},
						Order:      order,
					})
					require.NoError(t, err)

					require.Equal(t, 1, voucherCalls)
					require.Equal(t, 1, giftcardCalls)
					require.Equal(t, orders.OrderStateAccepted, order.State)
					require.True(t, order.Completed)
					require.True(t, order.Processed)

					// Verify Order persisted in DB
					reloadedOrder, err := orders.Query().Filter("ID", order.ID).Get()
					require.NoError(t, err)
					require.Equal(t, orders.OrderStateAccepted, reloadedOrder.Object.State)
					require.True(t, reloadedOrder.Object.Completed)
					require.True(t, reloadedOrder.Object.Processed)

					// Verify OrderItems persisted SupplierData and bind to OrderBlock on reload
					itemRows, err := queries.GetQuerySet(&orders.OrderItem{}).
						Filter(expr.Q("Order", order.ID)).
						OrderBy("ID").
						All()
					require.NoError(t, err)
					require.Len(t, itemRows, 2)

					for _, row := range itemRows {
						item := row.Object
						require.True(t, item.SupplierData.HasData(), "item %s missing SupplierData", item.ID)
						switch string(item.Supplier) {
						case "e2e_voucher_supplier":
							require.Same(t, voucherBlock, item.SupplierData.Block())
							codeField, ok := item.SupplierData.V["code"].(blocks.BoundBlockValue)
							require.True(t, ok)
							require.Equal(t, fmt.Sprintf("VOUCHER-%s", item.ID.String()[:6]), codeField.Data())
							serialField, ok := item.SupplierData.V["serial"].(blocks.BoundBlockValue)
							require.True(t, ok)
							require.Equal(t, "SER-1001", serialField.Data())
						case "e2e_giftcard_supplier":
							require.Same(t, giftcardBlock, item.SupplierData.Block())
							pinField, ok := item.SupplierData.V["pin"].(blocks.BoundBlockValue)
							require.True(t, ok)
							require.Equal(t, "PIN-9988", pinField.Data())
						default:
							t.Fatalf("unexpected supplier %q", item.Supplier)
						}
					}

					// Verify coupon reconciliation
					reloadedCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.EqualValues(t, 1, reloadedCoupon.Object.UsageCount)

					reloadedRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
					require.NoError(t, err)
					require.True(t, reloadedRedemption.Object.Counted)
					require.False(t, reloadedRedemption.Object.Invalidated)

					// Verify audit log entries for create, accept, process, complete
					audits := auditEntriesForOrder(t, order.ID)
					for _, kind := range []string{
						"shop:orders:create",
						"shop:orders:accept",
						"shop:orders:process",
						"shop:orders:complete",
					} {
						require.Len(t, audits[kind], 1, "expected 1 audit entry for %s", kind)
						def := auditlogs.DefinitionForType(kind)
						require.NotNil(t, def)
						msg := fmt.Sprint(def.FormatMessage(req, audits[kind][0]))
						require.Contains(t, msg, order.ID.String())
						require.NotContains(t, msg, "%!")
					}
					require.Equal(t, string(orders.OrderStateAccepted), audits["shop:orders:accept"][0].Data()["state"])
					require.EqualValues(t, 2, audits["shop:orders:accept"][0].Data()["line_count"])
					require.Equal(t, string(orders.OrderStateAccepted), audits["shop:orders:process"][0].Data()["state"])
					require.Equal(t, string(orders.OrderStateAccepted), audits["shop:orders:complete"][0].Data()["state"])

					// Idempotency: re-sending Completed does not re-call suppliers or double-count coupons
					err = orders.Signals.Completed.Send(ctx, orders.OrderCompletedSignal{
						BaseSignal: signals.BaseSignal{Request: req},
						Order:      order,
					})
					require.NoError(t, err)
					require.Equal(t, 1, voucherCalls)
					require.Equal(t, 1, giftcardCalls)

					reloadedCoupon, err = queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.EqualValues(t, 1, reloadedCoupon.Object.UsageCount)

					auditsAfter := auditEntriesForOrder(t, order.ID)
					require.Len(t, auditsAfter["shop:orders:accept"], 1)
					require.Len(t, auditsAfter["shop:orders:process"], 1)
				},
			},
			&djester.BasicTest{
				Label: "Failure in 2nd supplier rolls back SupplierData, Order state, coupons, and audit logs",
				Function: func(_ *djester.Tester, t *testing.T) {
					ctx := t.Context()
					voucherCalls = 0
					giftcardCalls = 0
					errGiftcardAPI := errors.New("giftcard upstream timeout")
					failGiftcard = errGiftcardAPI
					defer func() { failGiftcard = nil }()

					now := time.Now()
					coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        coupons.CouponCode("E2EROLLBACK"),
						Title:       "E2E Rollback",
						UsageLimit:  1,
						MaxDiscount: decimal.NewFromInt(5),
						ValidFrom:   now.Add(-time.Hour),
						ValidUntil:  now.Add(time.Hour),
					})
					require.NoError(t, err)

					req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
					order, _, err := orders.CreateOrder(
						req,
						customer,
						nil,
						&carts.CartItem{Product: voucherProd, Amount: 1},
						&carts.CartItem{Product: giftProd, Amount: 1},
					)
					require.NoError(t, err)

					redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Coupon:         coupon,
						Customer:       customer,
						Order:          order,
						DiscountAmount: decimal.NewFromInt(5),
						RedeemedAt:     now,
					})
					require.NoError(t, err)

					// Wrap completion in a caller transaction so all completion listeners (including shop:orders:complete)
					// roll back atomically on error.
					txCtx, tx, err := queries.StartTransaction(ctx)
					require.NoError(t, err)
					defer tx.Rollback(txCtx)

					err = orders.Signals.Completed.Send(txCtx, orders.OrderCompletedSignal{
						BaseSignal: signals.BaseSignal{Request: req.WithContext(txCtx)},
						Order:      order,
					})
					requireSignalErrorContains(t, err, errGiftcardAPI)
					require.NoError(t, tx.Rollback(txCtx))

					require.Equal(t, 1, voucherCalls)
					require.Equal(t, 1, giftcardCalls)

					reloadedOrder, err := orders.Query().Filter("ID", order.ID).Get()
					require.NoError(t, err)
					require.Equal(t, orders.OrderStatePending, reloadedOrder.Object.State)
					require.False(t, reloadedOrder.Object.Completed)
					require.False(t, reloadedOrder.Object.Processed)

					itemRows, err := queries.GetQuerySet(&orders.OrderItem{}).
						Filter(expr.Q("Order", order.ID)).
						All()
					require.NoError(t, err)
					require.Len(t, itemRows, 2)
					for _, row := range itemRows {
						require.False(t, row.Object.SupplierData.HasData(), "item %s must not have partial SupplierData after rollback", row.Object.ID)
					}

					reloadedCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.Zero(t, reloadedCoupon.Object.UsageCount)

					reloadedRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
					require.NoError(t, err)
					require.False(t, reloadedRedemption.Object.Counted)
					require.False(t, reloadedRedemption.Object.Invalidated)

					audits := auditEntriesForOrder(t, order.ID)
					require.Len(t, audits["shop:orders:create"], 1)
					require.Empty(t, audits["shop:orders:accept"])
					require.Empty(t, audits["shop:orders:process"])
					require.Empty(t, audits["shop:orders:complete"])
				},
			},
			&djester.BasicTest{
				Label: "Downstream Signals.Processed failure rolls back fulfillment, coupons, and shop:orders:accept audit log",
				Function: func(_ *djester.Tester, t *testing.T) {
					ctx := t.Context()
					voucherCalls = 0
					giftcardCalls = 0
					failGiftcard = nil

					now := time.Now()
					coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        coupons.CouponCode("E2ESIGFAIL"),
						Title:       "E2E Signal Fail",
						UsageLimit:  1,
						MaxDiscount: decimal.NewFromInt(5),
						ValidFrom:   now.Add(-time.Hour),
						ValidUntil:  now.Add(time.Hour),
					})
					require.NoError(t, err)

					req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
					order, _, err := orders.CreateOrder(
						req,
						customer,
						nil,
						&carts.CartItem{Product: voucherProd, Amount: 1},
					)
					require.NoError(t, err)

					redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Coupon:         coupon,
						Customer:       customer,
						Order:          order,
						DiscountAmount: decimal.NewFromInt(5),
						RedeemedAt:     now,
					})
					require.NoError(t, err)

					errProcessedListener := errors.New("downstream processed hook failed")
					listener, err := orders.Signals.Processed.Listen(ctx, func(ctx context.Context, _ signals.Signal[orders.OrderProcessedSignal], sig orders.OrderProcessedSignal) error {
						if sig.Order != nil && sig.Order.ID == order.ID {
							return errProcessedListener
						}
						return nil
					})
					require.NoError(t, err)
					defer orders.Signals.Processed.Disconnect(context.Background(), listener)

					txCtx, tx, err := queries.StartTransaction(ctx)
					require.NoError(t, err)
					defer tx.Rollback(txCtx)

					err = orders.Signals.Completed.Send(txCtx, orders.OrderCompletedSignal{
						BaseSignal: signals.BaseSignal{Request: req.WithContext(txCtx)},
						Order:      order,
					})
					requireSignalErrorContains(t, err, errProcessedListener)
					require.NoError(t, tx.Rollback(txCtx))

					reloadedOrder, err := orders.Query().Filter("ID", order.ID).Get()
					require.NoError(t, err)
					require.Equal(t, orders.OrderStatePending, reloadedOrder.Object.State)
					require.False(t, reloadedOrder.Object.Completed)
					require.False(t, reloadedOrder.Object.Processed)

					itemRows, err := queries.GetQuerySet(&orders.OrderItem{}).
						Filter(expr.Q("Order", order.ID)).
						All()
					require.NoError(t, err)
					require.Len(t, itemRows, 1)
					require.False(t, itemRows[0].Object.SupplierData.HasData())

					reloadedCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.Zero(t, reloadedCoupon.Object.UsageCount)

					reloadedRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
					require.NoError(t, err)
					require.False(t, reloadedRedemption.Object.Counted)

					audits := auditEntriesForOrder(t, order.ID)
					require.Len(t, audits["shop:orders:create"], 1)
					require.Empty(t, audits["shop:orders:accept"])
					require.Empty(t, audits["shop:orders:process"])
					require.Empty(t, audits["shop:orders:complete"])
				},
			},
			&djester.BasicTest{
				Label: "Disabled supplier in DB prevents fulfillment and rolls back order",
				Function: func(_ *djester.Tester, t *testing.T) {
					ctx := t.Context()
					voucherCalls = 0
					giftcardCalls = 0
					failGiftcard = nil

					_, err := orders.SuppliersQuery().
						Filter("InternalName", "e2e_giftcard_supplier").
						BulkUpdate(expr.As("Enabled", expr.Value(false)))
					require.NoError(t, err)
					t.Cleanup(func() {
						_, _ = orders.SuppliersQuery().
							Filter("InternalName", "e2e_giftcard_supplier").
							BulkUpdate(expr.As("Enabled", expr.Value(true)))
					})

					now := time.Now()
					coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
						Code:        coupons.CouponCode("E2EDISABLED"),
						Title:       "E2E Disabled Supplier",
						UsageLimit:  1,
						MaxDiscount: decimal.NewFromInt(5),
						ValidFrom:   now.Add(-time.Hour),
						ValidUntil:  now.Add(time.Hour),
					})
					require.NoError(t, err)

					req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
					order, _, err := orders.CreateOrder(
						req,
						customer,
						nil,
						&carts.CartItem{Product: voucherProd, Amount: 1},
						&carts.CartItem{Product: giftProd, Amount: 1},
					)
					require.NoError(t, err)

					redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Coupon:         coupon,
						Customer:       customer,
						Order:          order,
						DiscountAmount: decimal.NewFromInt(5),
						RedeemedAt:     now,
					})
					require.NoError(t, err)

					txCtx, tx, err := queries.StartTransaction(ctx)
					require.NoError(t, err)
					defer tx.Rollback(txCtx)

					err = orders.Signals.Completed.Send(txCtx, orders.OrderCompletedSignal{
						BaseSignal: signals.BaseSignal{Request: req.WithContext(txCtx)},
						Order:      order,
					})
					requireSignalErrorContains(t, err, orders.ErrSupplierDisabled)
					require.NoError(t, tx.Rollback(txCtx))

					require.LessOrEqual(t, voucherCalls, 1)
					require.Equal(t, 0, giftcardCalls)

					reloadedOrder, err := orders.Query().Filter("ID", order.ID).Get()
					require.NoError(t, err)
					require.Equal(t, orders.OrderStatePending, reloadedOrder.Object.State)
					require.False(t, reloadedOrder.Object.Completed)
					require.False(t, reloadedOrder.Object.Processed)

					itemRows, err := queries.GetQuerySet(&orders.OrderItem{}).
						Filter(expr.Q("Order", order.ID)).
						All()
					require.NoError(t, err)
					require.Len(t, itemRows, 2)
					for _, row := range itemRows {
						require.False(t, row.Object.SupplierData.HasData())
					}

					reloadedCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
					require.NoError(t, err)
					require.Zero(t, reloadedCoupon.Object.UsageCount)

					reloadedRedemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", redemption.ID).Get()
					require.NoError(t, err)
					require.False(t, reloadedRedemption.Object.Counted)

					audits := auditEntriesForOrder(t, order.ID)
					require.Len(t, audits["shop:orders:create"], 1)
					require.Empty(t, audits["shop:orders:accept"])
					require.Empty(t, audits["shop:orders:process"])
					require.Empty(t, audits["shop:orders:complete"])
				},
			},
		},
	}

	d.Test(t)
}
