//go:build test
// +build test

package generators

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	_ "github.com/Nigel2392/go-django/contrib/shop/core/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

func init() {
	coupons.MIN_CODE_LENGTH = 0
	coupons.MAX_CODE_LENGTH = 128
}

type generatorTestUser struct {
	ID              int `attrs:"primary"`
	Name            string
	LoggedIn        bool
	IsAdministrator bool
}

func (u *generatorTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, u).(*attrs.ObjectDefinitions).
		WithTableName("generator_test_users")
}

func (u *generatorTestUser) IsAuthenticated() bool {
	return u != nil && u.LoggedIn
}

func (u *generatorTestUser) IsAdmin() bool {
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

type generatorTestSupplierProduct struct {
	*products.Product
	SKU string
}

func (p *generatorTestSupplierProduct) ProductPrice(ctx context.Context) price.Money {
	return price.Money{
		Currency: currency.EUR,
		Amount:   p.Price,
	}
}

func (p *generatorTestSupplierProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	if p.Product == nil {
		p.Product = &products.Product{Instance: p}
	}
	return attrs.Make[*generatorTestSupplierProduct, any](ctx, p,
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
		fattrs.Field(p, "SKU", &p.SKU, func() fattrs.PtrFieldConfig[*generatorTestSupplierProduct, string] {
			return fattrs.PtrFieldConfig[*generatorTestSupplierProduct, string]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
	)
}

func setupGeneratorTestEnv(t *testing.T) {
	t.Helper()

	_, db := testdb.Open()

	mediafiles.RegisterBackend("memory", memory.NewBackend(5))
	mediafiles.SetDefault("memory")

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:  &generatorTestUser{},
		GetObject:      func() any { return &generatorTestUser{} },
		GetLabel:       func(ctx context.Context) string { return "Generator Test User" },
		GetPluralLabel: func(ctx context.Context) string { return "Generator Test Users" },
	})

	orders.RegisterSupplier(&orders.Supplier{
		Order: 1,
		Name:  "generator_test_supplier",
		TextLabel: func(ctx context.Context) string {
			return "Generator Test Supplier"
		},
		ProductModel: func(ctx context.Context) products.ProductModel {
			return &generatorTestSupplierProduct{}
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
					&generatorTestUser{},
					&categories.Category{},
					&orders.SupplierModel{},
					&generatorTestSupplierProduct{},
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
		&generatorTestUser{},
		&categories.Category{},
		&orders.SupplierModel{},
		&generatorTestSupplierProduct{},
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
		clearGeneratorTables(t)
		_ = db.Close()
	})
}

func clearGeneratorTables(t *testing.T) {
	t.Helper()
	modelsToClear := []attrs.Definer{
		&coupons.Redemption{},
		&coupons.CouponProductThrough{},
		&coupons.Coupon{},
		&carts.CartItem{},
		&carts.Cart{},
		&products.Product{},
	}
	for _, m := range modelsToClear {
		if _, err := queries.GetQuerySet(m).Delete(); err != nil {
			t.Fatalf("failed to clear %T table: %v", m, err)
		}
	}
}

func boundedCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(t.Context(), 5*time.Second)
}

func seedCouponWithContext(t *testing.T, ctx context.Context, code string) *coupons.Coupon {
	t.Helper()
	now := time.Now()
	c, err := queries.GetQuerySetWithContext(ctx, &coupons.Coupon{}).Create(&coupons.Coupon{
		Title:          "Test Coupon",
		Code:           coupons.CouponCode(code),
		IsPercentage:   false,
		MaxDiscount:    decimal.RequireFromString("10.00"),
		MinOrderAmount: decimal.Zero,
		ValidFrom:      now.Add(-time.Hour),
		ValidUntil:     now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create coupon %q: %v", code, err)
	}
	return c
}

func seedCoupon(t *testing.T, code string) *coupons.Coupon {
	t.Helper()
	return seedCouponWithContext(t, t.Context(), code)
}

func TestGenerateCouponCode(t *testing.T) {
	setupGeneratorTestEnv(t)

	t.Run("Generates valid code on empty table without inserting a coupon row", func(t *testing.T) {
		clearGeneratorTables(t)

		parts := []CodePart{
			{Type: CPT_STATIC, Value: "SHOP-"},
			{Type: CPT_YEAR},
			{Type: CPT_STATIC, Value: "-"},
			{Type: CPT_RANDOM, Length: 6},
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		code, err := GenerateCouponCode(ctx, parts)
		if err != nil {
			t.Fatalf("GenerateCouponCode failed: %v", err)
		}

		pattern := regexp.MustCompile(`^SHOP-\d{4}-[A-Z]{6}$`)
		if !pattern.MatchString(code) {
			t.Fatalf("generated code %q did not match expected pattern", code)
		}

		count, err := queries.GetQuerySetWithContext(t.Context(), &coupons.Coupon{}).Count()
		if err != nil {
			t.Fatalf("Count() failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected GenerateCouponCode not to persist a Coupon row, got count=%d", count)
		}
	})

	t.Run("Empty or nil parts on empty table returns empty string", func(t *testing.T) {
		clearGeneratorTables(t)

		for _, tc := range []struct {
			name  string
			parts []CodePart
		}{
			{"nil", nil},
			{"empty", []CodePart{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := boundedCtx(t)
				defer cancel()

				code, err := GenerateCouponCode(ctx, tc.parts)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if code != "" {
					t.Fatalf("want empty string, got %q", code)
				}
			})
		}
	})

	t.Run("Invalid CodePartType returns wrapped ErrInvalidPartType", func(t *testing.T) {
		clearGeneratorTables(t)
		_ = seedCoupon(t, "EXISTING1")

		cases := []struct {
			name  string
			parts []CodePart
		}{
			{
				name:  "Single invalid part",
				parts: []CodePart{{Type: CodePartType(999)}},
			},
			{
				name: "Valid parts followed by invalid part",
				parts: []CodePart{
					{Type: CPT_STATIC, Value: "OK-"},
					{Type: CPT_RANDOM, Length: 4},
					{Type: CPT_UUID + 1},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				code, err := GenerateCouponCode(t.Context(), tc.parts)
				if err == nil {
					t.Fatal("expected error for invalid CodePartType")
				}
				if !errors.Is(err, ErrInvalidPartType) {
					t.Fatalf("expected errors.Is(err, ErrInvalidPartType), got: %v", err)
				}
				if !strings.Contains(err.Error(), "GenerateCouponCode") {
					t.Fatalf("expected wrapped error containing GenerateCouponCode, got: %q", err.Error())
				}
				if code != "" {
					t.Fatalf("expected empty code on error, got %q", code)
				}
			})
		}
	})

	t.Run("Static code succeeds when unused and does not collide with substrings or superstrings", func(t *testing.T) {
		clearGeneratorTables(t)

		_ = seedCoupon(t, "EXACT12345")

		for _, candidate := range []string{"EXACT1234", "EXACT123456", "XACT12345", "OTHER12345"} {
			ctx, cancel := boundedCtx(t)
			got, err := GenerateCouponCode(ctx, []CodePart{
				{Type: CPT_STATIC, Value: candidate},
			})
			cancel()
			if err != nil {
				t.Fatalf("GenerateCouponCode(%q) failed: %v", candidate, err)
			}
			if got != candidate {
				t.Fatalf("want %q, got %q", candidate, got)
			}
		}
	})

	t.Run("Deleted coupon code becomes available again", func(t *testing.T) {
		clearGeneratorTables(t)

		seeded := seedCoupon(t, "REUSABLE-01")
		if _, err := queries.GetQuerySetWithContext(t.Context(), &coupons.Coupon{}).
			Filter("ID", seeded.ID).
			Delete(); err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "REUSABLE-01"},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode after delete failed: %v", err)
		}
		if got != "REUSABLE-01" {
			t.Fatalf("want %q, got %q", "REUSABLE-01", got)
		}
	})

	t.Run("Collision retry skips 25 existing codes and deterministically returns sole remaining random code", func(t *testing.T) {
		clearGeneratorTables(t)

		for ch := 'A'; ch <= 'Y'; ch++ {
			_ = seedCoupon(t, fmt.Sprintf("SEED-%c", ch))
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "SEED-"},
			{Type: CPT_RANDOM, Length: 1},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode failed: %v", err)
		}
		if got != "SEED-Z" {
			t.Fatalf("expected sole remaining code %q, got %q", "SEED-Z", got)
		}
	})

	t.Run("Collision retry works with random part in the middle of static prefix and suffix", func(t *testing.T) {
		clearGeneratorTables(t)

		for ch := 'B'; ch <= 'Z'; ch++ {
			_ = seedCoupon(t, fmt.Sprintf("PRE-%c-SUF", ch))
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "PRE-"},
			{Type: CPT_RANDOM, Length: 1},
			{Type: CPT_STATIC, Value: "-SUF"},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode failed: %v", err)
		}
		if got != "PRE-A-SUF" {
			t.Fatalf("want %q, got %q", "PRE-A-SUF", got)
		}
	})

	t.Run("Collision retry skips 15 existing 1-char UUID codes and returns sole remaining hex char", func(t *testing.T) {
		clearGeneratorTables(t)

		const hexChars = "0123456789abcde" // leave 'f' open
		for i := 0; i < len(hexChars); i++ {
			_ = seedCoupon(t, fmt.Sprintf("HEX-%c", hexChars[i]))
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "HEX-"},
			{Type: CPT_UUID, Length: 1},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode failed: %v", err)
		}
		if got != "HEX-f" {
			t.Fatalf("expected sole remaining hex code %q, got %q", "HEX-f", got)
		}
	})

	t.Run("Sequentially exhausts all 26 codes in a 1-letter random keyspace without duplicates", func(t *testing.T) {
		clearGeneratorTables(t)

		parts := []CodePart{
			{Type: CPT_STATIC, Value: "FULL-"},
			{Type: CPT_RANDOM, Length: 1},
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		seen := make(map[string]struct{}, 26)
		for i := range 26 {
			code, err := GenerateCouponCode(ctx, parts)
			if err != nil {
				t.Fatalf("iteration %d: GenerateCouponCode failed: %v", i, err)
			}
			if _, dup := seen[code]; dup {
				t.Fatalf("iteration %d: returned duplicate code %q", i, code)
			}
			seen[code] = struct{}{}
			_ = seedCoupon(t, code)
		}

		for ch := 'A'; ch <= 'Z'; ch++ {
			want := fmt.Sprintf("FULL-%c", ch)
			if _, ok := seen[want]; !ok {
				t.Fatalf("missing expected code %q after exhausting keyspace", want)
			}
		}
	})

	t.Run("Collision retry works with mixed date and random parts", func(t *testing.T) {
		clearGeneratorTables(t)

		year := strconv.Itoa(time.Now().Year())
		for ch := 'A'; ch <= 'Y'; ch++ {
			_ = seedCoupon(t, fmt.Sprintf("Y%s-%c", year, ch))
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "Y"},
			{Type: CPT_YEAR},
			{Type: CPT_STATIC, Value: "-"},
			{Type: CPT_RANDOM, Length: 1},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode failed: %v", err)
		}
		want := fmt.Sprintf("Y%s-Z", year)
		if got != want {
			t.Fatalf("want %q, got %q", want, got)
		}
	})

	t.Run("UUID-based generation (full and truncated) produces unique codes and persists cleanly", func(t *testing.T) {
		clearGeneratorTables(t)

		ctx, cancel := boundedCtx(t)
		defer cancel()

		cases := []struct {
			name    string
			parts   []CodePart
			pattern *regexp.Regexp
		}{
			{
				name:    "Full32",
				parts:   []CodePart{{Type: CPT_UUID, Length: 32}},
				pattern: regexp.MustCompile(`^[0-9a-f]{32}$`),
			},
			{
				name:    "DefaultLength0",
				parts:   []CodePart{{Type: CPT_UUID, Length: 0}},
				pattern: regexp.MustCompile(`^[0-9a-f]{32}$`),
			},
			{
				name: "PrefixedTruncated8",
				parts: []CodePart{
					{Type: CPT_STATIC, Value: "UID-"},
					{Type: CPT_UUID, Length: 8},
				},
				pattern: regexp.MustCompile(`^UID-[0-9a-f]{8}$`),
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				seen := make(map[string]struct{}, 15)
				for i := range 15 {
					code, err := GenerateCouponCode(ctx, tc.parts)
					if err != nil {
						t.Fatalf("iteration %d: GenerateCouponCode failed: %v", i, err)
					}
					if !tc.pattern.MatchString(code) {
						t.Fatalf("iteration %d: code %q did not match %s", i, code, tc.pattern)
					}
					if _, dup := seen[code]; dup {
						t.Fatalf("iteration %d: duplicate UUID code %q", i, code)
					}
					seen[code] = struct{}{}
					_ = seedCoupon(t, code)
				}
			})
		}
	})

	t.Run("Batch generation of 50 realistic coupons persists and reads back cleanly", func(t *testing.T) {
		clearGeneratorTables(t)

		parts := []CodePart{
			{Type: CPT_STATIC, Value: "PROMO-"},
			{Type: CPT_YEAR},
			{Type: CPT_STATIC, Value: "-"},
			{Type: CPT_RANDOM, Length: 8},
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		const batchSize = 50
		seen := make(map[string]struct{}, batchSize)
		for i := range batchSize {
			code, err := GenerateCouponCode(ctx, parts)
			if err != nil {
				t.Fatalf("batch %d: GenerateCouponCode failed: %v", i, err)
			}
			if len(code) < 6 || len(code) > 32 {
				t.Fatalf("batch %d: code %q length %d outside Coupon.Code bounds [6, 32]", i, code, len(code))
			}
			if _, dup := seen[code]; dup {
				t.Fatalf("batch %d: duplicate code %q", i, code)
			}
			seen[code] = struct{}{}
			_ = seedCoupon(t, code)
		}

		qs := queries.GetQuerySetWithContext(t.Context(), &coupons.Coupon{})
		count, err := qs.Count()
		if err != nil {
			t.Fatalf("Count() failed: %v", err)
		}
		if count != batchSize {
			t.Fatalf("expected %d persisted coupons, got %d", batchSize, count)
		}

		for code := range seen {
			exists, err := qs.Filter("Code", code).Exists()
			if err != nil {
				t.Fatalf("Exists(%q) failed: %v", code, err)
			}
			if !exists {
				t.Fatalf("expected persisted coupon %q to exist in DB", code)
			}
		}
	})

	t.Run("Pre-cancelled context returns wrapped context.Canceled", func(t *testing.T) {
		clearGeneratorTables(t)

		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()

		code, err := GenerateCouponCode(cancelledCtx, []CodePart{
			{Type: CPT_STATIC, Value: "CANCELLED1"},
		})
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected error wrapping context.Canceled, got: %v", err)
		}
		if !strings.Contains(err.Error(), "GenerateCouponCode") {
			t.Fatalf("expected wrapped error containing GenerateCouponCode, got: %q", err.Error())
		}
		if code != "" {
			t.Fatalf("expected empty code on error, got %q", code)
		}
	})

	t.Run("Already-expired deadline returns wrapped context.DeadlineExceeded", func(t *testing.T) {
		clearGeneratorTables(t)

		expiredCtx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()

		code, err := GenerateCouponCode(expiredCtx, []CodePart{
			{Type: CPT_STATIC, Value: "EXPIRED01"},
		})
		if err == nil {
			t.Fatal("expected error for expired deadline context")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected error wrapping context.DeadlineExceeded, got: %v", err)
		}
		if code != "" {
			t.Fatalf("expected empty code on error, got %q", code)
		}
	})

	t.Run("DeadlineExceeded aborts collision loop on exhausted static code", func(t *testing.T) {
		clearGeneratorTables(t)

		_ = seedCoupon(t, "TAKEN-STATIC")

		timeoutCtx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
		defer cancel()

		code, err := GenerateCouponCode(timeoutCtx, []CodePart{
			{Type: CPT_STATIC, Value: "TAKEN-STATIC"},
		})
		if err == nil {
			t.Fatal("expected timeout error when static code already exists")
		}
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrCouponExists) {
			t.Fatalf("expected error wrapping context.DeadlineExceeded, got: %v", err)
		}
		if code != "" {
			t.Fatalf("expected empty code on timeout, got %q", code)
		}
	})

	t.Run("DeadlineExceeded aborts collision loop on fully exhausted random keyspace", func(t *testing.T) {
		clearGeneratorTables(t)

		for ch := 'A'; ch <= 'Z'; ch++ {
			_ = seedCoupon(t, fmt.Sprintf("DEAD-%c", ch))
		}

		timeoutCtx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
		defer cancel()

		code, err := GenerateCouponCode(timeoutCtx, []CodePart{
			{Type: CPT_STATIC, Value: "DEAD-"},
			{Type: CPT_RANDOM, Length: 1},
		})
		if err == nil {
			t.Fatal("expected timeout error when entire 26-code keyspace is exhausted")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected error wrapping context.DeadlineExceeded, got: %v", err)
		}
		if code != "" {
			t.Fatalf("expected empty code on timeout, got %q", code)
		}
	})

	t.Run("DeadlineExceeded aborts collision loop on fully exhausted 1-char UUID keyspace", func(t *testing.T) {
		clearGeneratorTables(t)

		const hexChars = "0123456789abcdef"
		for i := 0; i < len(hexChars); i++ {
			_ = seedCoupon(t, fmt.Sprintf("HDEAD-%c", hexChars[i]))
		}

		timeoutCtx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
		defer cancel()

		code, err := GenerateCouponCode(timeoutCtx, []CodePart{
			{Type: CPT_STATIC, Value: "HDEAD-"},
			{Type: CPT_UUID, Length: 1},
		})
		if err == nil {
			t.Fatal("expected timeout error when entire 16-code hex keyspace is exhausted")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected error wrapping context.DeadlineExceeded, got: %v", err)
		}
		if code != "" {
			t.Fatalf("expected empty code on timeout, got %q", code)
		}
	})

	t.Run("Transaction rollback releases uncommitted codes", func(t *testing.T) {
		clearGeneratorTables(t)

		txCtx, tx, err := queries.StartTransaction(t.Context())
		if err != nil {
			t.Fatalf("StartTransaction failed: %v", err)
		}
		rolledBack := false
		defer func() {
			if !rolledBack {
				_ = tx.Rollback(txCtx)
			}
		}()

		for ch := 'A'; ch <= 'Y'; ch++ {
			_ = seedCouponWithContext(t, txCtx, fmt.Sprintf("TXCP-%c", ch))
		}

		got, err := GenerateCouponCode(txCtx, []CodePart{
			{Type: CPT_STATIC, Value: "TXCP-"},
			{Type: CPT_RANDOM, Length: 1},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode inside tx failed: %v", err)
		}
		if got != "TXCP-Z" {
			t.Fatalf("expected %q inside tx, got %q", "TXCP-Z", got)
		}

		if err := tx.Rollback(txCtx); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}
		rolledBack = true

		ctx, cancel := boundedCtx(t)
		defer cancel()

		afterRollback, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "TXCP-A"},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode after rollback failed: %v", err)
		}
		if afterRollback != "TXCP-A" {
			t.Fatalf("want %q after rollback, got %q", "TXCP-A", afterRollback)
		}
	})

	t.Run("Transaction commit makes generated codes visible to subsequent calls", func(t *testing.T) {
		clearGeneratorTables(t)

		txCtx, tx, err := queries.StartTransaction(t.Context())
		if err != nil {
			t.Fatalf("StartTransaction failed: %v", err)
		}

		for ch := 'A'; ch <= 'Y'; ch++ {
			_ = seedCouponWithContext(t, txCtx, fmt.Sprintf("CMTX-%c", ch))
		}
		if err := tx.Commit(txCtx); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		ctx, cancel := boundedCtx(t)
		defer cancel()

		got, err := GenerateCouponCode(ctx, []CodePart{
			{Type: CPT_STATIC, Value: "CMTX-"},
			{Type: CPT_RANDOM, Length: 1},
		})
		if err != nil {
			t.Fatalf("GenerateCouponCode after commit failed: %v", err)
		}
		if got != "CMTX-Z" {
			t.Fatalf("want %q after commit, got %q", "CMTX-Z", got)
		}
	})

	t.Run("Concurrent read-only generation across mixed patterns succeeds without errors", func(t *testing.T) {
		clearGeneratorTables(t)

		_ = seedCoupon(t, "PAR-AAAAAAAA")

		patterns := []struct {
			parts []CodePart
			re    *regexp.Regexp
		}{
			{
				parts: []CodePart{
					{Type: CPT_STATIC, Value: "PAR-"},
					{Type: CPT_RANDOM, Length: 8},
				},
				re: regexp.MustCompile(`^PAR-[A-Z]{8}$`),
			},
			{
				parts: []CodePart{
					{Type: CPT_STATIC, Value: "PUID-"},
					{Type: CPT_UUID, Length: 12},
				},
				re: regexp.MustCompile(`^PUID-[0-9a-f]{12}$`),
			},
		}

		const workers = 8
		const perWorker = 10
		errCh := make(chan error, workers*perWorker)

		var wg sync.WaitGroup
		for w := range workers {
			p := patterns[w%len(patterns)]
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range perWorker {
					ctx, cancel := boundedCtx(t)
					code, err := GenerateCouponCode(ctx, p.parts)
					cancel()
					if err != nil {
						errCh <- err
						return
					}
					if !p.re.MatchString(code) || code == "PAR-AAAAAAAA" {
						errCh <- fmt.Errorf("unexpected generated code: %q", code)
						return
					}
				}
			}()
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Fatalf("concurrent GenerateCouponCode failed: %v", err)
		}
	})
}
