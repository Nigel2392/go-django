//go:build test
// +build test

package chooser_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
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
	return attrs.
		AutoDefinitions[*TestModel](t).(*attrs.ObjectDefinitions).
		WithTableName("chooser_testmodel")
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

	chooser.Register(&chooser.ChooserDefinition[*TestModel]{
		Model: &TestModel{},
		Title: trans.S("Test Model"),
		PreviewString: func(ctx context.Context, instance *TestModel) string {
			return fmt.Sprintf("TestModel(%s)", instance.Name)
		},
		ListPage: &chooser.ChooserListPage[*TestModel]{
			Fields: []string{"ID", "Name"},
		},
		CreatePage: &chooser.ChooserFormPage[*TestModel]{},
	})

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
				Label: "Chooser create",
				Function: func(dj *djester.Tester, t *testing.T) {
					_, err := dj.Login("admin_user")
					asserter := dj.Assert(t, true)
					asserter.Assert(err == nil, "Login failed: %v", err)

					_, err = queries.GetQuerySet(&TestModel{}).Delete()
					asserter.Assert(err == nil, "Failed to delete old objects: %v", err)

					var names = []string{
						"First TestModel",
						"Second TestModel",
						"Third TestModel",
						"Fourth TestModel",
						"Fifth TestModel",
						"Sixth TestModel",
					}

					for _, name := range names {
						form := map[string]interface{}{
							"Name": name,
						}

						var chooserResp = new(chooser.ChooserResponse)
						resp, err := dj.PostForm("/admin/apps/admin_test/TestModel/chooser/default/create", nil, nil, form)
						asserter.Assert(err == nil, "POST add failed: %v", err)
						err = resp.JSON(chooserResp)
						asserter.Assert(err == nil, "Decode chooser response failed: %v", err)

						asserter.AssertHTMLString(
							chooserResp.HTML,
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

						var row *queries.Row[*TestModel]
						row, err = queries.GetQuerySet(&TestModel{}).Filter("ID", chooserResp.PK).Get()
						asserter.Assert(err == nil, "Failed to fetch new object: %v", err)
						asserter.AssertEqual(name, row.Object.Name)
					}

					cnt, err := queries.GetQuerySet(&TestModel{}).Count()
					asserter.Assert(err == nil, "Failed to count new objects: %v", err)
					asserter.AssertEqual(len(names), int(cnt))

					dj.Logout()
				},
			},
			&djester.BasicTest{
				Label: "Chooser list",
				Function: func(d *djester.Tester, t *testing.T) {
					_, err := d.Login("admin_user")
					asserter := d.Assert(t, true)
					asserter.Assert(err == nil, "Login failed: %v", err)

					// Check add page HTML
					var chooserResp = new(chooser.ChooserResponse)
					resp, err := d.Get("/admin/apps/admin_test/TestModel/chooser/default/list", nil, nil)
					asserter.Assert(err == nil, "GET add failed: %v", err)
					err = resp.JSON(chooserResp)
					asserter.Assert(err == nil, "Decode chooser response failed: %v", err)

					rows, err := queries.GetQuerySet(&TestModel{}).All()
					asserter.Assert(err == nil, "Query all test models failed: %v", err)
					asserter.Assert(len(rows) > 0, "No rows were retrieved with all query.")

					asserter.AssertHTMLString(
						chooserResp.HTML,
						func(doc *goquery.Document) error {
							for _, row := range rows {
								var s = doc.Find(fmt.Sprintf("[data-chooser-value=\"%d\"]", row.Object.ID))
								if s.Length() == 0 {
									return fmt.Errorf("could not find chooser value for id %d", row.Object.ID)
								}

								s = doc.Find(fmt.Sprintf("[data-chooser-preview=\"TestModel(%s)\"]", row.Object.Name))
								if s.Length() == 0 {
									return fmt.Errorf("could not find chooser preview for id %d", row.Object.ID)
								}
							}
							return nil
						},
					)

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
