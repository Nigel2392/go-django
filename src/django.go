package django

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Nigel2392/go-django/src/components"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	utils_text "github.com/Nigel2392/go-django/src/utils/text"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/justinas/nosurf"
	"github.com/pkg/errors"
)

// The interface for our multiplexer
//
// This is a wrapper around the nigel2392/mux.Mux interface
type Mux interface {
	Use(middleware ...mux.Middleware)
	Handle(method string, path string, handler mux.Handler, name ...string) *mux.Route
	AddRoute(route *mux.Route)

	Any(path string, handler mux.Handler, name ...string) *mux.Route
	Get(path string, handler mux.Handler, name ...string) *mux.Route
	Post(path string, handler mux.Handler, name ...string) *mux.Route
	Put(path string, handler mux.Handler, name ...string) *mux.Route
	Patch(path string, handler mux.Handler, name ...string) *mux.Route
	Delete(path string, handler mux.Handler, name ...string) *mux.Route
}

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

	// Dependencies for the application.
	//
	// This can be used to define dependencies for the application.
	//
	// All of theses dependencies must be registered before the application is initialized.
	Dependencies() []string

	// Commands for the application.
	//
	// These commands can be used to run tasks from the command line.
	//
	// The commands are registered in the Django command registry.
	Commands() []command.Command

	// BuildRouting is used to define the routes for the application.
	// It can also be used to define middleware for the application.
	//
	// A Mux object is passed to the function which can be used to define routes.
	BuildRouting(mux Mux)

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
	Mux         *mux.Mux
	Log         logger.Log
	flags       AppFlag
	quitter     func() error
	initialized *atomic.Bool
}

type Option func(*Application) error
type AppFlag uint64

var (
	Global       *Application
	AppInstalled = Global.AppInstalled
	Reverse      = Global.Reverse
	Static       = Global.Static
	Task         = Global.Task
)

const (
	FlagSkipDepsCheck AppFlag = 1 << iota
	FlagSkipCmds
)

func GetApp[T AppConfig](name string) T {
	var app, ok = Global.Apps.Get(name)
	assert.True(ok, "App %s not found", name)
	a, ok := app.(T)
	assert.True(ok, "Invalid app type: %T", app)
	return a
}

func App(opts ...Option) *Application {
	if Global == nil {
		Global = &Application{
			Apps: orderedmap.NewOrderedMap[string, AppConfig](),
			Mux:  mux.New(),

			initialized: new(atomic.Bool),
		}

		AppInstalled = Global.AppInstalled
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

	var handler, ok = a.Settings.Get(fmt.Sprintf("EntryHandler%d", code))
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
		utils_text.Trunc(r.URL.String(), 75),
		serverError.UserMessage(),
	)

	a.handleErrorCodePure(w, r, serverError)
}

func (a *Application) Reverse(name string, args ...any) string {
	var rt, err = a.Mux.Reverse(name, args...)

	if err != nil {
		panic(fmt.Sprintf("Error reversing URL %s: %s", name, err))
	}

	if len(rt) == 0 {
		panic(fmt.Sprintf("Error reversing URL %s: %s", name, err))
	}

	var l = len(rt)
	if !strings.HasPrefix(rt, "/") {
		l += 1
	}
	if !strings.HasSuffix(rt, "/") {
		l += 1
	}

	if l == len(rt) {
		return rt
	}

	var buf = make([]byte, l)
	if !strings.HasPrefix(rt, "/") {
		buf[0] = '/'
		copy(buf[1:], rt)
	} else {
		copy(buf, rt)
	}

	if !strings.HasSuffix(rt, "/") {
		buf[l-1] = '/'
	}

	return string(buf)
}

func (a *Application) staticURL() string {
	var staticURL = []byte(
		ConfigGet(
			a.Settings,
			APPVAR_STATIC_URL,
			"/static/",
		),
	)
	if staticURL[len(staticURL)-1] != '/' {
		staticURL = append(staticURL, '/')
	}
	return string(staticURL)
}

func (a *Application) Static(path string) string {
	var u, err = url.Parse(path)
	if err != nil {
		panic(errors.Wrapf(err, "Invalid static URL path '%s'", path))
	}

	if u.Scheme != "" || u.Host != "" {
		return path
	}

	if strings.HasPrefix(path, "/") {
		return path
	}

	var staticUrl = a.staticURL()
	return fmt.Sprintf("%s%s", staticUrl, path)
}

