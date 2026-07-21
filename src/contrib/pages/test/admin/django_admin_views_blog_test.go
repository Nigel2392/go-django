package pages_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/google/uuid"
)

var (
	testAdminUser = &auth.User{
		ID:   1,
		Base: users.Base{},
	}

	blogPageAddData = url.Values{
		"Title":   {"Blog entry"},
		"Slug":    {"blog-entry"},
		"Summary": {"Initial summary"},
	}

	blogPageEditData = withInlineManagement(url.Values{
		"Title":   {"Blog entry updated"},
		"Slug":    {"blog-entry-updated"},
		"Summary": {"Updated summary"},
	})

	blogPageRevisionEditData = withInlineManagement(url.Values{
		"Title":   {"Blog entry from revision"},
		"Slug":    {"blog-entry-from-revision"},
		"Summary": {"Revision summary"},
	})
)

func withInlineManagement(data url.Values) url.Values {
	if data == nil {
		data = url.Values{}
	}
	data.Set("TestBlogImageSet-management-TOTAL_FORMS", "0")
	data.Set("TestBlogImageSet-management-INITIAL_FORMS", "0")
	data.Set("TestBlogImageSet-management-MIN_NUM_FORMS", "0")
	data.Set("TestBlogImageSet-management-MAX_NUM_FORMS", "0")
	return data
}

func init() {
	testAdminUser.Email = drivers.MustParseEmail("admin@example.com")
	testAdminUser.Username = "admin"
	testAdminUser.Password = auth.NewPassword("admin-password")
	testAdminUser.IsAdministrator = true
	testAdminUser.IsActive = true
	testAdminUser.IsLoggedIn = true
}

type TestBlogPage struct {
	models.Model `table:"test_blog_pages"`
	Page         *pages.PageNode `proxy:"true"`
	Summary      string
}

func (b *TestBlogPage) ID() int64 {
	if b.Page == nil {
		return 0
	}
	return b.Page.PageID
}

func (b *TestBlogPage) Reference() *pages.PageNode {
	if b.Page == nil {
		b.Page = &pages.PageNode{}
	}
	return b.Page
}

func (b *TestBlogPage) Save(ctx context.Context) error {
	if b.Page == nil {
		b.Page = &pages.PageNode{}
	}
	if b.Page.PageID == 0 {
		_, err := queries.GetQuerySetWithContext(ctx, &TestBlogPage{}).ExplicitSave().Create(b)
		return err
	}
	_, err := queries.GetQuerySetWithContext(ctx, &TestBlogPage{}).ExplicitSave().Update(b)
	return err
}

func (b *TestBlogPage) FieldDefs(ctx context.Context) attrs.Definitions {
	if b.Page == nil {
		b.Page = &pages.PageNode{}
	}
	return b.Model.Define(ctx, b,
		attrs.NewField(b.Page, "PageID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.NewField(b.Page, "Title", &attrs.FieldConfig{Embedded: true}),
		attrs.NewField(b.Page, "Slug", &attrs.FieldConfig{Embedded: true, Blank: true}),
		attrs.NewField(b.Page, "UrlPath", &attrs.FieldConfig{Embedded: true, ReadOnly: true}),
		attrs.NewField(b, "Summary", &attrs.FieldConfig{Blank: true}),
	)
}

type TestBlogImage struct {
	models.Model `table:"test_blog_images"`
	ID           int64
	BlogPage     *TestBlogPage
	ImageText    string
}

func (b *TestBlogImage) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true, ReadOnly: true, Column: "id"}),
		attrs.Unbound("BlogPage", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&TestBlogPage{}, "", nil),
			Blank:         true,
			FormField: func(opts ...func(fields.Field)) fields.Field {
				return fields.CharField(fields.Hide(true))
			},
		}),
		attrs.Unbound("ImageText"),
	)
}

func newTestBlogAppConfig() django.AppConfig {
	app := apps.NewAppConfig("test_blog_pages_admin")
	app.ModelObjects = []attrs.Definer{
		&TestBlogPage{},
		&TestBlogImage{},
	}
	return app
}

