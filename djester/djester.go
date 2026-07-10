//go:build test
// +build test

package djester

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
)

/*
	This package is meant to be used for testing purposes only.

	It is not meant to be used in production.

	It is a test suite for the go-django framework, and it is meant to be used
	in conjunction with the go-django framework.
	It is not meant to be, and cannot be used as a standalone package.
*/

const (
	USER_SESSION_VAR             = "djester.session.user"
	ErrNoResponseBody errs.Error = "operation failed: no response body"
)

type TB interface {
	Cleanup(f func())
	Context() context.Context
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Run(name string, f func(TB)) bool
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
}

type tWrap struct {
	*testing.T
}

func TW(t *testing.T) TB {
	return &tWrap{t}
}

func (t *tWrap) Run(name string, f func(TB)) bool {
	t.Helper()
	return t.T.Run(name, func(nt *testing.T) {
		t.Helper()
		f(&tWrap{nt})
	})
}

type bWrap struct {
	*testing.B
}

func BW(b *testing.B) TB {
	return &bWrap{b}
}

func (t *bWrap) Run(name string, f func(TB)) bool {
	t.Helper()
	return t.B.Run(name, func(nt *testing.B) {
		t.Helper()
		f(&bWrap{nt})
	})
}

type (
	AppInitFuncOrAppConfig interface{}
	Database               struct {
		Engine           string
		ConnectionString string
	}
	Handler struct {
		Name    string
		Method  string
		Route   string
		Handler mux.Handler
	}
	TesterAuth struct {
		UnauthenticatedUser func() authentication.User
		Users               map[string]authentication.User
	}
	Tester struct {
		Verbose      bool
		BeforeSetup  func(dj *Tester) error
		Database     Database
		Auth         *TesterAuth
		Flags        []django.AppFlag
		Settings     map[string]any
		ExtraOptions []django.Option
		Apps         []AppInitFuncOrAppConfig
		Handlers     []Handler
		Tests        []Test

		// extra models that do not require an app nescessarily
		// they will be added to a custom app called 'djester' (if models exist).
		// so they can be used in queries without too much hassle.
		ExtraModels []attrs.Definer

		// Private fields to be used by the Tester struct
		db         drivers.Database
		app        *django.Application
		testClient *django.HTTPTestClient
		testServer *django.HTTPTestServer
		test       TB
	}
)

func (d *Tester) Setup(t TB) error {

	d.test = t

	if d.BeforeSetup != nil {
		if err := d.BeforeSetup(d); err != nil {
			return err
		}
	}

	var settings = make(map[string]any)
	maps.Copy(settings, d.Settings)

	if d.Database.Engine != "" {
		var db, err = drivers.Open(context.Background(), d.Database.Engine, d.Database.ConnectionString)
		if err != nil {
			return err
		}
		d.db = db
		settings[django.APPVAR_DATABASE] = db
	}
	if d.Settings == nil {
		d.Settings = make(map[string]any)
	}
	if d.Apps == nil {
		d.Apps = make([]AppInitFuncOrAppConfig, 0)
	}
	if d.Flags == nil {
		d.Flags = make([]django.AppFlag, 0)
	}
	if d.ExtraOptions == nil {
		d.ExtraOptions = make([]django.Option, 0)
	}
	if d.Tests == nil {
		d.Tests = make([]Test, 0)
	}

	var appConfigs = make([]any, 0, len(d.Apps)+1)
	for _, app := range d.Apps {
		appConfigs = append(appConfigs, app)
	}

	if len(d.Handlers) > 0 || len(d.ExtraModels) > 0 {
		var app = apps.NewAppConfig("djester")
		app.Routing = func(mux mux.Multiplexer) {
			for _, handler := range d.Handlers {
				if handler.Name == "" {
					mux.Handle(handler.Method, handler.Route, handler.Handler)
				} else {
					mux.Handle(handler.Method, handler.Route, handler.Handler, handler.Name)
				}
			}
		}
		app.ModelObjects = d.ExtraModels
		appConfigs = append(appConfigs, app)
	}

	d.Flags = append(d.Flags, django.FlagSkipCmds)
	d.Flags = append(d.Flags, django.FlagSkipChecks)

	var opts = make([]django.Option, 0, len(d.ExtraOptions))
	opts = append(opts, django.Configure(settings))
	opts = append(opts, django.Apps(appConfigs...))
	opts = append(opts, django.Flag(d.Flags...))
	opts = append(opts, d.ExtraOptions...)
	d.app = django.App(opts...)

	if err := d.app.Initialize(); err != nil {
		return err
	}

	if d.Auth != nil {
		if !django.AppInstalled("session") {
			t.Fatal("cannot initialize authentication without app 'session'")
		}

		d.app.Mux.Use(
			authentication.AddUserMiddleware(d.getUserFromRequest),
		)

		var djester = d.app.Mux.Get("/djester/auth", nil, "djester")
		djester.Post("/login/<<map_key>>", mux.NewHandler(d.loginHandler), "login")
		djester.Post("/logout", mux.NewHandler(d.logoutHandler), "logout")
	}

	var server, err = d.app.TestServe(true)
	if err != nil {
		return err
	}

	d.testServer = server
	d.testClient = server.Client()

	return nil
}

