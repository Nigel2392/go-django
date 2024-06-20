package django

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/Nigel2392/django/components"
	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/utils"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/justinas/nosurf"
	"github.com/pkg/errors"
)

// AppConfig is the interface that must be implemented by all Django applications.
//
// The AppConfig interface is used to define the structure of a Django application.
//
// It can be used to define routes, middleware, templates, and other options / handlers.
//
// The implementation of this interface can be found in django/apps/apps.go.
type AppConfig interface {
	// The application name.
	//
	// This is used to identify the application.
	//
	// An application cannot be registered twice - the name MUST be unique.
	Name() string

	// A list of callback functions to interact with the router / a sub- route.
	//
	// This can be used to register URLs for your application's handlers.
	//
	// These callback functions must take the core.Mux interface as the only argument
	// and return nothing.
	URLs() []core.URL

	// The base path for your application.
	//
	// If this is a non- empty string, a sub- route will automatically be created.
	//
	// This sub-route will then be passed to the above-mentioned list of callback functions.
	//
	// If the string is empty - direct access to the application's multiplexer will be given
	// (through the core.Mux interface).
	URLPath() string

	// An alias for core.URL
	//
	// This is just for semantics - use this to register middleware for your application.
	//
	// The implementation might change in the future to make this something more meaningful.
	//
	// We do not actively prevent you from also registering middleware in the URLs() callback.
	Middleware() []core.Middleware

	// Initialize your application.
	//
	// This can be used to retrieve variables / objects from settings (like a database).
	//
	// Generally we recommend you use this method for your applications
	// as opposed to doing stuff in toplevel init().
	//
	// Depending on the order of the registered applications, apps can depend on one- another.
	//
	// For example, this is used internally for authentication.
	//
	// I.E.: The 'sessions' app must always be registered before 'auth' in order for the auth app to work.
	Initialize(settings Settings) error
	Processors() []func(tpl.RequestContext)
	Templates() *tpl.Config

	// All apps have been initialized before OnReady() is called.
	OnReady() error
}

// The global application struct.
//
// This is a singleton object.
//
// It can only be configured once, any calls to the
// initialization function will return the old instance.
//
// This allows for any registered application to freely call the initializer function
// to work with the application instance.
//
// The application object should only be initialized once by calling `(*Application).Initialize()`
type Application struct {
	Settings    Settings
	Apps        *orderedmap.OrderedMap[string, AppConfig]
	Middleware  []core.Middleware
	URLs        []core.URL
	Mux         *mux.Mux
	Log         logger.Log
	quitter     func() error
	initialized *atomic.Bool
}

type Option func(*Application) error

var (
	Global  *Application
	Reverse = Global.Reverse
	Static  = Global.Static
	Task    = Global.Task
)

