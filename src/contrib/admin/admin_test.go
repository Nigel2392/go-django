package admin_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type TestModelStruct struct {
	ID   int
	Name string
}

func (t *TestModelStruct) String() string {
	return t.Name
}

func (t *TestModelStruct) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions[*TestModelStruct](t)
}

func getUserFromRequest(r *http.Request) authentication.User {
	var session = sessions.Retrieve(r)
	// fmt.Println("add user middleware", session)
	// return nil
	var user = session.Get("user")
	if user == nil {
		return &User{
			LoggedIn:        false,
			IsAdministrator: false,
		}
	}
	var (
		userID = user.(int)
		isLoggedIn,
		isAdmin bool
	)
	if userID >= 1 {
		isLoggedIn = true
	}
	if userID == 2 {
		isAdmin = true
	}
	return &User{
		LoggedIn:        isLoggedIn,
		IsAdministrator: isAdmin,
	}
}

var (
	HOST                = "localhost:22392"
	ADD_USER_MIDDLEWARE = authentication.AddUserMiddleware(getUserFromRequest)
)

var runTests = true

func init() {
	//admin.AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) {
	//	w.Write([]byte("testing app"))
	//	w.Write([]byte("\n"))
	//	w.Write([]byte(app.Name))
	//}
	//
	//admin.ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
	//	w.Write([]byte("testing list"))
	//	w.Write([]byte("\n"))
	//	w.Write([]byte(model.Name))
	//}
	//
	//admin.ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
	//	w.Write([]byte("testing add"))
	//	w.Write([]byte("\n"))
	//	w.Write([]byte(model.Name))
	//}
	//
	//admin.ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) {
	//	w.Write([]byte("testing edit"))
	//	w.Write([]byte("\n"))
	//	w.Write([]byte(model.Name))
	//	w.Write([]byte(" ("))
	//	w.Write([]byte(strconv.Itoa(instance.(*TestModelStruct).ID)))
	//	w.Write([]byte(")"))
	//}
	//
	//admin.ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) {
	//	w.Write([]byte("testing delete"))
	//	w.Write([]byte("\n"))
	//	w.Write([]byte(model.Name))
	//	w.Write([]byte(" ("))
	//	w.Write([]byte(strconv.Itoa(instance.(*TestModelStruct).ID)))
	//	w.Write([]byte(")"))
	//}

	// Configure the auth system for the admin
	admin.ConfigureAuth(admin.AuthConfig{
		GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
			return auth.UserLoginForm(r, formOpts...)
		},
		Logout: auth.Logout,
	})

	attrs.RegisterModel(&TestModelStruct{})

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &TestModelStruct{},
		GetInstance: func(ctx context.Context, identifier any) (interface{}, error) {
			return &TestModelStruct{
				ID:   1,
				Name: "Test",
			}, nil
		},
		GetInstances: func(ctx context.Context, amount, offset uint) ([]interface{}, error) {
			return []interface{}{
				&TestModelStruct{
					ID:   1,
					Name: "Test",
				},
			}, nil
		},
	})

	admin.RegisterApp("test",
		admin.AppOptions{},
		admin.ModelOptions{
			Model: &TestModelStruct{},
		})

	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         true,
			"HOST":          strings.Split(HOST, ":")[0],
			"PORT":          strings.Split(HOST, ":")[1],
			"DATABASE": func() drivers.Database {
				var _, db = testdb.Open()
				return db
			}(),
		}),
		django.Apps(
			session.NewAppConfig,
			admin.NewAppConfig,
		),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
			django.FlagSkipCmds,
		),
	)

	app.Mux.Any("/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this page should not be hit!"))
		panic("this page should not be hit!")
	}), "index")

	// register the auth errors hook to redirect to the login page
	// on login failed - normally this is setup by an auth package itself
	// but we don't need it here
	// the hook normally redirects to the login page of the auth package,
	// we redirect to an index page (which should not be hit,
	// hook functionality can be changed, like in admin_auth_errors.go).
	autherrors.RegisterHook("index")

	var err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Mux.Use(ADD_USER_MIDDLEWARE)

	app.Mux.Handle(mux.ANY, "tests/auto-login/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var session = sessions.Retrieve(r)
		session.RenewToken()
		session.Set("user", 2)
		w.Write([]byte("Logged in"))
	}))

	app.Mux.Handle(mux.ANY, "tests/user-level/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var session = sessions.Retrieve(r)
		session.RenewToken()
		session.Set("user", 1)
		w.Write([]byte("Logged in"))
	}))

	app.Mux.Handle(mux.ANY, "tests/auto-logout/", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var session = sessions.Retrieve(r)
		session.RenewToken()
		session.Set("user", 0)
		w.Write([]byte("Logged out"))
	}))

	gob.Register(&User{})

	http.DefaultClient.Jar, err = cookiejar.New(nil)

	if err != nil {
		panic(err)
	}

	go func() {
		var err = app.Serve()
		if err != nil {
			runTests = false
		}
	}()

	time.Sleep(1 * time.Second)
}