func (d *Tester) Server() *django.HTTPTestServer {
	return d.testServer
}

func (d *Tester) Client() *django.HTTPTestClient {
	return d.testClient
}

func (d *Tester) DB() drivers.Database {
	return d.db
}

func (d *Tester) getUserFromRequest(r *http.Request) authentication.User {
	var session = sessions.Retrieve(r)
	var iuserKey = session.Get(USER_SESSION_VAR)
	if iuserKey == nil {
		return d.Auth.UnauthenticatedUser()
	}

	userKey, ok := iuserKey.(string)
	if !ok {
		d.test.Fatalf("userKey has wrong type %T", userKey)
	}

	user, ok := d.Auth.Users[userKey]
	if !ok {
		d.test.Fatalf("userKey %q not present in Auth object", userKey)
	}

	return user
}

func writeAuthResponse(w http.ResponseWriter, r authResponse) error {
	var enc = json.NewEncoder(w)
	return enc.Encode(r)
}

const (
	AUTH_SESSION_NIL  = "[djester.auth.missing.session]"
	AUTH_USER_MISSING = "[djester.auth.missing]"
	AUTH_ERROR        = "[djester.auth.error]"
)

func (d *Tester) loginHandler(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)
	if session == nil {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: AUTH_SESSION_NIL,
		})
		return
	}

	var err = session.RenewToken()
	if err != nil {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: fmt.Sprintf("%s: %s", AUTH_ERROR, err.Error()),
		})
		return
	}

	userKey := mux.Vars(r).Get("map_key")
	u, ok := d.Auth.Users[userKey]
	if !ok {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: AUTH_USER_MISSING,
		})
		return
	}

	session.Set(USER_SESSION_VAR, userKey)

	if definer, ok := u.(interface {
		authentication.User
		attrs.Definer
	}); ok {
		core.SIGNAL_USER_LOGGED_IN.Send(core.UserWithRequest{
			User: definer,
			Req:  r,
		})
	}

	writeAuthResponse(w, authResponse{
		Success: true,
	})
}

func (d *Tester) logoutHandler(w http.ResponseWriter, r *http.Request) {
	var session = sessions.Retrieve(r)
	// except.Assert(session != nil, 500, "session is nil")
	if session == nil {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: AUTH_SESSION_NIL,
		})
		return
	}

	if err := session.Destroy(); err != nil {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: fmt.Sprintf("%s: %s", AUTH_ERROR, err.Error()),
		})
		return
	}

	err := core.SIGNAL_USER_LOGGED_OUT.Send(core.UserWithRequest{
		User: nil,
		Req:  r,
	})
	if err != nil {
		writeAuthResponse(w, authResponse{
			Success: false,
			Message: fmt.Sprintf("%s: %s", AUTH_ERROR, err.Error()),
		})
		return
	}

	writeAuthResponse(w, authResponse{
		Success: true,
	})
}

func (d *Tester) Close() error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return err
		}
	}
	if d.testServer != nil {
		d.testServer.Close()
	}
	if d.app != nil {
		if err := d.app.Quit(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Tester) Assert(verbose bool) Assertion {
	return &assertion{t: d.test, verbose: verbose}
}

func (d *Tester) Test(t *testing.T) {
	t.Helper()
	if err := d.Setup(TW(t)); err != nil {
		t.Errorf("failed to setup djester tests: %v", err)
		return
	}
	for _, test := range d.Tests {
		t.Run(test.Name(), func(t *testing.T) {
			test.Test(d, t)
		})
	}
}

func (d *Tester) Bench(b *testing.B) {
	b.Helper()
	b.StopTimer()
	b.ResetTimer()
	if err := d.Setup(BW(b)); err != nil {
		b.Errorf("failed to setup djester tests: %v", err)
		return
	}
	b.StartTimer()
	for _, test := range d.Tests {
		b.Run(test.Name(), func(b *testing.B) {
			test.Bench(d, b)
		})
	}
}
