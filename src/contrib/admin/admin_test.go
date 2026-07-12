//go:build test
// +build test

package admin_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/PuerkitoBio/goquery"
)

type TestModel struct {
	ID   int `attrs:"primary;readonly"`
	Name string
}

func (t *TestModel) String() string {
	return t.Name
}

func (t *TestModel) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions[*TestModel](t)
}

func (t *TestModel) Save() (err error) {
	if t.ID == 0 {
		_, err = queries.GetQuerySet(t).ExplicitSave().Create(t)
	} else {
		_, err = queries.GetQuerySet(t).ExplicitSave().Update(t)
	}
	return err
}

func (t *TestModel) Delete() error {
	_, err := queries.GetQuerySet(t).
		Filter("ID", t.ID).
		ExplicitSave().
		Delete()
	return err
}

// User does not need to be a DB model.
type User struct {
	Name            string `attrs:"primary"`
	LoggedIn        bool
	IsAdministrator bool
}

func (l *User) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions[*User](l)
}

func (l *User) IsAuthenticated() bool {
	return l.LoggedIn
}

func (l *User) IsAdmin() bool {
	return l.IsAdministrator
}

func initAdminApp() django.AppConfig {
	// Configure the auth system for the admin
	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	attrs.RegisterModel(&TestModel{})
	attrs.RegisterModel(&User{})
	admin.RegisterApp(
		"admin_test",
		admin.AppOptions{
			EnableIndexView: true,
		},
		admin.ModelOptions{Name: "TestModel", Model: &TestModel{}},
		admin.ModelOptions{Name: "User", Model: &User{}},
	)

	// hook index
	autherrors.RegisterHook("index")
	return admin.NewAppConfig()
}