func testBufferEquals(t *testing.T, buf *bytes.Buffer, expected string) bool {
	if buf.String() != expected {
		if t != nil {
			t.Errorf("Expected %s, got %s (%d != %d)", expected, buf.String(), len(expected), buf.Len())
		}
		return false
	}
	return true
}

type User struct {
	LoggedIn        bool
	IsAdministrator bool
}

func (l *User) IsAuthenticated() bool {
	return l.LoggedIn
}

func (l *User) IsAdmin() bool {
	return l.IsAdministrator
}

type HandlerTest struct {
	Name            string
	LoggedIn        bool
	IsAdministrator bool
	URL             string
	Expected        string
}

func AuthURL(url string, expected string) func(t *testing.T) {
	return func(t *testing.T) {
		var req *http.Request
		var err error

		req, err = http.NewRequest("GET", "http://"+HOST+url, nil)
		if err != nil {
			t.Error(err)
			os.Exit(1)
		}

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			os.Exit(1)
		}
		if response.StatusCode != http.StatusOK {
			t.Error("Expected status OK")
			os.Exit(1)
		}

		defer response.Body.Close()

		var buf bytes.Buffer
		buf.ReadFrom(response.Body)

		if !testBufferEquals(t, &buf, expected) {
			t.Error("Expected", expected, "got", buf.String())
			os.Exit(1)
		}
	}
}

var (
	login         = AuthURL("/tests/auto-login/", "Logged in")
	logout        = AuthURL("/tests/auto-logout/", "Logged out")
	asRegularUser = AuthURL("/tests/user-level/", "Logged in")
)

var handlerTests = []HandlerTest{
	//{
	//	Name:            "App",
	//	LoggedIn:        true,
	//	IsAdministrator: true,
	//	URL:             "/admin/apps/test",
	//	Expected:        "testing app\ntest",
	//},
	//{
	//	Name:            "List",
	//	LoggedIn:        true,
	//	IsAdministrator: true,
	//	URL:             "/admin/apps/test/TestModel",
	//	Expected:        "testing list\nTestModel",
	//},
	//{
	//	Name:            "Add",
	//	LoggedIn:        true,
	//	IsAdministrator: true,
	//	URL:             "/admin/apps/test/TestModel/add",
	//	Expected:        "testing add\nTestModel",
	//},
	//{
	//	Name:            "Edit",
	//	LoggedIn:        true,
	//	IsAdministrator: true,
	//	URL:             "/admin/apps/test/TestModel/edit/1",
	//	Expected:        "testing edit\nTestModel (1)",
	//},
	//{
	//	Name:            "Delete",
	//	LoggedIn:        true,
	//	IsAdministrator: true,
	//	URL:             "/admin/apps/test/TestModel/delete/1",
	//	Expected:        "testing delete\nTestModel (1)",
	//},
	{
		Name:            "Admin",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "/admin/login/",
	},
	{
		Name:            "App",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "/admin/login/",
	},
	{
		Name:            "List",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel",
		Expected:        "/admin/login/",
	},
	{
		Name:            "Add",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/add",
		Expected:        "/admin/login/",
	},
	{
		Name:            "Edit",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/edit/1",
		Expected:        "/admin/login/",
	},
	{
		Name:            "Delete",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/delete/1",
		Expected:        "/admin/login/",
	},
	{
		Name:            "App",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "/admin/relogin/",
	},
	{
		Name:            "List",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel",
		Expected:        "/admin/relogin/",
	},
	{
		Name:            "Admin",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "/admin/relogin/",
	},
	{
		Name:            "Add",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/add",
		Expected:        "/admin/relogin/",
	},
	{
		Name:            "Edit",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/edit/1",
		Expected:        "/admin/relogin/",
	},
	{
		Name:            "Delete",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/TestModel/delete/1",
		Expected:        "/admin/relogin/",
	},
}