func (a *Application) Task(description string, fn func(*Application) error) error {
	var (
		startTime = time.Now()
		w         = a.Log.NameSpace("Task")
	)

	w.Infof("Starting task: %s", description)

	var (
		err       = fn(a)
		timeTaken = time.Since(startTime)
	)
	if err != nil {
		w.Errorf("Task failed: %s (%s)", description, timeTaken)
	} else {
		w.Infof("Task completed: %s (%s)", description, timeTaken)
	}

	return err
}

func (a *Application) AppInstalled(name string) bool {
	_, ok := a.Apps.Get(name)
	return ok
}

func (a *Application) Flagged(flag AppFlag) bool {
	return (a.flags & flag) != 0
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
		a.ServerError(except.NewServerError(
			http.StatusNotFound,
			"Page not found",
		), w, r)
	}

	a.Mux.Use(
		RequestSignalMiddleware,
		// middleware.Recoverer(a.veryBadServerError),
		middleware.AllowedHosts(
			ConfigGet(a.Settings, APPVAR_ALLOWED_HOSTS, []string{"*"})...,
		),
		a.loggerMiddleware,
	)

	var staticUrl = a.staticURL()

	a.Log.Debugf(
		"Initializing static files at '%s'",
		staticUrl,
	)

	if staticUrl != "" {
		a.Mux.Handle(
			mux.GET,
			fmt.Sprintf("%s*", staticUrl),
			LoggingDisabledMiddleware(
				http.StripPrefix(staticUrl, staticfiles.EntryHandler),
			),
		)
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
			switch {
			case errors.Is(err, mux.ErrRouteNotFound):
				panic(fmt.Sprintf("URL %s not found", name))
			case errors.Is(err, mux.ErrTooManyVariables):
				panic(fmt.Sprintf("Too many variables for URL %s", name))
			case err != nil:
				panic(fmt.Sprintf("Error reversing URL %s: %s", name, err))
			}
			return rt
		},
		"has_object_perm": permissions.HasObjectPermission,
		"has_perm":        permissions.HasPermission,
		"component": func(name string, args ...interface{}) template.HTML {
			var c = components.Render(name, args...)
			var buf = new(bytes.Buffer)
			var ctx = context.Background()
			c.Render(ctx, buf)
			return template.HTML(buf.String())
		},
		"T": trans.T,
	})

	var err error
	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		var deps = app.Dependencies()

		if !a.Flagged(FlagSkipDepsCheck) {
			for _, dep := range deps {
				var _, ok = a.Apps.Get(dep)
				if !ok {
					return errors.Errorf("Dependency %q not found for app %q", dep, app.Name())
				}
			}
		}

		if err = app.Initialize(a.Settings); err != nil {
			return errors.Wrapf(err, "Error initializing app %s", app.Name())
		}
	}

	var commandRegistry = command.NewRegistry(
		"django",
		flag.ContinueOnError,
	)

	commandRegistry.Register(&command.Cmd[interface{}]{
		ID:   "help",
		Desc: "List all available commands and their usage information",
		Execute: func(m command.Manager, stored interface{}, args []string) error {
			var (
				buf  = new(bytes.Buffer)
				cmds = commandRegistry.Commands()
			)
			buf.WriteString("Available commands:\n")
			for _, cmd := range cmds {
				var description string
				var commandName = cmd.Name()
				if d, ok := cmd.(command.CommandDescriptor); ok {
					description = d.Description()
				}

				if description != "" {
					fmt.Fprintf(buf, "[%s]:\n  %s\n", commandName, description)
				} else {
					fmt.Fprintf(buf, "[%s]\n", commandName)
				}

				var flagger = flag.NewFlagSet(
					commandName,
					flag.ContinueOnError,
				)
				if err := cmd.AddFlags(m, flagger); err != nil {
					return err
				}

				flagger.SetOutput(buf)
				flagger.PrintDefaults()
				fmt.Fprintln(buf)
			}
			m.Log(buf.String())
			os.Exit(0)
			return nil
		},
	})

	commandRegistry.Register(sqlShellCommand)

	var r Mux = a.Mux
	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value

		app.BuildRouting(r)

		var processors = app.Processors()
		tpl.Processors(processors...)

		var templates = app.Templates()
		if templates != nil {
			tpl.Add(*templates)
		}

		var commands = app.Commands()
		for _, cmd := range commands {
			commandRegistry.Register(cmd)
		}
	}

	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		if err = app.OnReady(); err != nil {
			return err
		}
	}

	if ConfigGet(a.Settings, APPVAR_RECOVERER, true) {
		a.Mux.Use(
			middleware.Recoverer(a.ServerError),
		)
	}

	a.initialized.Store(true)

	// Check if running commands is disabled
	if a.Flagged(FlagSkipCmds) {
		return nil
	}

	if len(os.Args) == 2 && slices.Contains([]string{"help", "--help", "-h"}, os.Args[1]) {
		os.Args[1] = "help"
	}

	err = commandRegistry.ExecCommand(os.Args[1:])
	switch {
	case errors.Is(err, command.ErrNoCommand):
		return nil
	case errors.Is(err, command.ErrUnknownCommand):
		a.Log.Fatalf(1, "Error running command: %s", err)
		return nil
	case errors.Is(err, flag.ErrHelp):
		os.Exit(0)
		return nil
	}
	if err != nil {
		return err
	}
	if !ConfigGet(a.Settings, APPVAR_CONTINUE_AFTER_COMMANDS, false) {
		os.Exit(0)
	}
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

	var handler = nosurf.New(a.Mux)

	var hooks = goldcrest.Get[NosurfSetupHook](HOOK_SETUP_NOSURF)
	for _, hook := range hooks {
		hook(a, handler)
	}

	var (
		HOST        = ConfigGet(a.Settings, APPVAR_HOST, "localhost")
		PORT        = ConfigGet(a.Settings, APPVAR_PORT, "8080")
		TLS_PORT    = ConfigGet(a.Settings, APPVAR_TLS_PORT, "")
		TLSCert     = ConfigGet[string](a.Settings, APPVAR_TLS_CERT)
		TLSKey      = ConfigGet[string](a.Settings, APPVAR_TLS_KEY)
		TLSConfig   = ConfigGet[*tls.Config](a.Settings, APPVAR_TLS_CONFIG)
		addr_http   = fmt.Sprintf("%s:%s", HOST, PORT)
		addr_https  = fmt.Sprintf("%s:%s", HOST, TLS_PORT)
		server_http = &http.Server{
			Addr:      addr_http,
			Handler:   handler,
			TLSConfig: TLSConfig,
		}
		server_https = &http.Server{
			Addr:      addr_https,
			Handler:   handler,
			TLSConfig: TLSConfig,
		}
		listening_https = TLSCert != "" && TLSKey != "" && TLS_PORT != ""
		listening_http  = PORT != "" && PORT != "0"
	)

	a.quitter = func() (err error) {
		var (
			err1, err2 error
		)

		if listening_https {
			a.Log.Logf(logger.INF, "Shutting down http server")
			err1 = server_http.Shutdown(context.Background())
		}

		if err1 != nil && listening_https {
			err = errors.Wrap(err1, "Error closing http server")
		}

		if listening_http {
			a.Log.Logf(logger.INF, "Shutting down https server")
			err2 = server_https.Shutdown(context.Background())
		}

		if err2 != nil && listening_http {
			// err = errors.Wrap(err2, "Error closing https server")
			if err != nil {
				err = errors.Wrap(err, err2.Error())
			} else {
				err = err2
			}
		}

		a.quitter = nil
		return err
	}

	var (
		chanCt = 0
		errCh  = make(chan error, 2)
	)

	if listening_https {
		chanCt++

		a.Log.Logf(logger.INF, "Listening on https://%s (TLS)", addr_https)

		if TLS_PORT == "" || TLS_PORT == "0" {
			a.Log.Fatalf(1, "TLS_PORT must be set to a valid port number, got %q", TLS_PORT)
		}

		// Start https server, if it exits close http server
		go func() {
			errCh <- server_https.ListenAndServeTLS(TLSCert, TLSKey)
			server_http.Close()
		}()
	}

	if listening_http {
		chanCt++

		a.Log.Logf(logger.INF, "Listening on http://%s", addr_http)

		// Start http server, if it exits close https server
		go func() {
			errCh <- server_http.ListenAndServe()
			server_https.Close()
		}()
	}

	if chanCt == 0 {
		a.Log.Fatalf(1, "Server cannot be started, no valid ports found: %q, %q", PORT, TLS_PORT)
	}

	// Wait for both servers to exit
	var err error
	for i := 0; i < chanCt; i++ {
		if e := <-errCh; e != nil {
			err = e
		}
	}

	return err
}
