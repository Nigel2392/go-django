package admin_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/Nigel2392/go-django/contrib/admin"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	qfields "github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	qmodels "github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/views"
)

// ─────────────────────────────────────────────────────────────────────────────
// Extended M2M Test Models
// ─────────────────────────────────────────────────────────────────────────────

// 1. Strict target with an unlisted non-null FK (mirrors shop/products.Product).

type M2MTestStrictProduct struct {
	ID       int64
	Title    string
	Price    int
	Category *FormTestCategory
	Coupons  *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *M2MTestStrictProduct) String() string { return p.Title }

func (p *M2MTestStrictProduct) Save(ctx context.Context) error {
	if p.ID == 0 {
		_, err := queries.GetQuerySetWithContext(ctx, p).ExplicitSave().Create(p)
		return err
	}
	_, err := queries.GetQuerySetWithContext(ctx, p).ExplicitSave().Filter("ID", p.ID).Update(p)
	return err
}

func (p *M2MTestStrictProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*M2MTestStrictProduct, int64] {
			return fattrs.PtrFieldConfig[*M2MTestStrictProduct, int64]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
					Column:   "id",
				},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*M2MTestStrictProduct, string] {
			return fattrs.PtrFieldConfig[*M2MTestStrictProduct, string]{
				Config: attrs.Config{
					MinLength: 2,
				},
			}
		}),
		fattrs.Field(p, "Price", &p.Price, func() fattrs.PtrFieldConfig[*M2MTestStrictProduct, int] {
			return fattrs.PtrFieldConfig[*M2MTestStrictProduct, int]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(p, "Category", &p.Category, func() fattrs.PtrFieldConfig[*M2MTestStrictProduct, *FormTestCategory] {
			return fattrs.PtrFieldConfig[*M2MTestStrictProduct, *FormTestCategory]{
				Config: attrs.Config{
					Column:        "category_id",
					Null:          false,
					RelForeignKey: attrs.Relate(&FormTestCategory{}, "", nil),
				},
			}
		}),
		fattrs.Field(p, "Coupons", &p.Coupons, func() fattrs.PtrFieldConfig[*M2MTestStrictProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]] {
			return fattrs.PtrFieldConfig[*M2MTestStrictProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.Config{
					Attributes: map[string]any{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	).WithTableName("admin_m2m_ext_strict_products")
}

type M2MTestStrictCouponProduct struct {
	qmodels.Model `table:"admin_m2m_ext_strict_coupon_products"`
	Coupon        *M2MTestStrictCoupon
	Product       *M2MTestStrictProduct
}

func (cp *M2MTestStrictCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return cp.Model.Define(ctx, cp,
		attrs.NewField(cp, "Coupon", &attrs.Config{
			Column: "coupon_id",
			Null:   false,
		}),
		attrs.NewField(cp, "Product", &attrs.Config{
			Column: "product_id",
			Null:   false,
		}),
	)
}

type M2MTestStrictCoupon struct {
	qmodels.Model `table:"admin_m2m_ext_strict_coupons"`
	ID            int64
	Title         string
	Code          string
	Products      *queries.RelM2M[*M2MTestStrictProduct, *M2MTestStrictCouponProduct]
}

func (c *M2MTestStrictCoupon) String() string { return c.Title }

func (c *M2MTestStrictCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return c.Model.Define(ctx, c,
		attrs.Unbound("ID", &attrs.Config{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title"),
		attrs.Unbound("Code", &attrs.Config{MinLength: 4}),
		qfields.NewManyToManyField[*queries.RelM2M[*M2MTestStrictProduct, *M2MTestStrictCouponProduct]](c, "Products", &qfields.FieldConfig{
			ScanTo:      &c.Products,
			ReverseName: "Coupons",
			Rel: attrs.Relate(
				&M2MTestStrictProduct{},
				"", &attrs.ThroughModel{
					This:   &M2MTestStrictCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		}),
	)
}

// 2. Guarded target with UNIQUE SKU and BeforeSave error hook.

var errM2MTargetBeforeSave = errors.New("m2m target BeforeSave rejected")

type M2MTestGuardedProduct struct {
	qmodels.Model `table:"admin_m2m_ext_guarded_products"`
	ID            int64
	Title         string
	SKU           string
	Price         int
	Coupons       *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *M2MTestGuardedProduct) String() string { return p.Title }

func (p *M2MTestGuardedProduct) BeforeSave(_ context.Context) error {
	if p.Title == "TRIGGER_BEFORE_SAVE_ERROR" {
		return errM2MTargetBeforeSave
	}
	return nil
}

func (p *M2MTestGuardedProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title", &attrs.Config{MinLength: 2}),
		attrs.Unbound("SKU", &attrs.Config{
			MinLength: 2,
			Attributes: map[string]any{
				attrs.AttrUniqueKey: true,
			},
		}),
		attrs.Unbound("Price", &attrs.Config{Blank: true}),
	)
}

type M2MTestGuardedCouponProduct struct {
	qmodels.Model `table:"admin_m2m_ext_guarded_coupon_products"`
	Coupon        *M2MTestGuardedCoupon
	Product       *M2MTestGuardedProduct
}

func (cp *M2MTestGuardedCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return cp.Model.Define(ctx, cp,
		attrs.NewField(cp, "Coupon", &attrs.Config{
			Column: "coupon_id",
			Null:   false,
		}),
		attrs.NewField(cp, "Product", &attrs.Config{
			Column: "product_id",
			Null:   false,
		}),
	)
}

type M2MTestGuardedCoupon struct {
	qmodels.Model `table:"admin_m2m_ext_guarded_coupons"`
	ID            int64
	Title         string
	Code          string
	Products      *queries.RelM2M[*M2MTestGuardedProduct, *M2MTestGuardedCouponProduct]
}

func (c *M2MTestGuardedCoupon) String() string { return c.Title }

func (c *M2MTestGuardedCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return c.Model.Define(ctx, c,
		attrs.Unbound("ID", &attrs.Config{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title"),
		attrs.Unbound("Code", &attrs.Config{MinLength: 4}),
		qfields.NewManyToManyField[*queries.RelM2M[*M2MTestGuardedProduct, *M2MTestGuardedCouponProduct]](c, "Products", &qfields.FieldConfig{
			ScanTo:      &c.Products,
			ReverseName: "Coupons",
			Rel: attrs.Relate(
				&M2MTestGuardedProduct{},
				"", &attrs.ThroughModel{
					This:   &M2MTestGuardedCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		}),
	)
}

// 3. fattrs.Field + drivers.ULID PK models (no qmodels.Model embedding).

type M2MTestFattrsULIDProduct struct {
	ID      drivers.ULID
	Title   string
	Price   int
	Coupons *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *M2MTestFattrsULIDProduct) String() string { return p.Title }

func (p *M2MTestFattrsULIDProduct) BeforeSave(_ context.Context) error {
	if p.ID.IsZero() {
		p.ID = drivers.NewULID()
	}
	return nil
}

func (p *M2MTestFattrsULIDProduct) Save(ctx context.Context) error {
	if p.ID.IsZero() {
		_, err := queries.GetQuerySetWithContext(ctx, p).ExplicitSave().Create(p)
		return err
	}
	_, err := queries.GetQuerySetWithContext(ctx, p).ExplicitSave().Filter("ID", p.ID).Update(p)
	return err
}

func (p *M2MTestFattrsULIDProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, drivers.ULID] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, drivers.ULID]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, string] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, string]{
				Config: attrs.Config{
					MinLength: 2,
				},
			}
		}),
		fattrs.Field(p, "Price", &p.Price, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, int] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, int]{
				Config: attrs.Config{
					Blank: true,
				},
			}
		}),
		fattrs.Field(p, "Coupons", &p.Coupons, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.Config{
					Attributes: map[string]any{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	).WithTableName("admin_m2m_ext_fattrs_ulid_products")
}

type M2MTestFattrsULIDCouponProduct struct {
	Coupon  *M2MTestFattrsULIDCoupon
	Product *M2MTestFattrsULIDProduct
}

func (cp *M2MTestFattrsULIDCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, cp,
		fattrs.Field(cp, "Coupon", &cp.Coupon, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCouponProduct, *M2MTestFattrsULIDCoupon] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCouponProduct, *M2MTestFattrsULIDCoupon]{
				Config: attrs.Config{
					Column:   "coupon_id",
					ReadOnly: true,
					Null:     false,
					Blank:    false,
				},
			}
		}),
		fattrs.Field(cp, "Product", &cp.Product, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCouponProduct, *M2MTestFattrsULIDProduct] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCouponProduct, *M2MTestFattrsULIDProduct]{
				Config: attrs.Config{
					Column: "product_id",
				},
			}
		}),
	).WithTableName("admin_m2m_ext_fattrs_ulid_through")
}