func TestAdminHandlers(t *testing.T) {

	if !runTests {
		t.Skip("Server failed to start")
		return
	}

	for _, test := range handlerTests {
		var testName = fmt.Sprintf("%sHandler", test.Name)
		if !test.LoggedIn {
			testName += "LoggedOut"
		}
		if !test.IsAdministrator && test.LoggedIn {
			testName += "NormalUser"
		}

		t.Run(testName, func(t *testing.T) {
			var req *http.Request
			var res *http.Response
			var err error

			if test.LoggedIn && test.IsAdministrator {
				login(t)
			} else if test.LoggedIn && !test.IsAdministrator {
				asRegularUser(t)
			} else {
				logout(t)
			}

			req, err = http.NewRequest("GET", "http://"+HOST+test.URL, nil)
			if err != nil {
				t.Error(err)
				return
			}

			res, err = http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			defer res.Body.Close()

			if test.Expected != "" {
				if strings.TrimSuffix(res.Request.URL.Path, "/") != strings.TrimSuffix(test.Expected, "/") {
					t.Errorf("Expected %s, got %s", test.Expected, res.Request.URL.Path)
					var buf bytes.Buffer
					buf.ReadFrom(res.Body)
					t.Error(buf.String())
					return
				}
			}
		})
	}
}

type TestClient interface {
	Do(req *http.Request) (*http.Response, error)
	Authenticate(user users.User)
}

type testClient struct {
	baseURL string
	client  *http.Client
	user    users.User
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {

	if c.user != nil {
		req = req.WithContext(authentication.ContextWithUser(
			req.Context(), c.user,
		))
	}

	return c.client.Do(req)
}

func (c *testClient) Authenticate(user users.User) {
	c.user = user
}

type ViewTest interface {
	Name() string
	Reverse() string
	Test(t *testing.T, c TestClient, rt *mux.Route) error
}

type BaseViewTest struct {
	TestName       string
	ReverseURL     string
	ExpectedStatus int
	Execute        func(t *testing.T, c TestClient, rt *mux.Route) error
	Validate       func(t *testing.T, c TestClient, rt *mux.Route) bool
}

func (b *BaseViewTest) Name() string {
	return b.TestName
}

func (b *BaseViewTest) Reverse() string {
	return b.ReverseURL
}

func (b *BaseViewTest) Test(t *testing.T, c TestClient, rt *mux.Route) error {
	if b.Execute != nil {
		if err := b.Execute(t, c, rt); err != nil {
			t.Error(err)
		}
	}

	if b.Validate != nil && !b.Validate(t, c, rt) {
		return fmt.Errorf("validation failed for test %s", b.TestName)
	}

	return nil
}

func ExecuteViewTests(t *testing.T, tests []ViewTest, client *http.Client) {
	if !runTests {
		t.Skip("Server failed to start")
		return
	}

	var mux = django.Global.Mux
	for _, test := range tests {
		var client = &testClient{client: client}
		var reverseURL = test.Reverse()
		var rt = mux.Find(reverseURL)
		if rt == nil {
			t.Errorf(
				"Failed to reverse URL for test %s, route is nil for %s",
				test.Name(), reverseURL,
			)
			continue
		}

		t.Run(test.Name(), func(t *testing.T) {
			if err := test.Test(t, client, rt); err != nil {
				t.Error(err)
			}
		})
	}
}

var ViewTests = []ViewTest{
	&BaseViewTest{
		TestName:   "AdminIndex",
		ReverseURL: "admin:home",
		Execute: func(t *testing.T, c TestClient, rt *mux.Route) error {
			return nil
		},
	},
}
