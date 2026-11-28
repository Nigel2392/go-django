//go:build test
// +build test

package categories_test

import (
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/models/categories"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	_ "github.com/Nigel2392/go-django/queries/src/fields/formfields"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

func TestMain(m *testing.M) {
	testing.Init()
	flag.Parse()

	var which, db = testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(
		django.Configure(settings),
		django.Flag(django.FlagSkipCmds, django.FlagSkipChecks),
	)

	logger.Debugf("Using %s database for shop/models tests", which)

	if !testing.Verbose() {
		logger.SetLevel(logger.WRN)
	}

	var tables = quest.Table[*testing.T](nil, &categories.Category{})
	tables.IfNotExists = true
	tables.Create()
	defer tables.Drop()

	_ = attrs.ResetDefinitions.Send(context.Background(), nil)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func clearCategories(t *testing.T) {
	t.Helper()
	// shop_category has ON DELETE RESTRICT on parent_id, so delete child rows before roots.
	if _, err := queries.GetQuerySet(&categories.Category{}).
		Filter("ParentCategory__isnull", false).
		Delete(); err != nil {
		t.Fatalf("failed to delete child Category rows: %v", err)
	}
	if _, err := queries.GetQuerySet(&categories.Category{}).Delete(); err != nil {
		t.Fatalf("failed to clear Category table: %v", err)
	}
}

func categoryAdminPanels() []admin.Panel {
	// Mirrors contrib/shop/app.go Category EditView panels.
	return []admin.Panel{
		admin.FieldPanel("Name"),
		admin.FieldPanel("Slug"),
		admin.FieldPanel("ParentCategory"),
		&admin.ModelFormPanel[*categories.Category, modelforms.ModelForm[*categories.Category]]{
			TargetType: &categories.Category{},
			FieldName:  "Children",
			Classname:  "collapsible",
			MinNum:     1,
			Panels: []admin.Panel{
				admin.FieldPanel("Name"),
				admin.FieldPanel("Slug"),
			},
		},
	}
}

func newCategoryAdminForm(req *http.Request, cat *categories.Category) *admin.AdminForm[modelforms.ModelForm[*categories.Category], *categories.Category] {
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/admin/shop/category", nil)
	}
	mf := modelforms.NewBaseModelForm(req.Context(), cat)
	var iface modelforms.ModelForm[*categories.Category] = mf
	af := admin.NewAdminForm(req, iface, categoryAdminPanels()...)
	af.Load()
	return af
}

func unwrapCategoryChildrenFormSet(t *testing.T, af *admin.AdminForm[modelforms.ModelForm[*categories.Category], *categories.Category]) formsets.ListFormSet[modelforms.ModelForm[*categories.Category]] {
	t.Helper()
	fs := af.FormSet()
	if fs == nil {
		t.Fatalf("expected non-nil FormSet() from Category AdminForm with Children ModelFormPanel")
	}
	subForms, err := fs.Forms()
	if err != nil {
		t.Fatalf("FormSet().Forms() failed: %v", err)
	}
	if len(subForms) != 1 {
		t.Fatalf("expected 1 panel formset in AdminForm.FormSet(), got %d", len(subForms))
	}
	if wrapper, ok := subForms[0].(interface {
		Form() formsets.BaseFormSetForm
	}); ok {
		subForms[0] = wrapper.Form()
	}
	childrenFS, ok := subForms[0].(formsets.ListFormSet[modelforms.ModelForm[*categories.Category]])
	if !ok {
		t.Fatalf("expected ListFormSet[ModelForm[*categories.Category]], got %T", subForms[0])
	}
	return childrenFS
}

func TestCategory_SelfReferentialRelationMeta(t *testing.T) {
	clearCategories(t)
	ctx := t.Context()

	defs := attrs.Define(ctx, &categories.Category{})
	childrenField, ok := defs.Field("Children")
	if !ok {
		t.Fatalf("expected Children field on Category")
	}

	rel := childrenField.Rel()
	if rel == nil {
		t.Fatalf("expected non-nil relation on Category.Children")
	}
	if rel.Type() != attrs.RelOneToMany {
		t.Errorf("Category.Children Rel().Type() = %v, want %v", rel.Type(), attrs.RelOneToMany)
	}
	if rel.Field() == nil {
		t.Fatalf("expected non-nil target field on Category.Children.Rel()")
	}
	if rel.Field().Name() != "ParentCategory" {
		t.Errorf("Category.Children Rel().Field().Name() = %q, want %q", rel.Field().Name(), "ParentCategory")
	}
	if rel.Field().IsPrimary() {
		t.Errorf("Category.Children Rel().Field().IsPrimary() = true, want false")
	}

	meta := attrs.GetModelMeta(&categories.Category{})
	revRel, ok := meta.Reverse("Children")
	if !ok || revRel == nil {
		t.Fatalf("expected reverse relation %q on Category meta", "Children")
	}
	if revRel.Field() == nil || revRel.Field().Name() != "ParentCategory" {
		t.Errorf("Category meta reverse %q Field() = %v, want ParentCategory", "Children", revRel.Field())
	}

	parent, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Electronics",
		Slug: "electronics",
	})
	if err != nil {
		t.Fatalf("create parent category: %v", err)
	}

	other, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Unrelated",
		Slug: "unrelated",
	})
	if err != nil {
		t.Fatalf("create unrelated category: %v", err)
	}
	_ = other

	// Root with 0 children must return 0 rows (must NOT return itself).
	parentFld, _ := attrs.Define(ctx, parent).Field("Children")
	_ = parentFld.GetValue()
	rows, err := parent.Children.Objects().WithContext(ctx).All()
	if err != nil {
		t.Fatalf("parent.Children.Objects().All() error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 children for root category without children, got %d (first ID=%d Name=%q)",
			len(rows), rows[0].Object.ID, rows[0].Object.Name)
	}

	c1, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "Phones",
		Slug:           "phones",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child 1: %v", err)
	}
	c2, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "Laptops",
		Slug:           "laptops",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child 2: %v", err)
	}

	rows, err = parent.Children.Objects().WithContext(ctx).OrderBy("ID").All()
	if err != nil {
		t.Fatalf("parent.Children.Objects().All() error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 children for parent, got %d", len(rows))
	}
	if rows[0].Object.ID != c1.ID || rows[1].Object.ID != c2.ID {
		t.Errorf("children IDs = [%d, %d], want [%d, %d]", rows[0].Object.ID, rows[1].Object.ID, c1.ID, c2.ID)
	}
}

