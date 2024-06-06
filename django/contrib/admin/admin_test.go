package admin_test

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/session"
	"github.com/Nigel2392/django/core/attrs"
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

var (
	HOST                = "localhost:22392"
	ADD_USER_MIDDLEWARE = authentication.AddUserMiddleware(func(r *http.Request) authentication.User {
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
	})
)

var runTests = true

func init() {
	admin.AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition) {
		w.Write([]byte("testing app"))
		w.Write([]byte("\n"))
		w.Write([]byte(app.Name))
	}

	admin.ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
		w.Write([]byte("testing list"))
		w.Write([]byte("\n"))
		w.Write([]byte(model.Name))
	}

	admin.ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
		w.Write([]byte("testing add"))
		w.Write([]byte("\n"))
		w.Write([]byte(model.Name))
	}

	admin.ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) {
		w.Write([]byte("testing edit"))
		w.Write([]byte("\n"))
		w.Write([]byte(model.Name))
		w.Write([]byte(" ("))
		w.Write([]byte(strconv.Itoa(instance.(*TestModelStruct).ID)))
		w.Write([]byte(")"))
	}

	admin.ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) {
		w.Write([]byte("testing delete"))
		w.Write([]byte("\n"))
		w.Write([]byte(model.Name))
		w.Write([]byte(" ("))
		w.Write([]byte(strconv.Itoa(instance.(*TestModelStruct).ID)))
		w.Write([]byte(")"))
	}

	admin.RegisterApp("test", admin.ModelOptions{
		Name:  "TestModel",
		Model: &TestModelStruct{},
		GetForID: func(identifier any) (attrs.Definer, error) {
			return &TestModelStruct{
				ID:   1,
				Name: "Test",
			}, nil
		},
		GetList: func(amount, offset uint) ([]attrs.Definer, error) {
			return []attrs.Definer{
				&TestModelStruct{
					ID:   1,
					Name: "Test",
				},
			}, nil
		},
	})

	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         true,
			"HOST":          strings.Split(HOST, ":")[0],
			"PORT":          strings.Split(HOST, ":")[1],
			"DATABASE": func() *sql.DB {
				var db, err = sql.Open("sqlite3", "file::memory:?cache=shared")
				if err != nil {
					panic(err)
				}
				return db
			}(),
		}),
		django.Apps(
			session.NewAppConfig,
			admin.NewAppConfig,
		),
	)

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

	http.DefaultClient.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: nil,
	})

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

func login(t *testing.T) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "http://"+HOST+"/tests/auto-login/", nil)
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
	if !testBufferEquals(t, &buf, "Logged in") {
		t.Error("Expected Logged in, got", buf.String())
		os.Exit(1)
	}
}

func logout(t *testing.T) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "http://"+HOST+"/tests/auto-logout/", nil)
	if err != nil {
		t.Error(err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Expected status OK")
		os.Exit(1)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)

	if !testBufferEquals(t, &buf, "Logged out") {
		t.Error("Expected Logged out, got", buf.String())
		os.Exit(1)
	}
}

func asRegularUser(t *testing.T) {

	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "http://"+HOST+"/tests/user-level/", nil)
	if err != nil {
		t.Error(err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Expected status OK")
		os.Exit(1)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)

	if !testBufferEquals(t, &buf, "Logged in") {
		t.Error("Expected Logged in, got", buf.String())
		os.Exit(1)
	}
}

var handlerTests = []HandlerTest{
	{
		Name:            "App",
		LoggedIn:        true,
		IsAdministrator: true,
		URL:             "/admin/apps/test",
		Expected:        "testing app\ntest",
	},
	{
		Name:            "List",
		LoggedIn:        true,
		IsAdministrator: true,
		URL:             "/admin/apps/test/model/TestModel",
		Expected:        "testing list\nTestModel",
	},
	{
		Name:            "Add",
		LoggedIn:        true,
		IsAdministrator: true,
		URL:             "/admin/apps/test/model/TestModel/add",
		Expected:        "testing add\nTestModel",
	},
	{
		Name:            "Edit",
		LoggedIn:        true,
		IsAdministrator: true,
		URL:             "/admin/apps/test/model/TestModel/edit/1",
		Expected:        "testing edit\nTestModel (1)",
	},
	{
		Name:            "Delete",
		LoggedIn:        true,
		IsAdministrator: true,
		URL:             "/admin/apps/test/model/TestModel/delete/1",
		Expected:        "testing delete\nTestModel (1)",
	},
	{
		Name:            "Admin",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "You need to login\n",
	},
	{
		Name:            "App",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "You need to login\n",
	},
	{
		Name:            "List",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel",
		Expected:        "You need to login\n",
	},
	{
		Name:            "Add",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/add",
		Expected:        "You need to login\n",
	},
	{
		Name:            "Edit",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/edit/1",
		Expected:        "You need to login\n",
	},
	{
		Name:            "Delete",
		LoggedIn:        false,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/delete/1",
		Expected:        "You need to login\n",
	},
	{
		Name:            "App",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "Unauthorized\n",
	},
	{
		Name:            "List",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel",
		Expected:        "Unauthorized\n",
	},
	{
		Name:            "Admin",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test",
		Expected:        "Unauthorized\n",
	},
	{
		Name:            "Add",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/add",
		Expected:        "Unauthorized\n",
	},
	{
		Name:            "Edit",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/edit/1",
		Expected:        "Unauthorized\n",
	},
	{
		Name:            "Delete",
		LoggedIn:        true,
		IsAdministrator: false,
		URL:             "/admin/apps/test/model/TestModel/delete/1",
		Expected:        "Unauthorized\n",
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
			var buf bytes.Buffer
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
				os.Exit(1)
			}

			res, err = http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				os.Exit(1)
			}

			buf.ReadFrom(res.Body)
			if !testBufferEquals(nil, &buf, test.Expected) {
				t.Errorf("Expected %s, got %s (%d != %d)", test.Expected, buf.String(), len(test.Expected), buf.Len())
			}
		})
	}
}
