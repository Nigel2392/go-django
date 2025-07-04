//go:build test
// +build test

package djester

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/mux"
)

/*
	This package is meant to be used for testing purposes only.

	It is not meant to be used in production.

	It is a test suite for the go-django framework, and it is meant to be used
	in conjunction with the go-django framework.
	It is not meant to be, and cannot be used as a standalone package.
*/

const (
	ErrNoResponseBody errs.Error = "operation failed: no response body"
)

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
	Tester struct {
		Verbose      bool
		BeforeSetup  func(dj *Tester) error
		Database     Database
		Flags        []django.AppFlag
		Settings     map[string]any
		ExtraOptions []django.Option
		Apps         []AppInitFuncOrAppConfig
		Handlers     []Handler
		Tests        []Test

		// Private fields to be used by the Tester struct
		db         drivers.Database
		app        *django.Application
		testClient *http.Client
		testServer *httptest.Server
	}
)

func (d *Tester) Setup() error {

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

	if len(d.Handlers) > 0 {
		var app = apps.NewAppConfig("djester")
		app.Routing = func(mux django.Mux) {
			for _, handler := range d.Handlers {
				if handler.Name == "" {
					mux.Handle(handler.Method, handler.Route, handler.Handler)
				} else {
					mux.Handle(handler.Method, handler.Route, handler.Handler, handler.Name)
				}
			}
		}
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

	var server, err = d.app.TestServe(true)
	if err != nil {
		return err
	}

	d.testServer = server
	d.testClient = server.Client()
	return nil
}

func (d *Tester) Server() *httptest.Server {
	return d.testServer
}

func (d *Tester) Client() *http.Client {
	return d.testClient
}

func (d *Tester) DB() drivers.Database {
	return d.db
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

func (d *Tester) Assert(t *testing.T, verbose bool) Assertion {
	return &assertion{t: t, verbose: verbose}
}

func (d *Tester) Test(t *testing.T) {
	t.Helper()
	if err := d.Setup(); err != nil {
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
	b.StopTimer()
	b.ResetTimer()
	if err := d.Setup(); err != nil {
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