func TestMain(m *testing.M) {
	_, sqlDB := testdb.Open()

	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	attrs.RegisterModel(&pages.PageNode{})
	attrs.RegisterModel(&TestBlogPage{})
	attrs.RegisterModel(&TestBlogImage{})

	admin.RegisterApp(
		"test_blog_pages_admin",
		admin.AppOptions{RegisterToAdminMenu: false},
		admin.ModelOptions{Name: "test_blog_page", Model: &TestBlogPage{}},
		admin.ModelOptions{Name: "test_blog_image", Model: &TestBlogImage{}},
	)

	pages.Register(&pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestBlogPage{},
		},
		AddPanels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.FieldPanel("Title"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Summary"),
			}
		},
		EditPanels: func(r *http.Request, page pages.Page) []admin.Panel {
			return []admin.Panel{
				admin.FieldPanel("Title"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Summary"),
				&admin.ModelFormPanel[*TestBlogImage, modelforms.ModelForm[*TestBlogImage]]{
					TargetType: &TestBlogImage{},
					FieldName:  "TestBlogImageSet",
					MinNum:     0,
					Extra:      1,
					Panels: []admin.Panel{
						admin.FieldPanel("ImageText"),
					},
				},
			}
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			row, err := queries.GetQuerySetWithContext(ctx, &TestBlogPage{}).
				Filter("PageID", id).
				Get()
			if err != nil {
				return nil, err
			}
			row.Object.Page = ref
			return row.Object, nil
		},
	})

	pages.SetRoutePrefix("/")

	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE:  sqlDB,
			django.APPVAR_RECOVERER: false,
			"ALLOWED_HOSTS":         []string{"*"},
		}),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			revisions.NewAppConfig,
			newTestBlogAppConfig,
			// apps.AppConfig{AppName: "migrator"}, // dud so no indexes get created in the tests
			pages.NewAppConfig,
		),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
			django.FlagSkipCmds,
		),
	)

	var log = &logger.Logger{
		Level:       logger.INF,
		OutputDebug: nil,
		OutputInfo:  os.Stdout,
		WrapPrefix:  logger.ColoredLogWrapper,
	}
	django.Global.Log = log
	logger.Setup(log)

	// bandaid because i am lazy and do not want to fix my tests to not violate the constraint for the content type field.
	queries.SignalPreModelCreate.Listen(func(s signals.Signal[queries.ModelSignal], ms queries.ModelSignal) error {
		p, ok := ms.Instance.(*pages.PageNode)
		if !ok {
			return nil
		}

		if p.ContentType == "" {
			p.ContentType = uuid.NewString()
		}

		return nil
	})

	if err := app.Initialize(); err != nil {
		panic(err)
	}

	//	/*
	//		START OF MIGRATOR
	//
	//		we need to incorporate migrator because we added the migrator app dud.
	//		otherwise tests fail due to constraint violations and im lazy
	//	*/
	//	var schemaEditor, err = migrator.GetSchemaEditor(sqlDB.Driver())
	//	if err != nil {
	//		panic(fmt.Sprintf("failed to get schema editor: %w", err))
	//	}
	//
	//	var models = make([]attrs.Definer, 0)
	//	for head := app.Apps.Front(); head != nil; head = head.Next() {
	//		models = append(models, head.Value.Models()...)
	//	}
	//
	//	/*
	//		END OF MIGRATOR
	//	*/

	// Test-only middleware that authenticates every request as an admin user.
	app.Mux.Use(authentication.AddUserMiddleware(func(r *http.Request) authentication.User {
		return testAdminUser
	}))

	_, _ = queries.GetQuerySet(&auth.User{}).Delete()
	if _, err := queries.GetQuerySet(&auth.User{}).ExplicitSave().Create(testAdminUser); err != nil {
		panic(err)
	}

	tbl := quest.Table[*testing.T](nil, &TestBlogPage{}, &TestBlogImage{})
	tbl.Create()

	exitCode := m.Run()

	tbl.Drop()
	os.Exit(exitCode)
}

func resetPagesAdminData(t *testing.T) {
	t.Helper()
	_, _ = queries.GetQuerySet(&revisions.Revision{}).Delete()
	_, _ = queries.GetQuerySet(&TestBlogImage{}).Delete()
	_, _ = queries.GetQuerySet(&TestBlogPage{}).Delete()
	_, _ = queries.GetQuerySet(&pages.PageNode{}).Delete()
}

