package django

import (
	"crypto/tls"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"text/template"

	"github.com/Nigel2392/django/core/http_"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/justinas/nosurf"
)

type AppConfig interface {
	Name() string
	URLs() []http_.URL
	Middleware() []http_.Middleware
	Initialize(settings Settings) error
	Processors() []func(tpl.RequestContext)
	Templates() fs.FS
	OnReady() error
}

type Application struct {
	Settings   Settings
	Apps       *orderedmap.OrderedMap[string, AppConfig]
	Middleware []http_.Middleware
	URLs       []http_.URL
	Mux        *mux.Mux
	quitter    func() error
}

type Option func(*Application) error

var Global *Application

func App(opts ...Option) *Application {
	if Global == nil {
		Global = &Application{
			Apps:       orderedmap.NewOrderedMap[string, AppConfig](),
			Middleware: make([]http_.Middleware, 0),
			URLs:       make([]http_.URL, 0),
			Mux:        mux.New(),
		}
	}

	var err error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err = opt(Global); err != nil {
			panic(err)
		}
	}

	return Global
}

func (a *Application) App(name string) AppConfig {
	var app, ok = a.Apps.Get(name)
	if !ok {
		panic(fmt.Sprintf("App %s not found", name))
	}
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
			if rVal.Kind() != reflect.Func {
				panic("Invalid type")
			}

			var retVal = rVal.Call(nil)
			if len(retVal) == 0 {
				panic("Invalid return type")
			}

			var vInt = retVal[0].Interface()
			if vInt == nil {
				panic("Invalid return type")
			}
			var ok bool
			app, ok = vInt.(AppConfig)
			if !ok {
				panic("Invalid return type")
			}
		}

		var appName = app.Name()
		if appName == "" {
			panic("App name cannot be empty")
		}

		if _, ok := a.Apps.Get(appName); ok {
			panic(fmt.Sprintf("App %s already registered", appName))
		}

		a.Apps.Set(appName, app)
	}
}

func (a *Application) ServerError(err error, w http.ResponseWriter, r *http.Request) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (a *Application) Initialize() error {

	a.Mux.Use(
		// middleware.Recoverer(a.ServerError),
		middleware.AllowedHosts(
			ConfigGet(a.Settings, "ALLOWED_HOSTS", []string{"*"})...,
		),
	)

	a.Mux.Handle(
		mux.GET,
		fmt.Sprintf("%s*", http_.STATIC_URL),
		http.StripPrefix(http_.STATIC_URL, staticfiles.Handler),
	)

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
			return fmt.Sprintf("%s%s", http_.STATIC_URL, path)
		},
	})

	var err error
	for h := a.Apps.Front(); h != nil; h = h.Next() {
		var app = h.Value
		if err = app.Initialize(a.Settings); err != nil {
			return err
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

	return nil
}

func (a *Application) Quit() error {
	if a.quitter != nil {
		return a.quitter()
	}
	return nil
}

func (a *Application) Serve() error {

	if err := a.Initialize(); err != nil {
		return err
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