func TestCategory_ChildrenOfQuery(t *testing.T) {
	clearCategories(t)

	parent, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Root",
		Slug: "root",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	c1, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "Child 1",
		Slug:           "child-1",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child 1: %v", err)
	}
	c2, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "Child 2",
		Slug:           "child-2",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child 2: %v", err)
	}
	_, err = queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Other Root",
		Slug: "other-root",
	})
	if err != nil {
		t.Fatalf("create other root: %v", err)
	}

	rows, err := categories.CategoryQueryset().ChildrenOf(parent).OrderBy("ID").All()
	if err != nil {
		t.Fatalf("ChildrenOf(parent) failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("ChildrenOf(parent) returned %d rows, want 2", len(rows))
	}
	if rows[0].Object.ID != c1.ID || rows[1].Object.ID != c2.ID {
		t.Errorf("ChildrenOf(parent) IDs = [%d, %d], want [%d, %d]", rows[0].Object.ID, rows[1].Object.ID, c1.ID, c2.ID)
	}

	// Child with no grandchildren should return 0 rows, not its siblings.
	grandChildren, err := categories.CategoryQueryset().ChildrenOf(c1).All()
	if err != nil {
		t.Fatalf("ChildrenOf(c1) failed: %v", err)
	}
	if len(grandChildren) != 0 {
		t.Errorf("ChildrenOf(c1) returned %d rows, want 0", len(grandChildren))
	}
}