func createRootNode(t *testing.T, title string) *pages.PageNode {
	t.Helper()
	root := &pages.PageNode{Title: title, Slug: strings.ToLower(title), ContentType: uuid.New().String()} //ctype is literally useless, it does not matter here
	if err := pages.NewPageQuerySet().AddRoot(root); err != nil {
		t.Fatalf("add root: %v", err)
	}
	return root
}

func performAdminRequest(t *testing.T, method, target string, data url.Values) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if data != nil {
		req = httptest.NewRequest(method, target, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	ctx := authentication.ContextWithUser(req.Context(), testAdminUser)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	django.Global.Mux.ServeHTTP(rr, req)
	return rr
}

func createChildPageViaAddView(t *testing.T, parent *pages.PageNode) *pages.PageNode {
	t.Helper()
	ct := contenttypes.NewContentType(&TestBlogPage{})
	addURL := django.Reverse("admin:pages:add", parent.ID(), ct.AppLabel(), ct.Model())

	rr := performAdminRequest(t, http.MethodPost, addURL, blogPageAddData)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected add view redirect, got %d", rr.Code)
	}

	children, err := pages.NewPageQuerySet().GetChildNodes(parent, pages.StatusFlagNone, 0, 50)
	if err != nil {
		t.Fatalf("get child nodes: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child after add, got %d", len(children))
	}
	return children[0]
}

func TestPagesAdminAddView(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")

	child := createChildPageViaAddView(t, root)
	if child.Title != "Blog entry" {
		t.Fatalf("expected child title to be saved, got %q", child.Title)
	}

	revCount, err := revisions.NewRevisionQuerySet().ForObjects(child).Count()
	if err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revCount == 0 {
		t.Fatalf("expected at least one revision after add")
	}
}

func TestPagesAdminEditView(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	editURL := django.Reverse("admin:pages:edit", child.ID())
	rr := performAdminRequest(t, http.MethodPost, editURL, blogPageEditData)
	if rr.Code != http.StatusSeeOther && rr.Code != http.StatusOK {
		t.Fatalf("expected edit view status 200/303, got %d", rr.Code)
	}

	if _, err := pages.NewPageQuerySet().GetNodeByID(child.ID()); err != nil {
		t.Fatalf("expected edited page to remain queryable: %v", err)
	}
}

func TestPagesAdminRevisionAddViewHandlesPost(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	rr := performAdminRequest(t, http.MethodPost, django.Reverse("admin:pages:edit", child.ID()), blogPageEditData)
	if rr.Code != http.StatusSeeOther && rr.Code != http.StatusOK {
		t.Fatalf("expected edit status 200/303 before revision add test, got %d", rr.Code)
	}

	revisionRows, err := revisions.NewRevisionQuerySet().ForObjects(child).OrderBy("-CreatedAt").All()
	if err != nil {
		t.Fatalf("load revisions: %v", err)
	}
	if len(revisionRows) < 1 {
		t.Fatalf("expected revisions to exist")
	}

	beforeCount := len(revisionRows)
	revisionID := revisionRows[len(revisionRows)-1].Object.ID

	revURL := django.Reverse("admin:pages:revisions:detail", child.ID(), revisionID)
	rr = performAdminRequest(t, http.MethodPost, revURL, blogPageRevisionEditData)
	if rr.Code != http.StatusSeeOther && rr.Code != http.StatusOK {
		t.Fatalf("expected revision detail status 200/303, got %d", rr.Code)
	}

	afterCount, err := revisions.NewRevisionQuerySet().ForObjects(child).Count()
	if err != nil {
		t.Fatalf("count revisions after revision post: %v", err)
	}
	if int(afterCount) < beforeCount {
		t.Fatalf("expected revision count to stay the same or increase after revision add-view POST")
	}
}

func TestPagesAdminRevisionEditView(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	revisionsList, err := revisions.NewRevisionQuerySet().ForObjects(child).All()
	if err != nil {
		t.Fatalf("load revisions: %v", err)
	}
	if len(revisionsList) == 0 {
		t.Fatalf("expected at least one revision")
	}

	revURL := django.Reverse("admin:pages:revisions:detail", child.ID(), revisionsList[0].Object.ID)
	rr := performAdminRequest(t, http.MethodGet, revURL, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected revision edit GET to return 200, got %d", rr.Code)
	}
}

