package pages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/mux/middleware/authentication"
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

	blogPageEditData = url.Values{
		"Title":   {"Blog entry updated"},
		"Slug":    {"blog-entry-updated"},
		"Summary": {"Updated summary"},
		"TestBlogImageSet-management-TOTAL_FORMS":   {"0"},
		"TestBlogImageSet-management-INITIAL_FORMS": {"0"},
		"TestBlogImageSet-management-MIN_NUM_FORMS": {"0"},
		"TestBlogImageSet-management-MAX_NUM_FORMS": {"0"},
	}

	blogPageRevisionEditData = url.Values{
		"Title":   {"Blog entry from revision"},
		"Slug":    {"blog-entry-from-revision"},
		"Summary": {"Revision summary"},
		"TestBlogImageSet-management-TOTAL_FORMS":   {"0"},
		"TestBlogImageSet-management-INITIAL_FORMS": {"0"},
		"TestBlogImageSet-management-MIN_NUM_FORMS": {"0"},
		"TestBlogImageSet-management-MAX_NUM_FORMS": {"0"},
	}
)

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
	Page         *PageNode `proxy:"true"`
	Summary      string
}

func (b *TestBlogPage) ID() int64 {
	if b.Page == nil {
		return 0
	}
	return b.Page.PageID
}

func (b *TestBlogPage) Reference() *PageNode {
	if b.Page == nil {
		b.Page = &PageNode{}
	}
	return b.Page
}

func (b *TestBlogPage) Save(ctx context.Context) error {
	if b.Page == nil {
		b.Page = &PageNode{}
	}
	if b.Page.PageID == 0 {
		_, err := queries.GetQuerySetWithContext(ctx, &TestBlogPage{}).ExplicitSave().Create(b)
		return err
	}
	_, err := queries.GetQuerySetWithContext(ctx, &TestBlogPage{}).ExplicitSave().Update(b)
	return err
}

func (b *TestBlogPage) FieldDefs() attrs.Definitions {
	if b.Page == nil {
		b.Page = &PageNode{}
	}
	return b.Model.Define(b,
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

func (b *TestBlogImage) FieldDefs() attrs.Definitions {
	return b.Model.Define(b,
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

	attrs.RegisterModel(&PageNode{})
	attrs.RegisterModel(&TestBlogPage{})
	attrs.RegisterModel(&TestBlogImage{})

	admin.RegisterApp(
		"test_blog_pages_admin",
		admin.AppOptions{RegisterToAdminMenu: false},
		admin.ModelOptions{Name: "test_blog_page", Model: &TestBlogPage{}},
		admin.ModelOptions{Name: "test_blog_image", Model: &TestBlogImage{}},
	)

	Register(&PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestBlogPage{},
		},
		AddPanels: func(r *http.Request, page Page) []admin.Panel {
			return []admin.Panel{
				admin.FieldPanel("Title"),
				admin.FieldPanel("Slug"),
				admin.FieldPanel("Summary"),
			}
		},
		EditPanels: func(r *http.Request, page Page) []admin.Panel {
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
		GetForID: func(ctx context.Context, ref *PageNode, id int64) (Page, error) {
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

	SetRoutePrefix("/")

	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: sqlDB,
			"ALLOWED_HOSTS":        []string{"*"},
		}),
		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			revisions.NewAppConfig,
			newTestBlogAppConfig,
			NewAppConfig,
		),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
			django.FlagSkipCmds,
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
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
	_, _ = queries.GetQuerySet(&PageNode{}).Delete()
}

func createRootNode(t *testing.T, title string) *PageNode {
	t.Helper()
	root := &PageNode{Title: title, Slug: strings.ToLower(title)}
	if err := NewPageQuerySet().AddRoot(root); err != nil {
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

func createChildPageViaAddView(t *testing.T, parent *PageNode) *PageNode {
	t.Helper()
	ct := contenttypes.NewContentType(&TestBlogPage{})
	addURL := django.Reverse("admin:pages:add", parent.ID(), ct.AppLabel(), ct.Model())

	rr := performAdminRequest(t, http.MethodPost, addURL, blogPageAddData)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected add view redirect, got %d", rr.Code)
	}

	children, err := NewPageQuerySet().GetChildNodes(parent, StatusFlagNone, 0, 50)
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

	if _, err := NewPageQuerySet().GetNodeByID(child.ID()); err != nil {
		t.Fatalf("expected edited page to remain queryable: %v", err)
	}
}

func TestPagesAdminRevisionAddView(t *testing.T) {
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
		t.Fatalf("expected revision count to stay the same or increase")
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

	specific, err := Specific(context.Background(), child, false)
	if err != nil {
		t.Fatalf("load specific page: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, django.Reverse("admin:pages:edit", child.ID()), nil)
	req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

	form := PageEditForm(req, specific)
	form.Load()

	formSet := form.FormSet()
	if formSet == nil {
		t.Fatalf("expected edit form to build a formset")
	}
}

func TestPagesAdminPageEditFormIncludesBlogImageParentField(t *testing.T) {
	resetPagesAdminData(t)
	root := createRootNode(t, "Root")
	child := createChildPageViaAddView(t, root)

	specific, err := Specific(context.Background(), child, false)
	if err != nil {
		t.Fatalf("load specific page: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, django.Reverse("admin:pages:edit", child.ID()), nil)
	req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

	form := PageEditForm(req, specific)
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