func TestCategory_AdminModelFormPanel_EditDoesNotIncludeSelf(t *testing.T) {
	clearCategories(t)

	parent, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Electronics",
		Slug: "electronics",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	loaded, err := queries.GetQuerySet(&categories.Category{}).Filter("ID", parent.ID).First()
	if err != nil {
		t.Fatalf("fetch parent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/shop/category/edit", nil)
	af := newCategoryAdminForm(req, loaded.Object)

	childrenFS := unwrapCategoryChildrenFormSet(t, af)
	inlineForms, err := childrenFS.Forms()
	if err != nil {
		t.Fatalf("childrenFS.Forms() failed: %v", err)
	}

	if len(inlineForms) != 1 {
		t.Fatalf("expected 1 empty form from MinNum=1 (0 existing children), got %d", len(inlineForms))
	}
	firstInst := inlineForms[0].Instance()
	if firstInst.ID == parent.ID || firstInst.Name == parent.Name {
		t.Errorf("Children ModelFormPanel included parent itself (ID=%d, Name=%q) in inline form[0]",
			firstInst.ID, firstInst.Name)
	}
	if firstInst.ID != 0 || firstInst.Name != "" {
		t.Errorf("expected blank new inline form instance, got ID=%d Name=%q", firstInst.ID, firstInst.Name)
	}
}

func TestCategory_AdminModelFormPanel_EditAddChildrenSetsParent(t *testing.T) {
	clearCategories(t)
	ctx := t.Context()

	parent, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Electronics",
		Slug: "electronics",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	loaded, err := queries.GetQuerySet(&categories.Category{}).Filter("ID", parent.ID).First()
	if err != nil {
		t.Fatalf("fetch parent: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/shop/category/edit", nil)
	af := newCategoryAdminForm(req, loaded.Object)

	data := url.Values{
		"Name":                                       {"Electronics"},
		"Slug":                                       {"electronics"},
		"ParentCategory":                             {""},
		"Children-management-TOTAL_FORMS":            {"2"},
		"Children-management-INITIAL_FORMS":          {"0"},
		"Children-management-MIN_NUM_FORMS":          {"1"},
		"Children-management-MAX_NUM_FORMS":          {"100"},
		"Children-0-Name":                            {"Phones"},
		"Children-0-Slug":                            {"phones"},
		"Children-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
		"Children-0-" + formsets.DELETION_FIELD_NAME: {"false"},
		"Children-1-Name":                            {"Laptops"},
		"Children-1-Slug":                            {"laptops"},
		"Children-1-ParentCategory":                  {strconv.FormatUint(parent.ID, 10)},
		"Children-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
		"Children-1-" + formsets.DELETION_FIELD_NAME: {"false"},
	}
	af.WithData(data, nil, req)

	if !forms.IsValid(req.Context(), af) {
		t.Fatalf("expected Category EditView AdminForm valid, errList=%v boundErrs=%v",
			af.ErrorList(), af.BoundErrors())
	}

	if _, err := af.Save(); err != nil {
		t.Fatalf("af.Save() failed: %v (errList=%v boundErrs=%v)", err, af.ErrorList(), af.BoundErrors())
	}
	if len(af.ErrorList()) > 0 || (af.BoundErrors() != nil && af.BoundErrors().Len() > 0) {
		t.Fatalf("expected no form errors after save, errList=%v boundErrs=%v", af.ErrorList(), af.BoundErrors())
	}

	for _, slug := range []string{"phones", "laptops"} {
		row, err := queries.GetQuerySet(&categories.Category{}).
			Select("*", "ParentCategory.*").
			Filter("Slug", slug).
			First()
		if err != nil {
			t.Fatalf("query child %q: %v", slug, err)
		}
		if row.Object.ParentCategory == nil {
			t.Fatalf("child %q has ParentCategory == nil (parent_id was NULL), want parent ID %d", slug, parent.ID)
		}
		if row.Object.ParentCategory.ID != parent.ID {
			t.Errorf("child %q ParentCategory.ID = %d, want %d", slug, row.Object.ParentCategory.ID, parent.ID)
		}
	}

	childFld, _ := attrs.Define(ctx, parent).Field("Children")
	_ = childFld.GetValue()
	childRows, err := parent.Children.Objects().WithContext(ctx).OrderBy("ID").All()
	if err != nil {
		t.Fatalf("parent.Children.Objects().All() error: %v", err)
	}
	if len(childRows) != 2 {
		t.Fatalf("expected 2 children in parent.Children, got %d", len(childRows))
	}
	for _, r := range childRows {
		if r.Object.ID == parent.ID {
			t.Errorf("parent.Children must not contain parent itself (ID=%d)", parent.ID)
		}
	}
}

func TestCategory_AdminModelFormPanel_AddParentWithChildSetsParent(t *testing.T) {
	clearCategories(t)

	newCat := &categories.Category{}
	req := httptest.NewRequest(http.MethodPost, "/admin/shop/category/add", nil)
	af := newCategoryAdminForm(req, newCat)

	data := url.Values{
		"Name":                                       {"Clothing"},
		"Slug":                                       {"clothing"},
		"ParentCategory":                             {""},
		"Children-management-TOTAL_FORMS":            {"1"},
		"Children-management-INITIAL_FORMS":          {"0"},
		"Children-management-MIN_NUM_FORMS":          {"1"},
		"Children-management-MAX_NUM_FORMS":          {"100"},
		"Children-0-Name":                            {"Shirts"},
		"Children-0-Slug":                            {"shirts"},
		"Children-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
		"Children-0-" + formsets.DELETION_FIELD_NAME: {"false"},
	}
	af.WithData(data, nil, req)

	if !forms.IsValid(req.Context(), af) {
		t.Fatalf("expected Category AddView AdminForm valid, errList=%v boundErrs=%v",
			af.ErrorList(), af.BoundErrors())
	}

	if _, err := af.Save(); err != nil {
		t.Fatalf("af.Save() failed: %v", err)
	}

	parentRow, err := queries.GetQuerySet(&categories.Category{}).Filter("Slug", "clothing").First()
	if err != nil {
		t.Fatalf("fetch saved parent Clothing: %v", err)
	}
	childRow, err := queries.GetQuerySet(&categories.Category{}).
		Select("*", "ParentCategory.*").
		Filter("Slug", "shirts").
		First()
	if err != nil {
		t.Fatalf("fetch saved child Shirts: %v", err)
	}
	if childRow.Object.ParentCategory == nil {
		t.Fatalf("added child Shirts has ParentCategory == nil, want Clothing (ID=%d)", parentRow.Object.ID)
	}
	if childRow.Object.ParentCategory.ID != parentRow.Object.ID {
		t.Errorf("Shirts.ParentCategory.ID = %d, want %d", childRow.Object.ParentCategory.ID, parentRow.Object.ID)
	}
}

func TestCategory_AdminModelFormPanel_EditExistingAndDeleteChild(t *testing.T) {
	clearCategories(t)
	ctx := t.Context()

	parent, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name: "Books",
		Slug: "books",
	})
	if err != nil {
		t.Fatalf("create parent Books: %v", err)
	}
	c1, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "Fiction",
		Slug:           "fiction",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child Fiction: %v", err)
	}
	c2, err := queries.GetQuerySet(&categories.Category{}).Create(&categories.Category{
		Name:           "History",
		Slug:           "history",
		ParentCategory: parent,
	})
	if err != nil {
		t.Fatalf("create child History: %v", err)
	}

	loaded, err := queries.GetQuerySet(&categories.Category{}).Filter("ID", parent.ID).First()
	if err != nil {
		t.Fatalf("fetch Books: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/shop/category/edit", nil)
	af := newCategoryAdminForm(req, loaded.Object)

	data := url.Values{
		"Name":                                       {"Books"},
		"Slug":                                       {"books"},
		"ParentCategory":                             {""},
		"Children-management-TOTAL_FORMS":            {"3"},
		"Children-management-INITIAL_FORMS":          {"2"},
		"Children-management-MIN_NUM_FORMS":          {"1"},
		"Children-management-MAX_NUM_FORMS":          {"100"},
		"Children-0-ID":                              {strconv.FormatUint(c1.ID, 10)},
		"Children-0-Name":                            {"Sci-Fi"},
		"Children-0-Slug":                            {"sci-fi"},
		"Children-0-ParentCategory":                  {strconv.FormatUint(parent.ID, 10)},
		"Children-0-" + formsets.ORDERING_FIELD_NAME: {"0"},
		"Children-0-" + formsets.DELETION_FIELD_NAME: {"false"},
		"Children-1-ID":                              {strconv.FormatUint(c2.ID, 10)},
		"Children-1-Name":                            {"History"},
		"Children-1-Slug":                            {"history"},
		"Children-1-ParentCategory":                  {strconv.FormatUint(parent.ID, 10)},
		"Children-1-" + formsets.ORDERING_FIELD_NAME: {"1"},
		"Children-1-" + formsets.DELETION_FIELD_NAME: {"true"},
		"Children-2-Name":                            {"Biography"},
		"Children-2-Slug":                            {"biography"},
		"Children-2-" + formsets.ORDERING_FIELD_NAME: {"2"},
		"Children-2-" + formsets.DELETION_FIELD_NAME: {"false"},
	}
	af.WithData(data, nil, req)

	if !forms.IsValid(req.Context(), af) {
		t.Fatalf("expected EditView form valid, errList=%v boundErrs=%v", af.ErrorList(), af.BoundErrors())
	}
	if _, err := af.Save(); err != nil {
		t.Fatalf("af.Save() failed: %v", err)
	}

	reloadedParent, err := queries.GetQuerySet(&categories.Category{}).
		Select("*", "ParentCategory.*").
		Filter("ID", parent.ID).
		First()
	if err != nil {
		t.Fatalf("fetch Books after edit: %v", err)
	}
	if reloadedParent.Object.ParentCategory != nil {
		t.Errorf("Books ParentCategory = %+v, want nil", reloadedParent.Object.ParentCategory)
	}

	childFld, _ := attrs.Define(ctx, reloadedParent.Object).Field("Children")
	_ = childFld.GetValue()
	rows, err := reloadedParent.Object.Children.Objects().WithContext(ctx).OrderBy("ID").All()
	if err != nil {
		t.Fatalf("reloadedParent.Children.Objects().All() error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 children after update+delete+add, got %d", len(rows))
	}
	if rows[0].Object.Name != "Sci-Fi" || rows[1].Object.Name != "Biography" {
		t.Errorf("children names = [%q, %q], want [Sci-Fi, Biography]",
			rows[0].Object.Name, rows[1].Object.Name)
	}
	for _, r := range rows {
		if r.Object.ParentCategory == nil || r.Object.ParentCategory.ID != parent.ID {
			t.Errorf("child %q ParentCategory = %v, want ID %d", r.Object.Name, r.Object.ParentCategory, parent.ID)
		}
	}

	deletedCount, err := queries.GetQuerySet(&categories.Category{}).Filter("ID", c2.ID).Count()
	if err != nil {
		t.Fatalf("count deleted child: %v", err)
	}
	if deletedCount != 0 {
		t.Errorf("expected History child (ID=%d) to be deleted, count=%d", c2.ID, deletedCount)
	}
}