func TestPagesAdminPageEditFormBuildsFormSet(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	specific, err := pages.Specific(context.Background(), child, false)
	if err != nil {
		t.Fatalf("load specific page: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, django.Reverse("admin:pages:edit", child.ID()), nil)
	req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

	form := pages.PageEditForm(req, specific)
	form.Load()

	formSet := form.FormSet()
	if formSet == nil {
		t.Fatalf("expected edit form to build a formset")
	}
}

// buildAndValidateEditForm constructs a PageEditForm for the given node,
// feeds it the supplied POST data, and runs it through IsValid so that
// CleanedData (and therefore HasChanged) can subsequently be called.
func buildAndValidateEditForm(t *testing.T, node *pages.PageNode, data url.Values) *admin.AdminForm[*modelforms.BaseModelForm[attrs.Definer], attrs.Definer] {
	t.Helper()

	specific, err := pages.Specific(context.Background(), node, false)
	if err != nil {
		t.Fatalf("load specific page: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		django.Reverse("admin:pages:edit", node.ID()),
		strings.NewReader(data.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

	form := pages.PageEditForm(req, specific)
	form.Load()
	form.WithData(data, nil, req)

	if !forms.IsValid(req.Context(), form) {
		var errrList = form.ErrorList()
		var boundErrors = form.BoundErrors()
		var sb = strings.Builder{}
		for _, err := range errrList {
			sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		}
		if boundErrors != nil {
			for head := boundErrors.Front(); head != nil; head = head.Next() {
				sb.WriteString(fmt.Sprintf("Bound Error: [%s] %v\n", head.Key, head.Value))
			}
		}

		var fset = form.FormSet()
		if fset != nil {
			fsetErrorsList := fset.ErrorLists()
			for _, errs := range fsetErrorsList {
				for _, err := range errs {
					sb.WriteString(fmt.Sprintf("FormSet Error: %v\n", err))
				}
			}

			fsetBoundErrs := fset.BoundErrorsList()
			for _, boundErrs := range fsetBoundErrs {
				for head := boundErrs.Front(); head != nil; head = head.Next() {
					sb.WriteString(fmt.Sprintf("FormSet Bound Error: [%s] %v\n", head.Key, head.Value))
				}
			}
		}

		t.Fatalf("edit form unexpectedly invalid: %v", sb.String())
	}

	return form
}

func TestPagesAdminEditFormHasChangedReturnsFalseForIdenticalData(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	// Submit exactly the same values that were used to create the page.
	sameData := withInlineManagement(url.Values{
		"Title":   {blogPageAddData.Get("Title")},
		"Slug":    {blogPageAddData.Get("Slug")},
		"Summary": {blogPageAddData.Get("Summary")},
	})

	form := buildAndValidateEditForm(t, child, sameData)
	if form.HasChanged() {
		t.Fatalf("expected HasChanged() = false when submitted data matches the saved page, got true")
	}
}

func TestPagesAdminEditFormHasChangedReturnsTrueWhenTitleChanged(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	changedData := withInlineManagement(url.Values{
		"Title":   {"A different title"},
		"Slug":    {blogPageAddData.Get("Slug")},
		"Summary": {blogPageAddData.Get("Summary")},
	})

	form := buildAndValidateEditForm(t, child, changedData)
	if !form.HasChanged() {
		t.Fatalf("expected HasChanged() = true when Title was changed, got false")
	}
}

func TestPagesAdminEditFormHasChangedReturnsTrueWhenSummaryChanged(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	changedData := withInlineManagement(url.Values{
		"Title":   {blogPageAddData.Get("Title")},
		"Slug":    {blogPageAddData.Get("Slug")},
		"Summary": {"A different summary"},
	})

	form := buildAndValidateEditForm(t, child, changedData)
	if !form.HasChanged() {
		t.Fatalf("expected HasChanged() = true when Summary was changed, got false")
	}
}

func TestPagesAdminPageEditFormIncludesBlogImageParentField(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	specific, err := pages.Specific(context.Background(), child, false)
	if err != nil {
		t.Fatalf("load specific page: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, django.Reverse("admin:pages:edit", child.ID()), nil)
	req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

	form := pages.PageEditForm(req, specific)
	form.Load()

	formSet := form.FormSet()
	formsList, err := formSet.Forms()
	if err != nil {
		t.Fatalf("load formset forms: %v", err)
	}

	foundParentField := false
	for _, f := range formsList {
		if formWithField, ok := any(f).(interface {
			Field(string) (fields.Field, bool)
		}); ok {
			if _, ok := formWithField.Field("BlogPage"); ok {
				foundParentField = true
				break
			}
		}
	}

	if !foundParentField {
		t.Fatalf("expected related blog image admin form to include BlogPage field")
	}
}

// withInlineManagementForms is like withInlineManagement but sets
// TOTAL_FORMS / INITIAL_FORMS to n, for tests that submit real inline data.
func withInlineManagementForms(data url.Values, n int) url.Values {
	if data == nil {
		data = url.Values{}
	}
	nStr := fmt.Sprintf("%d", n)
	data.Set("TestBlogImageSet-management-TOTAL_FORMS", nStr)
	data.Set("TestBlogImageSet-management-INITIAL_FORMS", nStr)
	data.Set("TestBlogImageSet-management-MIN_NUM_FORMS", "0")
	data.Set("TestBlogImageSet-management-MAX_NUM_FORMS", "0")
	return data
}

// createBlogImage inserts a TestBlogImage row for the given page node and
// returns the saved object.
func createBlogImage(t *testing.T, page *pages.PageNode) *TestBlogImage {
	t.Helper()
	specific, err := pages.Specific(context.Background(), page, false)
	if err != nil {
		t.Fatalf("createBlogImage: load specific page: %v", err)
	}
	blogPage, ok := specific.(*TestBlogPage)
	if !ok {
		t.Fatalf("createBlogImage: expected *TestBlogPage, got %T", specific)
	}
	img := &TestBlogImage{
		BlogPage:  blogPage,
		ImageText: "initial image text",
	}
	saved, err := queries.GetQuerySet(&TestBlogImage{}).ExplicitSave().Create(img)
	if err != nil {
		t.Fatalf("createBlogImage: %v", err)
	}
	return saved
}

func TestPagesAdminEditFormHasChangedReturnsFalseForUnchangedInlineData(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)
	img := createBlogImage(t, child)

	// Re-submit exactly what is stored in the DB (page fields + inline image).
	data := withInlineManagementForms(url.Values{
		"Title":   {blogPageAddData.Get("Title")},
		"Slug":    {blogPageAddData.Get("Slug")},
		"Summary": {blogPageAddData.Get("Summary")},
		// Inline form 0: same ImageText and parent FK as was saved.
		// child.PageID is the test_blog_pages row-ID that the FK references;
		// child.ID() is the pages.PageNode PK, which is a different value.
		"TestBlogImageSet-0-ImageText":      {img.ImageText},
		"TestBlogImageSet-0-BlogPage":       {fmt.Sprintf("%d", child.PageID)},
		"TestBlogImageSet-0-ID":             {fmt.Sprintf("%d", img.ID)},
		"TestBlogImageSet-0-Content--total": {"0"},
		"TestBlogImageSet-0-__DELETE__":     {"false"},
		"TestBlogImageSet-0-__ORDER__":      {"0"},
	}, 1)

	form := buildAndValidateEditForm(t, child, data)
	if form.HasChanged() {
		t.Fatalf("expected HasChanged() = false when inline data is identical to DB, got true")
	}
}

func TestPagesAdminEditFormHasChangedReturnsTrueWhenInlineImageTextChanged(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)
	img := createBlogImage(t, child)
	_ = img // created so the formset loads one existing image

	// Submit a different ImageText value.
	data := withInlineManagementForms(url.Values{
		"Title":                             {blogPageAddData.Get("Title")},
		"Slug":                              {blogPageAddData.Get("Slug")},
		"Summary":                           {blogPageAddData.Get("Summary")},
		"TestBlogImageSet-0-ImageText":      {"changed image text"},
		"TestBlogImageSet-0-BlogPage":       {fmt.Sprintf("%d", child.PageID)},
		"TestBlogImageSet-0-ID":             {fmt.Sprintf("%d", img.ID)},
		"TestBlogImageSet-0-Content--total": {"0"},
		"TestBlogImageSet-0-__DELETE__":     {"false"},
		"TestBlogImageSet-0-__ORDER__":      {"0"},
	}, 1)

	form := buildAndValidateEditForm(t, child, data)
	if !form.HasChanged() {
		t.Fatalf("expected HasChanged() = true when inline ImageText was changed, got false")
	}
}