type M2MTestFattrsULIDCoupon struct {
	ID       uint64
	Title    string
	Code     string
	Products *queries.RelM2M[*M2MTestFattrsULIDProduct, *M2MTestFattrsULIDCouponProduct]
}

func (c *M2MTestFattrsULIDCoupon) String() string { return c.Title }

func (c *M2MTestFattrsULIDCoupon) Save(ctx context.Context) error {
	if c.ID == 0 {
		_, err := queries.GetQuerySetWithContext(ctx, c).ExplicitSave().Create(c)
		return err
	}
	_, err := queries.GetQuerySetWithContext(ctx, c).ExplicitSave().Filter("ID", c.ID).Update(c)
	return err
}

func (c *M2MTestFattrsULIDCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, uint64] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, uint64]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Title", &c.Title, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, string] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, string]{
				Config: attrs.Config{
					MinLength: 2,
				},
			}
		}),
		fattrs.Field(c, "Code", &c.Code, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, string] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, string]{
				Config: attrs.Config{
					MinLength: 4,
				},
			}
		}),
		fattrs.Field(c, "Products", &c.Products, func() fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, *queries.RelM2M[*M2MTestFattrsULIDProduct, *M2MTestFattrsULIDCouponProduct]] {
			return fattrs.PtrFieldConfig[*M2MTestFattrsULIDCoupon, *queries.RelM2M[*M2MTestFattrsULIDProduct, *M2MTestFattrsULIDCouponProduct]]{
				Config: attrs.Config{
					Blank: true,
					RelManyToMany: attrs.Relate(
						&M2MTestFattrsULIDProduct{},
						"", &attrs.ThroughModel{
							This:   &M2MTestFattrsULIDCouponProduct{},
							Source: "Coupon",
							Target: "Product",
						},
					),
					ReverseAlias: "Coupons",
				},
			}
		}),
	).WithTableName("admin_m2m_ext_fattrs_ulid_coupons")
}

// 4. Target without ContextSaver / Saver and opted out of queries save hook.

type M2MTestNoSaverProduct struct {
	ID      int64
	Title   string
	Coupons *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *M2MTestNoSaverProduct) String() string        { return p.Title }
func (p *M2MTestNoSaverProduct) ForUseInQueries() bool { return false }