func TestAdmin(t *testing.T) {
	var _, db = testdb.Open()
	defer db.Close()

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
			initAdminApp,
			&apps.AppConfig{
				AppName: "admin_test",
				ModelObjects: []attrs.Definer{
					&TestModel{},
					&User{},
				},
			},
		},
		Auth: &djester.TesterAuth{
			UnauthenticatedUser: func() authentication.User {
				return &User{LoggedIn: false, IsAdministrator: false}
			},
			Users: map[string]authentication.User{
				"valid_user": &User{LoggedIn: true, IsAdministrator: false},
				"admin_user": &User{LoggedIn: true, IsAdministrator: true},
			},
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Unauthenticated redirects",
				Function: func(d *djester.Tester, t *testing.T) {
					endpoints := []string{
						"/admin/apps/admin_test",
						"/admin/apps/admin_test/TestModel",
						"/admin/apps/admin_test/TestModel/add",
						"/admin/apps/admin_test/TestModel/edit/1",
						"/admin/apps/admin_test/TestModel/delete/1",
					}

					for _, ep := range endpoints {
						t.Run(ep, func(t *testing.T) {
							resp, err := d.Get(ep, nil, nil)
							if err != nil {
								t.Fatalf("GET failed: %v", err)
							}
							defer resp.Body.Close()
							d.Assert(true).AssertEqual(http.StatusOK, resp.StatusCode)
							d.Assert(true).Assert(strings.HasPrefix(resp.Request.URL.Path, "/admin/login/"), "expected redirect to /admin/login/, got %s", resp.Request.URL.Path)
						})
					}
				},
			},
			&djester.BasicTest{
				Label: "Logged in normal user redirects",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := d.Login("valid_user")
					d.Assert(true).Assert(err == nil, "Login failed: %v", err)

					endpoints := []string{
						"/admin/apps/admin_test",
						"/admin/apps/admin_test/TestModel",
						"/admin/apps/admin_test/TestModel/add",
						"/admin/apps/admin_test/TestModel/edit/1",
						"/admin/apps/admin_test/TestModel/delete/1",
					}

					for _, ep := range endpoints {
						t.Run(ep, func(t *testing.T) {
							resp, err := d.Get(ep, nil, nil)
							if err != nil {
								t.Fatalf("GET failed: %v", err)
							}
							defer resp.Body.Close()
							d.Assert(true).AssertEqual(http.StatusOK, resp.StatusCode)
							d.Assert(true).Assert(strings.HasPrefix(resp.Request.URL.Path, "/admin/relogin/"), "expected redirect to /admin/relogin/, got %s", resp.Request.URL.Path)
						})
					}

					d.Logout()
				},
			},
			&djester.BasicTest{
				Label: "Logged in admin access",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := d.Login("admin_user")
					d.Assert(true).Assert(err == nil, "Login failed: %v", err)

					endpoints := []string{
						"/admin/apps/admin_test",
						"/admin/apps/admin_test/TestModel",
						"/admin/apps/admin_test/TestModel/add",
						"/admin/apps/admin_test/TestModel/edit/1",
						"/admin/apps/admin_test/TestModel/delete/1",
					}

					for _, ep := range endpoints {
						t.Run(ep, func(t *testing.T) {
							resp, err := d.Get(ep, nil, nil)
							if err != nil {
								t.Fatalf("GET failed: %v", err)
							}
							defer resp.Body.Close()
							d.Assert(true).AssertEqual(http.StatusOK, resp.StatusCode)
						})
					}

					d.Logout()
				},
			},
			&djester.BasicTest{
				Label: "Admin create, edit, delete cycle",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := d.Login("admin_user")
					var asserter = d.Assert(true)
					asserter.Assert(err == nil, "Login failed: %v", err)

					// Check add page HTML
					resp, err := d.Get("/admin/apps/admin_test/TestModel/add", nil, nil)
					asserter.Assert(err == nil, "GET add failed: %v", err)
					resp.Assert(true).AssertHTML(
						djester.HasElement("form"),
						djester.HasElement("input#id_ID"),
						djester.HasElement("input#id_Name"),
					)
					resp.Body.Close()

					// Create new object
					form := map[string]interface{}{
						"Name": "New Test Name",
					}
					resp, err = d.PostForm("/admin/apps/admin_test/TestModel/add", nil, nil, form)
					asserter.Assert(err == nil, "POST add failed: %v", err)
					resp.Assert(true).AssertHTML(
						func(doc *goquery.Document) error {
							var s = doc.Find(".panel__error")
							var label = s.Parent().Parent().Parent().Find("label")
							lHtml, _ := label.Html()
							html, err := s.Html()
							if err != nil {
								return err
							}
							if html != "" {
								return fmt.Errorf("%s: %s", lHtml, html)
							}
							return nil
						},
					)
					resp.Body.Close()

					var row *queries.Row[*TestModel]
					row, err = queries.GetQuerySet(&TestModel{}).Filter("ID", 2).Get()
					asserter.Assert(err == nil, "Failed to fetch new object: %v", err)
					asserter.AssertEqual("New Test Name", row.Object.Name)

					// Edit object
					form["Name"] = "Edited Test Name"
					resp, err = d.PostForm("/admin/apps/admin_test/TestModel/edit/2", nil, nil, form)
					asserter.Assert(err == nil, "POST edit failed: %v", err)
					resp.Assert(true).AssertHTML(
						func(doc *goquery.Document) error {
							var s = doc.Find(".panel__error")
							var label = s.ParentsUntil(".panel").ChildrenFiltered("label")
							lHtml, _ := label.Html()
							html, err := s.Html()
							if err != nil {
								return err
							}
							if html != "" {
								return fmt.Errorf("%s: %s", lHtml, html)
							}
							return nil
						},
					)
					resp.Body.Close()

					// Verify it was edited
					row, err = queries.GetQuerySet(&TestModel{}).Filter("ID", 2).Get()
					asserter.Assert(err == nil, "Failed to fetch edited object: %v", err)
					asserter.AssertEqual("Edited Test Name", row.Object.Name)

					// Delete object
					resp, err = d.PostForm("/admin/apps/admin_test/TestModel/delete/2", nil, nil, map[string]interface{}{"confirm": "yes"})
					asserter.Assert(err == nil, "POST delete failed: %v", err)
					resp.Assert(true).AssertHTML(
						func(doc *goquery.Document) error {
							var s = doc.Find(".panel__error")
							var label = s.ParentsUntil(".panel").ChildrenFiltered("label")
							lHtml, _ := label.Html()
							html, err := s.Html()
							if err != nil {
								return err
							}
							if html != "" {
								return fmt.Errorf("%s: %s", lHtml, html)
							}
							return nil
						},
					)
					resp.Body.Close()

					// Verify it was deleted
					_, err = queries.GetQuerySet(&TestModel{}).Filter("ID", 2).Get()
					asserter.Assert(err != nil, "Object was not deleted")

					d.Logout()
				},
			},
		},
	}

	d.Setup(djester.TW(t))

	var tables = quest.Table(t, &TestModel{})
	tables.IfNotExists = true
	tables.Create()
	defer tables.Drop()

	attrs.ResetDefinitions.Send(nil)

	// insert a test model
	var testModel = &TestModel{Name: "Test 1"}
	err := queries.SaveObject(testModel)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	d.Test(t)
}
