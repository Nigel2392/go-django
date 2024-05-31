package django

import (
	"crypto/tls"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"sync/atomic"
	"text/template"

	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/internal/http_"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/justinas/nosurf"
	"github.com/pkg/errors"
)

type AppConfig interface {
	Name() string
	URLs() []core.URL
	Middleware() []core.Middleware
	Initialize(settings Settings) error
	Processors() []func(tpl.RequestContext)
	Templates() fs.FS
	OnReady() error
}

type Application struct {
	Settings    Settings
	Apps        *orderedmap.OrderedMap[string, AppConfig]
	Middleware  []core.Middleware
	URLs        []core.URL
	Mux         *mux.Mux
	quitter     func() error
	initialized *atomic.Bool
}

type Option func(*Application) error

var Global *Application

func App(opts ...Option) *Application {
	if Global == nil {
		Global = &Application{
			Apps:       orderedmap.NewOrderedMap[string, AppConfig](),
			Middleware: make([]core.Middleware, 0),
			URLs:       make([]core.URL, 0),
			Mux:        mux.New(),

			initialized: new(atomic.Bool),
		}
	}

	for i, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(Global); err != nil {
			assert.Fail("Error initializing django application %d: %s", i, err)
		}
	}

	return Global
}

func (a *Application) App(name string) AppConfig {
	var app, ok = a.Apps.Get(name)
	assert.True(ok, "App %s not found", name)
	return app
}

// Config returns the value of the key from the settings
// Shortcut for Application.Settings.Get(key)
func (a *Application) Config(key string) interface{} {
	return ConfigGet[interface{}](a.Settings, key)
}

func (a *Application) Register(apps ...any) {
	for _, appType := range apps {

		var app AppConfig
		switch v := appType.(type) {
		case AppConfig:
			app = v
		case func() AppConfig:
			app = v()
		default:
			var rVal = reflect.ValueOf(appType)

			assert.True(rVal.Kind() == reflect.Func, "Invalid type")

			var retVal = rVal.Call(nil)

			assert.True(len(retVal) == 1, "Invalid return type")

			var vInt = retVal[0].Interface()

			assert.False(vInt == nil, "Invalid return type")

			var ok bool
			app, ok = vInt.(AppConfig)
			assert.True(ok, "Invalid return type")
		}

		var appName = app.Name()
		assert.Truthy(appName, "App name cannot be empty")

		var _, ok = a.Apps.Get(appName)
		assert.False(ok, "App %s already registered", appName)

		a.Apps.Set(appName, app)
	}
}

func (a *Application) handleErrorCodePure(w http.ResponseWriter, r *http.Request, err any, code int) {
	assert.False(code == 0, "code cannot be 0")

	var pure error = errs.Convert(err, errs.ErrUnknown)
	var handler, ok = a.Settings.Get(fmt.Sprintf("Handler%d", code))
	if handler != nil && ok {
		handler.(func(http.ResponseWriter, *http.Request, error))(w, r, pure)
		return
	}

	http.Error(w, pure.Error(), int(code))
}

func (a *Application) ServerError(err error, w http.ResponseWriter, r *http.Request) {
	var serverErrInt = except.GetServerError(err)
	if serverErrInt == nil {
		a.handleErrorCodePure(w, r, nil, http.StatusInternalServerError)
		return
	}

	var serverErr = serverErrInt.(*except.HttpError)
	a.handleErrorCodePure(w, r, serverErr.Message, serverErr.Code)
}

func (a *Application) Initialize() error {

	if err := assert.False(a.Settings == nil, "Settings cannot be nil"); err != nil {
		return err
	}

	a.Mux.Use(
		http_.RequestSignalMiddleware,
		middleware.Recoverer(a.ServerError),
		middleware.AllowedHosts(
			ConfigGet(a.Settings, "ALLOWED_HOSTS", []string{"*"})...,
		),
	)

	if core.STATIC_URL != "" {
		a.Mux.Handle(
			mux.GET,
			fmt.Sprintf("%s*", core.STATIC_URL),
			http.StripPrefix(core.STATIC_URL, staticfiles.Handler),
		)
	}

	for _, m := range a.Middleware {
		m.Register(a.Mux)
	}

	for _, u := range a.URLs {
		u.Register(a.Mux)
	}

	tpl.Processors(func(rc tpl.RequestContext) {
		rc.Set("Application", a)
		rc.Set("Settings", a.Settings)
		rc.Set("CsrfToken", nosurf.Token(rc.Request()))
	})

	tpl.Funcs(template.FuncMap{
		"static": func(path string) string {
			return fmt.Sprintf("%s%s", core.STATIC_URL, path)
		},
	})

	var err error
	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		if err = app.Initialize(a.Settings); err != nil {
			return errors.Wrapf(err, "Error initializing app %s", app.Name())
		}
	}

	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		var urls = app.URLs()
		for _, url := range urls {
			url.Register(a.Mux)
		}

		var middleware = app.Middleware()
		for _, m := range middleware {
			m.Register(a.Mux)
		}

		var processors = app.Processors()
		tpl.Processors(processors...)

		var templates = app.Templates()
		if templates != nil {
			tpl.AddFS(templates, nil)
		}
	}

	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		if err = app.OnReady(); err != nil {
			return err
		}
	}

	a.initialized.Store(true)

	return nil
}

func (a *Application) Quit() error {
	if a.quitter != nil {
		return a.quitter()
	}
	return nil
}

func (a *Application) Serve() error {
	if !a.initialized.Load() {
		if err := a.Initialize(); err != nil {
			return err
		}
	}

	var (
		HOST      = ConfigGet(a.Settings, "HOST", "localhost")
		PORT      = ConfigGet(a.Settings, "PORT", "8080")
		TLSCert   = ConfigGet[string](a.Settings, "TLS_CERT")
		TLSKey    = ConfigGet[string](a.Settings, "TLS_KEY")
		TLSConfig = ConfigGet[*tls.Config](a.Settings, "TLS_CONFIG")
		addr      = fmt.Sprintf("%s:%s", HOST, PORT)
		server    = &http.Server{
			Addr:      addr,
			Handler:   nosurf.New(a.Mux),
			TLSConfig: TLSConfig,
		}
	)

	a.quitter = func() (err error) {
		err = server.Close()
		a.quitter = nil
		return err
	}

	if TLSCert != "" && TLSKey != "" {
		fmt.Printf("Listening on https://%s\n", addr)
		return server.ListenAndServeTLS(TLSCert, TLSKey)
	} else {
		fmt.Printf("Listening on http://%s\n", addr)
		return server.ListenAndServe()
	}
}