func (p *M2MTestNoSaverProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, int64] {
			return fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, int64]{
				Config: attrs.Config{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, string] {
			return fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, string]{
				Config: attrs.Config{
					MinLength: 2,
				},
			}
		}),
		fattrs.Field(p, "Coupons", &p.Coupons, func() fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]] {
			return fattrs.PtrFieldConfig[*M2MTestNoSaverProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.Config{
					Attributes: map[string]any{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	).WithTableName("admin_m2m_ext_nosaver_products")
}

type M2MTestNoSaverCouponProduct struct {
	qmodels.Model `table:"admin_m2m_ext_nosaver_through"`
	Coupon        *M2MTestNoSaverCoupon
	Product       *M2MTestNoSaverProduct
}

func (cp *M2MTestNoSaverCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return cp.Model.Define(ctx, cp,
		attrs.NewField(cp, "Coupon", &attrs.Config{
			Column: "coupon_id",
			Null:   false,
		}),
		attrs.NewField(cp, "Product", &attrs.Config{
			Column: "product_id",
			Null:   false,
		}),
	)
}

type M2MTestNoSaverCoupon struct {
	qmodels.Model `table:"admin_m2m_ext_nosaver_coupons"`
	ID            int64
	Title         string
	Code          string
	Products      *queries.RelM2M[*M2MTestNoSaverProduct, *M2MTestNoSaverCouponProduct]
}

func (c *M2MTestNoSaverCoupon) String() string { return c.Title }

func (c *M2MTestNoSaverCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return c.Model.Define(ctx, c,
		attrs.Unbound("ID", &attrs.Config{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title"),
		attrs.Unbound("Code", &attrs.Config{MinLength: 4}),
		qfields.NewManyToManyField[*queries.RelM2M[*M2MTestNoSaverProduct, *M2MTestNoSaverCouponProduct]](c, "Products", &qfields.FieldConfig{
			ScanTo:      &c.Products,
			ReverseName: "Coupons",
			Rel: attrs.Relate(
				&M2MTestNoSaverProduct{},
				"", &attrs.ThroughModel{
					This:   &M2MTestNoSaverCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		}),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Extended Setup & Helpers
// ─────────────────────────────────────────────────────────────────────────────

var m2mExtTestDBOnce sync.Once

func setupM2MExtendedTestDB(t *testing.T) drivers.Database {
	t.Helper()
	db := setupFormsTestDB(t)

	m2mExtTestDBOnce.Do(func() {
		extModels := []attrs.Definer{
			&M2MTestStrictProduct{},
			&M2MTestStrictCoupon{},
			&M2MTestStrictCouponProduct{},
			&M2MTestGuardedProduct{},
			&M2MTestGuardedCoupon{},
			&M2MTestGuardedCouponProduct{},
			&M2MTestFattrsULIDProduct{},
			&M2MTestFattrsULIDCoupon{},
			&M2MTestFattrsULIDCouponProduct{},
			&M2MTestNoSaverProduct{},
			&M2MTestNoSaverCoupon{},
			&M2MTestNoSaverCouponProduct{},
		}
		for _, mdl := range extModels {
			attrs.RegisterModel(mdl)
			contenttypes.Register(&contenttypes.ContentTypeDefinition{
				ContentObject:  mdl,
				GetLabel:       trans.S(fmt.Sprintf("%T", mdl)),
				GetPluralLabel: trans.S(fmt.Sprintf("%T", mdl)),
			})
		}

		if err := attrs.ResetDefinitions.Send(context.Background(), nil); err != nil {
			panic(fmt.Errorf("failed to reset extended M2M model definitions: %w", err))
		}

		schemaEditor, err := migrator.GetSchemaEditor(db.Driver())
		if err != nil {
			panic(fmt.Errorf("failed to get schema editor: %w", err))
		}

		for _, mdl := range extModels {
			table := migrator.NewModelTable(mdl)
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				panic(fmt.Errorf("failed to create table %s: %w", table.TableName(), err))
			}
		}

		tpl.Add(tpl.Config{
			AppName: "m2m_test_views",
			FS: fstest.MapFS{
				"m2m_test_views/form.tmpl": {
					Data: []byte("{{ define \"form\" }}{{ $f := (.Get \"form\") }}{{ range $f.ErrorList }}FORM_ERR:{{ . }}\n{{ end }}{{ $f.AsP }}{{ end }}"),
				},
			},
			Matches: filesystem.MatchPrefix("m2m_test_views/"),
		})
	})

	_, _ = queries.GetQuerySet(&M2MTestStrictCouponProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestStrictCoupon{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestStrictProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestGuardedCouponProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestGuardedCoupon{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestGuardedProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestFattrsULIDCouponProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestFattrsULIDCoupon{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestFattrsULIDProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestNoSaverCouponProduct{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestNoSaverCoupon{}).Delete()
	_, _ = queries.GetQuerySet(&M2MTestNoSaverProduct{}).Delete()

	return db
}

func assertAdminFormSaveReportedError[T1 modelforms.ModelForm[T2], T2 attrs.Definer](
	t *testing.T,
	saveErr error,
	af *admin.AdminForm[T1, T2],
	wantErr error,
	wantSubstr string,
) {
	t.Helper()

	allErrs := append([]error(nil), af.ErrorList()...)
	if saveErr != nil {
		allErrs = append(allErrs, saveErr)
	}

	if saveErr == nil && !forms.HasErrors(af) {
		t.Fatalf("expected inline M2M save failure to surface on Save() error or AdminForm errors, got saveErr=<nil>, ErrorList()=%v, BoundErrors()=%v",
			af.ErrorList(), af.BoundErrors())
	}
	if len(allErrs) == 0 {
		t.Fatalf("expected non-empty error list after failed M2M save")
	}

	if wantErr != nil {
		found := false
		for _, e := range allErrs {
			if errors.Is(e, wantErr) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected reported errors to match %v, got %v", wantErr, allErrs)
		}
	}

	if wantSubstr != "" {
		var joined strings.Builder
		for _, e := range allErrs {
			joined.WriteString(e.Error())
			joined.WriteByte('\n')
		}
		if !strings.Contains(strings.ToLower(joined.String()), strings.ToLower(wantSubstr)) {
			t.Fatalf("expected reported errors to contain %q, got %v", wantSubstr, allErrs)
		}
	}
}

func couponEditPOST(c *FormTestCoupon) url.Values {
	v := url.Values{
		"Title":       {c.Title},
		"Description": {c.Description},
		"Code":        {c.Code},
		"MaxDiscount": {fmt.Sprint(c.MaxDiscount)},
		"ValidFrom":   {c.ValidFrom},
		"ValidUntil":  {c.ValidUntil},
	}
	if c.IsPercentage {
		v.Set("IsPercentage", "on")
	}
	return v
}

func buildStrictCouponM2MPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Code"),
		&admin.ModelFormPanel[*M2MTestStrictProduct, modelforms.ModelForm[*M2MTestStrictProduct]]{
			TargetType: &M2MTestStrictProduct{},
			FieldName:  "Products",
			MinNum:     minNum,
			MaxNum:     maxNum,
			Extra:      extra,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Price"),
			},
		},
	}
}

func buildGuardedCouponM2MPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Code"),
		&admin.ModelFormPanel[*M2MTestGuardedProduct, modelforms.ModelForm[*M2MTestGuardedProduct]]{
			TargetType: &M2MTestGuardedProduct{},
			FieldName:  "Products",
			MinNum:     minNum,
			MaxNum:     maxNum,
			Extra:      extra,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("SKU"),
				admin.FieldPanel("Price"),
			},
		},
	}
}

func buildFattrsULIDCouponM2MPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Code"),
		&admin.ModelFormPanel[*M2MTestFattrsULIDProduct, modelforms.ModelForm[*M2MTestFattrsULIDProduct]]{
			TargetType: &M2MTestFattrsULIDProduct{},
			FieldName:  "Products",
			MinNum:     minNum,
			MaxNum:     maxNum,
			Extra:      extra,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Price"),
			},
		},
	}
}

func buildNoSaverCouponM2MPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Code"),
		&admin.ModelFormPanel[*M2MTestNoSaverProduct, modelforms.ModelForm[*M2MTestNoSaverProduct]]{
			TargetType: &M2MTestNoSaverProduct{},
			FieldName:  "Products",
			MinNum:     minNum,
			MaxNum:     maxNum,
			Extra:      extra,
			Panels: []admin.Panel{
				admin.FieldPanel("ID"),
				admin.FieldPanel("Title"),
			},
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Extended ManyToMany ModelFormPanel Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_ModelFormPanelManyToMany_Extended(t *testing.T) {
	setupM2MExtendedTestDB(t)

	// 1. Reproduces shop Coupon Add bug: target has non-null unlisted FK (Category / Image / Editor)
	// causing BulkCreate to roll back with FieldNull, which BaseFormSet.Save() must not swallow.
	t.Run("Add_TargetUnlistedNonNullFieldFailsAndSurfacesError", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &M2MTestStrictCoupon{}
		panels := buildStrictCouponM2MPanels(0, 5, 1)

		req := httptest.NewRequest(http.MethodPost, "/admin/strict-coupon/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			url.Values{
				"Title": {"Strict Promo"},
				"Code":  {"STRICT1"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Strict Phone"},
				"Products-0-Price":                           {"499"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected strict coupon AdminForm valid before save, errs=%v bound=%v",
				af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		prodCount, err := queries.GetQuerySet(&M2MTestStrictProduct{}).Count()
		if err != nil {
			t.Fatalf("count M2MTestStrictProduct: %v", err)
		}
		if prodCount != 0 {
			t.Errorf("expected 0 M2MTestStrictProduct rows after rolled-back BulkCreate, got %d", prodCount)
		}
		throughCount, err := queries.GetQuerySet(&M2MTestStrictCouponProduct{}).Count()
		if err != nil {
			t.Fatalf("count M2MTestStrictCouponProduct: %v", err)
		}
		if throughCount != 0 {
			t.Errorf("expected 0 M2MTestStrictCouponProduct rows after failed save, got %d", throughCount)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, nil, "Category")
	})

	// 2. SaveInstance hook failure on Add must be surfaced on AdminForm/Save.
	t.Run("Add_SaveInstanceErrorSurfacedOnAdminForm", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		sentinelErr := errors.New("custom SaveInstance failed on add")
		coupon := &FormTestCoupon{}
		panels := []admin.Panel{
			admin.FieldPanel("Title"),
			admin.FieldPanel("Code"),
			&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
				TargetType: &FormTestProduct{},
				FieldName:  "Products",
				MinNum:     0,
				MaxNum:     5,
				Extra:      1,
				SaveInstance: func(_ context.Context, _ modelforms.ModelForm[*FormTestProduct], _ *FormTestProduct) error {
					return sentinelErr
				},
				Panels: []admin.Panel{
					admin.FieldPanel("ID"),
					admin.FieldPanel("Title"),
					admin.FieldPanel("Slug"),
					admin.FieldPanel("Price"),
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			baseCouponPOST("Add Hook Fail", "HOOKFAIL"),
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Failing Product"},
				"Products-0-Slug":                            {"failing-product"},
				"Products-0-Price":                           {"50"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		prodCount, _ := queries.GetQuerySet(&FormTestProduct{}).Count()
		throughCount, _ := queries.GetQuerySet(&FormTestCouponProduct{}).Count()
		if prodCount != 0 || throughCount != 0 {
			t.Errorf("expected 0 products and 0 through rows, got products=%d through=%d", prodCount, throughCount)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, sentinelErr, "")
	})

	// 3. SaveInstance hook failure when editing an existing M2M target must be surfaced.
	t.Run("Edit_SaveInstanceErrorSurfacedOnAdminForm", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		sentinelErr := errors.New("custom SaveInstance failed on edit")

		coupon := &FormTestCoupon{
			Title:       "Edit Hook Promo",
			Description: "Desc",
			Code:        "EDITHOOK",
			MaxDiscount: 10,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		prod := &FormTestProduct{Title: "Original", Slug: "original", Price: 100}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(prod); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(prod); err != nil {
			t.Fatalf("link product: %v", err)
		}

		panels := []admin.Panel{
			admin.FieldPanel("Title"),
			admin.FieldPanel("Description"),
			admin.FieldPanel("Code"),
			admin.FieldPanel("MaxDiscount"),
			admin.FieldPanel("IsPercentage"),
			admin.FieldPanel("ValidFrom"),
			admin.FieldPanel("ValidUntil"),
			&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
				TargetType: &FormTestProduct{},
				FieldName:  "Products",
				MinNum:     0,
				MaxNum:     5,
				Extra:      0,
				SaveInstance: func(_ context.Context, _ modelforms.ModelForm[*FormTestProduct], _ *FormTestProduct) error {
					return sentinelErr
				},
				Panels: []admin.Panel{
					admin.FieldPanel("ID"),
					admin.FieldPanel("Title"),
					admin.FieldPanel("Slug"),
					admin.FieldPanel("Price"),
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/edit", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(prod.ID)},
				"Products-0-Title": {"Mutated"},
				"Products-0-Slug":  {"original"},
				"Products-0-Price": {"100"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid edit form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if !af.HasChanged() {
			t.Fatalf("expected HasChanged() == true for mutated inline M2M product")
		}

		_, saveErr := af.Save()

		reloaded, err := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", prod.ID).Get()
		if err != nil {
			t.Fatalf("reload product: %v", err)
		}
		if reloaded.Object.Title != "Original" {
			t.Errorf("expected DB product Title to remain 'Original', got %q", reloaded.Object.Title)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, sentinelErr, "")
	})

	// 4. Partial failure across multiple inline M2M forms: form 0 succeeds, form 1 fails.
	t.Run("Add_PartialChildSaveFailureSurfacesError", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		errSecondFailed := errors.New("second product save failed")
		coupon := &FormTestCoupon{}

		panels := []admin.Panel{
			admin.FieldPanel("Title"),
			admin.FieldPanel("Code"),
			&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
				TargetType: &FormTestProduct{},
				FieldName:  "Products",
				MinNum:     0,
				MaxNum:     5,
				Extra:      0,
				SaveInstance: func(ctx context.Context, _ modelforms.ModelForm[*FormTestProduct], inst *FormTestProduct) error {
					if inst.Title == "Bad Product" {
						return errSecondFailed
					}
					return inst.Save(ctx)
				},
				Panels: []admin.Panel{
					admin.FieldPanel("ID"),
					admin.FieldPanel("Title"),
					admin.FieldPanel("Slug"),
					admin.FieldPanel("Price"),
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			baseCouponPOST("Partial M2M", "PARTIAL1"),
			inlineMgmtValues("Products", 2, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Good Product"},
				"Products-0-Slug":                            {"good-product"},
				"Products-0-Price":                           {"25"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-Title":                           {"Bad Product"},
				"Products-1-Slug":                            {"bad-product"},
				"Products-1-Price":                           {"30"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		assertAdminFormSaveReportedError(t, saveErr, af, errSecondFailed, "")
	})

	// 5. Target model BeforeSave returning an error must surface on AdminForm/Save.
	t.Run("Add_TargetBeforeSaveErrorSurfacesError", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &M2MTestGuardedCoupon{}
		panels := buildGuardedCouponM2MPanels(0, 5, 1)

		req := httptest.NewRequest(http.MethodPost, "/admin/guarded/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			url.Values{
				"Title": {"Guarded Promo"},
				"Code":  {"GUARD1"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title": {"TRIGGER_BEFORE_SAVE_ERROR"},
				"Products-0-SKU":   {"SKU-BS-1"},
				"Products-0-Price": {"10"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		prodCount, _ := queries.GetQuerySet(&M2MTestGuardedProduct{}).Count()
		throughCount, _ := queries.GetQuerySet(&M2MTestGuardedCouponProduct{}).Count()
		if prodCount != 0 || throughCount != 0 {
			t.Errorf("expected 0 guarded products and 0 through rows, got products=%d through=%d", prodCount, throughCount)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, errM2MTargetBeforeSave, "")
	})

	// 6. DB UNIQUE constraint violation on inline M2M target must surface instead of silent rollback.
	t.Run("Add_TargetUniqueConstraintViolationSurfacesError", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		existing := &M2MTestGuardedProduct{Title: "Existing", SKU: "DUP-SKU-001", Price: 10}
		if _, err := queries.GetQuerySet(&M2MTestGuardedProduct{}).Create(existing); err != nil {
			t.Fatalf("seed existing product: %v", err)
		}

		coupon := &M2MTestGuardedCoupon{}
		panels := buildGuardedCouponM2MPanels(0, 5, 1)
		req := httptest.NewRequest(http.MethodPost, "/admin/guarded/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			url.Values{
				"Title": {"Dup SKU Promo"},
				"Code":  {"DUPSKU1"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title": {"Conflicting Product"},
				"Products-0-SKU":   {"DUP-SKU-001"},
				"Products-0-Price": {"99"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		prodCount, _ := queries.GetQuerySet(&M2MTestGuardedProduct{}).Count()
		throughCount, _ := queries.GetQuerySet(&M2MTestGuardedCouponProduct{}).Count()
		if prodCount != 1 {
			t.Errorf("expected only the 1 seeded M2MTestGuardedProduct row, got %d", prodCount)
		}
		if throughCount != 0 {
			t.Errorf("expected 0 M2MTestGuardedCouponProduct rows after unique failure, got %d", throughCount)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, nil, "")
	})

	// 7. Target model without ContextSaver/Saver and opted out of queries save hook returns "not saved" error.
	t.Run("Add_TargetWithoutContextSaverSurfacesError", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &M2MTestNoSaverCoupon{}
		panels := buildNoSaverCouponM2MPanels(0, 5, 1)

		req := httptest.NewRequest(http.MethodPost, "/admin/nosaver/add", nil)
		af := newTestAdminForm(req, coupon, nil, panels...)
		data := mergeFormValues(
			url.Values{
				"Title": {"No Saver Promo"},
				"Code":  {"NOSAVE1"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Unsaveable Item"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af.WithData(data, nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}

		_, saveErr := af.Save()

		prodCount, _ := queries.GetQuerySet(&M2MTestNoSaverProduct{}).Count()
		throughCount, _ := queries.GetQuerySet(&M2MTestNoSaverCouponProduct{}).Count()
		if prodCount != 0 || throughCount != 0 {
			t.Errorf("expected 0 products and 0 through rows, got products=%d through=%d", prodCount, throughCount)
		}

		assertAdminFormSaveReportedError(t, saveErr, af, nil, "not saved")
	})

	// 8. End-to-end views.FormView: inline M2M save failure must NOT call SuccessFn and must re-render with errors.
	t.Run("FormView_PostInlineM2MSaveFailureDoesNotInvokeSuccessFn", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		panels := buildStrictCouponM2MPanels(0, 5, 1)
		var successCalled bool

		fv := &views.FormView[*admin.AdminForm[modelforms.ModelForm[*M2MTestStrictCoupon], *M2MTestStrictCoupon]]{
			BaseView: views.BaseView{
				AllowedMethods:  []string{http.MethodGet, http.MethodPost},
				BaseTemplateKey: "m2m_test_views",
				TemplateName:    "m2m_test_views/form.tmpl",
			},
			GetFormFn: func(req *http.Request) *admin.AdminForm[modelforms.ModelForm[*M2MTestStrictCoupon], *M2MTestStrictCoupon] {
				return newTestAdminForm(req, &M2MTestStrictCoupon{}, nil, panels...)
			},
			SuccessFn: func(w http.ResponseWriter, _ *http.Request, _ *admin.AdminForm[modelforms.ModelForm[*M2MTestStrictCoupon], *M2MTestStrictCoupon]) {
				successCalled = true
				w.WriteHeader(http.StatusSeeOther)
			},
		}

		data := mergeFormValues(
			url.Values{
				"Title": {"FormView Strict Failure"},
				"Code":  {"FVFAIL1"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Strict Inline Product"},
				"Products-0-Price":                           {"199"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/admin/strict-coupon/add", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		if err := views.Invoke(fv, rec, req); err != nil {
			t.Fatalf("views.Invoke returned unexpected error: %v", err)
		}

		if successCalled {
			t.Errorf("expected SuccessFn NOT to be called when inline M2M save failed")
		}
		if !strings.Contains(rec.Body.String(), "FORM_ERR:") {
			t.Errorf("expected re-rendered FormView HTML to contain FORM_ERR:, body:\n%s", rec.Body.String())
		}
	})

	// 9. End-to-end views.FormView happy path: valid coupon + inline M2M product invokes SuccessFn and persists.
	t.Run("FormView_PostInlineM2MSaveSuccessInvokesSuccessFnAndPersists", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		panels := buildShopCouponM2MPanels(0, 5, 1)
		var successCalled bool
		var savedCoupon *FormTestCoupon

		fv := &views.FormView[*admin.AdminForm[modelforms.ModelForm[*FormTestCoupon], *FormTestCoupon]]{
			BaseView: views.BaseView{
				AllowedMethods:  []string{http.MethodGet, http.MethodPost},
				BaseTemplateKey: "m2m_test_views",
				TemplateName:    "m2m_test_views/form.tmpl",
			},
			GetFormFn: func(req *http.Request) *admin.AdminForm[modelforms.ModelForm[*FormTestCoupon], *FormTestCoupon] {
				return newTestAdminForm(req, &FormTestCoupon{}, nil, panels...)
			},
			SuccessFn: func(w http.ResponseWriter, _ *http.Request, form *admin.AdminForm[modelforms.ModelForm[*FormTestCoupon], *FormTestCoupon]) {
				successCalled = true
				savedCoupon = form.Instance()
				w.WriteHeader(http.StatusSeeOther)
			},
		}

		data := mergeFormValues(
			baseCouponPOST("FormView Happy Coupon", "FVGOOD1"),
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"FormView Gadget"},
				"Products-0-Slug":                            {"formview-gadget"},
				"Products-0-Price":                           {"321"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/add", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		if err := views.Invoke(fv, rec, req); err != nil {
			t.Fatalf("views.Invoke returned unexpected error: %v", err)
		}
		if !successCalled {
			t.Fatalf("expected SuccessFn to be called on valid M2M FormView submit, body:\n%s", rec.Body.String())
		}
		if savedCoupon == nil || savedCoupon.ID == 0 {
			t.Fatalf("expected persisted coupon with non-zero ID, got %+v", savedCoupon)
		}

		linked, err := savedCoupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query savedCoupon.Products: %v", err)
		}
		if len(linked) != 1 || linked[0].Object.Title != "FormView Gadget" || linked[0].Object.Price != 321 {
			t.Fatalf("expected 1 persisted linked product (FormView Gadget, $321), got %+v", linked)
		}
	})

	// 10. EditView on target with unlisted non-null FK preserves existing FK on in-place Update.
	t.Run("Edit_TargetWithUnlistedNonNullFKPreservesFKOnInPlaceUpdate", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		cat := &FormTestCategory{Name: "Electronics"}
		if _, err := queries.GetQuerySet(&FormTestCategory{}).Create(cat); err != nil {
			t.Fatalf("create category: %v", err)
		}
		prod := &M2MTestStrictProduct{Title: "Old Phone", Price: 500, Category: cat}
		if _, err := queries.GetQuerySet(&M2MTestStrictProduct{}).Create(prod); err != nil {
			t.Fatalf("create strict product: %v", err)
		}
		coupon := &M2MTestStrictCoupon{Title: "Promo", Code: "STRC1"}
		if _, err := queries.GetQuerySet(&M2MTestStrictCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create strict coupon: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(prod); err != nil {
			t.Fatalf("link strict product: %v", err)
		}

		data := mergeFormValues(
			url.Values{
				"Title": {"Promo"},
				"Code":  {"STRC1"},
			},
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(prod.ID)},
				"Products-0-Title": {"New Phone"},
				"Products-0-Price": {"650"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildStrictCouponM2MPanels(0, 5, 0)...)

		reloaded, err := queries.GetQuerySet(&M2MTestStrictProduct{}).Filter("ID", prod.ID).Get()
		if err != nil {
			t.Fatalf("reload strict product: %v", err)
		}
		if reloaded.Object.Title != "New Phone" || reloaded.Object.Price != 650 {
			t.Errorf("expected Title='New Phone', Price=650, got %+v", reloaded.Object)
		}
		if reloaded.Object.Category == nil || reloaded.Object.Category.ID != cat.ID {
			t.Errorf("expected preserved Category ID=%d, got %+v", cat.ID, reloaded.Object.Category)
		}
		throughCount, err := queries.GetQuerySet(&M2MTestStrictCouponProduct{}).Count()
		if err != nil {
			t.Fatalf("count strict through rows: %v", err)
		}
		if throughCount != 1 {
			t.Errorf("expected 1 strict through row after in-place update, got %d", throughCount)
		}
	})

	// 11. fattrs.Field + drivers.ULID PK M2M lifecycle without qmodels.Model: Add, Edit in place, Unlink.
	t.Run("FattrsULIDM2M_AddEditAndUnlinkThroughModelFormPanel", func(t *testing.T) {
		setupM2MExtendedTestDB(t)

		// Phase A: Add coupon with 1 inline ULID product (ID omitted).
		coupon := &M2MTestFattrsULIDCoupon{}
		addData := mergeFormValues(
			url.Values{
				"Title": {"ULID Promo"},
				"Code":  {"ULID01"},
			},
			inlineMgmtValues("Products", 1, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"ULID Speaker"},
				"Products-0-Price":                           {"150"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, addData, buildFattrsULIDCouponM2MPanels(0, 5, 1)...)
		if coupon.ID == 0 {
			t.Fatalf("expected non-zero created ULID coupon ID")
		}

		prodRows, err := queries.GetQuerySet(&M2MTestFattrsULIDProduct{}).All()
		if err != nil {
			t.Fatalf("list ULID products: %v", err)
		}
		if len(prodRows) != 1 {
			t.Fatalf("expected 1 ULID product created, got %d", len(prodRows))
		}
		origID := prodRows[0].Object.ID
		if origID.IsZero() {
			t.Fatalf("expected non-zero BeforeSave ULID assigned on product")
		}
		linkedAdd, err := coupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query linked ULID products after add: %v", err)
		}
		if len(linkedAdd) != 1 || linkedAdd[0].Object.ID != origID {
			t.Fatalf("expected 1 linked ULID product with ID %v, got %+v", origID, linkedAdd)
		}

		// Phase B: Reload coupon from DB and update ULID product in place without duplicating through row.
		loadedForEdit, err := queries.GetQuerySet(&M2MTestFattrsULIDCoupon{}).Filter("ID", coupon.ID).Get()
		if err != nil {
			t.Fatalf("reload ULID coupon for edit: %v", err)
		}
		editData := mergeFormValues(
			url.Values{
				"Title": {loadedForEdit.Object.Title},
				"Code":  {loadedForEdit.Object.Code},
			},
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {origID.String()},
				"Products-0-Title": {"ULID Studio Monitor"},
				"Products-0-Price": {"299"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, loadedForEdit.Object, editData, buildFattrsULIDCouponM2MPanels(0, 5, 0)...)

		prodRowsEdit, err := queries.GetQuerySet(&M2MTestFattrsULIDProduct{}).All()
		if err != nil {
			t.Fatalf("list ULID products after edit: %v", err)
		}
		if len(prodRowsEdit) != 1 {
			t.Fatalf("expected 1 ULID product after in-place edit, got %d", len(prodRowsEdit))
		}
		if prodRowsEdit[0].Object.ID != origID || prodRowsEdit[0].Object.Title != "ULID Studio Monitor" || prodRowsEdit[0].Object.Price != 299 {
			t.Errorf("unexpected ULID product after edit: %+v (want ID=%v)", prodRowsEdit[0].Object, origID)
		}
		throughAfterEdit, err := queries.GetQuerySet(&M2MTestFattrsULIDCouponProduct{}).Count()
		if err != nil {
			t.Fatalf("count ULID through rows after edit: %v", err)
		}
		if throughAfterEdit != 1 {
			t.Errorf("expected 1 ULID through row after in-place edit, got %d", throughAfterEdit)
		}

		// Phase C: Reload coupon and unlink via DELETE=true; through row deleted, product remains.
		loadedForDel, err := queries.GetQuerySet(&M2MTestFattrsULIDCoupon{}).Filter("ID", coupon.ID).Get()
		if err != nil {
			t.Fatalf("reload ULID coupon for unlink: %v", err)
		}
		delData := mergeFormValues(
			url.Values{
				"Title": {loadedForDel.Object.Title},
				"Code":  {loadedForDel.Object.Code},
			},
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {origID.String()},
				"Products-0-Title": {"ULID Studio Monitor"},
				"Products-0-Price": {"299"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"true"},
			},
		)
		submitAndSaveAdminForm(t, loadedForDel.Object, delData, buildFattrsULIDCouponM2MPanels(0, 5, 0)...)

		throughAfterDel, err := queries.GetQuerySet(&M2MTestFattrsULIDCouponProduct{}).Count()
		if err != nil {
			t.Fatalf("count ULID through rows after unlink: %v", err)
		}
		if throughAfterDel != 0 {
			t.Errorf("expected 0 ULID through rows after unlink, got %d", throughAfterDel)
		}
		prodCountAfterDel, err := queries.GetQuerySet(&M2MTestFattrsULIDProduct{}).Count()
		if err != nil {
			t.Fatalf("count ULID products after unlink: %v", err)
		}
		if prodCountAfterDel != 1 {
			t.Errorf("expected ULID product row to remain in DB after M2M unlink, got %d", prodCountAfterDel)
		}
	})

	// 12. Parent unchanged, only inline M2M product modified -> HasChanged() true & persists.
	t.Run("Edit_ParentUnchangedOnlyInlineM2MProductModifiedSaves", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Unchanged Parent",
			Description: "Desc",
			Code:        "UNCHP1",
			MaxDiscount: 20,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		prod := &FormTestProduct{Title: "Chair", Slug: "chair", Price: 100}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(prod); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(prod); err != nil {
			t.Fatalf("link product: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/edit", nil)
		af := newTestAdminForm(req, coupon, nil, buildShopCouponM2MPanels(0, 5, 0)...)
		af.WithData(mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(prod.ID)},
				"Products-0-Title": {prod.Title},
				"Products-0-Slug":  {prod.Slug},
				"Products-0-Price": {"250"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if !af.HasChanged() {
			t.Fatalf("expected HasChanged() == true when only inline M2M product price changed")
		}
		if _, err := af.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		reloaded, err := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", prod.ID).Get()
		if err != nil {
			t.Fatalf("reload product: %v", err)
		}
		if reloaded.Object.Price != 250 {
			t.Errorf("expected product Price=250, got %d", reloaded.Object.Price)
		}
	})

	// 13. Parent unchanged, only a new inline M2M product added -> HasChanged() true & persists.
	t.Run("Edit_ParentUnchangedOnlyInlineM2MProductAddedSaves", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Unchanged Parent Add Child",
			Description: "Desc",
			Code:        "UNCHP2",
			MaxDiscount: 20,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "First Desk", Slug: "first-desk", Price: 300}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create p1: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link p1: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/coupons/edit", nil)
		af := newTestAdminForm(req, coupon, nil, buildShopCouponM2MPanels(0, 5, 0)...)
		af.WithData(mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 2, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {p1.Title},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"300"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-Title":                           {"Second Desk"},
				"Products-1-Slug":                            {"second-desk"},
				"Products-1-Price":                           {"450"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if !af.HasChanged() {
			t.Fatalf("expected HasChanged() == true when only a new inline M2M product is added")
		}
		if _, err := af.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		linked, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 2 {
			t.Fatalf("expected 2 linked products, got %d", len(linked))
		}
	})

	// 14. Single Edit POST simultaneously updates p1, unlinks p2, and adds new p3.
	t.Run("Edit_SimultaneousUpdateUnlinkAndAddInSinglePost", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Mixed Ops",
			Description: "Mixed",
			Code:        "MIXED1",
			MaxDiscount: 30,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Keep & Edit", Slug: "keep-edit", Price: 111}
		p2 := &FormTestProduct{Title: "Unlink Me", Slug: "unlink-me", Price: 222}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create p1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p2); err != nil {
			t.Fatalf("create p2: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1, p2); err != nil {
			t.Fatalf("link p1,p2: %v", err)
		}

		data := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 3, 2, 0, 10),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {"Edited P1"},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"999"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-ID":    {fmt.Sprint(p2.ID)},
				"Products-1-Title": {p2.Title},
				"Products-1-Slug":  {p2.Slug},
				"Products-1-Price": {"222"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"true"},
				"Products-2-Title":                           {"Brand New P3"},
				"Products-2-Slug":                            {"brand-new-p3"},
				"Products-2-Price":                           {"42"},
				"Products-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Products-2-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 10, 0)...)

		linked, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 2 {
			t.Fatalf("expected 2 linked products after mixed edit, got %d", len(linked))
		}
		if linked[0].Object.ID != p1.ID || linked[0].Object.Title != "Edited P1" || linked[0].Object.Price != 999 {
			t.Errorf("expected linked[0] Edited P1 ($999), got %+v", linked[0].Object)
		}
		if linked[1].Object.Title != "Brand New P3" || linked[1].Object.Price != 42 {
			t.Errorf("expected linked[1] Brand New P3 ($42), got %+v", linked[1].Object)
		}

		totalProducts, err := queries.GetQuerySet(&FormTestProduct{}).Count()
		if err != nil {
			t.Fatalf("count products: %v", err)
		}
		if totalProducts != 3 {
			t.Errorf("expected p2 to remain in product table (3 total products), got %d", totalProducts)
		}
	})

	// 15. Unlinking all M2M products leaves 0 through rows while preserving target rows.
	t.Run("Edit_UnlinkAllM2MProductsLeavesZeroThroughRowsAndPreservesTargets", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Unlink All",
			Description: "Clear links",
			Code:        "CLRALL",
			MaxDiscount: 15,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Target One", Slug: "target-one", Price: 10}
		p2 := &FormTestProduct{Title: "Target Two", Slug: "target-two", Price: 20}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create p1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p2); err != nil {
			t.Fatalf("create p2: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1, p2); err != nil {
			t.Fatalf("link p1,p2: %v", err)
		}

		data := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 2, 2, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {p1.Title},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"10"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"true"},
				"Products-1-ID":    {fmt.Sprint(p2.ID)},
				"Products-1-Title": {p2.Title},
				"Products-1-Slug":  {p2.Slug},
				"Products-1-Price": {"20"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"true"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)

		throughCount, err := queries.GetQuerySet(&FormTestCouponProduct{}).Filter("Coupon", coupon.ID).Count()
		if err != nil {
			t.Fatalf("count through rows: %v", err)
		}
		if throughCount != 0 {
			t.Errorf("expected 0 through rows after unlinking all, got %d", throughCount)
		}
		targetCount, err := queries.GetQuerySet(&FormTestProduct{}).Count()
		if err != nil {
			t.Fatalf("count FormTestProduct rows: %v", err)
		}
		if targetCount != 2 {
			t.Errorf("expected both target rows preserved in DB (2), got %d", targetCount)
		}
	})

	// 16. Extra blank slot marked deleted on Edit does not create a spurious product.
	t.Run("Edit_ExtraBlankSlotMarkedDeletedDoesNotCreateSpuriousProduct", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Extra Slot Edit",
			Description: "Extra 1",
			Code:        "EXTRA1",
			MaxDiscount: 10,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Initial P1", Slug: "initial-p1", Price: 80}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create p1: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link p1: %v", err)
		}

		panels := buildShopCouponM2MPanels(0, 5, 2)
		reqGet := httptest.NewRequest(http.MethodGet, "/admin/coupons/edit", nil)
		afGet := newTestAdminForm(reqGet, coupon, nil, panels...)
		prodFS := unwrapProductFormSet(t, afGet)
		getForms, err := prodFS.Forms()
		if err != nil {
			t.Fatalf("GET FormSet Forms() failed: %v", err)
		}
		if len(getForms) != 2 {
			t.Fatalf("expected 2 forms on Edit GET with Extra=2 (1 existing + 1 blank), got %d", len(getForms))
		}

		postData := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 2, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {"Kept P1"},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"80"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-Title":                           {""},
				"Products-1-Slug":                            {""},
				"Products-1-Price":                           {""},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"true"},
			},
		)
		submitAndSaveAdminForm(t, coupon, postData, panels...)

		totalProducts, _ := queries.GetQuerySet(&FormTestProduct{}).Count()
		if totalProducts != 1 {
			t.Errorf("expected exactly 1 product in DB, got %d", totalProducts)
		}
		linked, err := coupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 1 || linked[0].Object.Title != "Kept P1" {
			t.Errorf("expected 1 linked product 'Kept P1', got %+v", linked)
		}
	})

	// 17. DisallowAdd and DisallowRemove omit deletion field, reject adding extra form, yet allow editing existing.
	t.Run("DisallowAddAndDisallowRemove_OmitControlFieldsAndUpdateStillSaves", func(t *testing.T) {
		setupM2MExtendedTestDB(t)
		coupon := &FormTestCoupon{
			Title:       "Locked Set",
			Description: "No add no remove",
			Code:        "LOCKED1",
			MaxDiscount: 25,
			ValidFrom:   "2026-01-01",
			ValidUntil:  "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Locked P1", Slug: "locked-p1", Price: 70}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create p1: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link p1: %v", err)
		}

		panels := []admin.Panel{
			admin.FieldPanel("Title"),
			admin.FieldPanel("Description"),
			admin.FieldPanel("Code"),
			admin.FieldPanel("MaxDiscount"),
			admin.FieldPanel("IsPercentage"),
			admin.FieldPanel("ValidFrom"),
			admin.FieldPanel("ValidUntil"),
			&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
				TargetType:     &FormTestProduct{},
				FieldName:      "Products",
				MinNum:         0,
				MaxNum:         5,
				Extra:          0,
				DisallowAdd:    true,
				DisallowRemove: true,
				Panels: []admin.Panel{
					admin.FieldPanel("ID"),
					admin.FieldPanel("Title"),
					admin.FieldPanel("Slug"),
					admin.FieldPanel("Price"),
				},
			},
		}

		reqGet := httptest.NewRequest(http.MethodGet, "/admin/coupons/edit", nil)
		afGet := newTestAdminForm(reqGet, coupon, nil, panels...)
		doc := parseHTMLDoc(t, afGet.BoundForm().AsP())
		if delLen := doc.Find(`input[name="Products-0-` + formsets.DELETION_FIELD_NAME + `"]`).Length(); delLen != 0 {
			t.Errorf("expected 0 deletion inputs when DisallowRemove=true, got %d", delLen)
		}

		// Attempting to add an extra form when DisallowAdd=true must fail validation.
		addAttempt := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 2, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {p1.Title},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"70"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-1-Title":                           {"Disallowed New"},
				"Products-1-Slug":                            {"disallowed-new"},
				"Products-1-Price":                           {"99"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			},
		)
		submitExpectInvalidAdminForm(t, coupon, addAttempt, panels...)

		// Updating the existing product in place must succeed.
		validEdit := mergeFormValues(
			couponEditPOST(coupon),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":    {fmt.Sprint(p1.ID)},
				"Products-0-Title": {"Unlocked Update"},
				"Products-0-Slug":  {p1.Slug},
				"Products-0-Price": {"85"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			},
		)
		submitAndSaveAdminForm(t, coupon, validEdit, panels...)

		reloaded, err := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", p1.ID).Get()
		if err != nil {
			t.Fatalf("reload p1: %v", err)
		}
		if reloaded.Object.Title != "Unlocked Update" || reloaded.Object.Price != 85 {
			t.Errorf("expected updated p1 Title='Unlocked Update', Price=85, got %+v", reloaded.Object)
		}
		throughCount, _ := queries.GetQuerySet(&FormTestCouponProduct{}).Filter("Coupon", coupon.ID).Count()
		if throughCount != 1 {
			t.Errorf("expected 1 through row, got %d", throughCount)
		}
	})

	// 18. Multiple inline ModelFormPanels (O2M Items + M2M Products) in the same TabbedPanel.
	t.Run("MultipleInlinePanels_O2MAndM2MInSameTabbedAdminForm", func(t *testing.T) {
		buildMultiInlinePanels := func(m2mSaveHook func(context.Context, modelforms.ModelForm[*FormTestProduct], *FormTestProduct) error) []admin.Panel {
			return []admin.Panel{
				admin.TabbedPanel(
					admin.PanelTab(
						trans.S("Coupon"),
						admin.FieldPanel("Title"),
						admin.FieldPanel("Description"),
						admin.FieldPanel("Code"),
						admin.FieldPanel("MaxDiscount"),
						admin.FieldPanel("IsPercentage"),
						admin.FieldPanel("ValidFrom"),
						admin.FieldPanel("ValidUntil"),
					),
					admin.PanelTab(
						trans.S("Items"),
						&admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]]{
							TargetType: &FormTestCouponItem{},
							FieldName:  "Items",
							MinNum:     0,
							MaxNum:     5,
							Panels: []admin.Panel{
								admin.FieldPanel("ID"),
								admin.FieldPanel("ProductName"),
								admin.FieldPanel("Quantity"),
							},
						},
					),
					admin.PanelTab(
						trans.S("Products"),
						&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
							TargetType:   &FormTestProduct{},
							FieldName:    "Products",
							MinNum:       0,
							MaxNum:       5,
							SaveInstance: m2mSaveHook,
							Panels: []admin.Panel{
								admin.FieldPanel("ID"),
								admin.FieldPanel("Title"),
								admin.FieldPanel("Slug"),
								admin.FieldPanel("Price"),
							},
						},
					),
				),
			}
		}

		t.Run("BothPanelsSaveAndLink", func(t *testing.T) {
			setupM2MExtendedTestDB(t)
			coupon := &FormTestCoupon{}
			data := mergeFormValues(
				baseCouponPOST("Dual Inline Success", "DUALOK1"),
				inlineMgmtValues("Items", 1, 0, 0, 5),
				url.Values{
					"Items-0-ProductName":                     {"Bonus Voucher"},
					"Items-0-Quantity":                        {"2"},
					"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
					"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				},
				inlineMgmtValues("Products", 1, 0, 0, 5),
				url.Values{
					"Products-0-Title":                           {"Bundled Headset"},
					"Products-0-Slug":                            {"bundled-headset"},
					"Products-0-Price":                           {"140"},
					"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
					"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				},
			)
			submitAndSaveAdminForm(t, coupon, data, buildMultiInlinePanels(nil)...)

			items, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).All()
			if err != nil {
				t.Fatalf("query items: %v", err)
			}
			if len(items) != 1 || items[0].Object.ProductName != "Bonus Voucher" || items[0].Object.Quantity != 2 {
				t.Errorf("expected 1 linked O2M item, got %+v", items)
			}

			linked, err := coupon.Products.Objects().All()
			if err != nil {
				t.Fatalf("query products: %v", err)
			}
			if len(linked) != 1 || linked[0].Object.Title != "Bundled Headset" || linked[0].Object.Price != 140 {
				t.Errorf("expected 1 linked M2M product, got %+v", linked)
			}
		})

		t.Run("M2MTabFailureSurfacedEvenWhenO2MTabSucceeds", func(t *testing.T) {
			setupM2MExtendedTestDB(t)
			errM2MTabFail := errors.New("m2m tab save failed")
			coupon := &FormTestCoupon{}
			panels := buildMultiInlinePanels(func(_ context.Context, _ modelforms.ModelForm[*FormTestProduct], _ *FormTestProduct) error {
				return errM2MTabFail
			})

			req := httptest.NewRequest(http.MethodPost, "/admin/coupons/add", nil)
			af := newTestAdminForm(req, coupon, nil, panels...)
			data := mergeFormValues(
				baseCouponPOST("Dual Inline M2M Fail", "DUALFAIL"),
				inlineMgmtValues("Items", 1, 0, 0, 5),
				url.Values{
					"Items-0-ProductName":                     {"Saved Item"},
					"Items-0-Quantity":                        {"1"},
					"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
					"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				},
				inlineMgmtValues("Products", 1, 0, 0, 5),
				url.Values{
					"Products-0-Title":                           {"Failing M2M Tab Product"},
					"Products-0-Slug":                            {"failing-m2m-tab-product"},
					"Products-0-Price":                           {"90"},
					"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
					"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				},
			)
			af.WithData(data, nil, req)
			if !forms.IsValid(req.Context(), af) {
				t.Fatalf("expected valid form before save, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
			}

			_, saveErr := af.Save()

			if throughCount, _ := queries.GetQuerySet(&FormTestCouponProduct{}).Count(); throughCount != 0 {
				t.Errorf("expected 0 M2M through rows on failure, got %d", throughCount)
			}
			assertAdminFormSaveReportedError(t, saveErr, af, errM2MTabFail, "")
		})
	})
}
