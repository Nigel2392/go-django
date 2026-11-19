package admin_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/compare"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	qfields "github.com/Nigel2392/go-django/queries/src/fields"
	_ "github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	qmodels "github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/PuerkitoBio/goquery"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test models & database setup
// ─────────────────────────────────────────────────────────────────────────────

type FormTestUser struct {
	qmodels.Model   `table:"admin_forms_test_users"`
	ID              int64
	Email           string
	Username        string
	FirstName       string
	LastName        string
	Password        string
	IsAdministrator bool
	IsActive        bool
	Data            string
	CreatedAt       string
	UpdatedAt       string
}

func (u *FormTestUser) String() string {
	return u.Username
}

func (u *FormTestUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return u.Model.Define(ctx, u,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Email", &attrs.FieldConfig{MinLength: 3, HelpText: "Primary user email"}),
		attrs.Unbound("Username", &attrs.FieldConfig{MinLength: 3}),
		attrs.Unbound("FirstName", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("LastName", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("Password"),
		attrs.Unbound("IsAdministrator", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("IsActive", &attrs.FieldConfig{Blank: true, Default: true}),
		attrs.Unbound("Data", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{ReadOnly: true}),
		attrs.Unbound("UpdatedAt", &attrs.FieldConfig{ReadOnly: true}),
	)
}

type FormTestCategory struct {
	qmodels.Model `table:"admin_forms_test_categories"`
	ID            int64
	Name          string
}

func (c *FormTestCategory) String() string {
	return c.Name
}

func (c *FormTestCategory) FieldDefs(ctx context.Context) attrs.Definitions {
	return c.Model.Define(ctx, c,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Name", &attrs.FieldConfig{MinLength: 2}),
	)
}

type FormTestProduct struct {
	qmodels.Model `table:"admin_forms_test_products"`
	ID            int64
	Title         string
	Slug          string
	Price         int
	Coupons       *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *FormTestProduct) String() string {
	return p.Title
}

func (p *FormTestProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title", &attrs.FieldConfig{MinLength: 2}),
		attrs.Unbound("Slug", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("Price", &attrs.FieldConfig{Blank: true}),
	)
}

type FormTestCouponProduct struct {
	qmodels.Model `table:"admin_forms_test_coupon_products"`
	Coupon        *FormTestCoupon
	Product       *FormTestProduct
}

func (cp *FormTestCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return cp.Model.Define(ctx, cp,
		attrs.NewField(cp, "Coupon", &attrs.FieldConfig{
			Column: "coupon_id",
			Null:   false,
		}),
		attrs.NewField(cp, "Product", &attrs.FieldConfig{
			Column: "product_id",
			Null:   false,
		}),
	)
}

type FormTestCoupon struct {
	qmodels.Model `table:"admin_forms_test_coupons"`
	ID            int64
	Title         string
	Description   string
	Code          string
	MaxDiscount   int
	IsPercentage  bool
	ValidFrom     string
	ValidUntil    string
	CreatedAt     string
	Category      *FormTestCategory
	Products      *queries.RelM2M[*FormTestProduct, *FormTestCouponProduct]
}

func (c *FormTestCoupon) String() string {
	return c.Title
}

func (c *FormTestCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return c.Model.Define(ctx, c,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title"),
		attrs.Unbound("Description", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("Code", &attrs.FieldConfig{MinLength: 4}),
		attrs.Unbound("MaxDiscount", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("IsPercentage", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("ValidFrom", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("ValidUntil", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{ReadOnly: true}),
		attrs.Unbound("Category", &attrs.FieldConfig{
			Blank:         true,
			Null:          true,
			RelForeignKey: attrs.Relate(&FormTestCategory{}, "", nil),
		}),
		qfields.NewManyToManyField[*queries.RelM2M[*FormTestProduct, *FormTestCouponProduct]](c, "Products", &qfields.FieldConfig{
			ScanTo:      &c.Products,
			ReverseName: "Coupons",
			Rel: attrs.Relate(
				&FormTestProduct{},
				"", &attrs.ThroughModel{
					This:   &FormTestCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		}),
	)
}

type FormTestCouponItem struct {
	qmodels.Model `table:"admin_forms_test_coupon_items"`
	ID            int64
	Coupon        *FormTestCoupon
	ProductName   string
	Quantity      int
}

func (i *FormTestCouponItem) String() string {
	return i.ProductName
}

func (i *FormTestCouponItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return i.Model.Define(ctx, i,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Coupon", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&FormTestCoupon{}, "", nil),
			Blank:         true,
			Null:          true,
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "Items",
			},
			FormField: func(opts ...func(fields.Field)) fields.Field {
				return fields.CharField(append([]func(fields.Field){fields.Hide(true)}, opts...)...)
			},
		}),
		attrs.Unbound("ProductName"),
		attrs.Unbound("Quantity", &attrs.FieldConfig{Blank: true}),
	)
}

type FormTestAuthorBio struct {
	qmodels.Model   `table:"admin_forms_test_author_bios"`
	ID              int64
	Headline        string
	Website         string
	ExperienceYears int
}

func (b *FormTestAuthorBio) String() string {
	return b.Headline
}

func (b *FormTestAuthorBio) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Headline", &attrs.FieldConfig{MinLength: 3}),
		attrs.Unbound("Website", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("ExperienceYears", &attrs.FieldConfig{Blank: true}),
	)
}

type FormTestAuthorBadge struct {
	qmodels.Model `table:"admin_forms_test_author_badges"`
	ID            int64
	BadgeName     string
	Tier          string
}

func (b *FormTestAuthorBadge) String() string {
	return b.BadgeName
}

func (b *FormTestAuthorBadge) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("BadgeName", &attrs.FieldConfig{MinLength: 2}),
		attrs.Unbound("Tier", &attrs.FieldConfig{Blank: true}),
	)
}

type FormTestAuthorBadgeThrough struct {
	qmodels.Model `table:"admin_forms_test_author_badge_links"`
	Author        *FormTestAuthor
	Badge         *FormTestAuthorBadge
}

func (ab *FormTestAuthorBadgeThrough) FieldDefs(ctx context.Context) attrs.Definitions {
	return ab.Model.Define(ctx, ab,
		attrs.NewField(ab, "Author", &attrs.FieldConfig{
			Column: "author_id",
			Null:   false,
		}),
		attrs.NewField(ab, "Badge", &attrs.FieldConfig{
			Column: "badge_id",
			Null:   false,
		}),
	)
}

type FormTestAuthor struct {
	qmodels.Model `table:"admin_forms_test_authors"`
	ID            int64
	Name          string
	Email         string
	Bio           *FormTestAuthorBio
	Badge         *queries.RelO2O[*FormTestAuthorBadge, *FormTestAuthorBadgeThrough]
}

func (a *FormTestAuthor) String() string {
	return a.Name
}

func (a *FormTestAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Name", &attrs.FieldConfig{MinLength: 2}),
		attrs.Unbound("Email", &attrs.FieldConfig{Blank: true}),
		attrs.Unbound("Bio", &attrs.FieldConfig{
			Blank:       true,
			Null:        true,
			Column:      "bio_id",
			RelOneToOne: attrs.Relate(&FormTestAuthorBio{}, "", nil),
		}),
		qfields.OneToOne[*queries.RelO2O[*FormTestAuthorBadge, *FormTestAuthorBadgeThrough]]("Badge", &qfields.FieldConfig{
			ReverseName: "AuthorBadgeReverse",
			Rel: attrs.Relate(
				&FormTestAuthorBadge{},
				"", &attrs.ThroughModel{
					This:   &FormTestAuthorBadgeThrough{},
					Source: "Author",
					Target: "Badge",
				},
			),
		}),
	)
}

type FormTestDefaultPanelModel struct {
	qmodels.Model `table:"admin_forms_test_default_panel_models"`
	ID            int64
	Title         string
	Summary       string
}

func (m *FormTestDefaultPanelModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("Title"),
		attrs.Unbound("Summary", &attrs.FieldConfig{Blank: true}),
	)
}

func (m *FormTestDefaultPanelModel) GetAddPanels() []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")),
	}
}

func (m *FormTestDefaultPanelModel) GetEditPanels() []admin.Panel {
	return []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Title")),
		admin.FieldPanel("Summary"),
	}
}

func (m *FormTestDefaultPanelModel) AdminForm(r *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) modelforms.ModelForm[*FormTestDefaultPanelModel] {
	f := modelforms.NewBaseModelForm(r.Context(), m)
	f.AddField("ExtraNote", fields.CharField(fields.Required(false), fields.Label("Extra Note")))
	return f
}