func App(opts ...Option) *Application {
	if Global == nil {
		Global = &Application{
			Apps:       orderedmap.NewOrderedMap[string, AppConfig](),
			Middleware: make([]core.Middleware, 0),
			URLs:       make([]core.URL, 0),
			Mux:        mux.New(),

			initialized: new(atomic.Bool),
		}

		Reverse = Global.Reverse
		Static = Global.Static
		Task = Global.Task
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

func (a *Application) handleErrorCodePure(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	var (
		code    = err.StatusCode()
		message = err.UserMessage()
	)

	assert.False(code == 0, "code cannot be 0")

	var handler, ok = a.Settings.Get(fmt.Sprintf("Handler%d", code))
	if handler != nil && ok {
		handler.(func(http.ResponseWriter, *http.Request, except.ServerError))(w, r, err)
		return
	}

	var markedWriter = &markedResponseWriter{
		ResponseWriter: w,
	}

	var hooks = goldcrest.Get[ServerErrorHook](HOOK_SERVER_ERROR)
	for _, hook := range hooks {
		hook(markedWriter, r, a, err)
	}

	if !markedWriter.wasWritten {
		http.Error(w, message, int(code))
	}
}

func (a *Application) veryBadServerError(err error, w http.ResponseWriter, r *http.Request) {
	a.Log.Errorf("An unexpected error occurred: %s (%s)", err, r.URL.String())
	http.Error(w, "An unexpected error occurred", http.StatusInternalServerError)
}

func (a *Application) ServerError(err error, w http.ResponseWriter, r *http.Request) {
	var serverError = except.GetServerError(err)
	if serverError == nil {
		a.veryBadServerError(err, w, r)
		return
	}

	a.Log.Errorf(
		"Error serving request (%d: %s) %s",
		serverError.StatusCode(),
		utils.Trunc(r.URL.String(), 75),
		serverError.UserMessage(),
	)
	a.handleErrorCodePure(w, r, serverError)
}

func (a *Application) Reverse(name string, args ...any) string {
	var rt, _ = a.Mux.Reverse(name, args...)
	return rt
}

func (a *Application) Static(path string) string {
	var u, err = url.Parse(path)
	if err != nil {
		panic(errors.Wrapf(err, "Invalid static URL path '%s'", path))
	}

	if u.Scheme != "" || u.Host != "" {
		return path
	}

	return fmt.Sprintf("%s%s", core.STATIC_URL, path)
}

func (a *Application) Task(description string, fn func(*Application) error) error {
	var startTime = time.Now()
	a.Log.Infof("Starting task: %s", description)
	var err = fn(a)
	var timeTaken = time.Since(startTime)
	if err != nil {
		a.Log.Errorf("Task failed: %s (%s)", description, timeTaken)
	} else {
		a.Log.Infof("Task completed: %s (%s)", description, timeTaken)
	}
	return err
}

func (a *Application) Initialize() error {

	if a.Log == nil {
		a.Log = &logger.Logger{
			Level:      logger.INF,
			OutputTime: true,
			Prefix:     "django",
			WrapPrefix: logger.ColoredLogWrapper,
		}

		a.Log.SetOutput(
			logger.OutputAll,
			os.Stdout,
		)

		logger.Setup(a.Log)
	}

	if err := assert.False(a.Settings == nil, "Settings cannot be nil"); err != nil {
		return err
	}

	a.Mux.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		a.Log.Debug("Page not found")
		a.ServerError(except.NewServerError(
			http.StatusNotFound,
			"Page not found",
		), w, r)
	}

	a.Mux.Use(
		RequestSignalMiddleware,
		// middleware.Recoverer(a.veryBadServerError),
		middleware.AllowedHosts(
			ConfigGet(a.Settings, "ALLOWED_HOSTS", []string{"*"})...,
		),
		a.loggerMiddleware,
	)

	a.Log.Debugf(
		"Initializing static files at '%s'",
		core.STATIC_URL,
	)

	if core.STATIC_URL != "" {
		a.Mux.Handle(
			mux.GET,
			fmt.Sprintf("%s*", core.STATIC_URL),
			http.StripPrefix(core.STATIC_URL, staticfiles.Handler),
		)
	}

	a.Log.Debug("Initializing 'Django' middleware")

	for _, m := range a.Middleware {
		m.Register(a.Mux)
	}

	a.Log.Debug("Initializing 'Django' URLs")

	for _, u := range a.URLs {
		u.Register(a.Mux)
	}

	tpl.Processors(func(rc tpl.RequestContext) {
		rc.Set("Application", a)
		rc.Set("Settings", a.Settings)
		rc.Set("CsrfToken", nosurf.Token(rc.Request()))
	})

	tpl.Funcs(template.FuncMap{
		"static": a.Static,
		"url": func(name string, args ...any) string {
			var rt, err = a.Mux.Reverse(name, args...)
			if err != nil {
				panic(fmt.Sprintf("URL %s not found", name))
			}
			return rt
		},
		"component": func(name string, args ...interface{}) template.HTML {
			var c = components.Render(name, args...)
			var buf = new(bytes.Buffer)
			var ctx = context.Background()
			c.Render(ctx, buf)
			return template.HTML(buf.String())
		},
		"T": fields.T,
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
		if len(urls) > 0 {
			var path = app.URLPath()

			var r core.Mux = a.Mux
			if path != "" {
				r = r.Handle(
					mux.ANY, path, nil, app.Name(),
				)
			}

			for _, url := range urls {
				url.Register(r)
			}
		}

		var middleware = app.Middleware()
		for _, m := range middleware {
			m.Register(a.Mux)
		}

		var processors = app.Processors()
		tpl.Processors(processors...)

		var templates = app.Templates()
		if templates != nil {
			tpl.Add(*templates)
		}
	}

	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		if err = app.OnReady(); err != nil {
			return err
		}
	}

	if ConfigGet(a.Settings, "RECOVERER", true) {
		a.Mux.Use(
			middleware.Recoverer(a.ServerError),
		)
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
		a.Log.Logf(logger.INF, "Listening on https://%s\n", addr)
		return server.ListenAndServeTLS(TLSCert, TLSKey)
	} else {
		a.Log.Logf(logger.INF, "Listening on http://%s\n", addr)
		return server.ListenAndServe()
	}
}