type customValidatorPanel struct {
	admin.Panel
	validateFn func(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error
}

func (c *customValidatorPanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	if c.validateFn != nil {
		return c.validateFn(r, ctx, form, data)
	}
	return nil
}

// Compile-time interface assertions.
var (
	_ admin.ValidatorPanel                          = (*customValidatorPanel)(nil)
	_ admin.FormDefiner[*FormTestDefaultPanelModel] = (*FormTestDefaultPanelModel)(nil)
)

var (
	formsTestDBOnce sync.Once
	formsTestDB     drivers.Database
)

func setupFormsTestDB(t *testing.T) drivers.Database {
	t.Helper()

	formsTestDBOnce.Do(func() {
		db, err := drivers.Open(
			context.Background(),
			"sqlite3",
			"file:admin_forms_test?mode=memory&cache=shared&_loc=auto",
		)
		if err != nil {
			panic(fmt.Errorf("failed to open sqlite3 memory db: %w", err))
		}
		formsTestDB = db

		if django.Global == nil {
			django.App(
				django.Configure(map[string]any{
					django.APPVAR_DATABASE:      db,
					django.APPVAR_ALLOWED_HOSTS: []string{"*"},
				}),
			)
		} else {
			django.Global.Settings.Set(django.APPVAR_DATABASE, db)
		}

		testModels := []attrs.Definer{
			&FormTestUser{},
			&FormTestCategory{},
			&FormTestProduct{},
			&FormTestCoupon{},
			&FormTestCouponProduct{},
			&FormTestCouponItem{},
			&FormTestDefaultPanelModel{},
			&FormTestAuthorBio{},
			&FormTestAuthorBadge{},
			&FormTestAuthor{},
			&FormTestAuthorBadgeThrough{},
		}
		for _, mdl := range testModels {
			attrs.RegisterModel(mdl)
		}

		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestUser{},
			GetLabel:       trans.S("Form Test User"),
			GetPluralLabel: trans.S("Form Test Users"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestCategory{},
			GetLabel:       trans.S("Form Test Category"),
			GetPluralLabel: trans.S("Form Test Categories"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestProduct{},
			GetLabel:       trans.S("Form Test Product"),
			GetPluralLabel: trans.S("Form Test Products"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestCoupon{},
			GetLabel:       trans.S("Form Test Coupon"),
			GetPluralLabel: trans.S("Form Test Coupons"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestCouponProduct{},
			GetLabel:       trans.S("Form Test Coupon Product"),
			GetPluralLabel: trans.S("Form Test Coupon Products"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestCouponItem{},
			GetLabel:       trans.S("Form Test Coupon Item"),
			GetPluralLabel: trans.S("Form Test Coupon Items"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestDefaultPanelModel{},
			GetLabel:       trans.S("Form Test Default Panel Model"),
			GetPluralLabel: trans.S("Form Test Default Panel Models"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestAuthorBio{},
			GetLabel:       trans.S("Form Test Author Bio"),
			GetPluralLabel: trans.S("Form Test Author Bios"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestAuthorBadge{},
			GetLabel:       trans.S("Form Test Author Badge"),
			GetPluralLabel: trans.S("Form Test Author Badges"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestAuthor{},
			GetLabel:       trans.S("Form Test Author"),
			GetPluralLabel: trans.S("Form Test Authors"),
		})
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &FormTestAuthorBadgeThrough{},
			GetLabel:       trans.S("Form Test Author Badge Through"),
			GetPluralLabel: trans.S("Form Test Author Badge Throughs"),
		})

		if err := attrs.ResetDefinitions.Send(context.Background(), nil); err != nil {
			panic(fmt.Errorf("failed to reset model definitions: %w", err))
		}

		schemaEditor, err := migrator.GetSchemaEditor(db.Driver())
		if err != nil {
			panic(fmt.Errorf("failed to get schema editor: %w", err))
		}

		for _, mdl := range testModels {
			table := migrator.NewModelTable(mdl)
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				panic(fmt.Errorf("failed to create table %s: %w", table.TableName(), err))
			}
		}
	})

	// Ensure settings always point to our active in-memory DB even if another
	// test (e.g. TestAdmin) opened and closed its own database earlier.
	django.Global.Settings.Set(django.APPVAR_DATABASE, formsTestDB)

	// Clean up rows from any prior test run.
	_, _ = queries.GetQuerySet(&FormTestCouponProduct{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestCoupon{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestProduct{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestCategory{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestUser{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestDefaultPanelModel{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestAuthorBadgeThrough{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestAuthor{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestAuthorBadge{}).Delete()
	_, _ = queries.GetQuerySet(&FormTestAuthorBio{}).Delete()

	return formsTestDB
}

func newTestAdminForm[T attrs.Definer](req *http.Request, instance T, exclude []string, panels ...admin.Panel) *admin.AdminForm[modelforms.ModelForm[T], T] {
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	}
	modelForm := modelforms.NewBaseModelForm(req.Context(), instance)
	if len(exclude) > 0 {
		modelForm.SetExclude(exclude...)
	}
	var mf modelforms.ModelForm[T] = modelForm
	af := admin.NewAdminForm(req, mf, panels...)
	af.Load()
	return af
}

func parseHTMLDoc(t *testing.T, htmlStr template.HTML) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlStr)))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v\nHTML:\n%s", err, htmlStr)
	}
	return doc
}

func bindSinglePanelToForm(t *testing.T, instance attrs.Definer, panels ...admin.Panel) (*admin.AdminForm[modelforms.ModelForm[attrs.Definer], attrs.Definer], *admin.PanelBoundForm) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	af := newTestAdminForm(req, instance, nil, panels...)
	bf := af.BoundForm().(*admin.PanelBoundForm)
	return af, bf
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Default Panels, Field Filtering, BoundForm & AdminForm Delegation
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_DefaultPanelsAndFieldFiltering(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("AutoPopulatesFieldPanelsWhenEmpty", func(t *testing.T) {
		user := &FormTestUser{}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		af := newTestAdminForm(req, user, []string{"ID", "CreatedAt", "UpdatedAt"})

		expectedFields := af.FieldMap().Keys()
		if len(af.Panels) != len(expectedFields) {
			t.Fatalf("expected %d auto panels, got %d", len(expectedFields), len(af.Panels))
		}
		for i, name := range expectedFields {
			if got := af.Panels[i].Fields(); len(got) != 1 || got[0] != name {
				t.Errorf("panel[%d].Fields() = %v, want [%s]", i, got, name)
			}
		}
	})

	t.Run("DeletesUnreferencedFieldsWhenPanelsSpecified", func(t *testing.T) {
		user := &FormTestUser{}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		af := newTestAdminForm(req, user, nil,
			admin.TitlePanel(admin.FieldPanel("Email")),
			admin.RowPanel(
				admin.FieldPanel("FirstName"),
				admin.FieldPanel("LastName"),
			),
		)

		kept := map[string]bool{
			"Email":     true,
			"FirstName": true,
			"LastName":  true,
		}
		for _, key := range af.FieldMap().Keys() {
			if !kept[key] {
				t.Errorf("expected unreferenced field %q to be deleted", key)
			}
		}
		for name := range kept {
			if _, ok := af.Field(name); !ok {
				t.Errorf("expected referenced field %q to remain in form", name)
			}
		}
		if _, ok := af.Field("Username"); ok {
			t.Errorf("expected Username to be deleted from form")
		}
	})

	t.Run("NewPanelBoundFormWithoutPanelsRespectsFieldOrderAndFallback", func(t *testing.T) {
		user := &FormTestUser{
			Email:     "ordered@example.com",
			Username:  "ordered",
			FirstName: "First",
			LastName:  "Last",
			Password:  "secret",
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		base := modelforms.NewBaseModelForm(req.Context(), user)
		base.SetFields("Email", "Username", "FirstName", "LastName")
		base.Load()

		// Without FieldOrder
		bf := forms.NewBoundForm(req.Context(), base, nil)
		pbf := admin.NewPanelBoundForm(req.Context(), req, user, base, bf, nil, nil)
		if len(pbf.BoundPanels) != 4 {
			t.Fatalf("expected 4 bound panels, got %d", len(pbf.BoundPanels))
		}

		// With explicit FieldOrder
		base.Ordering([]string{"LastName", "Email"})
		pbfOrdered := admin.NewPanelBoundForm(req.Context(), req, user, base, bf, nil, nil)
		if len(pbfOrdered.BoundPanels) != 2 {
			t.Fatalf("expected 2 bound panels for FieldOrder, got %d", len(pbfOrdered.BoundPanels))
		}

		asP := pbfOrdered.AsP()
		asUL := pbfOrdered.AsUL()
		if asP == "" || asP != asUL {
			t.Errorf("expected non-empty matching AsP() and AsUL() output")
		}
		doc := parseHTMLDoc(t, asP)
		titles := make([]string, 0)
		doc.Find(".panel__title").Each(func(_ int, s *goquery.Selection) {
			titles = append(titles, strings.TrimSpace(s.Text()))
		})
		if len(titles) != 2 || titles[0] != "Last Name" || titles[1] != "Email" {
			t.Errorf("expected ordered titles [Last Name Email], got %v", titles)
		}

		if pbfOrdered.Media() == nil {
			t.Errorf("expected non-nil Media()")
		}
		if len(pbfOrdered.Fields()) != 4 {
			t.Errorf("expected 4 bound fields, got %d", len(pbfOrdered.Fields()))
		}
		if pbfOrdered.Errors() != nil && pbfOrdered.Errors().Len() != 0 {
			t.Errorf("expected no errors, got %v", pbfOrdered.Errors())
		}
		if len(pbfOrdered.ErrorList()) != 0 {
			t.Errorf("expected empty error list, got %v", pbfOrdered.ErrorList())
		}
	})

	t.Run("AdminFormDelegationMethods", func(t *testing.T) {
		user := &FormTestUser{Username: "delegate"}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		af := newTestAdminForm(req, user, []string{"ID", "CreatedAt", "UpdatedAt"},
			admin.FieldPanel("Email"),
			admin.FieldPanel("Username"),
		)

		af.SetPrefix("usr")
		if af.Prefix() != "usr" {
			t.Errorf("Prefix() = %q, want usr", af.Prefix())
		}
		if af.PrefixName("Email") != "usr-Email" {
			t.Errorf("PrefixName(Email) = %q, want usr-Email", af.PrefixName("Email"))
		}

		af.Ordering([]string{"Username", "Email"})
		if order := af.FieldOrder(); len(order) != 2 || order[0] != "Username" {
			t.Errorf("FieldOrder() = %v, want [Username Email]", order)
		}

		customField := fields.CharField(fields.Label("Nickname"), fields.Required(false))
		af.AddField("Nickname", customField)
		if f, ok := af.Field("Nickname"); !ok || f.Label(req.Context()) != "Nickname" {
			t.Errorf("AddField/Field failed for Nickname")
		}
		if len(af.Fields()) < 3 {
			t.Errorf("expected at least 3 fields")
		}
		if !af.DeleteField("Nickname") {
			t.Errorf("expected DeleteField(Nickname) to return true")
		}

		customWidget := widgets.NewTextarea(nil)
		af.AddWidget("Email", customWidget)
		if w, ok := af.Widget("Email"); !ok || w != customWidget {
			t.Errorf("AddWidget/Widget failed for Email")
		}
		if len(af.Widgets()) == 0 {
			t.Errorf("expected non-empty Widgets()")
		}

		af.SetInitial(map[string]any{"Email": "init@example.com", "Username": "delegate"})
		if init := af.InitialData(); init["Email"] != "init@example.com" {
			t.Errorf("InitialData[Email] = %v, want init@example.com", init["Email"])
		}

		newCtx := context.WithValue(req.Context(), struct{ k string }{"k"}, "v")
		af.WithContext(newCtx)
		if af.Context() != newCtx {
			t.Errorf("WithContext/Context mismatch")
		}

		if af.Renderer() == nil {
			t.Errorf("expected non-nil Renderer()")
		}
		af.SetRenderer(af.Renderer())

		if af.Media() == nil {
			t.Errorf("expected non-nil Media()")
		}
		if af.BoundFields().Len() == 0 {
			t.Errorf("expected non-empty BoundFields()")
		}
		if af.FormValue("Email") != "init@example.com" {
			t.Errorf("FormValue(Email) = %v, want init@example.com", af.FormValue("Email"))
		}

		c := ctx.NewContext(nil)
		af.EditContext("form", c)
		if _, ok := c.Get("form").(*admin.PanelBoundForm); !ok {
			t.Errorf("EditContext expected *admin.PanelBoundForm, got %T", c.Get("form"))
		}

		other := &FormTestUser{ID: 88, Username: "other_instance"}
		unloadedMF := modelforms.ModelForm[*FormTestUser](modelforms.NewBaseModelForm(req.Context(), user))
		unloadedAF := admin.NewAdminForm(req, unloadedMF, admin.FieldPanel("Email"))
		unloadedAF.SetInstance(other)
		unloadedAF.Load()
		if unloadedAF.Instance() != other || unloadedAF.Instance().Username != "other_instance" {
			t.Errorf("expected Instance() to return distinct other instance after SetInstance before Load, got %+v", unloadedAF.Instance())
		}

		af.AddError("Email", errors.New("bad email"))
		af.AddFormError(errors.New("form level error"))
		if af.BoundErrors() == nil || af.BoundErrors().Len() == 0 {
			t.Errorf("expected BoundErrors after AddError")
		}
		if len(af.ErrorList()) != 1 {
			t.Errorf("expected 1 form error, got %d", len(af.ErrorList()))
		}
		unpacked := af.UnpackErrors(af.BoundForm(), af.BoundErrors())
		if len(unpacked) == 0 {
			t.Errorf("expected UnpackErrors to return field errors")
		}

		if !af.WasCleaned() {
			t.Errorf("expected WasCleaned() to be true when errors exist")
		}

		unwrapped := af.Unwrap()
		if len(unwrapped) != 1 || unwrapped[0] != af {
			t.Errorf("expected Unwrap() to return [af] when no formset, got %v", unwrapped)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. FieldPanel & HiddenFieldPanel
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_FieldPanelAndHiddenFieldPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("FieldPanelPropertiesAndHTML", func(t *testing.T) {
		p := admin.FieldPanel("Email", "custom-initial")
		if got := p.Fields(); len(got) != 1 || got[0] != "Email" {
			t.Fatalf("Fields() = %v, want [Email]", got)
		}
		if p.ClassName() != "custom-initial" {
			t.Fatalf("ClassName() = %q, want custom-initial", p.ClassName())
		}
		admin.PanelClass("extra-class", p)
		if p.ClassName() != "extra-class" {
			t.Fatalf("PanelClass did not update ClassName: got %q", p.ClassName())
		}
		if vp, ok := p.(admin.ValidatorPanel); !ok || vp.Validate(nil, context.Background(), nil, nil) != nil {
			t.Errorf("FieldPanel.Validate expected nil")
		}

		user := &FormTestUser{Email: "alice@example.com"}
		_, bf := bindSinglePanelToForm(t, user, p)
		html := bf.AsP()
		doc := parseHTMLDoc(t, html)

		panelDiv := doc.Find(".panel.panel-default.collapsible.extra-class")
		if panelDiv.Length() != 1 {
			t.Fatalf("expected 1 .panel.panel-default.collapsible.extra-class, got %d\nHTML: %s", panelDiv.Length(), html)
		}
		if v, _ := panelDiv.Attr("data-controller"); v != "panel" {
			t.Errorf("data-controller = %q, want panel", v)
		}
		if v, _ := panelDiv.Attr("data-panel-panel-value"); v != "panel-email" {
			t.Errorf("data-panel-panel-value = %q, want panel-email", v)
		}
		if doc.Find(".panel__heading .panel__title#panel-email").Length() != 1 {
			t.Errorf("expected .panel__heading .panel__title#panel-email")
		}
		if doc.Find(`a.panel__icon[href="#panel-email"]`).Length() != 1 {
			t.Errorf("expected link icon a.panel__icon[href='#panel-email']")
		}
		if help := strings.TrimSpace(doc.Find(".panel__help").Text()); help != "Primary user email" {
			t.Errorf("help text = %q, want 'Primary user email'", help)
		}
		inp := doc.Find(`input#id_Email[name="Email"]`)
		if inp.Length() != 1 {
			t.Errorf("expected input#id_Email[name='Email']")
		}
		if val, _ := inp.Attr("value"); val != "alice@example.com" {
			t.Errorf("input value = %q, want alice@example.com", val)
		}
	})

	t.Run("FieldPanelRendersValidationErrors", func(t *testing.T) {
		user := &FormTestUser{}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		af := newTestAdminForm(req, user, nil, admin.FieldPanel("Email"))
		af.WithData(url.Values{"Email": {""}}, nil, req)

		if forms.IsValid(req.Context(), af) {
			t.Fatalf("expected empty required Email field to be invalid")
		}

		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		errEl := doc.Find(".panel__errors li.panel__error")
		if errEl.Length() == 0 {
			t.Fatalf("expected .panel__errors li.panel__error to be rendered")
		}
	})

	t.Run("HiddenFieldPanelAndHiddenField", func(t *testing.T) {
		hp := admin.HiddenFieldPanel("Username", "hidden-cls")
		if hp.ClassName() != "hidden-cls" {
			t.Fatalf("HiddenFieldPanel ClassName = %q, want hidden-cls", hp.ClassName())
		}

		user := &FormTestUser{Username: "secret_user"}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		af := newTestAdminForm(req, user, nil, hp)
		f, _ := af.Field("Username")
		f.Hide(true)

		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(".panel__heading").Length() != 0 {
			t.Errorf("expected hidden field panel to omit .panel__heading")
		}
		if doc.Find(".panel__help").Length() != 0 {
			t.Errorf("expected hidden field panel to omit .panel__help")
		}
		if doc.Find(`input[type="hidden"][name="Username"]`).Length() != 1 {
			t.Errorf("expected hidden input for Username")
		}
	})

	t.Run("MissingFieldPanicsOnBind", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected FieldPanel.Bind on missing field to panic")
			}
		}()
		p := admin.FieldPanel("NonExistentField")
		p.Bind(nil, nil, nil, context.Background(), &FormTestUser{}, map[string]forms.BoundField{}, nil)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. TitlePanel
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_TitlePanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("SingleFieldWithAndWithoutOutputFields", func(t *testing.T) {
		tp := admin.TitlePanel(admin.FieldPanel("Email"), "title-cls")
		if got := tp.Fields(); len(got) != 1 || got[0] != "Email" {
			t.Fatalf("TitlePanel.Fields() = %v, want [Email]", got)
		}
		if tp.ClassName() != "title-cls" {
			t.Fatalf("TitlePanel.ClassName() = %q, want title-cls", tp.ClassName())
		}
		tp.Class("updated-title-cls")
		if tp.ClassName() != "updated-title-cls" {
			t.Fatalf("TitlePanel.Class() failed, got %q", tp.ClassName())
		}

		_, bf := bindSinglePanelToForm(t, &FormTestUser{Email: "t@example.com"}, tp)
		doc := parseHTMLDoc(t, bf.AsP())
		tpDiv := doc.Find(`.panel.panel-title.updated-title-cls[data-controller="titlepanel"]`)
		if tpDiv.Length() != 1 {
			t.Fatalf("expected .panel.panel-title.updated-title-cls[data-controller='titlepanel']")
		}
		if val, _ := tpDiv.Attr("data-titlepanel-outputids-value"); val != "[]" {
			t.Errorf("data-titlepanel-outputids-value = %q, want []", val)
		}

		// WithOutputFields
		tpWithOut := admin.TitlePanel(admin.FieldPanel("Title")).WithOutputFields("Code", "Description")
		coupon := &FormTestCoupon{Title: "Spring Sale", Code: "SPRING26"}
		_, bfOut := bindSinglePanelToForm(t, coupon, tpWithOut, admin.FieldPanel("Code"), admin.FieldPanel("Description"))
		docOut := parseHTMLDoc(t, bfOut.AsP())
		tpOutDiv := docOut.Find(`.panel.panel-title[data-controller="titlepanel"]`)
		if val, _ := tpOutDiv.Attr("data-titlepanel-outputids-value"); val != `["id_Code","id_Description"]` {
			t.Errorf("data-titlepanel-outputids-value = %q, want [\"id_Code\",\"id_Description\"]", val)
		}
	})

	t.Run("TitlePanelWrappingRowPanel", func(t *testing.T) {
		// Pattern from contrib/openauth2/openauth2.go
		tp := admin.TitlePanel(
			admin.RowPanel(
				admin.FieldPanel("FirstName"),
				admin.FieldPanel("LastName"),
			),
		)
		if got := tp.Fields(); len(got) != 2 || got[0] != "FirstName" || got[1] != "LastName" {
			t.Fatalf("Fields() = %v, want [FirstName LastName]", got)
		}
		_, bf := bindSinglePanelToForm(t, &FormTestUser{FirstName: "Ada", LastName: "Lovelace"}, tp)
		doc := parseHTMLDoc(t, bf.AsP())
		if doc.Find(`.panel.panel-title .panel.panel-row`).Length() != 1 {
			t.Errorf("expected .panel.panel-row inside .panel.panel-title")
		}
		if doc.Find(`input[name="FirstName"][value="Ada"]`).Length() != 1 {
			t.Errorf("expected FirstName input with value Ada")
		}
		if doc.Find(`input[name="LastName"][value="Lovelace"]`).Length() != 1 {
			t.Errorf("expected LastName input with value Lovelace")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. RowPanel & LabeledRowPanel
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_RowPanelAndLabeledRowPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("UnlabeledRowPanel", func(t *testing.T) {
		rp := admin.RowPanel(
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
		).Class("two-col")

		if got := rp.Fields(); len(got) != 2 || got[0] != "FirstName" || got[1] != "LastName" {
			t.Fatalf("RowPanel.Fields() = %v, want [FirstName LastName]", got)
		}
		mp, ok := rp.(admin.MultiPanel)
		if !ok || len(mp.Children()) != 2 {
			t.Fatalf("expected MultiPanel with 2 children")
		}

		_, bf := bindSinglePanelToForm(t, &FormTestUser{FirstName: "Grace", LastName: "Hopper"}, rp)
		doc := parseHTMLDoc(t, bf.AsP())

		rowDiv := doc.Find(".panel.panel-row.collapsible.two-col")
		if rowDiv.Length() != 1 {
			t.Fatalf("expected 1 .panel.panel-row.collapsible.two-col")
		}
		if val, _ := rowDiv.Attr("data-panel-panel-value"); val != "" {
			t.Errorf("expected empty data-panel-panel-value for unlabeled RowPanel, got %q", val)
		}
		// Only child field panels should have headings, not the row panel itself.
		if rowDiv.ChildrenFiltered(".panel__heading").Length() != 0 {
			t.Errorf("expected unlabeled RowPanel not to have direct .panel__heading")
		}
	})

	t.Run("LabeledRowPanelAndIncrementingIDs", func(t *testing.T) {
		rp1 := admin.LabeledRowPanel(
			trans.S("Full Name"),
			trans.S("Enter your given and family names"),
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
		)
		rp2 := admin.LabeledRowPanel(
			"Credentials",
			"Account credentials",
			admin.FieldPanel("Username"),
			admin.FieldPanel("Email"),
		)

		_, bf := bindSinglePanelToForm(t, &FormTestUser{}, rp1, rp2)
		doc := parseHTMLDoc(t, bf.AsP())

		rows := doc.Find(".panel.panel-row.collapsible")
		if rows.Length() != 2 {
			t.Fatalf("expected 2 row panels, got %d", rows.Length())
		}

		first := rows.Eq(0)
		if id, _ := first.Attr("data-panel-panel-value"); id != "panel-row" {
			t.Errorf("first row data-panel-panel-value = %q, want panel-row", id)
		}
		if lbl := strings.TrimSpace(first.ChildrenFiltered(".panel__heading").Find("label").Text()); lbl != "Full Name" {
			t.Errorf("first row label = %q, want Full Name", lbl)
		}
		if help := strings.TrimSpace(first.ChildrenFiltered(".panel__help").Text()); help != "Enter your given and family names" {
			t.Errorf("first row help = %q, want 'Enter your given and family names'", help)
		}

		second := rows.Eq(1)
		if id, _ := second.Attr("data-panel-panel-value"); id != "panel-row-1" {
			t.Errorf("second row data-panel-panel-value = %q, want panel-row-1", id)
		}
	})

	t.Run("LabeledRowPanelWithPanelOverloadArgs", func(t *testing.T) {
		rp := admin.LabeledRowPanel(
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
			admin.FieldPanel("Username"),
		)
		if got := rp.Fields(); len(got) != 3 || got[0] != "LastName" || got[1] != "FirstName" || got[2] != "Username" {
			t.Fatalf("expected Fields() to preserve prepended panel args [LastName FirstName Username], got %v", got)
		}
		_, bf := bindSinglePanelToForm(t, &FormTestUser{}, rp)
		doc := parseHTMLDoc(t, bf.AsP())
		if doc.Find(".panel.panel-row").ChildrenFiltered(".panel__heading").Length() != 0 {
			t.Errorf("expected no heading when label/helpText slots held Panels")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. PanelGroup & LabeledPanelGroup
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_PanelGroupAndLabeledPanelGroup(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("LabeledPanelGroupWithPanelClass", func(t *testing.T) {
		// Pattern from contrib/documents/admin.go
		pg1 := admin.PanelClass("collapsed", admin.LabeledPanelGroup(
			trans.S("File Metadata"),
			trans.S("Metadata about the record"),
			admin.FieldPanel("CreatedAt"),
			admin.FieldPanel("UpdatedAt"),
		))
		pg2 := admin.LabeledPanelGroup(
			"Flags",
			nil,
			admin.FieldPanel("IsAdministrator"),
			admin.FieldPanel("IsActive"),
		)

		if got := pg1.Fields(); len(got) != 2 || got[0] != "CreatedAt" || got[1] != "UpdatedAt" {
			t.Fatalf("Fields() = %v, want [CreatedAt UpdatedAt]", got)
		}
		if mp, ok := pg1.(admin.MultiPanel); !ok || len(mp.Children()) != 2 {
			t.Fatalf("expected MultiPanel with 2 children")
		}
		if pg1.ClassName() != "collapsed" {
			t.Fatalf("ClassName() = %q, want collapsed", pg1.ClassName())
		}

		_, bf := bindSinglePanelToForm(t, &FormTestUser{}, pg1, pg2)
		doc := parseHTMLDoc(t, bf.AsP())

		groups := doc.Find(".panel.panel-group.collapsible")
		if groups.Length() != 2 {
			t.Fatalf("expected 2 panel groups, got %d", groups.Length())
		}

		g1 := groups.Eq(0)
		if !g1.HasClass("collapsed") {
			t.Errorf("expected first panel group to have 'collapsed' class")
		}
		if id, _ := g1.Attr("data-panel-panel-value"); id != "panel-group" {
			t.Errorf("first group data-panel-panel-value = %q, want panel-group", id)
		}
		if lbl := strings.TrimSpace(g1.ChildrenFiltered(".panel__heading").Find("label").Text()); lbl != "File Metadata" {
			t.Errorf("first group label = %q, want File Metadata", lbl)
		}
		if help := strings.TrimSpace(g1.ChildrenFiltered(".panel__help").Text()); help != "Metadata about the record" {
			t.Errorf("first group help = %q, want 'Metadata about the record'", help)
		}

		g2 := groups.Eq(1)
		if id, _ := g2.Attr("data-panel-panel-value"); id != "panel-group-1" {
			t.Errorf("second group data-panel-panel-value = %q, want panel-group-1", id)
		}
	})

	t.Run("UnlabeledPanelGroupAndPanelOverloadArgs", func(t *testing.T) {
		unlabeled := admin.PanelGroup(admin.FieldPanel("Email"))
		overloaded := admin.LabeledPanelGroup(
			admin.FieldPanel("Username"),
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
		)
		if got := overloaded.Fields(); len(got) != 3 || got[0] != "FirstName" || got[1] != "Username" || got[2] != "LastName" {
			t.Fatalf("expected Fields [FirstName Username LastName], got %v", got)
		}

		_, bf := bindSinglePanelToForm(t, &FormTestUser{}, unlabeled, overloaded)
		doc := parseHTMLDoc(t, bf.AsP())
		groups := doc.Find(".panel.panel-group")
		if groups.Length() != 2 {
			t.Fatalf("expected 2 panel groups, got %d", groups.Length())
		}
		for i := 0; i < groups.Length(); i++ {
			if groups.Eq(i).ChildrenFiltered(".panel__heading").Length() != 0 {
				t.Errorf("group %d expected no direct .panel__heading", i)
			}
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. AlertPanel
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_AlertPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("AlertTypesAndTextFormats", func(t *testing.T) {
		defaultAlert := &admin.AlertPanel{
			Label: "Notice",
			HTML:  "Static notice body",
		}
		if len(defaultAlert.Fields()) != 0 {
			t.Errorf("expected AlertPanel.Fields() to be empty")
		}
		comp, err := defaultAlert.Comparison(context.Background(), &FormTestUser{}, &FormTestUser{})
		if comp != nil || err != nil {
			t.Errorf("expected nil comparison for AlertPanel, got %v, %v", comp, err)
		}
		if defaultAlert.GetType() != admin.ALERT_INFO {
			t.Errorf("default GetType() = %q, want %q", defaultAlert.GetType(), admin.ALERT_INFO)
		}
		defaultAlert.Class("alert-extra")
		if defaultAlert.ClassName() != "alert-extra" {
			t.Errorf("ClassName() = %q, want alert-extra", defaultAlert.ClassName())
		}

		cases := []struct {
			name      string
			panel     *admin.AlertPanel
			wantClass string
			wantTitle string
			wantBody  string
		}{
			{
				name:      "DefaultInfoString",
				panel:     defaultAlert,
				wantClass: "info",
				wantTitle: "Notice",
				wantBody:  "Static notice body",
			},
			{
				name: "SuccessFuncZeroArgs",
				panel: &admin.AlertPanel{
					Type:  admin.ALERT_SUCCESS,
					Label: func() string { return "Saved" },
					HTML:  func() string { return "<b>All good</b>" },
				},
				wantClass: "success",
				wantTitle: "Saved",
				wantBody:  "All good",
			},
			{
				name: "WarningTransS",
				panel: &admin.AlertPanel{
					Type:  admin.ALERT_WARNING,
					Label: trans.S("Heads up"),
					HTML:  trans.S("Check fields carefully"),
				},
				wantClass: "warning",
				wantTitle: "Heads up",
				wantBody:  "Check fields carefully",
			},
			{
				name: "DangerWithoutLabel",
				panel: &admin.AlertPanel{
					Type:  admin.ALERT_ERROR,
					Label: nil,
					HTML:  "Critical error occurred",
				},
				wantClass: "danger",
				wantTitle: "",
				wantBody:  "Critical error occurred",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				bp := tc.panel.Bind(req, nil, nil, req.Context(), nil, nil, nil)
				if bp.Hidden() {
					t.Errorf("expected BoundAlertPanel.Hidden() = false")
				}
				rendered := bp.Render()
				doc := parseHTMLDoc(t, rendered)
				alertBox := doc.Find(".panel.panel-alert .alert." + tc.wantClass)
				if alertBox.Length() != 1 {
					t.Fatalf("expected .panel.panel-alert .alert.%s\nHTML: %s", tc.wantClass, rendered)
				}
				titleSel := alertBox.Find(".alert-title")
				if tc.wantTitle == "" {
					if titleSel.Length() != 0 {
						t.Errorf("expected no .alert-title when Label is nil")
					}
				} else if got := strings.TrimSpace(titleSel.Text()); got != tc.wantTitle {
					t.Errorf("alert title = %q, want %q", got, tc.wantTitle)
				}
				if !strings.Contains(alertBox.Text(), tc.wantBody) {
					t.Errorf("expected alert body to contain %q, got %q", tc.wantBody, alertBox.Text())
				}
			})
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. JSONDetailPanel
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_JSONDetailPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("ValidJSONWithOrderingLabelsAndCustomWidgets", func(t *testing.T) {
		jp := admin.JSONDetailPanel{
			FieldName: "Data",
			Ordering: func(r *http.Request, fields map[string]forms.BoundField) []string {
				return []string{"id", "login"}
			},
			Labels: func(r *http.Request, fields map[string]forms.BoundField) map[string]any {
				return map[string]any{
					"id":    "User ID",
					"login": trans.S("Login Handle"),
				}
			},
			Widgets: func(r *http.Request, fields map[string]forms.BoundField) map[string]widgets.Widget {
				return map[string]widgets.Widget{
					"avatar": widgets.NewBaseWidget("url", "", nil),
				}
			},
		}

		if got := jp.Fields(); len(got) != 1 || got[0] != "Data" {
			t.Fatalf("Fields() = %v, want [Data]", got)
		}
		jpUpdated := jp.Class("json-custom").(admin.JSONDetailPanel)
		if jpUpdated.ClassName() != "json-custom" {
			t.Fatalf("ClassName() = %q, want json-custom", jpUpdated.ClassName())
		}
		if comp, err := jp.Comparison(context.Background(), &FormTestUser{}, &FormTestUser{}); comp != nil || err != nil {
			t.Errorf("expected nil comparison for JSONDetailPanel")
		}

		user := &FormTestUser{
			Data: `{"provider":"github","login":"nigel","id":42,"avatar":"https://example.com/a.png"}`,
		}
		_, bf := bindSinglePanelToForm(t, user, jpUpdated)
		if len(bf.BoundPanels) != 1 {
			t.Fatalf("expected 1 bound panel, got %d", len(bf.BoundPanels))
		}

		bjp, ok := bf.BoundPanels[0].(*admin.BoundJSONDetailPanel)
		if !ok {
			t.Fatalf("expected *admin.BoundJSONDetailPanel, got %T", bf.BoundPanels[0])
		}
		if bjp.Error != nil {
			t.Fatalf("unexpected error in BoundJSONDetailPanel: %v", bjp.Error)
		}
		if bjp.Hidden() {
			t.Errorf("expected Hidden() = false")
		}

		wantNames := []string{"Data__id", "Data__login", "Data__avatar", "Data__provider"}
		if len(bjp.BoundFields) != len(wantNames) {
			t.Fatalf("expected %d JSON bound fields, got %d", len(wantNames), len(bjp.BoundFields))
		}
		for i, want := range wantNames {
			if got := bjp.BoundFields[i].Name(); got != want {
				t.Errorf("BoundFields[%d].Name() = %q, want %q", i, got, want)
			}
		}

		doc := parseHTMLDoc(t, bjp.Render())
		if doc.Find(`.panel.panel-default.collapsible.json-custom`).Length() != 1 {
			t.Errorf("expected outer JSON detail panel with json-custom class")
		}
		for _, key := range []string{"id", "login", "avatar", "provider"} {
			inp := doc.Find(fmt.Sprintf(`input[data-field="%s"][readonly]`, key))
			if inp.Length() != 1 {
				t.Errorf("expected readonly input with data-field=%q", key)
			}
		}
		avatarInp := doc.Find(`input[data-field="avatar"]`)
		if typ, _ := avatarInp.Attr("type"); typ != "url" {
			t.Errorf("expected custom URL widget (type='url') for avatar, got %q", typ)
		}
		htmlText := doc.Text()
		if !strings.Contains(htmlText, "User ID") || !strings.Contains(htmlText, "Login Handle") {
			t.Errorf("expected custom labels 'User ID' and 'Login Handle' in HTML: %s", htmlText)
		}
	})

	t.Run("InvalidJSONReturnsErrorAndHiddenClass", func(t *testing.T) {
		jp := admin.JSONDetailPanel{FieldName: "Data"}.Class("hidden").(admin.JSONDetailPanel)
		user := &FormTestUser{Data: `{invalid json`}
		_, bf := bindSinglePanelToForm(t, user, jp)

		bjp, ok := bf.BoundPanels[0].(*admin.BoundJSONDetailPanel)
		if !ok {
			t.Fatalf("expected *admin.BoundJSONDetailPanel, got %T", bf.BoundPanels[0])
		}
		if bjp.Error == nil {
			t.Errorf("expected JSON unmarshal error in BoundJSONDetailPanel")
		}
		var buf bytes.Buffer
		if err := bjp.Component().Render(context.Background(), &buf); err == nil {
			t.Errorf("expected Render to return unmarshal error")
		}

		// Valid JSON with "hidden" class checks Hidden() == true
		validJP := admin.JSONDetailPanel{FieldName: "Data"}.Class("hidden")
		_, bfValid := bindSinglePanelToForm(t, &FormTestUser{Data: `{"k":"v"}`}, validJP)
		if !bfValid.BoundPanels[0].Hidden() {
			t.Errorf("expected Hidden() == true when class is 'hidden'")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. TabbedPanel & PanelTab
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_TabbedPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("MultiTabWithNestedPanelsAndEmptyTabSkipping", func(t *testing.T) {
		tp := admin.TabbedPanel(
			admin.PanelTab(
				trans.S("General"),
				admin.TitlePanel(admin.FieldPanel("Email")),
				admin.RowPanel(
					admin.FieldPanel("FirstName"),
					admin.FieldPanel("LastName"),
				),
			),
			admin.PanelTab(
				trans.S("EmptyTab"),
			),
			admin.PanelTab(
				trans.S("Details"),
				&admin.AlertPanel{
					Type:  admin.ALERT_INFO,
					Label: trans.S("User Data"),
					HTML:  trans.S("Read-only metadata."),
				},
				admin.LabeledPanelGroup(
					trans.S("JSON Metadata"),
					nil,
					&admin.JSONDetailPanel{FieldName: "Data"},
				),
			),
		).Class("tabs-custom")

		if tp.ClassName() != "tabs-custom" {
			t.Fatalf("ClassName() = %q, want tabs-custom", tp.ClassName())
		}
		wantFields := []string{"Email", "FirstName", "LastName", "Data"}
		if got := tp.Fields(); len(got) != len(wantFields) {
			t.Fatalf("Fields() = %v, want %v", got, wantFields)
		} else {
			for i := range wantFields {
				if got[i] != wantFields[i] {
					t.Errorf("Fields()[%d] = %q, want %q", i, got[i], wantFields[i])
				}
			}
		}
		if mp, ok := tp.(admin.MultiPanel); !ok || len(mp.Children()) != 4 {
			t.Fatalf("expected MultiPanel with 4 children across tabs, got %d", len(mp.Children()))
		}

		user := &FormTestUser{
			Email:     "tab@example.com",
			FirstName: "Tab",
			LastName:  "User",
			Data:      `{"role":"admin"}`,
		}
		_, bf := bindSinglePanelToForm(t, user, tp)
		if bf.BoundPanels[0].Hidden() {
			t.Errorf("expected non-empty TabbedPanel Hidden() = false")
		}

		doc := parseHTMLDoc(t, bf.AsP())
		if doc.Find(`.panel.tabpanel[data-controller="tabpanel"]`).Length() != 1 {
			t.Fatalf("expected .panel.tabpanel[data-controller='tabpanel']")
		}
		controls := doc.Find(`.tabpanel__controls button.tabpanel__control`)
		if controls.Length() != 2 {
			t.Fatalf("expected 2 non-empty tab controls (empty tab skipped), got %d", controls.Length())
		}
		if got := strings.TrimSpace(controls.Eq(0).Text()); got != "General" {
			t.Errorf("tab 0 title = %q, want General", got)
		}
		if got := strings.TrimSpace(controls.Eq(1).Text()); got != "Details" {
			t.Errorf("tab 1 title = %q, want Details", got)
		}
		tabs := doc.Find(`.tabpanel__content .tabpanel__panel[data-tabpanel-target="tab"]`)
		if tabs.Length() != 2 {
			t.Fatalf("expected 2 tab panels, got %d", tabs.Length())
		}
	})

	t.Run("EmptyTabbedPanelAndPanelTabOverload", func(t *testing.T) {
		empty := admin.TabbedPanel()
		_, bfEmpty := bindSinglePanelToForm(t, &FormTestUser{}, empty)
		if !bfEmpty.BoundPanels[0].Hidden() {
			t.Errorf("expected empty TabbedPanel to report Hidden() = true")
		}

		overloaded := admin.TabbedPanel(
			admin.PanelTab(
				admin.FieldPanel("Email"),
				admin.FieldPanel("Username"),
			),
		)
		if got := overloaded.Fields(); len(got) != 2 || got[0] != "Email" || got[1] != "Username" {
			t.Fatalf("PanelTab(Panel, ...) Fields() = %v, want [Email Username]", got)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. Auth User AddView & EditView Panels + FormInit (contrib/auth/app.go)
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_AuthUserPanelsAndFormInit(t *testing.T) {
	setupFormsTestDB(t)

	addPanels := []admin.Panel{
		admin.TitlePanel(admin.FieldPanel("Email")),
		admin.FieldPanel("Username"),
		admin.RowPanel(
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
		),
		admin.FieldPanel("Password"),
		admin.FieldPanel("IsAdministrator"),
		admin.FieldPanel("IsActive"),
	}

	t.Run("AddViewSaveAndHasChanged", func(t *testing.T) {
		user := &FormTestUser{}
		req := httptest.NewRequest(http.MethodPost, "/admin/auth/user/add", nil)
		af := newTestAdminForm(req, user, []string{"ID", "CreatedAt", "UpdatedAt"}, addPanels...)

		if _, ok := af.Field("Data"); ok {
			t.Errorf("expected unreferenced field Data to be removed by Load()")
		}

		postData := url.Values{
			"Email":           {"jane@example.com"},
			"Username":        {"janedoe"},
			"FirstName":       {"Jane"},
			"LastName":        {"Doe"},
			"Password":        {"StrongPass1!"},
			"IsAdministrator": {"on"},
			"IsActive":        {"on"},
		}
		af.WithData(postData, nil, req)

		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected AddView form valid, errors: %v / %v", af.ErrorList(), af.BoundErrors())
		}
		if !af.HasChanged() {
			t.Errorf("expected HasChanged() = true for newly populated AddView form")
		}

		if _, err := af.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		row, err := queries.GetQuerySet(&FormTestUser{}).Filter("Username", "janedoe").Get()
		if err != nil {
			t.Fatalf("failed to load saved user: %v", err)
		}
		if row.Object.Email != "jane@example.com" || !row.Object.IsAdministrator || !row.Object.IsActive {
			t.Errorf("unexpected saved user values: %+v", row.Object)
		}
	})

	t.Run("EditViewFormInitPasswordConfirmValidator", func(t *testing.T) {
		savedUser := &FormTestUser{
			Email:     "edit@example.com",
			Username:  "edituser",
			FirstName: "Edit",
			LastName:  "User",
			Password:  "OldPass123!",
			IsActive:  true,
		}
		if _, err := queries.GetQuerySet(&FormTestUser{}).Create(savedUser); err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}

		editPanels := []admin.Panel{
			admin.TitlePanel(admin.FieldPanel("Email")),
			admin.FieldPanel("Username"),
			admin.RowPanel(
				admin.FieldPanel("FirstName"),
				admin.FieldPanel("LastName"),
			),
			admin.FieldPanel("Password"),
			admin.FieldPanel("PasswordConfirm"),
			admin.FieldPanel("IsAdministrator"),
			admin.FieldPanel("IsActive"),
		}

		initAuthEdit := func(instance *FormTestUser, form modelforms.ModelForm[*FormTestUser]) {
			form.Ordering([]string{
				"Email", "Username", "FirstName", "LastName",
				"IsAdministrator", "IsActive", "Password", "PasswordConfirm",
			})
			form.AddField("PasswordConfirm", fields.CharField(
				fields.Label(trans.S("Password Confirm")),
				fields.Required(false),
			))
			form.SetValidators(func(f forms.Form, cleaned map[string]any) []error {
				p1, _ := cleaned["Password"].(string)
				p2, _ := cleaned["PasswordConfirm"].(string)
				if p2 == "" {
					if p1 == instance.Password {
						delete(cleaned, "PasswordConfirm")
						return nil
					}
					return []error{errors.New("password confirmation is required when changing password")}
				}
				if p1 != p2 {
					return []error{errors.New("passwords do not match")}
				}
				return nil
			})
		}

		// 1. Mismatched Password & PasswordConfirm fails validation.
		reqBad := httptest.NewRequest(http.MethodPost, "/admin/auth/user/edit", nil)
		badModelForm := modelforms.NewBaseModelForm(reqBad.Context(), savedUser)
		badModelForm.SetExclude("ID", "CreatedAt", "UpdatedAt")
		initAuthEdit(savedUser, badModelForm)
		afBad := admin.NewAdminForm[modelforms.ModelForm[*FormTestUser]](reqBad, badModelForm, editPanels...)
		afBad.Load()
		afBad.WithData(url.Values{
			"Email":           {"edit@example.com"},
			"Username":        {"edituser"},
			"FirstName":       {"Edit"},
			"LastName":        {"User"},
			"Password":        {"NewPass1!"},
			"PasswordConfirm": {"DifferentPass2!"},
			"IsActive":        {"on"},
		}, nil, reqBad)

		if forms.IsValid(reqBad.Context(), afBad) {
			t.Fatalf("expected EditView with mismatched PasswordConfirm to be invalid")
		}
		if len(afBad.ErrorList()) == 0 || !strings.Contains(afBad.ErrorList()[0].Error(), "passwords do not match") {
			t.Errorf("expected mismatch error in ErrorList, got %v", afBad.ErrorList())
		}

		// 2. Unchanged data returns HasChanged() == false.
		reqSame := httptest.NewRequest(http.MethodPost, "/admin/auth/user/edit", nil)
		sameModelForm := modelforms.NewBaseModelForm(reqSame.Context(), savedUser)
		sameModelForm.SetExclude("ID", "CreatedAt", "UpdatedAt")
		initAuthEdit(savedUser, sameModelForm)
		afSame := admin.NewAdminForm[modelforms.ModelForm[*FormTestUser]](reqSame, sameModelForm, editPanels...)
		afSame.Load()
		afSame.WithData(url.Values{
			"Email":           {"edit@example.com"},
			"Username":        {"edituser"},
			"FirstName":       {"Edit"},
			"LastName":        {"User"},
			"Password":        {"OldPass123!"},
			"PasswordConfirm": {""},
			"IsActive":        {"on"},
		}, nil, reqSame)
		if !forms.IsValid(reqSame.Context(), afSame) {
			t.Fatalf("expected unchanged EditView valid, errors: %v / %v", afSame.ErrorList(), afSame.BoundErrors())
		}
		if afSame.HasChanged() {
			t.Errorf("expected HasChanged() = false when submitted values match existing user")
		}

		// 3. Matching changed password succeeds and updates DB.
		reqGood := httptest.NewRequest(http.MethodPost, "/admin/auth/user/edit", nil)
		goodModelForm := modelforms.NewBaseModelForm(reqGood.Context(), savedUser)
		goodModelForm.SetExclude("ID", "CreatedAt", "UpdatedAt")
		initAuthEdit(savedUser, goodModelForm)
		afGood := admin.NewAdminForm[modelforms.ModelForm[*FormTestUser]](reqGood, goodModelForm, editPanels...)
		afGood.Load()
		afGood.WithData(url.Values{
			"Email":           {"edit@example.com"},
			"Username":        {"edituser"},
			"FirstName":       {"Edited"},
			"LastName":        {"User"},
			"Password":        {"NewPass1!"},
			"PasswordConfirm": {"NewPass1!"},
			"IsActive":        {"on"},
		}, nil, reqGood)
		if !forms.IsValid(reqGood.Context(), afGood) {
			t.Fatalf("expected EditView valid, errors: %v / %v", afGood.ErrorList(), afGood.BoundErrors())
		}
		if !afGood.HasChanged() {
			t.Errorf("expected HasChanged() = true when FirstName and Password changed")
		}
		if _, err := afGood.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		updated, err := queries.GetQuerySet(&FormTestUser{}).Filter("ID", savedUser.ID).Get()
		if err != nil {
			t.Fatalf("failed to query updated user: %v", err)
		}
		if updated.Object.FirstName != "Edited" || updated.Object.Password != "NewPass1!" {
			t.Errorf("unexpected updated user: %+v", updated.Object)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 10. Shop Coupon TabbedPanel + ModelFormPanel (contrib/shop/app.go)
// ─────────────────────────────────────────────────────────────────────────────

func buildShopCouponPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Coupon"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Description"),
				admin.FieldPanel("Code"),
				admin.RowPanel(
					admin.FieldPanel("MaxDiscount"),
					admin.FieldPanel("IsPercentage"),
				),
			),
			admin.PanelTab(
				trans.S("Constraints"),
				admin.FieldPanel("ValidFrom"),
				admin.FieldPanel("ValidUntil"),
				admin.FieldPanel("CreatedAt"),
			),
			admin.PanelTab(
				trans.S("Items"),
				&admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]]{
					TargetType: &FormTestCouponItem{},
					FieldName:  "Items",
					Classname:  "collapsible",
					MinNum:     minNum,
					MaxNum:     maxNum,
					Extra:      extra,
					Panels: []admin.Panel{
						admin.FieldPanel("ID"),
						admin.FieldPanel("ProductName"),
						admin.FieldPanel("Quantity"),
					},
				},
			),
		),
	}
}

func TestAdminForm_ShopCouponTabbedModelFormPanel(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("LoadFormSetUnwrapAndHTMLRendering", func(t *testing.T) {
		coupon := &FormTestCoupon{}
		req := httptest.NewRequest(http.MethodGet, "/admin/shop/coupons/add", nil)
		panels := buildShopCouponPanels(1, 0, 0)
		af := newTestAdminForm(req, coupon, nil, panels...)

		fs := af.FormSet()
		if fs == nil {
			t.Fatalf("expected non-nil FormSet() when TabbedPanel contains ModelFormPanel")
		}
		unwrapped := af.Unwrap()
		if len(unwrapped) != 2 || unwrapped[0] != af || unwrapped[1] != fs {
			t.Fatalf("expected Unwrap() to return [adminForm, formset], got %v", unwrapped)
		}

		// SetPrefix propagates to FormSet()
		af.SetPrefix("cp")
		if fs.Prefix() != "cp" {
			t.Errorf("expected FormSet prefix 'cp', got %q", fs.Prefix())
		}
		af.SetPrefix("")

		// Render BoundForm HTML
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`.modelform-panel.collapsible`).Length() != 1 {
			t.Errorf("expected .modelform-panel.collapsible in rendered HTML")
		}
		if doc.Find(`template.modelform-panel__empty-form`).Length() != 1 {
			t.Errorf("expected template.modelform-panel__empty-form in rendered HTML")
		}
		if doc.Find(`.modelform-panel__management`).Length() != 1 {
			t.Errorf("expected .modelform-panel__management in rendered HTML")
		}
		if doc.Find(`.modelform-panel__forms .modelform-panel__forms__panel`).Length() != 1 {
			t.Errorf("expected 1 rendered inline form panel (MinNum=1)")
		}
		if doc.Find(`.modelform-panel__form__controls--add`).Length() == 0 {
			t.Errorf("expected add controls in rendered inline form panel")
		}
		if doc.Find(`.modelform-panel__form__controls--remove`).Length() == 0 {
			t.Errorf("expected remove control in rendered inline form panel")
		}
	})

	t.Run("AddCouponAndCreateInlineItemsOnEdit", func(t *testing.T) {
		coupon := &FormTestCoupon{}
		reqAdd := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/add", nil)
		afAdd := newTestAdminForm(reqAdd, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		afAdd.WithData(url.Values{
			"Title":                          {"Summer Discount"},
			"Description":                    {"20 percent off"},
			"Code":                           {"SUMMER20"},
			"MaxDiscount":                    {"20"},
			"IsPercentage":                   {"on"},
			"ValidFrom":                      {"2026-06-01"},
			"ValidUntil":                     {"2026-09-01"},
			"Items-management-TOTAL_FORMS":   {"0"},
			"Items-management-INITIAL_FORMS": {"0"},
			"Items-management-MIN_NUM_FORMS": {"0"},
			"Items-management-MAX_NUM_FORMS": {"10"},
		}, nil, reqAdd)

		if !forms.IsValid(reqAdd.Context(), afAdd) {
			t.Fatalf("expected Coupon AddView valid, errs=%v bound=%v fsErrs=%v",
				afAdd.ErrorList(), afAdd.BoundErrors(), afAdd.FormSet().ErrorList())
		}
		if _, err := afAdd.Save(); err != nil {
			t.Fatalf("AddView Save() failed: %v", err)
		}
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}

		// Now add 2 new inline FormTestCouponItem rows on EditView for the saved coupon.
		reqEdit := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afEdit := newTestAdminForm(reqEdit, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		afEdit.WithData(url.Values{
			"Title":                          {"Summer Discount"},
			"Description":                    {"20 percent off"},
			"Code":                           {"SUMMER20"},
			"MaxDiscount":                    {"20"},
			"IsPercentage":                   {"on"},
			"ValidFrom":                      {"2026-06-01"},
			"ValidUntil":                     {"2026-09-01"},
			"Items-management-TOTAL_FORMS":   {"2"},
			"Items-management-INITIAL_FORMS": {"0"},
			"Items-management-MIN_NUM_FORMS": {"0"},
			"Items-management-MAX_NUM_FORMS": {"10"},
			"Items-0-ProductName":            {"Keyboard"},
			"Items-0-Quantity":               {"2"},
			"Items-0-Coupon":                 {fmt.Sprint(coupon.ID)},
			"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Items-1-ProductName":                     {"Mouse"},
			"Items-1-Quantity":                        {"3"},
			"Items-1-Coupon":                          {fmt.Sprint(coupon.ID)},
			"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqEdit)

		if !forms.IsValid(reqEdit.Context(), afEdit) {
			t.Fatalf("expected Coupon EditView with 2 inline items valid, errs=%v bound=%v fsErrs=%v",
				afEdit.ErrorList(), afEdit.BoundErrors(), afEdit.FormSet().ErrorList())
		}
		if _, err := afEdit.Save(); err != nil {
			t.Fatalf("EditView Save() failed: %v", err)
		}
		if len(afEdit.ErrorList()) > 0 {
			t.Fatalf("unexpected errors after EditView Save(): %v", afEdit.ErrorList())
		}

		items, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query saved coupon items: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 saved coupon items, got %d", len(items))
		}
		if items[0].Object.ProductName != "Keyboard" || items[0].Object.Quantity != 2 {
			t.Errorf("unexpected item[0]: %+v", items[0].Object)
		}
		if items[1].Object.ProductName != "Mouse" || items[1].Object.Quantity != 3 {
			t.Errorf("unexpected item[1]: %+v", items[1].Object)
		}
		if items[0].Object.Coupon == nil || items[0].Object.Coupon.ID != coupon.ID {
			t.Errorf("expected item[0].Coupon.ID == %d, got %+v", coupon.ID, items[0].Object.Coupon)
		}
	})

	t.Run("AddCouponWithInlineItemsInSinglePost", func(t *testing.T) {
		coupon := &FormTestCoupon{}
		reqAdd := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/add", nil)
		afAdd := newTestAdminForm(reqAdd, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		afAdd.WithData(url.Values{
			"Title":                                   {"Single Post Coupon"},
			"Description":                             {"Parent and inline items together"},
			"Code":                                    {"SINGLE10"},
			"MaxDiscount":                             {"10"},
			"IsPercentage":                            {"on"},
			"ValidFrom":                               {"2026-07-01"},
			"ValidUntil":                              {"2026-08-01"},
			"Items-management-TOTAL_FORMS":            {"2"},
			"Items-management-INITIAL_FORMS":          {"0"},
			"Items-management-MIN_NUM_FORMS":          {"0"},
			"Items-management-MAX_NUM_FORMS":          {"10"},
			"Items-0-ProductName":                     {"Headphones"},
			"Items-0-Quantity":                        {"1"},
			"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Items-1-ProductName":                     {"USB-C Hub"},
			"Items-1-Quantity":                        {"2"},
			"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqAdd)

		if !forms.IsValid(reqAdd.Context(), afAdd) {
			t.Fatalf("expected Coupon AddView with inline items valid, errs=%v bound=%v fsErrs=%v",
				afAdd.ErrorList(), afAdd.BoundErrors(), afAdd.FormSet().ErrorList())
		}
		if _, err := afAdd.Save(); err != nil {
			t.Fatalf("AddView Save() with inline items failed: %v", err)
		}
		if len(afAdd.ErrorList()) > 0 {
			t.Fatalf("unexpected errors after AddView Save(): %v", afAdd.ErrorList())
		}
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}

		items, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query saved coupon items: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 saved coupon items linked to coupon %d, got %d", coupon.ID, len(items))
		}
		if items[0].Object.ProductName != "Headphones" || items[0].Object.Quantity != 1 {
			t.Errorf("unexpected item[0]: %+v", items[0].Object)
		}
		if items[1].Object.ProductName != "USB-C Hub" || items[1].Object.Quantity != 2 {
			t.Errorf("unexpected item[1]: %+v", items[1].Object)
		}
		if items[0].Object.Coupon == nil || items[0].Object.Coupon.ID != coupon.ID {
			t.Errorf("expected item[0].Coupon.ID == %d, got %+v", coupon.ID, items[0].Object.Coupon)
		}
		if items[1].Object.Coupon == nil || items[1].Object.Coupon.ID != coupon.ID {
			t.Errorf("expected item[1].Coupon.ID == %d, got %+v", coupon.ID, items[1].Object.Coupon)
		}
	})

	t.Run("EditFlowHasChangedUpdateAndDelete", func(t *testing.T) {
		coupon := &FormTestCoupon{
			Title:        "Winter Sale",
			Description:  "Cold weather deal",
			Code:         "WINTER15",
			MaxDiscount:  15,
			IsPercentage: false,
			ValidFrom:    "2026-11-01",
			ValidUntil:   "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("failed to create coupon: %v", err)
		}
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Monitor", Quantity: 1}
		item2 := &FormTestCouponItem{Coupon: coupon, ProductName: "Webcam", Quantity: 4}
		if _, err := queries.GetQuerySet(&FormTestCouponItem{}).Create(item1); err != nil {
			t.Fatalf("failed to create item1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestCouponItem{}).Create(item2); err != nil {
			t.Fatalf("failed to create item2: %v", err)
		}

		// 1. Identical parent and inline data -> HasChanged() == false
		reqUnchanged := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afUnchanged := newTestAdminForm(reqUnchanged, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		unchangedData := url.Values{
			"Title":                          {coupon.Title},
			"Description":                    {coupon.Description},
			"Code":                           {coupon.Code},
			"MaxDiscount":                    {"15"},
			"ValidFrom":                      {coupon.ValidFrom},
			"ValidUntil":                     {coupon.ValidUntil},
			"Items-management-TOTAL_FORMS":   {"2"},
			"Items-management-INITIAL_FORMS": {"2"},
			"Items-management-MIN_NUM_FORMS": {"0"},
			"Items-management-MAX_NUM_FORMS": {"10"},
			"Items-0-ID":                     {fmt.Sprint(item1.ID)},
			"Items-0-ProductName":            {"Monitor"},
			"Items-0-Quantity":               {"1"},
			"Items-0-Coupon":                 {fmt.Sprint(coupon.ID)},
			"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Items-1-ID":          {fmt.Sprint(item2.ID)},
			"Items-1-ProductName": {"Webcam"},
			"Items-1-Quantity":    {"4"},
			"Items-1-Coupon":      {fmt.Sprint(coupon.ID)},
			"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}
		afUnchanged.WithData(unchangedData, nil, reqUnchanged)
		if !forms.IsValid(reqUnchanged.Context(), afUnchanged) {
			t.Fatalf("expected unchanged edit form valid, errs=%v bound=%v fs=%v",
				afUnchanged.ErrorList(), afUnchanged.BoundErrors(), afUnchanged.FormSet().BoundErrors())
		}
		if afUnchanged.HasChanged() {
			t.Errorf("expected HasChanged() = false for identical parent and inline data")
		}

		// 2. Modifying only an inline item's ProductName -> HasChanged() == true and updates DB
		reqUpdate := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afUpdate := newTestAdminForm(reqUpdate, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		updateData := url.Values{
			"Title":                          {coupon.Title},
			"Description":                    {coupon.Description},
			"Code":                           {coupon.Code},
			"MaxDiscount":                    {"15"},
			"ValidFrom":                      {coupon.ValidFrom},
			"ValidUntil":                     {coupon.ValidUntil},
			"Items-management-TOTAL_FORMS":   {"2"},
			"Items-management-INITIAL_FORMS": {"2"},
			"Items-management-MIN_NUM_FORMS": {"0"},
			"Items-management-MAX_NUM_FORMS": {"10"},
			"Items-0-ID":                     {fmt.Sprint(item1.ID)},
			"Items-0-ProductName":            {"4K Monitor"},
			"Items-0-Quantity":               {"1"},
			"Items-0-Coupon":                 {fmt.Sprint(coupon.ID)},
			"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Items-1-ID":          {fmt.Sprint(item2.ID)},
			"Items-1-ProductName": {"Webcam"},
			"Items-1-Quantity":    {"4"},
			"Items-1-Coupon":      {fmt.Sprint(coupon.ID)},
			"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}
		afUpdate.WithData(updateData, nil, reqUpdate)
		if !forms.IsValid(reqUpdate.Context(), afUpdate) {
			t.Fatalf("expected updated inline form valid, errs=%v bound=%v", afUpdate.ErrorList(), afUpdate.BoundErrors())
		}
		if !afUpdate.HasChanged() {
			t.Errorf("expected HasChanged() = true when only inline ProductName changed")
		}
		if _, err := afUpdate.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		updatedItem1, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("ID", item1.ID).Get()
		if err != nil {
			t.Fatalf("failed to fetch updated item1: %v", err)
		}
		if updatedItem1.Object.ProductName != "4K Monitor" {
			t.Errorf("expected ProductName '4K Monitor', got %q", updatedItem1.Object.ProductName)
		}

		// 3. Deleting an inline item via DELETION_FIELD_NAME and via SaveForms(__DELETED__)
		reqDelete := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afDelete := newTestAdminForm(reqDelete, coupon, nil, buildShopCouponPanels(0, 10, 0)...)
		deleteData := url.Values{
			"Title":                          {coupon.Title},
			"Description":                    {coupon.Description},
			"Code":                           {coupon.Code},
			"MaxDiscount":                    {"15"},
			"ValidFrom":                      {coupon.ValidFrom},
			"ValidUntil":                     {coupon.ValidUntil},
			"Items-management-TOTAL_FORMS":   {"2"},
			"Items-management-INITIAL_FORMS": {"2"},
			"Items-management-MIN_NUM_FORMS": {"0"},
			"Items-management-MAX_NUM_FORMS": {"10"},
			"Items-0-ID":                     {fmt.Sprint(item1.ID)},
			"Items-0-ProductName":            {"4K Monitor"},
			"Items-0-Quantity":               {"1"},
			"Items-0-Coupon":                 {fmt.Sprint(coupon.ID)},
			"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Items-1-ID":          {fmt.Sprint(item2.ID)},
			"Items-1-ProductName": {"Webcam"},
			"Items-1-Quantity":    {"4"},
			"Items-1-Coupon":      {fmt.Sprint(coupon.ID)},
			"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Items-1-" + formsets.DELETION_FIELD_NAME: {"on"},
		}
		afDelete.WithData(deleteData, nil, reqDelete)
		if !forms.IsValid(reqDelete.Context(), afDelete) {
			t.Fatalf("expected delete inline form valid, errs=%v bound=%v", afDelete.ErrorList(), afDelete.BoundErrors())
		}
		if !afDelete.HasChanged() {
			t.Errorf("expected HasChanged() = true when inline item marked for deletion")
		}
		if _, err := afDelete.Save(); err != nil {
			t.Fatalf("Save() failed on delete: %v", err)
		}

		remaining, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).All()
		if err != nil {
			t.Fatalf("failed to query remaining items: %v", err)
		}
		if len(remaining) != 1 || remaining[0].Object.ID != item1.ID {
			t.Errorf("expected only item1 (%d) to remain after formset delete, got %d items", item1.ID, len(remaining))
		}

		// Also verify AdminForm.SaveForms handles __DELETED__ == "true"
		delStub := &deleteMarkerForm{
			BaseForm: forms.NewBaseForm(context.Background()),
			instance: item1,
		}
		delStub.BindCleanedData(nil, nil, map[string]any{"__DELETED__": "true"})
		if err := afDelete.SaveForms(delStub); err != nil {
			t.Fatalf("SaveForms(delStub) returned error: %v", err)
		}
		if len(afDelete.ErrorList()) > 0 {
			t.Fatalf("unexpected errors after SaveForms delete: %v", afDelete.ErrorList())
		}
		countAfter, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).Count()
		if err != nil {
			t.Fatalf("failed to count remaining items: %v", err)
		}
		if countAfter != 0 {
			t.Errorf("expected 0 items after __DELETED__ SaveForms, got %d", countAfter)
		}
	})

	t.Run("ModelPanelManyToOneClampingAndClusterOrdering", func(t *testing.T) {
		cat := &FormTestCategory{Name: "Electronics"}
		if _, err := queries.GetQuerySet(&FormTestCategory{}).Create(cat); err != nil {
			t.Fatalf("failed to create category: %v", err)
		}

		mPanel := admin.ModelPanel("Category", &FormTestCategory{}, func(p *admin.ModelFormPanel[*FormTestCategory, modelforms.ModelForm[*FormTestCategory]]) {
			p.MinNum = 3
			p.MaxNum = 5
			p.Extra = 2
			p.Panels = []admin.Panel{
				admin.FieldPanel("Name"),
			}
		})

		// Unsaved parent -> FORM_ORDERING_PRE (target FK column lives on parent)
		newCoupon := &FormTestCoupon{Category: cat}
		_, formsNew, err := mPanel.Forms(httptest.NewRequest(http.MethodGet, "/", nil), context.Background(), newCoupon)
		if err != nil {
			t.Fatalf("mPanel.Forms(new) failed: %v", err)
		}
		if len(formsNew) != 1 {
			t.Fatalf("expected 1 form wrapper, got %d", len(formsNew))
		}
		if orderable, ok := formsNew[0].(admin.ClusterOrderableForm); !ok || orderable.FormOrder() != admin.FORM_ORDERING_PRE {
			t.Errorf("expected FORM_ORDERING_PRE for unsaved parent, got %v", orderable.FormOrder())
		}

		// Saved parent -> FORM_ORDERING_PRE and Bind clamps MinNum=1, MaxNum=1, Extra=0
		savedCoupon := &FormTestCoupon{Title: "Cat Coupon", Code: "CAT123", Category: cat}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(savedCoupon); err != nil {
			t.Fatalf("failed to create coupon: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		af := newTestAdminForm(req, savedCoupon, nil,
			admin.FieldPanel("Title"),
			admin.FieldPanel("Code"),
			mPanel,
		)
		_, formsSaved, err := mPanel.Forms(req, req.Context(), savedCoupon)
		if err != nil {
			t.Fatalf("mPanel.Forms(saved) failed: %v", err)
		}
		if orderable, ok := formsSaved[0].(admin.ClusterOrderableForm); !ok || orderable.FormOrder() != admin.FORM_ORDERING_PRE {
			t.Errorf("expected FORM_ORDERING_PRE for saved parent, got %v", orderable.FormOrder())
		}

		// Trigger Bind to verify clamping on ManyToOne
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if mPanel.MinNum != 1 || mPanel.MaxNum != 1 || mPanel.Extra != 0 {
			t.Errorf("expected ManyToOne clamping MinNum=1 MaxNum=1 Extra=0, got Min=%d Max=%d Extra=%d",
				mPanel.MinNum, mPanel.MaxNum, mPanel.Extra)
		}
		if doc.Find(`input[name="Category-0-Name"][value="Electronics"]`).Length() != 1 {
			t.Errorf("expected bound Category-0-Name input with value Electronics")
		}
	})
}

type deleteMarkerForm struct {
	*forms.BaseForm
	instance attrs.Definer
}

func (d *deleteMarkerForm) Instance() attrs.Definer {
	return d.instance
}

// ─────────────────────────────────────────────────────────────────────────────
// 11. ValidatorPanels (top-level and nested)
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_ValidatorPanels(t *testing.T) {
	setupFormsTestDB(t)

	rejectCode := func(badVal string, msg string) func(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
		return func(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
			if v, _ := data["Code"].(string); v == badVal {
				return []error{errors.New(msg)}
			}
			return nil
		}
	}

	cases := []struct {
		name      string
		build     func() []admin.Panel
		wantError string
	}{
		{
			name: "TopLevelValidatorPanel",
			build: func() []admin.Panel {
				return []admin.Panel{
					admin.FieldPanel("Title"),
					&customValidatorPanel{
						Panel:      admin.FieldPanel("Code"),
						validateFn: rejectCode("BAD1", "top-level validator failed"),
					},
				}
			},
			wantError: "top-level validator failed",
		},
		{
			name: "NestedInTitlePanel",
			build: func() []admin.Panel {
				return []admin.Panel{
					admin.FieldPanel("Title"),
					admin.TitlePanel(&customValidatorPanel{
						Panel:      admin.FieldPanel("Code"),
						validateFn: rejectCode("BAD1", "title validator failed"),
					}),
				}
			},
			wantError: "title validator failed",
		},
		{
			name: "NestedInRowPanel",
			build: func() []admin.Panel {
				return []admin.Panel{
					admin.RowPanel(
						admin.FieldPanel("Title"),
						&customValidatorPanel{
							Panel:      admin.FieldPanel("Code"),
							validateFn: rejectCode("BAD1", "row validator failed"),
						},
					),
				}
			},
			wantError: "row validator failed",
		},
		{
			name: "NestedInPanelGroup",
			build: func() []admin.Panel {
				return []admin.Panel{
					admin.PanelGroup(
						admin.FieldPanel("Title"),
						&customValidatorPanel{
							Panel:      admin.FieldPanel("Code"),
							validateFn: rejectCode("BAD1", "group validator failed"),
						},
					),
				}
			},
			wantError: "group validator failed",
		},
		{
			name: "NestedInTabbedPanel",
			build: func() []admin.Panel {
				return []admin.Panel{
					admin.TabbedPanel(
						admin.PanelTab(
							"Main",
							admin.FieldPanel("Title"),
							&customValidatorPanel{
								Panel:      admin.FieldPanel("Code"),
								validateFn: rejectCode("BAD1", "tabbed validator failed"),
							},
						),
					),
				}
			},
			wantError: "tabbed validator failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			af := newTestAdminForm(req, &FormTestCoupon{}, nil, tc.build()...)
			af.WithData(url.Values{
				"Title": {"Promo"},
				"Code":  {"BAD1"},
			}, nil, req)

			if forms.IsValid(req.Context(), af) {
				t.Fatalf("expected ValidatorPanel to fail validation")
			}
			errs := af.ErrorList()
			if len(errs) == 0 || !strings.Contains(errs[0].Error(), tc.wantError) {
				t.Errorf("expected error %q in ErrorList, got %v", tc.wantError, errs)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 12. PanelTree, PanelComparison & BuildPanelID
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_PanelTreeAndPanelComparison(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("NewPanelTreeAndFindNode", func(t *testing.T) {
		tree := admin.NewPanelTree(
			admin.FieldPanel("Title"),
			admin.RowPanel(
				admin.FieldPanel("ValidFrom"),
				admin.FieldPanel("ValidUntil"),
			),
			admin.PanelGroup(
				admin.FieldPanel("Code"),
			),
			admin.TabbedPanel(
				admin.PanelTab("Meta", admin.FieldPanel("Description")),
			),
		)

		if node := tree.FindNode([]string{}); node != tree {
			t.Errorf("empty path should return root tree")
		}
		if node := tree.FindNode([]string{"fieldpanel-0"}); node == nil || len(node.Panels) != 1 || node.Panels[0].Fields()[0] != "Title" {
			t.Errorf("expected leaf node for fieldpanel-0 (Title), got %+v", node)
		}
		if node := tree.FindNode([]string{"rowpanel-1", "fieldpanel-1"}); node == nil || node.Panels[0].Fields()[0] != "ValidUntil" {
			t.Errorf("expected nested node rowpanel-1/fieldpanel-1 (ValidUntil), got %+v", node)
		}
		if node := tree.FindNode([]string{"panelgroup-2", "fieldpanel-0"}); node == nil || node.Panels[0].Fields()[0] != "Code" {
			t.Errorf("expected nested node panelgroup-2/fieldpanel-0 (Code), got %+v", node)
		}
		if node := tree.FindNode([]string{"tabbedpanel-3", "fieldpanel-0"}); node == nil || node.Panels[0].Fields()[0] != "Description" {
			t.Errorf("expected nested node tabbedpanel-3/fieldpanel-0 (Description), got %+v", node)
		}
		if node := tree.FindNode([]string{"missing-0"}); node != nil {
			t.Errorf("expected nil for unknown path, got %+v", node)
		}
	})

	t.Run("PanelComparisonAcrossPanels", func(t *testing.T) {
		panels := []admin.Panel{
			admin.TitlePanel(admin.FieldPanel("Title")),
			admin.RowPanel(
				admin.FieldPanel("Code"),
				admin.FieldPanel("MaxDiscount"),
			),
			admin.PanelGroup(
				admin.FieldPanel("ValidFrom"),
			),
			admin.TabbedPanel(
				admin.PanelTab("Extra", admin.FieldPanel("Description")),
			),
			&admin.AlertPanel{Type: admin.ALERT_INFO, HTML: "ignored"},
			admin.JSONDetailPanel{FieldName: "CreatedAt"},
			&admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]]{
				TargetType: &FormTestCouponItem{},
				FieldName:  "Items",
			},
		}

		oldCoupon := &FormTestCoupon{
			Title:       "Spring",
			Code:        "SPR26",
			MaxDiscount: 10,
			ValidFrom:   "2026-03-01",
			Description: "Initial desc",
		}
		sameCoupon := &FormTestCoupon{
			Title:       "Spring",
			Code:        "SPR26",
			MaxDiscount: 10,
			ValidFrom:   "2026-03-01",
			Description: "Initial desc",
		}
		newCoupon := &FormTestCoupon{
			Title:       "Summer",
			Code:        "SPR26",
			MaxDiscount: 25,
			ValidFrom:   "2026-03-01",
			Description: "Updated desc",
		}

		// Identical instances -> HasChanged == false
		compSame, err := admin.PanelComparison(context.Background(), panels, oldCoupon, sameCoupon)
		if err != nil {
			t.Fatalf("PanelComparison(same) failed: %v", err)
		}
		changed, err := compSame.HasChanged()
		if err != nil {
			t.Fatalf("HasChanged(same) error: %v", err)
		}
		if changed {
			t.Errorf("expected HasChanged() = false for identical instances")
		}

		// Changed instances -> HasChanged == true and HTMLDiff markup present
		compDiff, err := admin.PanelComparison(context.Background(), panels, oldCoupon, newCoupon)
		if err != nil {
			t.Fatalf("PanelComparison(diff) failed: %v", err)
		}
		changed, err = compDiff.HasChanged()
		if err != nil {
			t.Fatalf("HasChanged(diff) error: %v", err)
		}
		if !changed {
			t.Errorf("expected HasChanged() = true for modified instances")
		}
		diffHTML, err := compDiff.HTMLDiff()
		if err != nil {
			t.Fatalf("HTMLDiff() error: %v", err)
		}
		diffStr := string(diffHTML)
		for _, expected := range []string{"<dl>", "<dt>", "<dd>", "diff-removed", "diff-added", "Spring", "Summer"} {
			if !strings.Contains(diffStr, expected) {
				t.Errorf("expected %q in HTMLDiff output:\n%s", expected, diffStr)
			}
		}

		// Only non-comparable panels -> returns nil, nil
		emptyComp, err := admin.PanelComparison(context.Background(), []admin.Panel{
			&admin.AlertPanel{HTML: "x"},
			admin.JSONDetailPanel{FieldName: "CreatedAt"},
		}, oldCoupon, newCoupon)
		if emptyComp != nil || err != nil {
			t.Errorf("expected nil, nil when only non-comparable panels are present, got %v, %v", emptyComp, err)
		}

		// Single panel without mustWrap vs with mustWrap=true
		singleUnwrapped, err := admin.PanelComparison(context.Background(), []admin.Panel{admin.FieldPanel("Title")}, oldCoupon, newCoupon)
		if err != nil {
			t.Fatalf("single PanelComparison failed: %v", err)
		}
		if _, isWrapper := singleUnwrapped.(compare.ComparisonWrapper); isWrapper {
			t.Errorf("expected unwrapped comparison when mustWrap is omitted")
		}
		singleWrapped, err := admin.PanelComparison(context.Background(), []admin.Panel{admin.FieldPanel("Title")}, oldCoupon, newCoupon, true)
		if err != nil {
			t.Fatalf("wrapped PanelComparison failed: %v", err)
		}
		if _, isWrapper := singleWrapped.(compare.ComparisonWrapper); !isWrapper {
			t.Errorf("expected ComparisonWrapper when mustWrap=true")
		}
	})

	t.Run("BuildPanelID", func(t *testing.T) {
		if id := admin.BuildPanelID("k", nil, "email"); id != "panel-email" {
			t.Errorf("BuildPanelID with nil map = %q, want panel-email", id)
		}
		counts := make(map[string]int)
		if id := admin.BuildPanelID("row", counts, "row"); id != "panel-row" {
			t.Errorf("first BuildPanelID = %q, want panel-row", id)
		}
		if id := admin.BuildPanelID("row", counts, "row"); id != "panel-row-1" {
			t.Errorf("second BuildPanelID = %q, want panel-row-1", id)
		}
		if id := admin.BuildPanelID("row", counts, "row"); id != "panel-row-2" {
			t.Errorf("third BuildPanelID = %q, want panel-row-2", id)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 13. FormViewOptions.SetupDefaults & GetAdminForm
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_FormViewOptionsAndGetAdminForm(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("SetupDefaultsDiscoversGetAddPanelsAndGetEditPanels", func(t *testing.T) {
		mdl := &FormTestDefaultPanelModel{}
		var addOpts admin.FormViewOptions
		addOpts.SetupDefaults(mdl, "GetAddPanels")
		if len(addOpts.Panels) != 1 || addOpts.Panels[0].Fields()[0] != "Title" {
			t.Errorf("expected GetAddPanels to populate [Title], got %+v", addOpts.Panels)
		}

		var editOpts admin.FormViewOptions
		editOpts.SetupDefaults(mdl, "GetEditPanels")
		if len(editOpts.Panels) != 2 || editOpts.Panels[1].Fields()[0] != "Summary" {
			t.Errorf("expected GetEditPanels to populate [Title, Summary], got %+v", editOpts.Panels)
		}
	})

	t.Run("GetAdminFormLabelsPrecedenceAndSaveInstance", func(t *testing.T) {
		modelDef := &admin.ModelDefinition{
			Name:  "form_test_user",
			Model: &FormTestUser{},
			Labels: map[string]func(context.Context) string{
				"Email":    trans.S("Model Email Label"),
				"Username": trans.S("Model Username Label"),
			},
		}

		var customSaveCalled bool
		opts := admin.FormViewOptions{
			Fields: []string{"Email", "Username"},
			Labels: map[string]func(context.Context) string{
				"Email": trans.S("View Email Override"),
			},
			SaveInstance: func(ctx context.Context, d attrs.Definer) error {
				customSaveCalled = true
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		form := admin.GetAdminForm[*FormTestUser](&FormTestUser{}, opts, nil, modelDef, req)
		form.SetFields(opts.ViewOptions.Fields...)
		form.Load()

		emailField, ok := form.Field("Email")
		if !ok {
			t.Fatalf("expected Email field in form")
		}
		if got := emailField.Label(req.Context()); got != "View Email Override" {
			t.Errorf("Email label = %q, want 'View Email Override'", got)
		}

		usernameField, ok := form.Field("Username")
		if !ok {
			t.Fatalf("expected Username field in form")
		}
		if got := usernameField.Label(req.Context()); got != "Model Username Label" {
			t.Errorf("Username label = %q, want 'Model Username Label'", got)
		}

		form.WithData(url.Values{
			"Email":    {"custom@example.com"},
			"Username": {"customuser"},
		}, nil, req)
		if !forms.IsValid(req.Context(), form) {
			t.Fatalf("expected form valid, errs=%v", form.BoundErrors())
		}
		if _, err := form.Save(); err != nil {
			t.Fatalf("form.Save() failed: %v", err)
		}
		if !customSaveCalled {
			t.Errorf("expected custom SaveInstance hook to be called")
		}
	})

	t.Run("GetAdminFormWithFormDefiner", func(t *testing.T) {
		mdl := &FormTestDefaultPanelModel{Title: "Defined"}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		form := admin.GetAdminForm(mdl, admin.FormViewOptions{}, nil, nil, req)
		form.Load()

		if _, ok := form.Field("ExtraNote"); !ok {
			t.Errorf("expected FormDefiner.AdminForm custom field ExtraNote to be present")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 14. ModelFormPanel with ManyToMany Relation (contrib/shop/app.go pattern)
// ─────────────────────────────────────────────────────────────────────────────

func buildShopCouponM2MPanels(minNum, maxNum, extra int) []admin.Panel {
	return []admin.Panel{
		admin.TabbedPanel(
			admin.PanelTab(
				trans.S("Coupon"),
				admin.FieldPanel("Title"),
				admin.FieldPanel("Description"),
				admin.FieldPanel("Code"),
				admin.RowPanel(
					admin.FieldPanel("MaxDiscount"),
					admin.FieldPanel("IsPercentage"),
				),
			),
			admin.PanelTab(
				trans.S("Constraints"),
				admin.FieldPanel("ValidFrom"),
				admin.FieldPanel("ValidUntil"),
				admin.FieldPanel("CreatedAt"),
			),
			admin.PanelTab(
				trans.S("Products"),
				&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
					TargetType: &FormTestProduct{},
					FieldName:  "Products",
					Classname:  "collapsible",
					MinNum:     minNum,
					MaxNum:     maxNum,
					Extra:      extra,
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

func unwrapProductFormSet(t *testing.T, af *admin.AdminForm[modelforms.ModelForm[*FormTestCoupon], *FormTestCoupon]) formsets.ListFormSet[modelforms.ModelForm[*FormTestProduct]] {
	t.Helper()
	fs := af.FormSet()
	if fs == nil {
		t.Fatalf("expected non-nil FormSet() from AdminForm with M2M ModelFormPanel")
	}
	subForms, err := fs.Forms()
	if err != nil {
		t.Fatalf("FormSet().Forms() failed: %v", err)
	}
	if len(subForms) != 1 {
		t.Fatalf("expected 1 panel formset in AdminForm.FormSet(), got %d", len(subForms))
	}
	if wrapper, ok := subForms[0].(interface{ Form() formsets.BaseFormSetForm }); ok {
		subForms[0] = wrapper.Form()
	}
	prodFS, ok := subForms[0].(formsets.ListFormSet[modelforms.ModelForm[*FormTestProduct]])
	if !ok {
		t.Fatalf("expected ListFormSet[ModelForm[*FormTestProduct]], got %T", subForms[0])
	}
	return prodFS
}

func TestAdminForm_ModelFormPanelManyToMany(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("M2MRelationMetadataAndThroughQuerySetSanity", func(t *testing.T) {
		setupFormsTestDB(t)
		defs := attrs.Define(context.Background(), &FormTestCoupon{})
		prodField, ok := defs.Field("Products")
		if !ok {
			t.Fatalf("expected Products field on FormTestCoupon")
		}
		rel := prodField.Rel()
		if rel == nil {
			t.Fatalf("expected non-nil relation on FormTestCoupon.Products")
		}
		if rel.Type() != attrs.RelManyToMany {
			t.Fatalf("Products relation type = %s, want %s", rel.Type(), attrs.RelManyToMany)
		}
		through := rel.Through()
		if through == nil {
			t.Fatalf("expected non-nil Through on FormTestCoupon.Products")
		}
		if _, ok := through.Model().(*FormTestCouponProduct); !ok {
			t.Fatalf("Through.Model() = %T, want *FormTestCouponProduct", through.Model())
		}
		if through.SourceField() != "Coupon" {
			t.Errorf("Through.SourceField() = %q, want Coupon", through.SourceField())
		}
		if through.TargetField() != "Product" {
			t.Errorf("Through.TargetField() = %q, want Product", through.TargetField())
		}

		coupon := &FormTestCoupon{
			Title:       "M2M Sanity Coupon",
			Description: "M2M relation check",
			Code:        "M2MSANITY",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("failed to create coupon: %v", err)
		}

		p1 := &FormTestProduct{Title: "Laptop", Slug: "laptop", Price: 1200}
		p2 := &FormTestProduct{Title: "Dock", Slug: "dock", Price: 180}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("failed to create p1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p2); err != nil {
			t.Fatalf("failed to create p2: %v", err)
		}

		if _, err := coupon.Products.Objects().AddTargets(p1, p2); err != nil {
			t.Fatalf("AddTargets failed: %v", err)
		}

		rows, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("Products.Objects().All() failed: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 linked products, got %d", len(rows))
		}
		if rows[0].Object.ID != p1.ID || rows[1].Object.ID != p2.ID {
			t.Errorf("unexpected linked product IDs: [%d, %d], want [%d, %d]",
				rows[0].Object.ID, rows[1].Object.ID, p1.ID, p2.ID)
		}
	})

	t.Run("UnboundAddViewAndPopulatedEditViewRendering", func(t *testing.T) {
		setupFormsTestDB(t)
		// AddView with MinNum=1, MaxNum=5, Extra=0
		newCoupon := &FormTestCoupon{}
		reqAdd := httptest.NewRequest(http.MethodGet, "/admin/shop/coupons/add", nil)
		afAdd := newTestAdminForm(reqAdd, newCoupon, nil, buildShopCouponM2MPanels(1, 5, 0)...)

		prodFSAdd := unwrapProductFormSet(t, afAdd)
		if prodFSAdd.Prefix() != "Products" {
			t.Errorf("AddView FormSet prefix = %q, want Products", prodFSAdd.Prefix())
		}
		addForms, err := prodFSAdd.Forms()
		if err != nil {
			t.Fatalf("AddView prodFSAdd.Forms() failed: %v", err)
		}
		if len(addForms) != 1 {
			t.Errorf("expected 1 inline form on AddView, got %d", len(addForms))
		}
		if got := prodFSAdd.ManagementForm().TotalFormsValue; got != 1 {
			t.Errorf("AddView TotalFormsValue = %d, want 1", got)
		}

		docAdd := parseHTMLDoc(t, afAdd.BoundForm().AsP())
		if docAdd.Find(`.modelform-panel.collapsible`).Length() != 1 {
			t.Errorf("expected .modelform-panel.collapsible in AddView HTML")
		}
		if docAdd.Find(`template.modelform-panel__empty-form`).Length() != 1 {
			t.Errorf("expected template.modelform-panel__empty-form in AddView HTML")
		}
		if docAdd.Find(`.modelform-panel__management`).Length() != 1 {
			t.Errorf("expected .modelform-panel__management in AddView HTML")
		}
		for _, field := range []string{"Products-0-Title", "Products-0-Slug", "Products-0-Price"} {
			if docAdd.Find(fmt.Sprintf(`input[name=%q]`, field)).Length() != 1 {
				t.Errorf("expected input %q in AddView rendered HTML", field)
			}
		}

		// EditView with 2 linked products via FormTestCouponProduct
		editCoupon := &FormTestCoupon{
			Title:       "Existing M2M Coupon",
			Description: "Has 2 products",
			Code:        "EDITM2M",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(editCoupon); err != nil {
			t.Fatalf("failed to create editCoupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Espresso Machine", Slug: "espresso-machine", Price: 450}
		p2 := &FormTestProduct{Title: "Coffee Grinder", Slug: "coffee-grinder", Price: 160}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("failed to create p1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p2); err != nil {
			t.Fatalf("failed to create p2: %v", err)
		}
		if _, err := editCoupon.Products.Objects().AddTargets(p1, p2); err != nil {
			t.Fatalf("failed to link products to editCoupon: %v", err)
		}

		reqEdit := httptest.NewRequest(http.MethodGet, "/admin/shop/coupons/edit", nil)
		afEdit := newTestAdminForm(reqEdit, editCoupon, nil, buildShopCouponM2MPanels(1, 5, 0)...)

		prodFSEdit := unwrapProductFormSet(t, afEdit)
		editForms, err := prodFSEdit.Forms()
		if err != nil {
			t.Fatalf("EditView prodFSEdit.Forms() failed: %v", err)
		}
		if len(editForms) != 2 {
			t.Fatalf("expected 2 default inline forms on EditView, got %d", len(editForms))
		}
		if got := prodFSEdit.ManagementForm().TotalFormsValue; got != 2 {
			t.Errorf("EditView TotalFormsValue = %d, want 2", got)
		}
		if init0 := editForms[0].InitialData()["Title"]; init0 != "Espresso Machine" {
			t.Errorf("editForms[0].InitialData[Title] = %v, want 'Espresso Machine'", init0)
		}
		if init1 := editForms[1].InitialData()["Title"]; init1 != "Coffee Grinder" {
			t.Errorf("editForms[1].InitialData[Title] = %v, want 'Coffee Grinder'", init1)
		}

		docEdit := parseHTMLDoc(t, afEdit.BoundForm().AsP())
		if docEdit.Find(`input[name="Products-0-Title"][value="Espresso Machine"]`).Length() != 1 {
			t.Errorf("expected input Products-0-Title with value 'Espresso Machine'")
		}
		if docEdit.Find(`input[name="Products-1-Title"][value="Coffee Grinder"]`).Length() != 1 {
			t.Errorf("expected input Products-1-Title with value 'Coffee Grinder'")
		}
	})

	t.Run("AddCouponWithM2MProductsInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		reqAdd := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/add", nil)
		afAdd := newTestAdminForm(reqAdd, coupon, nil, buildShopCouponM2MPanels(0, 10, 0)...)
		afAdd.WithData(url.Values{
			"Title":                                      {"Bundle Coupon"},
			"Description":                                {"Created with M2M products"},
			"Code":                                       {"BUNDLE20"},
			"MaxDiscount":                                {"20"},
			"IsPercentage":                               {"on"},
			"ValidFrom":                                  {"2026-07-01"},
			"ValidUntil":                                 {"2026-08-01"},
			"Products-management-TOTAL_FORMS":            {"2"},
			"Products-management-INITIAL_FORMS":          {"0"},
			"Products-management-MIN_NUM_FORMS":          {"0"},
			"Products-management-MAX_NUM_FORMS":          {"10"},
			"Products-0-Title":                           {"Mechanical Keyboard"},
			"Products-0-Slug":                            {"mech-kb"},
			"Products-0-Price":                           {"120"},
			"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Products-1-Title":                           {"Gaming Mouse"},
			"Products-1-Slug":                            {"gaming-mouse"},
			"Products-1-Price":                           {"65"},
			"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqAdd)

		if !forms.IsValid(reqAdd.Context(), afAdd) {
			t.Fatalf("expected AddView form valid, errs=%v bound=%v fsErrs=%v",
				afAdd.ErrorList(), afAdd.BoundErrors(), afAdd.FormSet().ErrorList())
		}
		if _, err := afAdd.Save(); err != nil {
			t.Fatalf("AddView Save() failed: %v", err)
		}
		if len(afAdd.ErrorList()) > 0 {
			t.Fatalf("unexpected errors after AddView Save(): %v", afAdd.ErrorList())
		}
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}

		allProducts, err := queries.GetQuerySet(&FormTestProduct{}).OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query products: %v", err)
		}
		if len(allProducts) != 2 {
			t.Fatalf("expected 2 created products in DB, got %d", len(allProducts))
		}

		linked, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query coupon.Products: %v", err)
		}
		if len(linked) != 2 {
			t.Fatalf("expected 2 linked M2M products for coupon %d, got %d", coupon.ID, len(linked))
		}
		if linked[0].Object.Title != "Mechanical Keyboard" || linked[0].Object.Price != 120 {
			t.Errorf("unexpected linked[0]: %+v", linked[0].Object)
		}
		if linked[1].Object.Title != "Gaming Mouse" || linked[1].Object.Price != 65 {
			t.Errorf("unexpected linked[1]: %+v", linked[1].Object)
		}
	})

	t.Run("EditCouponM2MProductsHasChangedUpdateAddAndDelete", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{
			Title:        "Hardware Promo",
			Description:  "M2M edit lifecycle",
			Code:         "HARD50",
			MaxDiscount:  50,
			IsPercentage: false,
			ValidFrom:    "2026-10-01",
			ValidUntil:   "2026-12-31",
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("failed to create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "USB Mic", Slug: "usb-mic", Price: 90}
		p2 := &FormTestProduct{Title: "Boom Arm", Slug: "boom-arm", Price: 45}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("failed to create p1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p2); err != nil {
			t.Fatalf("failed to create p2: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1, p2); err != nil {
			t.Fatalf("failed to link initial products: %v", err)
		}

		// 1. Unchanged submission -> HasChanged() == false
		reqUnchanged := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afUnchanged := newTestAdminForm(reqUnchanged, coupon, nil, buildShopCouponM2MPanels(0, 10, 0)...)
		afUnchanged.WithData(url.Values{
			"Title":                                      {coupon.Title},
			"Description":                                {coupon.Description},
			"Code":                                       {coupon.Code},
			"MaxDiscount":                                {"50"},
			"ValidFrom":                                  {coupon.ValidFrom},
			"ValidUntil":                                 {coupon.ValidUntil},
			"Products-management-TOTAL_FORMS":            {"2"},
			"Products-management-INITIAL_FORMS":          {"2"},
			"Products-management-MIN_NUM_FORMS":          {"0"},
			"Products-management-MAX_NUM_FORMS":          {"10"},
			"Products-0-ID":                              {fmt.Sprint(p1.ID)},
			"Products-0-Title":                           {p1.Title},
			"Products-0-Slug":                            {p1.Slug},
			"Products-0-Price":                           {"90"},
			"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Products-1-ID":                              {fmt.Sprint(p2.ID)},
			"Products-1-Title":                           {p2.Title},
			"Products-1-Slug":                            {p2.Slug},
			"Products-1-Price":                           {"45"},
			"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqUnchanged)
		if !forms.IsValid(reqUnchanged.Context(), afUnchanged) {
			t.Fatalf("expected unchanged edit form valid, errs=%v bound=%v",
				afUnchanged.ErrorList(), afUnchanged.BoundErrors())
		}
		if afUnchanged.HasChanged() {
			t.Errorf("expected HasChanged() = false for identical parent and M2M product data")
		}

		// 2. Modify product 0 and add new product 2
		reqUpdate := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afUpdate := newTestAdminForm(reqUpdate, coupon, nil, buildShopCouponM2MPanels(0, 10, 0)...)
		afUpdate.WithData(url.Values{
			"Title":                                      {coupon.Title},
			"Description":                                {coupon.Description},
			"Code":                                       {coupon.Code},
			"MaxDiscount":                                {"50"},
			"ValidFrom":                                  {coupon.ValidFrom},
			"ValidUntil":                                 {coupon.ValidUntil},
			"Products-management-TOTAL_FORMS":            {"3"},
			"Products-management-INITIAL_FORMS":          {"2"},
			"Products-management-MIN_NUM_FORMS":          {"0"},
			"Products-management-MAX_NUM_FORMS":          {"10"},
			"Products-0-ID":                              {fmt.Sprint(p1.ID)},
			"Products-0-Title":                           {"Studio USB Mic"},
			"Products-0-Slug":                            {p1.Slug},
			"Products-0-Price":                           {"115"},
			"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Products-1-ID":                              {fmt.Sprint(p2.ID)},
			"Products-1-Title":                           {p2.Title},
			"Products-1-Slug":                            {p2.Slug},
			"Products-1-Price":                           {"45"},
			"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Products-2-Title":                           {"Shock Mount"},
			"Products-2-Slug":                            {"shock-mount"},
			"Products-2-Price":                           {"30"},
			"Products-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
			"Products-2-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqUpdate)
		if !forms.IsValid(reqUpdate.Context(), afUpdate) {
			t.Fatalf("expected updated/added M2M form valid, errs=%v bound=%v",
				afUpdate.ErrorList(), afUpdate.BoundErrors())
		}
		if !afUpdate.HasChanged() {
			t.Errorf("expected HasChanged() = true when M2M product updated and added")
		}
		if _, err := afUpdate.Save(); err != nil {
			t.Fatalf("Edit Save() (update+add) failed: %v", err)
		}

		updatedP1, err := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", p1.ID).Get()
		if err != nil {
			t.Fatalf("failed to fetch updated p1: %v", err)
		}
		if updatedP1.Object.Title != "Studio USB Mic" || updatedP1.Object.Price != 115 {
			t.Errorf("expected updated p1 Studio USB Mic ($115), got %+v", updatedP1.Object)
		}

		linkedAfterAdd, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query coupon.Products after update+add: %v", err)
		}
		if len(linkedAfterAdd) != 3 {
			t.Fatalf("expected 3 linked products after adding Products-2, got %d", len(linkedAfterAdd))
		}
		p3 := linkedAfterAdd[2].Object
		if p3.Title != "Shock Mount" || p3.Price != 30 {
			t.Errorf("unexpected created p3: %+v", p3)
		}

		// 3. Delete product 0 from coupon via DELETION_FIELD_NAME
		reqDelete := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		afDelete := newTestAdminForm(reqDelete, coupon, nil, buildShopCouponM2MPanels(0, 10, 0)...)
		afDelete.WithData(url.Values{
			"Title":                                      {coupon.Title},
			"Description":                                {coupon.Description},
			"Code":                                       {coupon.Code},
			"MaxDiscount":                                {"50"},
			"ValidFrom":                                  {coupon.ValidFrom},
			"ValidUntil":                                 {coupon.ValidUntil},
			"Products-management-TOTAL_FORMS":            {"3"},
			"Products-management-INITIAL_FORMS":          {"3"},
			"Products-management-MIN_NUM_FORMS":          {"0"},
			"Products-management-MAX_NUM_FORMS":          {"10"},
			"Products-0-ID":                              {fmt.Sprint(p1.ID)},
			"Products-0-Title":                           {"Studio USB Mic"},
			"Products-0-Slug":                            {p1.Slug},
			"Products-0-Price":                           {"115"},
			"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Products-0-" + formsets.DELETION_FIELD_NAME: {"on"},
			"Products-1-ID":                              {fmt.Sprint(p2.ID)},
			"Products-1-Title":                           {p2.Title},
			"Products-1-Slug":                            {p2.Slug},
			"Products-1-Price":                           {"45"},
			"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
			"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			"Products-2-ID":                              {fmt.Sprint(p3.ID)},
			"Products-2-Title":                           {p3.Title},
			"Products-2-Slug":                            {p3.Slug},
			"Products-2-Price":                           {"30"},
			"Products-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
			"Products-2-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, reqDelete)
		if !forms.IsValid(reqDelete.Context(), afDelete) {
			t.Fatalf("expected delete M2M form valid, errs=%v bound=%v",
				afDelete.ErrorList(), afDelete.BoundErrors())
		}
		if !afDelete.HasChanged() {
			t.Errorf("expected HasChanged() = true when M2M product marked for deletion")
		}
		if _, err := afDelete.Save(); err != nil {
			t.Fatalf("Edit Save() (delete) failed: %v", err)
		}

		linkedAfterDelete, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("failed to query coupon.Products after delete: %v", err)
		}
		if len(linkedAfterDelete) != 2 {
			t.Fatalf("expected 2 linked products after deleting p1, got %d", len(linkedAfterDelete))
		}
		if linkedAfterDelete[0].Object.ID != p2.ID || linkedAfterDelete[1].Object.ID != p3.ID {
			t.Errorf("expected remaining linked product IDs [%d, %d], got [%d, %d]",
				p2.ID, p3.ID, linkedAfterDelete[0].Object.ID, linkedAfterDelete[1].Object.ID)
		}
	})

	t.Run("M2MModelFormPanelValidationErrors", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		req := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/add", nil)
		af := newTestAdminForm(req, coupon, nil, buildShopCouponM2MPanels(1, 5, 0)...)
		af.WithData(url.Values{
			"Title":                                      {"Valid Coupon Title"},
			"Description":                                {"Invalid product title test"},
			"Code":                                       {"VALERR1"},
			"MaxDiscount":                                {"10"},
			"ValidFrom":                                  {"2026-07-01"},
			"ValidUntil":                                 {"2026-08-01"},
			"Products-management-TOTAL_FORMS":            {"1"},
			"Products-management-INITIAL_FORMS":          {"0"},
			"Products-management-MIN_NUM_FORMS":          {"1"},
			"Products-management-MAX_NUM_FORMS":          {"5"},
			"Products-0-Title":                           {"A"}, // MinLength is 2
			"Products-0-Slug":                            {"a"},
			"Products-0-Price":                           {"10"},
			"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
			"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
		}, nil, req)

		if forms.IsValid(req.Context(), af) {
			t.Fatalf("expected M2M ModelFormPanel form validation to fail when Products-0-Title violates MinLength")
		}

		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected rendered panel HTML to display validation error element .panel__error")
		}
	})

	t.Run("Add_CreateCouponWithZeroM2MProductsWhenMinNumZero", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Zero M2M Coupon", "ZEROM2M"),
			inlineMgmtValues("Products", 0, 0, 0, 5),
		)
		af := submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}
		_ = af

		prodCount, err := queries.GetQuerySet(&FormTestProduct{}).Count()
		if err != nil {
			t.Fatalf("count products: %v", err)
		}
		if prodCount != 0 {
			t.Errorf("expected 0 products created, got %d", prodCount)
		}
		linked, err := coupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query coupon.Products: %v", err)
		}
		if len(linked) != 0 {
			t.Errorf("expected 0 linked products, got %d", len(linked))
		}
	})

	t.Run("Add_CreateCouponSkippingDeletedM2MProductSlot", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Skip Deleted M2M", "SKIPM2M"),
			inlineMgmtValues("Products", 2, 0, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Discarded Product"},
				"Products-0-Slug":                            {"discarded"},
				"Products-0-Price":                           {"999"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"on"},
				"Products-1-Title":                           {"Kept Product"},
				"Products-1-Slug":                            {"kept-product"},
				"Products-1-Price":                           {"75"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)

		allProducts, err := queries.GetQuerySet(&FormTestProduct{}).OrderBy("ID").All()
		if err != nil {
			t.Fatalf("query products: %v", err)
		}
		if len(allProducts) != 1 || allProducts[0].Object.Title != "Kept Product" {
			t.Fatalf("expected only 'Kept Product' in DB, got %+v", allProducts)
		}
		linked, err := coupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 1 || linked[0].Object.ID != allProducts[0].Object.ID {
			t.Errorf("expected coupon linked only to 'Kept Product', got %+v", linked)
		}
	})

	t.Run("Add_CreateCouponWithCustomFormAndSaveInstanceOnM2MPanel", func(t *testing.T) {
		setupFormsTestDB(t)
		var saveHookCalled bool
		panels := []admin.Panel{
			admin.FieldPanel("Title"),
			admin.FieldPanel("Code"),
			&admin.ModelFormPanel[*FormTestProduct, modelforms.ModelForm[*FormTestProduct]]{
				TargetType: &FormTestProduct{},
				FieldName:  "Products",
				MinNum:     1,
				MaxNum:     5,
				Form: func() modelforms.ModelForm[*FormTestProduct] {
					f := modelforms.NewBaseModelForm(context.Background(), &FormTestProduct{})
					f.SetValidators(func(frm forms.Form, m map[string]any) []error {
						if p, _ := m["Price"].(int); p < 10 {
							return []error{errors.New("product price must be at least 10")}
						}
						return nil
					})
					return f
				},
				SaveInstance: func(ctx context.Context, form modelforms.ModelForm[*FormTestProduct], inst *FormTestProduct) error {
					saveHookCalled = true
					inst.Slug = strings.ToUpper(inst.Slug)
					return inst.Save(ctx)
				},
				Panels: []admin.Panel{
					admin.FieldPanel("Title"),
					admin.FieldPanel("Slug"),
					admin.FieldPanel("Price"),
				},
			},
		}

		// Custom Form validator rejects Price < 10
		badCoupon := &FormTestCoupon{}
		badData := mergeFormValues(
			baseCouponPOST("Custom M2M Bad", "CUSTBAD"),
			inlineMgmtValues("Products", 1, 0, 1, 5),
			url.Values{
				"Products-0-Title":                           {"Cheap Item"},
				"Products-0-Slug":                            {"cheap-item"},
				"Products-0-Price":                           {"5"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitExpectInvalidAdminForm(t, badCoupon, badData, panels...)
		if saveHookCalled {
			t.Fatalf("did not expect SaveInstance to run on invalid form")
		}

		// Valid Price >= 10 invokes SaveInstance and links via M2M through model
		goodCoupon := &FormTestCoupon{}
		goodData := mergeFormValues(
			baseCouponPOST("Custom M2M Good", "CUSTGOOD"),
			inlineMgmtValues("Products", 1, 0, 1, 5),
			url.Values{
				"Products-0-Title":                           {"Pro Desk"},
				"Products-0-Slug":                            {"pro-desk"},
				"Products-0-Price":                           {"350"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, goodCoupon, goodData, panels...)
		if !saveHookCalled {
			t.Fatalf("expected custom SaveInstance callback to be invoked")
		}
		linked, err := goodCoupon.Products.Objects().All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 1 || linked[0].Object.Slug != "PRO-DESK" {
			t.Errorf("expected custom SaveInstance slug 'PRO-DESK', got %+v", linked)
		}
	})

	t.Run("Add_CreateCouponFailsOnM2MManagementFormMinMaxBounds", func(t *testing.T) {
		setupFormsTestDB(t)
		panels := buildShopCouponM2MPanels(2, 3, 0)

		// Below MinNum=2
		belowCoupon := &FormTestCoupon{}
		belowData := mergeFormValues(
			baseCouponPOST("Below Min", "BELOW2"),
			inlineMgmtValues("Products", 1, 0, 2, 3),
			url.Values{
				"Products-0-Title":                           {"Solo"},
				"Products-0-Slug":                            {"solo"},
				"Products-0-Price":                           {"20"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitExpectInvalidAdminForm(t, belowCoupon, belowData, panels...)

		// Above MaxNum=3
		aboveCoupon := &FormTestCoupon{}
		aboveData := mergeFormValues(
			baseCouponPOST("Above Max", "ABOVE3"),
			inlineMgmtValues("Products", 4, 0, 2, 3),
			url.Values{
				"Products-0-Title":                           {"One"},
				"Products-0-Slug":                            {"one"},
				"Products-0-Price":                           {"10"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-Title":                           {"Two"},
				"Products-1-Slug":                            {"two"},
				"Products-1-Price":                           {"20"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-2-Title":                           {"Three"},
				"Products-2-Slug":                            {"three"},
				"Products-2-Price":                           {"30"},
				"Products-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Products-2-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-3-Title":                           {"Four"},
				"Products-3-Slug":                            {"four"},
				"Products-3-Price":                           {"40"},
				"Products-3-" + formsets.ORDERING_FIELD_NAME: {"3"},
				"Products-3-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitExpectInvalidAdminForm(t, aboveCoupon, aboveData, panels...)
		if cnt, _ := queries.GetQuerySet(&FormTestCoupon{}).Count(); cnt != 0 {
			t.Errorf("expected 0 coupons persisted on invalid bounds, got %d", cnt)
		}
	})

	t.Run("Update_EditCouponWithUnchangedM2MProductsHasChangedFalse", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Fixed M2M", Code: "FIXED1", ValidFrom: "2026-01-01", ValidUntil: "2026-12-31"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Lamp", Slug: "lamp", Price: 40}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link product: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/shop/coupons/edit", nil)
		af := newTestAdminForm(req, coupon, nil, buildShopCouponM2MPanels(0, 5, 0)...)
		af.WithData(mergeFormValues(
			url.Values{
				"Title":       {"Fixed M2M"},
				"Description": {""},
				"Code":        {"FIXED1"},
				"MaxDiscount": {"0"},
				"ValidFrom":   {"2026-01-01"},
				"ValidUntil":  {"2026-12-31"},
			},
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":                              {fmt.Sprint(p1.ID)},
				"Products-0-Title":                           {"Lamp"},
				"Products-0-Slug":                            {"lamp"},
				"Products-0-Price":                           {"40"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if af.HasChanged() {
			t.Errorf("expected HasChanged() == false on identical coupon and M2M product data")
		}
	})

	t.Run("Update_EditCouponUpdatingExistingM2MProductFieldsInPlace", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Edit Place", Code: "INPLACE1"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Old Chair", Slug: "old-chair", Price: 100}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link product: %v", err)
		}

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-Title":                           {"Ergonomic Chair"},
				"Products-0-Slug":                            {"ergonomic-chair"},
				"Products-0-Price":                           {"240"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)

		got, err := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", p1.ID).Get()
		if err != nil {
			t.Fatalf("fetch updated product: %v", err)
		}
		if got.Object.Title != "Ergonomic Chair" || got.Object.Slug != "ergonomic-chair" || got.Object.Price != 240 {
			t.Errorf("expected in-place update on product %d, got %+v", p1.ID, got.Object)
		}
		totalProds, _ := queries.GetQuerySet(&FormTestProduct{}).Count()
		if totalProds != 1 {
			t.Errorf("expected 1 total product row, got %d", totalProds)
		}
	})

	t.Run("Update_EditCouponAddingNewM2MProductsToExistingCoupon", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Expand M2M", Code: "EXPANDM2M"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Base Item", Slug: "base-item", Price: 50}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link product: %v", err)
		}

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Products", 3, 1, 0, 5),
			url.Values{
				"Products-0-ID":                              {fmt.Sprint(p1.ID)},
				"Products-0-Title":                           {p1.Title},
				"Products-0-Slug":                            {p1.Slug},
				"Products-0-Price":                           {"50"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-1-Title":                           {"Addon One"},
				"Products-1-Slug":                            {"addon-one"},
				"Products-1-Price":                           {"25"},
				"Products-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Products-1-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Products-2-Title":                           {"Addon Two"},
				"Products-2-Slug":                            {"addon-two"},
				"Products-2-Price":                           {"35"},
				"Products-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Products-2-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)

		linked, err := coupon.Products.Objects().OrderBy("ID").All()
		if err != nil {
			t.Fatalf("query linked products: %v", err)
		}
		if len(linked) != 3 {
			t.Fatalf("expected 3 linked products, got %d", len(linked))
		}
		if linked[1].Object.Title != "Addon One" || linked[2].Object.Title != "Addon Two" {
			t.Errorf("unexpected added products: %+v, %+v", linked[1].Object, linked[2].Object)
		}
	})

	t.Run("Update_EditCouponUnlinkingDeletedM2MProductViaThroughTable", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Unlink M2M", Code: "UNLINK1"}
		otherCoupon := &FormTestCoupon{Title: "Other M2M", Code: "OTHERM2M"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(otherCoupon); err != nil {
			t.Fatalf("create otherCoupon: %v", err)
		}
		shared := &FormTestProduct{Title: "Shared Cable", Slug: "shared-cable", Price: 15}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(shared); err != nil {
			t.Fatalf("create shared product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(shared); err != nil {
			t.Fatalf("link to coupon: %v", err)
		}
		if _, err := otherCoupon.Products.Objects().AddTargets(shared); err != nil {
			t.Fatalf("link to otherCoupon: %v", err)
		}

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":                              {fmt.Sprint(shared.ID)},
				"Products-0-Title":                           {shared.Title},
				"Products-0-Slug":                            {shared.Slug},
				"Products-0-Price":                           {"15"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"on"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)

		linkedCoupon, _ := coupon.Products.Objects().All()
		if len(linkedCoupon) != 0 {
			t.Errorf("expected 0 linked products on coupon after unlink, got %d", len(linkedCoupon))
		}
		linkedOther, _ := otherCoupon.Products.Objects().All()
		if len(linkedOther) != 1 || linkedOther[0].Object.ID != shared.ID {
			t.Errorf("expected shared product still linked to otherCoupon, got %+v", linkedOther)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", shared.ID).Count(); cnt != 1 {
			t.Errorf("expected shared FormTestProduct row to remain in DB after M2M unlink, count=%d", cnt)
		}
	})

	t.Run("Update_EditCouponFailsOnInvalidM2MProductAndPreservesBoundInput", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Original Coupon", Code: "ORIGM2M"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		p1 := &FormTestProduct{Title: "Valid Name", Slug: "valid-name", Price: 60}
		if _, err := queries.GetQuerySet(&FormTestProduct{}).Create(p1); err != nil {
			t.Fatalf("create product: %v", err)
		}
		if _, err := coupon.Products.Objects().AddTargets(p1); err != nil {
			t.Fatalf("link product: %v", err)
		}

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Products", 1, 1, 0, 5),
			url.Values{
				"Products-0-ID":                              {fmt.Sprint(p1.ID)},
				"Products-0-Title":                           {"X"}, // MinLength is 2
				"Products-0-Slug":                            {"x-slug"},
				"Products-0-Price":                           {"60"},
				"Products-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Products-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af := submitExpectInvalidAdminForm(t, coupon, data, buildShopCouponM2MPanels(0, 5, 0)...)
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`input[name="Products-0-Title"][value="X"]`).Length() != 1 {
			t.Errorf("expected invalid POST value 'X' preserved in re-rendered HTML")
		}
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered in HTML")
		}

		inDB, _ := queries.GetQuerySet(&FormTestProduct{}).Filter("ID", p1.ID).Get()
		if inDB.Object.Title != "Valid Name" {
			t.Errorf("expected DB product title preserved as 'Valid Name', got %q", inDB.Object.Title)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 15. Shared helpers for ModelFormPanel relationship CRUD tests
// ─────────────────────────────────────────────────────────────────────────────

func mergeFormValues(sets ...url.Values) url.Values {
	out := make(url.Values)
	for _, s := range sets {
		for k, v := range s {
			out[k] = slices.Clone(v)
		}
	}
	return out
}

func baseCouponPOST(title, code string) url.Values {
	return url.Values{
		"Title":       {title},
		"Description": {"Promo description"},
		"Code":        {code},
		"MaxDiscount": {"15"},
		"ValidFrom":   {"2026-01-01"},
		"ValidUntil":  {"2026-12-31"},
	}
}

func baseAuthorPOST(name, email string) url.Values {
	return url.Values{
		"Name":  {name},
		"Email": {email},
	}
}

func inlineMgmtValues(prefix string, total, initial, minNum, maxNum int) url.Values {
	return url.Values{
		prefix + "-management-TOTAL_FORMS":   {fmt.Sprint(total)},
		prefix + "-management-INITIAL_FORMS": {fmt.Sprint(initial)},
		prefix + "-management-MIN_NUM_FORMS": {fmt.Sprint(minNum)},
		prefix + "-management-MAX_NUM_FORMS": {fmt.Sprint(maxNum)},
	}
}

func submitAndSaveAdminForm[T attrs.Definer](t *testing.T, instance T, data url.Values, panels ...admin.Panel) *admin.AdminForm[modelforms.ModelForm[T], T] {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/test/submit", nil)
	af := newTestAdminForm(req, instance, nil, panels...)
	af.WithData(data, nil, req)
	if !forms.IsValid(req.Context(), af) {
		var fsErrs []error
		if fs := af.FormSet(); fs != nil {
			fsErrs = fs.ErrorList()
		}
		t.Fatalf("expected AdminForm[%T] valid, errs=%v bound=%v fsErrs=%v",
			instance, af.ErrorList(), af.BoundErrors(), fsErrs)
	}
	if _, err := af.Save(); err != nil {
		t.Fatalf("AdminForm[%T].Save() failed: %v", instance, err)
	}
	if len(af.ErrorList()) > 0 {
		t.Fatalf("unexpected ErrorList() after AdminForm[%T].Save(): %v", instance, af.ErrorList())
	}
	return af
}

func submitExpectInvalidAdminForm[T attrs.Definer](t *testing.T, instance T, data url.Values, panels ...admin.Panel) *admin.AdminForm[modelforms.ModelForm[T], T] {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/test/submit", nil)
	af := newTestAdminForm(req, instance, nil, panels...)
	af.WithData(data, nil, req)
	if forms.IsValid(req.Context(), af) {
		t.Fatalf("expected AdminForm[%T] validation to fail, but IsValid returned true", instance)
	}
	return af
}

func buildCouponOneToManyPanels(minNum, maxNum, extra int, opts ...func(*admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]])) []admin.Panel {
	itemPanel := &admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]]{
		TargetType: &FormTestCouponItem{},
		FieldName:  "Items",
		MinNum:     minNum,
		MaxNum:     maxNum,
		Extra:      extra,
		Panels: []admin.Panel{
			admin.FieldPanel("ID"),
			admin.FieldPanel("ProductName"),
			admin.FieldPanel("Quantity"),
		},
	}
	for _, opt := range opts {
		opt(itemPanel)
	}
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Description"),
		admin.FieldPanel("Code"),
		admin.FieldPanel("MaxDiscount"),
		admin.FieldPanel("IsPercentage"),
		admin.FieldPanel("ValidFrom"),
		admin.FieldPanel("ValidUntil"),
		itemPanel,
	}
}

func buildCouponManyToOnePanels(minNum, maxNum, extra int, opts ...func(*admin.ModelFormPanel[*FormTestCategory, modelforms.ModelForm[*FormTestCategory]])) []admin.Panel {
	catPanel := &admin.ModelFormPanel[*FormTestCategory, modelforms.ModelForm[*FormTestCategory]]{
		TargetType: &FormTestCategory{},
		FieldName:  "Category",
		MinNum:     minNum,
		MaxNum:     maxNum,
		Extra:      extra,
		Panels: []admin.Panel{
			admin.FieldPanel("Name"),
		},
	}
	for _, opt := range opts {
		opt(catPanel)
	}
	return []admin.Panel{
		admin.FieldPanel("Title"),
		admin.FieldPanel("Description"),
		admin.FieldPanel("Code"),
		admin.FieldPanel("MaxDiscount"),
		admin.FieldPanel("IsPercentage"),
		admin.FieldPanel("ValidFrom"),
		admin.FieldPanel("ValidUntil"),
		catPanel,
	}
}

func buildAuthorOneToOnePanels(
	includeBio bool, bioMin, bioMax, bioExtra int, bioOpts []func(*admin.ModelFormPanel[*FormTestAuthorBio, modelforms.ModelForm[*FormTestAuthorBio]]),
	includeBadge bool, badgeMin, badgeMax, badgeExtra int, badgeOpts []func(*admin.ModelFormPanel[*FormTestAuthorBadge, modelforms.ModelForm[*FormTestAuthorBadge]]),
) []admin.Panel {
	panels := []admin.Panel{
		admin.FieldPanel("Name"),
		admin.FieldPanel("Email"),
	}
	if includeBio {
		bioPanel := &admin.ModelFormPanel[*FormTestAuthorBio, modelforms.ModelForm[*FormTestAuthorBio]]{
			TargetType: &FormTestAuthorBio{},
			FieldName:  "Bio",
			MinNum:     bioMin,
			MaxNum:     bioMax,
			Extra:      bioExtra,
			Panels: []admin.Panel{
				admin.FieldPanel("Headline"),
				admin.FieldPanel("Website"),
				admin.FieldPanel("ExperienceYears"),
			},
		}
		for _, opt := range bioOpts {
			opt(bioPanel)
		}
		panels = append(panels, bioPanel)
	}
	if includeBadge {
		badgePanel := &admin.ModelFormPanel[*FormTestAuthorBadge, modelforms.ModelForm[*FormTestAuthorBadge]]{
			TargetType: &FormTestAuthorBadge{},
			FieldName:  "Badge",
			MinNum:     badgeMin,
			MaxNum:     badgeMax,
			Extra:      badgeExtra,
			Panels: []admin.Panel{
				admin.FieldPanel("BadgeName"),
				admin.FieldPanel("Tier"),
			},
		}
		for _, opt := range badgeOpts {
			opt(badgePanel)
		}
		panels = append(panels, badgePanel)
	}
	return panels
}

// ─────────────────────────────────────────────────────────────────────────────
// 16. ModelFormPanel RelOneToMany Create & Update Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_ModelFormPanelOneToMany(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("Add_CreateParentWithMultipleInlineChildren", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Triple Pack", "TRIPLE3"),
			inlineMgmtValues("Items", 3, 0, 1, 5),
			url.Values{
				"Items-0-ProductName":                     {"Dock"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"Cable"},
				"Items-1-Quantity":                        {"3"},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-2-ProductName":                     {"Adapter"},
				"Items-2-Quantity":                        {"2"},
				"Items-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Items-2-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(1, 5, 0)...)
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}

		items, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if err != nil {
			t.Fatalf("query coupon items: %v", err)
		}
		if len(items) != 3 {
			t.Fatalf("expected 3 inline items, got %d", len(items))
		}
		if items[0].Object.ProductName != "Dock" || items[1].Object.ProductName != "Cable" || items[2].Object.Quantity != 2 {
			t.Errorf("unexpected created items: %+v, %+v, %+v", items[0].Object, items[1].Object, items[2].Object)
		}
	})

	t.Run("Add_CreateParentWithZeroInlineChildrenWhenMinNumZero", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("No Inline Items", "NOITEMS0"),
			inlineMgmtValues("Items", 0, 0, 0, 5),
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)
		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0")
		}
		cnt, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).Count()
		if err != nil {
			t.Fatalf("count coupon items: %v", err)
		}
		if cnt != 0 {
			t.Errorf("expected 0 items, got %d", cnt)
		}
	})

	t.Run("Add_CreateParentSkippingDeletedInlineChildInSamePost", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Skip Item", "SKIPITEM1"),
			inlineMgmtValues("Items", 2, 0, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {""},
				"Items-0-Quantity":                        {"1"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"on"},
				"Items-1-ProductName":                     {"Mechanical Keycaps"},
				"Items-1-Quantity":                        {"3"},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)

		items, err := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).All()
		if err != nil {
			t.Fatalf("query items: %v", err)
		}
		if len(items) != 1 || items[0].Object.ProductName != "Mechanical Keycaps" || items[0].Object.Quantity != 3 {
			t.Fatalf("expected single kept item Mechanical Keycaps x3, got %+v", items)
		}
	})

	t.Run("Add_CreateParentWithOrderedInlineChildrenAndCustomSaveInstance", func(t *testing.T) {
		setupFormsTestDB(t)
		var savedOrder []string
		panels := buildCouponOneToManyPanels(1, 5, 0, func(p *admin.ModelFormPanel[*FormTestCouponItem, modelforms.ModelForm[*FormTestCouponItem]]) {
			p.Form = func() modelforms.ModelForm[*FormTestCouponItem] {
				f := modelforms.NewBaseModelForm(context.Background(), &FormTestCouponItem{})
				f.SetValidators(func(frm forms.Form, m map[string]any) []error {
					if qty, _ := m["Quantity"].(int); qty <= 0 {
						return []error{errors.New("quantity must be positive")}
					}
					return nil
				})
				return f
			}
			p.SaveInstance = func(ctx context.Context, form modelforms.ModelForm[*FormTestCouponItem], inst *FormTestCouponItem) error {
				savedOrder = append(savedOrder, inst.ProductName)
				inst.ProductName = strings.ToUpper(inst.ProductName)
				return inst.Save(ctx)
			}
		})

		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Ordered Custom", "ORDCUST1"),
			inlineMgmtValues("Items", 2, 0, 1, 5),
			url.Values{
				"Items-0-ProductName":                     {"second-item"},
				"Items-0-Quantity":                        {"2"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"first-item"},
				"Items-1-Quantity":                        {"1"},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, panels...)

		if !slices.Equal(savedOrder, []string{"first-item", "second-item"}) {
			t.Errorf("expected SaveInstance order [first-item, second-item], got %v", savedOrder)
		}
		items, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if len(items) != 2 || items[0].Object.ProductName != "FIRST-ITEM" || items[1].Object.ProductName != "SECOND-ITEM" {
			t.Errorf("unexpected persisted items: %+v", items)
		}
	})

	t.Run("Add_CreateParentFailsOnChildValidationError", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Child Invalid", "CHLDINV"),
			inlineMgmtValues("Items", 1, 0, 1, 5),
			url.Values{
				"Items-0-ProductName":                     {""}, // Required
				"Items-0-Quantity":                        {"2"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af := submitExpectInvalidAdminForm(t, coupon, data, buildCouponOneToManyPanels(1, 5, 0)...)
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered in HTML")
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCoupon{}).Count(); cnt != 0 {
			t.Errorf("expected 0 coupons in DB, got %d", cnt)
		}
	})

	t.Run("Add_CreateParentFailsWhenTotalFormsBelowMinNumOrAboveMaxNum", func(t *testing.T) {
		setupFormsTestDB(t)
		panels := buildCouponOneToManyPanels(2, 3, 0)

		below := &FormTestCoupon{}
		submitExpectInvalidAdminForm(t, below, mergeFormValues(
			baseCouponPOST("Below Min Items", "O2MBELOW"),
			inlineMgmtValues("Items", 1, 0, 2, 3),
			url.Values{
				"Items-0-ProductName":                     {"Only One"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		above := &FormTestCoupon{}
		submitExpectInvalidAdminForm(t, above, mergeFormValues(
			baseCouponPOST("Above Max Items", "O2MABOVE"),
			inlineMgmtValues("Items", 4, 0, 2, 3),
			url.Values{
				"Items-0-ProductName":                     {"A"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"B"},
				"Items-1-Quantity":                        {"1"},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-2-ProductName":                     {"C"},
				"Items-2-Quantity":                        {"1"},
				"Items-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Items-2-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-3-ProductName":                     {"D"},
				"Items-3-Quantity":                        {"1"},
				"Items-3-" + formsets.ORDERING_FIELD_NAME: {"3"},
				"Items-3-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		if cnt, _ := queries.GetQuerySet(&FormTestCoupon{}).Count(); cnt != 0 {
			t.Errorf("expected 0 coupons persisted on min/max error, got %d", cnt)
		}
	})

	t.Run("Update_EditUnchangedParentAndChildrenReportsNoChanges", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Unchanged O2M", Description: "Promo description", Code: "UNCHO2M", MaxDiscount: 15, ValidFrom: "2026-01-01", ValidUntil: "2026-12-31"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Monitor", Quantity: 1}
		item2 := &FormTestCouponItem{Coupon: coupon, ProductName: "Stand", Quantity: 2}
		if _, err := queries.GetQuerySet(&FormTestCouponItem{}).Create(item1); err != nil {
			t.Fatalf("create item1: %v", err)
		}
		if _, err := queries.GetQuerySet(&FormTestCouponItem{}).Create(item2); err != nil {
			t.Fatalf("create item2: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/test/edit", nil)
		af := newTestAdminForm(req, coupon, nil, buildCouponOneToManyPanels(0, 5, 0)...)
		af.WithData(mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Items", 2, 2, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {"Monitor"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"Stand"},
				"Items-1-Quantity":                        {"2"},
				"Items-1-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid unchanged form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if af.HasChanged() {
			t.Errorf("expected HasChanged() == false")
		}
	})

	t.Run("Update_EditExistingInlineChildrenInPlace", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Edit Children", Code: "EDCHLD"}
		if _, err := queries.GetQuerySet(&FormTestCoupon{}).Create(coupon); err != nil {
			t.Fatalf("create coupon: %v", err)
		}
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Desk Mat", Quantity: 1}
		item2 := &FormTestCouponItem{Coupon: coupon, ProductName: "Wrist Rest", Quantity: 1}
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item1)
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item2)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Items", 2, 2, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {"Desk Mat"},
				"Items-0-Quantity":                        {"5"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"Leather Wrist Rest"},
				"Items-1-Quantity":                        {"1"},
				"Items-1-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)

		got1, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("ID", item1.ID).Get()
		got2, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("ID", item2.ID).Get()
		if got1.Object.Quantity != 5 {
			t.Errorf("expected item1.Quantity == 5, got %d", got1.Object.Quantity)
		}
		if got2.Object.ProductName != "Leather Wrist Rest" {
			t.Errorf("expected item2.ProductName == 'Leather Wrist Rest', got %q", got2.Object.ProductName)
		}
		cnt, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).Count()
		if cnt != 2 {
			t.Errorf("expected 2 items total, got %d", cnt)
		}
	})

	t.Run("Update_EditParentAddingNewInlineChildrenToExistingSet", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Add Children", Code: "ADDCHLD"}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Router", Quantity: 1}
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item1)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Items", 3, 1, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {"Router"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"Switch"},
				"Items-1-Quantity":                        {"2"},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-2-ProductName":                     {"Patch Cable"},
				"Items-2-Quantity":                        {"6"},
				"Items-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Items-2-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)

		items, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if len(items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(items))
		}
		if items[1].Object.ProductName != "Switch" || items[2].Object.ProductName != "Patch Cable" || items[2].Object.Quantity != 6 {
			t.Errorf("unexpected added items: %+v, %+v", items[1].Object, items[2].Object)
		}
	})

	t.Run("Update_EditParentDeletingSpecificInlineChild", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Delete Child", Code: "DELCHLD"}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "SSD", Quantity: 1}
		item2 := &FormTestCouponItem{Coupon: coupon, ProductName: "HDD", Quantity: 2}
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item1)
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item2)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Items", 2, 2, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {"SSD"},
				"Items-0-Quantity":                        {"1"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"on"},
				"Items-1-ProductName":                     {"HDD"},
				"Items-1-Quantity":                        {"2"},
				"Items-1-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)

		remaining, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).All()
		if len(remaining) != 1 || remaining[0].Object.ID != item2.ID {
			t.Errorf("expected only HDD (ID=%d) remaining, got %+v", item2.ID, remaining)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("ID", item1.ID).Count(); cnt != 0 {
			t.Errorf("expected SSD row deleted from DB, count=%d", cnt)
		}
	})

	t.Run("Update_EditParentSimultaneouslyUpdateAddAndDeleteChildren", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Combined O2M", Code: "COMBO2M"}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Old A", Quantity: 1}
		item2 := &FormTestCouponItem{Coupon: coupon, ProductName: "Delete B", Quantity: 2}
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item1)
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item2)

		data := mergeFormValues(
			baseCouponPOST("Combined O2M Edited", coupon.Code),
			inlineMgmtValues("Items", 3, 2, 0, 5),
			url.Values{
				"Items-0-ProductName":                     {"Updated A"},
				"Items-0-Quantity":                        {"10"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Items-1-ProductName":                     {"Delete B"},
				"Items-1-Quantity":                        {"2"},
				"Items-1-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Items-1-" + formsets.DELETION_FIELD_NAME: {"on"},
				"Items-2-ProductName":                     {"Added C"},
				"Items-2-Quantity":                        {"7"},
				"Items-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
				"Items-2-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponOneToManyPanels(0, 5, 0)...)

		gotCoupon, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		if gotCoupon.Object.Title != "Combined O2M Edited" {
			t.Errorf("expected updated parent title 'Combined O2M Edited', got %q", gotCoupon.Object.Title)
		}
		items, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("Coupon", coupon.ID).OrderBy("ID").All()
		if len(items) != 2 {
			t.Fatalf("expected 2 items after update+delete+add, got %d", len(items))
		}
		if items[0].Object.ID != item1.ID || items[0].Object.ProductName != "Updated A" || items[0].Object.Quantity != 10 {
			t.Errorf("unexpected items[0]: %+v", items[0].Object)
		}
		if items[1].Object.ProductName != "Added C" || items[1].Object.Quantity != 7 {
			t.Errorf("unexpected items[1]: %+v", items[1].Object)
		}
	})

	t.Run("Update_EditParentFailsOnInvalidChildPreservingExistingDBState", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Keep Coupon", Code: "KEEPC1"}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)
		item1 := &FormTestCouponItem{Coupon: coupon, ProductName: "Original Item", Quantity: 2}
		_, _ = queries.GetQuerySet(&FormTestCouponItem{}).Create(item1)

		data := mergeFormValues(
			baseCouponPOST("Should Not Save", coupon.Code),
			inlineMgmtValues("Items", 1, 1, 1, 5),
			url.Values{
				"Items-0-ProductName":                     {""}, // Required
				"Items-0-Quantity":                        {"9"},
				"Items-0-Coupon":                          {fmt.Sprint(coupon.ID)},
				"Items-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Items-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af := submitExpectInvalidAdminForm(t, coupon, data, buildCouponOneToManyPanels(1, 5, 0)...)
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`input[name="Title"][value="Should Not Save"]`).Length() != 1 {
			t.Errorf("expected bound parent Title preserved in re-rendered HTML")
		}
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered in HTML")
		}

		dbCoupon, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		dbItem, _ := queries.GetQuerySet(&FormTestCouponItem{}).Filter("ID", item1.ID).Get()
		if dbCoupon.Object.Title != "Keep Coupon" || dbItem.Object.ProductName != "Original Item" || dbItem.Object.Quantity != 2 {
			t.Errorf("expected DB state untouched, got coupon=%+v item=%+v", dbCoupon.Object, dbItem.Object)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 17. ModelFormPanel RelManyToOne Create & Update Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_ModelFormPanelManyToOne(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("Add_CreateParentWithNewInlineManyToOneTargetInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Peripherals Promo", "PERIPH1"),
			url.Values{
				"Category-0-Name":                            {"Peripherals"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(1, 1, 0)...)

		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0 after save")
		}
		cats, err := queries.GetQuerySet(&FormTestCategory{}).All()
		if err != nil {
			t.Fatalf("query categories: %v", err)
		}
		if len(cats) != 1 || cats[0].Object.Name != "Peripherals" {
			t.Fatalf("expected 1 created category 'Peripherals', got %+v", cats)
		}
		reloaded, err := queries.GetQuerySet(&FormTestCoupon{}).Select("*", "Category.*").Filter("ID", coupon.ID).Get()
		if err != nil {
			t.Fatalf("reload coupon: %v", err)
		}
		if reloaded.Object.Category == nil || reloaded.Object.Category.ID != cats[0].Object.ID || reloaded.Object.Category.Name != "Peripherals" {
			t.Errorf("expected coupon.Category linked to 'Peripherals' (ID=%d), got %+v", cats[0].Object.ID, reloaded.Object.Category)
		}
	})

	t.Run("Add_CreateParentWithNoManyToOneTargetWhenMinNumZero", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Uncategorized Coupon", "UNCAT1"),
			inlineMgmtValues("Category", 0, 0, 0, 1),
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(0, 1, 0)...)

		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0")
		}
		reloaded, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		if reloaded.Object.Category != nil {
			t.Errorf("expected nil Category, got %+v", reloaded.Object.Category)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Count(); cnt != 0 {
			t.Errorf("expected 0 categories in DB, got %d", cnt)
		}
	})

	t.Run("Add_CreateParentWithDeletedManyToOneSlotInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Deleted FK Slot", "DELFK1"),
			inlineMgmtValues("Category", 1, 0, 0, 1),
			url.Values{
				"Category-0-Name":                            {"X"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"on"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(0, 1, 0)...)

		if coupon.ID == 0 {
			t.Fatalf("expected coupon.ID > 0")
		}
		reloaded, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		if reloaded.Object.Category != nil {
			t.Errorf("expected nil Category when inline slot marked deleted, got %+v", reloaded.Object.Category)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Count(); cnt != 0 {
			t.Errorf("expected 0 categories created, got %d", cnt)
		}
	})

	t.Run("Add_CreateParentWithCustomFormAndSaveInstanceOnManyToOnePanel", func(t *testing.T) {
		setupFormsTestDB(t)
		var hookCalled bool
		panels := buildCouponManyToOnePanels(1, 1, 0, func(p *admin.ModelFormPanel[*FormTestCategory, modelforms.ModelForm[*FormTestCategory]]) {
			p.Form = func() modelforms.ModelForm[*FormTestCategory] {
				f := modelforms.NewBaseModelForm(context.Background(), &FormTestCategory{})
				f.SetValidators(func(frm forms.Form, m map[string]any) []error {
					if n, _ := m["Name"].(string); strings.Contains(n, "forbidden") {
						return []error{errors.New("category name contains forbidden word")}
					}
					return nil
				})
				return f
			}
			p.SaveInstance = func(ctx context.Context, form modelforms.ModelForm[*FormTestCategory], inst *FormTestCategory) error {
				hookCalled = true
				inst.Name = strings.ToUpper(strings.TrimSpace(inst.Name))
				return inst.Save(ctx)
			}
		})

		badCoupon := &FormTestCoupon{}
		submitExpectInvalidAdminForm(t, badCoupon, mergeFormValues(
			baseCouponPOST("Bad Cat", "BADCAT1"),
			url.Values{
				"Category-0-Name":                            {"forbidden dept"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)
		if hookCalled {
			t.Fatalf("did not expect SaveInstance on invalid form")
		}

		goodCoupon := &FormTestCoupon{}
		submitAndSaveAdminForm(t, goodCoupon, mergeFormValues(
			baseCouponPOST("Good Cat", "GOODCAT1"),
			url.Values{
				"Category-0-Name":                            {"  audio gear  "},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)
		if !hookCalled {
			t.Fatalf("expected SaveInstance to be called")
		}
		reloaded, _ := queries.GetQuerySet(&FormTestCoupon{}).Select("*", "Category.*").Filter("ID", goodCoupon.ID).Get()
		if reloaded.Object.Category == nil || reloaded.Object.Category.Name != "AUDIO GEAR" {
			t.Errorf("expected normalized Category 'AUDIO GEAR', got %+v", reloaded.Object.Category)
		}
	})

	t.Run("Add_CreateParentFailsOnManyToOneTargetValidationError", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Invalid Cat", "INVCAT1"),
			url.Values{
				"Category-0-Name":                            {"A"}, // MinLength is 2
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		af := submitExpectInvalidAdminForm(t, coupon, data, buildCouponManyToOnePanels(1, 1, 0)...)
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error in HTML")
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCoupon{}).Count(); cnt != 0 {
			t.Errorf("expected 0 coupons created, got %d", cnt)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Count(); cnt != 0 {
			t.Errorf("expected 0 categories created, got %d", cnt)
		}
	})

	t.Run("Add_CreateParentFailsWhenManyToOneExceedsClampedMaxNumOne", func(t *testing.T) {
		setupFormsTestDB(t)
		// MinNum=0, MaxNum=5, Extra=3 clamps to MinNum=0, MaxNum=1, Extra=0 with ManagementForm enforcing MaxNum=1
		panels := buildCouponManyToOnePanels(0, 5, 3)
		coupon := &FormTestCoupon{}
		data := mergeFormValues(
			baseCouponPOST("Two Cats Invalid", "TWOCAT1"),
			inlineMgmtValues("Category", 2, 0, 0, 1),
			url.Values{
				"Category-0-Name":                            {"First"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Category-1-Name":                            {"Second"},
				"Category-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Category-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitExpectInvalidAdminForm(t, coupon, data, panels...)
		catPanel := panels[len(panels)-1].(*admin.ModelFormPanel[*FormTestCategory, modelforms.ModelForm[*FormTestCategory]])
		if catPanel.MinNum != 0 || catPanel.MaxNum != 1 || catPanel.Extra != 0 {
			t.Errorf("expected clamped bounds MinNum=0 MaxNum=1 Extra=0, got Min=%d Max=%d Extra=%d",
				catPanel.MinNum, catPanel.MaxNum, catPanel.Extra)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Count(); cnt != 0 {
			t.Errorf("expected 0 categories created, got %d", cnt)
		}
	})

	t.Run("Update_EditParentWithUnchangedManyToOneTargetHasChangedFalse", func(t *testing.T) {
		setupFormsTestDB(t)
		cat := &FormTestCategory{Name: "Monitors"}
		_, _ = queries.GetQuerySet(&FormTestCategory{}).Create(cat)
		coupon := &FormTestCoupon{Title: "Mon Deal", Description: "Promo description", Code: "MONDEAL", MaxDiscount: 15, ValidFrom: "2026-01-01", ValidUntil: "2026-12-31", Category: cat}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		req := httptest.NewRequest(http.MethodPost, "/admin/test/edit", nil)
		af := newTestAdminForm(req, coupon, nil, buildCouponManyToOnePanels(1, 1, 0)...)
		af.WithData(mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			url.Values{
				"Category-0-Name":                            {"Monitors"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid unchanged form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if af.HasChanged() {
			t.Errorf("expected HasChanged() == false on identical coupon + category data")
		}
	})

	t.Run("Update_EditExistingManyToOneTargetInPlace", func(t *testing.T) {
		setupFormsTestDB(t)
		cat := &FormTestCategory{Name: "Storage"}
		_, _ = queries.GetQuerySet(&FormTestCategory{}).Create(cat)
		coupon := &FormTestCoupon{Title: "Drive Sale", Code: "DRVSALE", Category: cat}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			url.Values{
				"Category-0-Name":                            {"NVMe Storage"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(1, 1, 0)...)

		gotCat, _ := queries.GetQuerySet(&FormTestCategory{}).Filter("ID", cat.ID).Get()
		if gotCat.Object.Name != "NVMe Storage" {
			t.Errorf("expected in-place update to 'NVMe Storage', got %q", gotCat.Object.Name)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Count(); cnt != 1 {
			t.Errorf("expected 1 category row in DB, got %d", cnt)
		}
	})

	t.Run("Update_EditParentWithNilManyToOneAddingNewTarget", func(t *testing.T) {
		setupFormsTestDB(t)
		coupon := &FormTestCoupon{Title: "Add Cat Later", Code: "ADDCATL"}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Category", 1, 0, 0, 1),
			url.Values{
				"Category-0-Name":                            {"Audio"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(0, 1, 0)...)

		reloaded, err := queries.GetQuerySet(&FormTestCoupon{}).Select("*", "Category.*").Filter("ID", coupon.ID).Get()
		if err != nil {
			t.Fatalf("reload coupon: %v", err)
		}
		if reloaded.Object.Category == nil || reloaded.Object.Category.Name != "Audio" {
			t.Errorf("expected coupon linked to new Category 'Audio', got %+v", reloaded.Object.Category)
		}
	})

	t.Run("Update_EditParentDeletingLinkedManyToOneTarget", func(t *testing.T) {
		setupFormsTestDB(t)
		cat := &FormTestCategory{Name: "Legacy Cat"}
		_, _ = queries.GetQuerySet(&FormTestCategory{}).Create(cat)
		coupon := &FormTestCoupon{Title: "Drop Cat", Code: "DROPCAT1", Category: cat}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		data := mergeFormValues(
			baseCouponPOST(coupon.Title, coupon.Code),
			inlineMgmtValues("Category", 1, 1, 0, 1),
			url.Values{
				"Category-0-Name":                            {"Legacy Cat"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"on"},
			},
		)
		submitAndSaveAdminForm(t, coupon, data, buildCouponManyToOnePanels(0, 1, 0)...)

		reloaded, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		if reloaded.Object.Category != nil {
			t.Errorf("expected reloaded coupon.Category == nil after delete, got %+v", reloaded.Object.Category)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestCategory{}).Filter("ID", cat.ID).Count(); cnt != 0 {
			t.Errorf("expected deleted category row removed from DB, count=%d", cnt)
		}
	})

	t.Run("Update_EditParentLoadedFromDBWithoutSelectRelatedAndUpdateBothParentAndTarget", func(t *testing.T) {
		setupFormsTestDB(t)
		cat := &FormTestCategory{Name: "Networking"}
		_, _ = queries.GetQuerySet(&FormTestCategory{}).Create(cat)
		coupon := &FormTestCoupon{Title: "Old Net Title", Code: "NETCODE", Category: cat}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		// Fetch WITHOUT Select("Category.*")
		row, err := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		if err != nil {
			t.Fatalf("fetch coupon without SelectRelated: %v", err)
		}
		loaded := row.Object

		// Verify GET form hydrates target fields from DB
		getReq := httptest.NewRequest(http.MethodGet, "/admin/test/edit", nil)
		afGet := newTestAdminForm(getReq, loaded, nil, buildCouponManyToOnePanels(1, 1, 0)...)
		getDoc := parseHTMLDoc(t, afGet.BoundForm().AsP())
		if getDoc.Find(`input[name="Category-0-Name"][value="Networking"]`).Length() != 1 {
			t.Errorf("expected hydrated Category-0-Name input value 'Networking' without SelectRelated")
		}

		// POST update to both parent and target
		row2, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		submitAndSaveAdminForm(t, row2.Object, mergeFormValues(
			baseCouponPOST("New Net Title", coupon.Code),
			url.Values{
				"Category-0-Name":                            {"Pro Networking"},
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), buildCouponManyToOnePanels(1, 1, 0)...)

		gotCoupon, _ := queries.GetQuerySet(&FormTestCoupon{}).Select("*", "Category.*").Filter("ID", coupon.ID).Get()
		if gotCoupon.Object.Title != "New Net Title" {
			t.Errorf("expected Title 'New Net Title', got %q", gotCoupon.Object.Title)
		}
		if gotCoupon.Object.Category == nil || gotCoupon.Object.Category.Name != "Pro Networking" {
			t.Errorf("expected Category 'Pro Networking', got %+v", gotCoupon.Object.Category)
		}
	})

	t.Run("Update_EditParentFailsOnInvalidManyToOneTargetPreservingDBState", func(t *testing.T) {
		setupFormsTestDB(t)
		cat := &FormTestCategory{Name: "Accessories"}
		_, _ = queries.GetQuerySet(&FormTestCategory{}).Create(cat)
		coupon := &FormTestCoupon{Title: "Good Title", Code: "GOODM2O", Category: cat}
		_, _ = queries.GetQuerySet(&FormTestCoupon{}).Create(coupon)

		af := submitExpectInvalidAdminForm(t, coupon, mergeFormValues(
			baseCouponPOST("Changed Title", coupon.Code),
			url.Values{
				"Category-0-Name":                            {"Z"}, // MinLength is 2
				"Category-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Category-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), buildCouponManyToOnePanels(1, 1, 0)...)
		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`input[name="Category-0-Name"][value="Z"]`).Length() != 1 {
			t.Errorf("expected invalid value 'Z' preserved in HTML")
		}
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered in HTML")
		}

		dbCoupon, _ := queries.GetQuerySet(&FormTestCoupon{}).Filter("ID", coupon.ID).Get()
		dbCat, _ := queries.GetQuerySet(&FormTestCategory{}).Filter("ID", cat.ID).Get()
		if dbCoupon.Object.Title != "Good Title" || dbCat.Object.Name != "Accessories" {
			t.Errorf("expected DB untouched, got coupon=%+v cat=%+v", dbCoupon.Object, dbCat.Object)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 18. ModelFormPanel RelOneToOne (Direct FK & Through Model) Create & Update Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminForm_ModelFormPanelOneToOne(t *testing.T) {
	setupFormsTestDB(t)

	t.Run("Add_CreateAuthorWithDirectOneToOneBioInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{}
		panels := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, false, 0, 0, 0, nil)
		data := mergeFormValues(
			baseAuthorPOST("Ada Lovelace", "ada@example.com"),
			url.Values{
				"Bio-0-Headline":                             {"Analytical Pioneer"},
				"Bio-0-Website":                              {"https://ada.example"},
				"Bio-0-ExperienceYears":                      {"12"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:      {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:      {"false"},
			},
		)
		submitAndSaveAdminForm(t, author, data, panels...)

		if author.ID == 0 {
			t.Fatalf("expected author.ID > 0")
		}
		bios, _ := queries.GetQuerySet(&FormTestAuthorBio{}).All()
		if len(bios) != 1 || bios[0].Object.Headline != "Analytical Pioneer" || bios[0].Object.ExperienceYears != 12 {
			t.Fatalf("expected 1 created FormTestAuthorBio, got %+v", bios)
		}
		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Select("*", "Bio.*").Filter("ID", author.ID).Get()
		if reloaded.Object.Bio == nil || reloaded.Object.Bio.ID != bios[0].Object.ID || reloaded.Object.Bio.Headline != "Analytical Pioneer" {
			t.Errorf("expected author.Bio linked to bio %d, got %+v", bios[0].Object.ID, reloaded.Object.Bio)
		}
	})

	t.Run("Add_CreateAuthorWithThroughOneToOneBadgeInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{}
		panels := buildAuthorOneToOnePanels(false, 0, 0, 0, nil, true, 1, 1, 0, nil)
		data := mergeFormValues(
			baseAuthorPOST("Grace Hopper", "grace@example.com"),
			url.Values{
				"Badge-0-BadgeName":                          {"Compiler Architect"},
				"Badge-0-Tier":                               {"Platinum"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME:    {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME:    {"false"},
			},
		)
		submitAndSaveAdminForm(t, author, data, panels...)

		if author.ID == 0 {
			t.Fatalf("expected author.ID > 0")
		}
		row, err := author.Badge.Objects().Get()
		if err != nil {
			t.Fatalf("author.Badge.Objects().Get() failed: %v", err)
		}
		if row.Object.BadgeName != "Compiler Architect" || row.Object.Tier != "Platinum" {
			t.Errorf("unexpected linked badge: %+v", row.Object)
		}
	})

	t.Run("Add_CreateAuthorWithBothDirectBioAndThroughBadgeInSinglePost", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{}
		panels := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, true, 1, 1, 0, nil)
		data := mergeFormValues(
			baseAuthorPOST("Niklaus Wirth", "wirth@example.com"),
			url.Values{
				"Bio-0-Headline":                             {"Pascal Creator"},
				"Bio-0-Website":                              {"https://pascal.example"},
				"Bio-0-ExperienceYears":                      {"40"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:      {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:      {"false"},
				"Badge-0-BadgeName":                          {"Turing Laureate"},
				"Badge-0-Tier":                               {"Diamond"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME:    {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME:    {"false"},
			},
		)
		submitAndSaveAdminForm(t, author, data, panels...)

		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Select("*", "Bio.*").Filter("ID", author.ID).Get()
		if reloaded.Object.Bio == nil || reloaded.Object.Bio.Headline != "Pascal Creator" {
			t.Errorf("expected direct Bio 'Pascal Creator', got %+v", reloaded.Object.Bio)
		}
		badgeRow, err := author.Badge.Objects().Get()
		if err != nil {
			t.Fatalf("expected linked Badge via through table: %v", err)
		}
		if badgeRow.Object.BadgeName != "Turing Laureate" || badgeRow.Object.Tier != "Diamond" {
			t.Errorf("unexpected through Badge: %+v", badgeRow.Object)
		}
	})

	t.Run("Add_CreateAuthorWithZeroOneToOneTargetsWhenMinNumZero", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{}
		panels := buildAuthorOneToOnePanels(true, 0, 1, 0, nil, true, 0, 1, 0, nil)
		data := mergeFormValues(
			baseAuthorPOST("Minimal Author", "min@example.com"),
			inlineMgmtValues("Bio", 0, 0, 0, 1),
			inlineMgmtValues("Badge", 0, 0, 0, 1),
		)
		submitAndSaveAdminForm(t, author, data, panels...)

		if author.ID == 0 {
			t.Fatalf("expected author.ID > 0")
		}
		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Filter("ID", author.ID).Get()
		if reloaded.Object.Bio != nil {
			t.Errorf("expected nil Bio, got %+v", reloaded.Object.Bio)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBio{}).Count(); cnt != 0 {
			t.Errorf("expected 0 bios, got %d", cnt)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBadge{}).Count(); cnt != 0 {
			t.Errorf("expected 0 badges, got %d", cnt)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBadgeThrough{}).Count(); cnt != 0 {
			t.Errorf("expected 0 badge links, got %d", cnt)
		}
	})

	t.Run("Add_CreateAuthorWithCustomFormAndSaveInstanceOnOneToOnePanel", func(t *testing.T) {
		setupFormsTestDB(t)
		var bioHook, badgeHook bool
		panels := buildAuthorOneToOnePanels(
			true, 1, 1, 0,
			[]func(*admin.ModelFormPanel[*FormTestAuthorBio, modelforms.ModelForm[*FormTestAuthorBio]]){
				func(p *admin.ModelFormPanel[*FormTestAuthorBio, modelforms.ModelForm[*FormTestAuthorBio]]) {
					p.Form = func() modelforms.ModelForm[*FormTestAuthorBio] {
						return modelforms.NewBaseModelForm(context.Background(), &FormTestAuthorBio{})
					}
					p.SaveInstance = func(ctx context.Context, form modelforms.ModelForm[*FormTestAuthorBio], inst *FormTestAuthorBio) error {
						bioHook = true
						inst.Website = strings.TrimSpace(strings.ToLower(inst.Website))
						return inst.Save(ctx)
					}
				},
			},
			true, 1, 1, 0,
			[]func(*admin.ModelFormPanel[*FormTestAuthorBadge, modelforms.ModelForm[*FormTestAuthorBadge]]){
				func(p *admin.ModelFormPanel[*FormTestAuthorBadge, modelforms.ModelForm[*FormTestAuthorBadge]]) {
					p.Form = func() modelforms.ModelForm[*FormTestAuthorBadge] {
						return modelforms.NewBaseModelForm(context.Background(), &FormTestAuthorBadge{})
					}
					p.SaveInstance = func(ctx context.Context, form modelforms.ModelForm[*FormTestAuthorBadge], inst *FormTestAuthorBadge) error {
						badgeHook = true
						inst.Tier = strings.ToUpper(inst.Tier)
						return inst.Save(ctx)
					}
				},
			},
		)

		author := &FormTestAuthor{}
		data := mergeFormValues(
			baseAuthorPOST("Custom Hooks", "hooks@example.com"),
			url.Values{
				"Bio-0-Headline":                          {"Systems Engineer"},
				"Bio-0-Website":                           {"  HTTPS://SYS.EXAMPLE  "},
				"Bio-0-ExperienceYears":                   {"8"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-0-BadgeName":                       {"Contributor"},
				"Badge-0-Tier":                            {"silver"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		)
		submitAndSaveAdminForm(t, author, data, panels...)

		if !bioHook || !badgeHook {
			t.Fatalf("expected both bioHook (%v) and badgeHook (%v) called", bioHook, badgeHook)
		}
		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Select("*", "Bio.*").Filter("ID", author.ID).Get()
		if reloaded.Object.Bio == nil || reloaded.Object.Bio.Website != "https://sys.example" {
			t.Errorf("expected normalized Website 'https://sys.example', got %+v", reloaded.Object.Bio)
		}
		badgeRow, _ := author.Badge.Objects().Get()
		if badgeRow.Object.Tier != "SILVER" {
			t.Errorf("expected uppercased Tier 'SILVER', got %+v", badgeRow.Object)
		}
	})

	t.Run("Add_CreateAuthorFailsOnOneToOneValidationAndClampedMaxNum", func(t *testing.T) {
		setupFormsTestDB(t)
		// Verify PRE/POST ClusterOrderableForm ordering for direct vs through O2O
		panelsOrder := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, true, 0, 5, 0, nil)
		bioPanel := panelsOrder[2].(*admin.ModelFormPanel[*FormTestAuthorBio, modelforms.ModelForm[*FormTestAuthorBio]])
		badgePanel := panelsOrder[3].(*admin.ModelFormPanel[*FormTestAuthorBadge, modelforms.ModelForm[*FormTestAuthorBadge]])

		_, bioForms, err := bioPanel.Forms(httptest.NewRequest(http.MethodGet, "/", nil), context.Background(), &FormTestAuthor{})
		if err != nil {
			t.Fatalf("bioPanel.Forms failed: %v", err)
		}
		if bioForms[0].(admin.ClusterOrderableForm).FormOrder() != admin.FORM_ORDERING_PRE {
			t.Errorf("expected direct O2O Bio FormOrder PRE, got %v", bioForms[0].(admin.ClusterOrderableForm).FormOrder())
		}
		_, badgeForms, err := badgePanel.Forms(httptest.NewRequest(http.MethodGet, "/", nil), context.Background(), &FormTestAuthor{})
		if err != nil {
			t.Fatalf("badgePanel.Forms failed: %v", err)
		}
		if badgeForms[0].(admin.ClusterOrderableForm).FormOrder() != admin.FORM_ORDERING_POST {
			t.Errorf("expected through O2O Badge FormOrder POST, got %v", badgeForms[0].(admin.ClusterOrderableForm).FormOrder())
		}
		if badgePanel.MaxNum != 1 {
			t.Errorf("expected through O2O Badge MaxNum clamped to 1, got %d", badgePanel.MaxNum)
		}

		// Case A: Field MinLength validation failure on Bio.Headline
		badBioAuthor := &FormTestAuthor{}
		afBioErr := submitExpectInvalidAdminForm(t, badBioAuthor, mergeFormValues(
			baseAuthorPOST("Invalid Bio", "badbio@example.com"),
			url.Values{
				"Bio-0-Headline":                          {"Hi"}, // MinLength 3
				"Bio-0-Website":                           {""},
				"Bio-0-ExperienceYears":                   {"1"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-management-TOTAL_FORMS":            {"0"},
				"Badge-management-INITIAL_FORMS":          {"0"},
				"Badge-management-MIN_NUM_FORMS":          {"0"},
				"Badge-management-MAX_NUM_FORMS":          {"1"},
			},
		), panelsOrder...)
		if doc := parseHTMLDoc(t, afBioErr.BoundForm().AsP()); doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered for Bio validation failure")
		}

		// Case B: ManagementForm failure when submitting 2 badges against clamped MaxNum=1
		badBadgeAuthor := &FormTestAuthor{}
		submitExpectInvalidAdminForm(t, badBadgeAuthor, mergeFormValues(
			baseAuthorPOST("Too Many Badges", "badges@example.com"),
			inlineMgmtValues("Badge", 2, 0, 0, 1),
			url.Values{
				"Bio-0-Headline":                          {"Valid Headline"},
				"Bio-0-Website":                           {""},
				"Bio-0-ExperienceYears":                   {"3"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-0-BadgeName":                       {"First Badge"},
				"Badge-0-Tier":                            {"Gold"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
				"Badge-1-BadgeName":                       {"Second Badge"},
				"Badge-1-Tier":                            {"Silver"},
				"Badge-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
				"Badge-1-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panelsOrder...)

		if cnt, _ := queries.GetQuerySet(&FormTestAuthor{}).Count(); cnt != 0 {
			t.Errorf("expected 0 authors created on failed validation, got %d", cnt)
		}
	})

	t.Run("Update_EditAuthorWithUnchangedDirectAndThroughOneToOneHasChangedFalse", func(t *testing.T) {
		setupFormsTestDB(t)
		bio := &FormTestAuthorBio{Headline: "Steady Bio", Website: "https://steady.example", ExperienceYears: 7}
		_, _ = queries.GetQuerySet(&FormTestAuthorBio{}).Create(bio)
		author := &FormTestAuthor{Name: "Steady Author", Email: "steady@example.com", Bio: bio}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)
		badge := &FormTestAuthorBadge{BadgeName: "Mentor", Tier: "Gold"}
		_, _ = queries.GetQuerySet(&FormTestAuthorBadge{}).Create(badge)
		_, _, _ = author.Badge.Objects().SetTarget(badge)

		req := httptest.NewRequest(http.MethodPost, "/admin/test/edit", nil)
		panels := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, true, 1, 1, 0, nil)
		af := newTestAdminForm(req, author, nil, panels...)
		af.WithData(mergeFormValues(
			baseAuthorPOST(author.Name, author.Email),
			url.Values{
				"Bio-0-Headline":                          {bio.Headline},
				"Bio-0-Website":                           {bio.Website},
				"Bio-0-ExperienceYears":                   {"7"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-0-BadgeName":                       {badge.BadgeName},
				"Badge-0-Tier":                            {badge.Tier},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), nil, req)
		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected valid unchanged O2O form, errs=%v bound=%v", af.ErrorList(), af.BoundErrors())
		}
		if af.HasChanged() {
			t.Errorf("expected HasChanged() == false on identical author, bio, and badge")
		}
	})

	t.Run("Update_EditExistingDirectOneToOneBioInPlace", func(t *testing.T) {
		setupFormsTestDB(t)
		bio := &FormTestAuthorBio{Headline: "Draft Bio", Website: "https://old.example", ExperienceYears: 2}
		_, _ = queries.GetQuerySet(&FormTestAuthorBio{}).Create(bio)
		author := &FormTestAuthor{Name: "Bio Editor", Email: "bio@example.com", Bio: bio}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)

		panels := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, false, 0, 0, 0, nil)
		submitAndSaveAdminForm(t, author, mergeFormValues(
			baseAuthorPOST(author.Name, author.Email),
			url.Values{
				"Bio-0-Headline":                        {"Principal Architect"},
				"Bio-0-Website":                         {"https://new.example"},
				"Bio-0-ExperienceYears":                 {"15"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		gotBio, _ := queries.GetQuerySet(&FormTestAuthorBio{}).Filter("ID", bio.ID).Get()
		if gotBio.Object.Headline != "Principal Architect" || gotBio.Object.Website != "https://new.example" || gotBio.Object.ExperienceYears != 15 {
			t.Errorf("expected in-place bio update, got %+v", gotBio.Object)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBio{}).Count(); cnt != 1 {
			t.Errorf("expected 1 bio row in DB, got %d", cnt)
		}
	})

	t.Run("Update_EditExistingThroughOneToOneBadgeInPlace", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{Name: "Badge Editor", Email: "badge@example.com"}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)
		badge := &FormTestAuthorBadge{BadgeName: "Reviewer", Tier: "Bronze"}
		_, _ = queries.GetQuerySet(&FormTestAuthorBadge{}).Create(badge)
		_, _, _ = author.Badge.Objects().SetTarget(badge)

		// Fetch fresh author WITHOUT preloading Badge so ModelFormPanel queries through table
		loaded, err := queries.GetQuerySet(&FormTestAuthor{}).Filter("ID", author.ID).Get()
		if err != nil {
			t.Fatalf("load author: %v", err)
		}

		panels := buildAuthorOneToOnePanels(false, 0, 0, 0, nil, true, 1, 1, 0, nil)
		getReq := httptest.NewRequest(http.MethodGet, "/admin/test/edit", nil)
		afGet := newTestAdminForm(getReq, loaded.Object, nil, panels...)
		getDoc := parseHTMLDoc(t, afGet.BoundForm().AsP())
		if getDoc.Find(`input[name="Badge-0-BadgeName"][value="Reviewer"]`).Length() != 1 {
			t.Errorf("expected GET form to query through O2O and populate Badge-0-BadgeName='Reviewer'")
		}

		loaded2, _ := queries.GetQuerySet(&FormTestAuthor{}).Filter("ID", author.ID).Get()
		submitAndSaveAdminForm(t, loaded2.Object, mergeFormValues(
			baseAuthorPOST(author.Name, author.Email),
			url.Values{
				"Badge-0-BadgeName":                       {"Lead Reviewer"},
				"Badge-0-Tier":                            {"Platinum"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		gotBadge, _ := queries.GetQuerySet(&FormTestAuthorBadge{}).Filter("ID", badge.ID).Get()
		if gotBadge.Object.BadgeName != "Lead Reviewer" || gotBadge.Object.Tier != "Platinum" {
			t.Errorf("expected updated badge ID=%d, got %+v", badge.ID, gotBadge.Object)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBadge{}).Count(); cnt != 1 {
			t.Errorf("expected 1 badge row in DB, got %d", cnt)
		}
	})

	t.Run("Update_EditAuthorWithNilOneToOneTargetsAddingBioAndBadge", func(t *testing.T) {
		setupFormsTestDB(t)
		author := &FormTestAuthor{Name: "Fresh Author", Email: "fresh@example.com"}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)

		panels := buildAuthorOneToOnePanels(true, 0, 1, 0, nil, true, 0, 1, 0, nil)
		submitAndSaveAdminForm(t, author, mergeFormValues(
			baseAuthorPOST("Fresh Author", "fresh@example.com"),
			inlineMgmtValues("Bio", 1, 0, 0, 1),
			inlineMgmtValues("Badge", 1, 0, 0, 1),
			url.Values{
				"Bio-0-Headline":                          {"Newly Added Bio"},
				"Bio-0-Website":                           {"https://added.example"},
				"Bio-0-ExperienceYears":                   {"4"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-0-BadgeName":                       {"Rising Star"},
				"Badge-0-Tier":                            {"Silver"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Select("*", "Bio.*").Filter("ID", author.ID).Get()
		if reloaded.Object.Bio == nil || reloaded.Object.Bio.Headline != "Newly Added Bio" {
			t.Errorf("expected added Bio linked on author, got %+v", reloaded.Object.Bio)
		}
		badgeRow, err := author.Badge.Objects().Get()
		if err != nil {
			t.Fatalf("expected added Badge linked via through table: %v", err)
		}
		if badgeRow.Object.BadgeName != "Rising Star" {
			t.Errorf("expected Badge 'Rising Star', got %+v", badgeRow.Object)
		}
	})

	t.Run("Update_EditAuthorDeletingDirectBioAndThroughBadge", func(t *testing.T) {
		setupFormsTestDB(t)
		bio := &FormTestAuthorBio{Headline: "Removable Bio", ExperienceYears: 5}
		_, _ = queries.GetQuerySet(&FormTestAuthorBio{}).Create(bio)
		author := &FormTestAuthor{Name: "Cleanup Author", Email: "cleanup@example.com", Bio: bio}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)
		badge := &FormTestAuthorBadge{BadgeName: "Removable Badge", Tier: "Bronze"}
		_, _ = queries.GetQuerySet(&FormTestAuthorBadge{}).Create(badge)
		_, _, _ = author.Badge.Objects().SetTarget(badge)

		panels := buildAuthorOneToOnePanels(true, 0, 1, 0, nil, true, 0, 1, 0, nil)
		submitAndSaveAdminForm(t, author, mergeFormValues(
			baseAuthorPOST(author.Name, author.Email),
			inlineMgmtValues("Bio", 1, 1, 0, 1),
			inlineMgmtValues("Badge", 1, 1, 0, 1),
			url.Values{
				"Bio-0-Headline":                          {bio.Headline},
				"Bio-0-Website":                           {""},
				"Bio-0-ExperienceYears":                   {"5"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"on"},
				"Badge-0-BadgeName":                       {badge.BadgeName},
				"Badge-0-Tier":                            {badge.Tier},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"on"},
			},
		), panels...)

		reloaded, _ := queries.GetQuerySet(&FormTestAuthor{}).Filter("ID", author.ID).Get()
		if reloaded.Object.Bio != nil {
			t.Errorf("expected reloaded author.Bio == nil after delete, got %+v", reloaded.Object.Bio)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBio{}).Filter("ID", bio.ID).Count(); cnt != 0 {
			t.Errorf("expected bio deleted from DB, count=%d", cnt)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBadgeThrough{}).Count(); cnt != 0 {
			t.Errorf("expected through link deleted from DB, count=%d", cnt)
		}
		if cnt, _ := queries.GetQuerySet(&FormTestAuthorBadge{}).Filter("ID", badge.ID).Count(); cnt != 0 {
			t.Errorf("expected badge deleted from DB, count=%d", cnt)
		}
	})

	t.Run("Update_EditAuthorFailsOnInvalidOneToOneTargetPreservingDBState", func(t *testing.T) {
		setupFormsTestDB(t)
		bio := &FormTestAuthorBio{Headline: "Go Developer", ExperienceYears: 6}
		_, _ = queries.GetQuerySet(&FormTestAuthorBio{}).Create(bio)
		author := &FormTestAuthor{Name: "Alice", Email: "alice@example.com", Bio: bio}
		_, _ = queries.GetQuerySet(&FormTestAuthor{}).Create(author)
		badge := &FormTestAuthorBadge{BadgeName: "Core", Tier: "Gold"}
		_, _ = queries.GetQuerySet(&FormTestAuthorBadge{}).Create(badge)
		_, _, _ = author.Badge.Objects().SetTarget(badge)

		panels := buildAuthorOneToOnePanels(true, 1, 1, 0, nil, true, 1, 1, 0, nil)
		af := submitExpectInvalidAdminForm(t, author, mergeFormValues(
			baseAuthorPOST("Should Not Persist", "alice@example.com"),
			url.Values{
				"Bio-0-Headline":                          {"No"}, // MinLength 3
				"Bio-0-Website":                           {""},
				"Bio-0-ExperienceYears":                   {"6"},
				"Bio-0-" + formsets.ORDERING_FIELD_NAME:   {"0"},
				"Bio-0-" + formsets.DELETION_FIELD_NAME:   {"false"},
				"Badge-0-BadgeName":                       {"New Badge"},
				"Badge-0-Tier":                            {"Silver"},
				"Badge-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
				"Badge-0-" + formsets.DELETION_FIELD_NAME: {"false"},
			},
		), panels...)

		doc := parseHTMLDoc(t, af.BoundForm().AsP())
		if doc.Find(`input[name="Bio-0-Headline"][value="No"]`).Length() != 1 {
			t.Errorf("expected invalid Bio-0-Headline value 'No' preserved in HTML")
		}
		if doc.Find(`.panel__error`).Length() == 0 {
			t.Errorf("expected .panel__error rendered in HTML")
		}

		dbAuthor, _ := queries.GetQuerySet(&FormTestAuthor{}).Filter("ID", author.ID).Get()
		dbBio, _ := queries.GetQuerySet(&FormTestAuthorBio{}).Filter("ID", bio.ID).Get()
		dbBadge, _ := queries.GetQuerySet(&FormTestAuthorBadge{}).Filter("ID", badge.ID).Get()
		if dbAuthor.Object.Name != "Alice" || dbBio.Object.Headline != "Go Developer" || dbBadge.Object.BadgeName != "Core" {
			t.Errorf("expected DB state untouched, got author=%+v bio=%+v badge=%+v",
				dbAuthor.Object, dbBio.Object, dbBadge.Object)
		}
	})
}
