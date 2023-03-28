package app

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/app/debug"
	"github.com/Nigel2392/go-django/core/app/tool"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/go-django/core/secret"
	"github.com/Nigel2392/go-django/core/tracer"
	"github.com/Nigel2392/go-django/logger"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/middleware/csrf"
	"github.com/Nigel2392/router/v3/middleware/sessions/scsmiddleware"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/router/v3/templates"
	"github.com/alexedwards/scs/gormstore"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"gorm.io/gorm"
)

var __app *Application

// Returns the application object after it has been initialized.
func App() *Application {
	if __app == nil {
		panic("You must initialize the app before calling app.App()")
	}
	return __app
}

type AdminSite admin.AdminSite

// Server is the configuration used to initialize the server.
type Server struct {
	// The address to listen on
	Host string
	Port int
	// Wether to skip trailing slashes
	SkipTrailingSlash bool
	// The server to use
	Server *http.Server
	// The handler to use when a route is not found
	NotFoundHandler router.Handler
	// SSL options
	CertFile string
	KeyFile  string
}

// RedirectServer is the configuration used to initialize the redirect server.
// This will only be used if the server is running in SSL mode.
type RedirectServer struct {
	Host       string
	Port       int
	URL        string
	StatusCode int
}

// Config is the configuration used to initialize the application.
// You must provide a server configuration!
type Config struct {
	// The hostnames that are allowed to access the application.
	AllowedHosts []string

	// Options for the rate limiter.
	// If not specified, the rate limiter will not be used.
	RateLimitOptions *middleware.RateLimitOptions

	// The secret key for the application.
	SecretKey string

	// The HTTP server configuration.
	Server         *Server
	RedirectServer *RedirectServer

	// Database settings:
	// GORM Config
	// Database.DB will be initialized with Database.Init()
	//
	//	Config *gorm.Config
	DBConfig *db.DatabasePoolItem

	// Email settings:
	// Timeout to wait for a response from the server
	// Errors if the timeout is reached.
	// TLS Config to use when USE_SSL is true.
	// Default smpt authentication method
	// Hook for when an email will be sent
	Mail *email.Manager

	// Size of the file queue
	// Hook for when a file is read, or written in the media directory.
	File *fs.Manager

	// Use template cache?
	// Template cache to use, if enabled
	// Default base template suffixes
	// Default directory to look in for base templates
	// Functions to add to templates
	// Template file system
	//
	//	USE_TEMPLATE_CACHE bool
	//	BASE_TEMPLATE_SUFFIXES []string
	//	BASE_TEMPLATE_DIRS []string
	//	TEMPLATE_DIRS      []string
	//	DEFAULT_FUNCS template.FuncMap
	//	TEMPLATEFS fs.FS
	Templates *templates.Manager

	Middlewares          []router.Middleware
	DefaultTemplateFuncs template.FuncMap

	// Adminsite settings
	// Name and URL of the admin site must be set in order for it to be used.
	Admin *AdminSite

	// DefaultFlags are the default flags that will be added to the application.
	// You can add more flags by using the Application.Flags().Register() function.
	DefaultFlags []*flag.Command

	// Default error handler on panics.
	ErrorHandler func(error, *request.Request)
}

// Application is the main application object.
//
// This is used to store all the application data.
type Application struct {
	// Wether the application is in debug mode.
	DEBUG bool

	// The secret key for the application.
	// This is used to encrypt, decrypt, and sign data.
	SecretKey secret.Key

	// The session manager to use.
	sessionManager *scs.SessionManager

	// A pool of database connections.
	// This is used to store multiple database connections.
	// This is a generic type, so you can use any type of database connection,
	// as long as it implements the PoolItem interface.
	Pool db.Pool[*gorm.DB]

	// The default database connection.
	defaultDatabase db.PoolItem[*gorm.DB]

	Router *router.Router
	Logger request.Logger

	initted bool
	config  *Config

	flags *flag.Flags

	adminSite *admin.AdminSite
}

// Initialize a new application.
//
// This will set up the database, sessionmanager and other things.
//
// This is also where to register extra command-line flags.
func New(c Config) *Application {
	var config = &c
	if config.Server == nil {
		panic("You must provide a server configuration.")
	}
	if len(config.AllowedHosts) == 0 {
		panic("You must provide at least one allowed host.")
	}
	if config.Templates == nil {
		panic("You must provide a template manager.")
	}
	if config.DBConfig == nil {
		config.DBConfig = &db.DatabasePoolItem{
			DEFAULT_DATABASE: "sqlite",
			DB_NAME:          "sqlite.db",
		}
	}

	// Initialize the default database.
	config.DBConfig.DBKey = db.DEFAULT_DATABASE_KEY
	config.DBConfig.Init()

	// Initialize the secret key.
	var key secret.Key
	if config.SecretKey == "" {
		key = secret.KEY
	} else {
		key = secret.New(config.SecretKey)
	}

	var conf = *config
	config = &conf

	// Set up the logger.
	var logger = logger.NewBatchLogger(logger.INFO, 25, 5*time.Second, os.Stdout, "Go-Django ")
	logger.Colorize = true

	// Initialize the application object.
	var a = &Application{
		SecretKey:       key,
		config:          config,
		defaultDatabase: config.DBConfig,
		Pool:            db.NewPool(config.DBConfig),
		Logger:          logger,
		flags:           flag.NewFlags("Go-Django", flag.ExitOnError),
	}
	a.flags.Info = `Go-Django is a web framework written in Go.
It is inspired by the Django web framework for Python.
This is Go-Django's default command line interface.`

	for _, f := range config.DefaultFlags {
		a.flags.RegisterCommand(f)
	}

	// Initialize default flags
	a.flags.Register("startapp", "", "Initialize a new application with the given name.", tool.StartApp)
	a.flags.RegisterPtr(&a.DEBUG, false, "debug", "Run the application in debug mode.", nil)

	__app = a
	a.initted = true

	auth.Init(a.Pool, a.flags)

	if config.File != nil {
		a.config.File.Init()
	}

	if config.Mail != nil {
		a.config.Mail.Init()
	}

	a.config.Templates.Init()

	a.setupSessionManager()

	a.setupRouter()

	response.TEMPLATE_MANAGER = a.config.Templates

	if a.config.Admin != nil && a.config.Admin.Name != "" && a.config.Admin.URL != "" {
		var adminSiteDePtr = *a.config.Admin
		a.adminSite = (*admin.AdminSite)(&adminSiteDePtr)
		a.adminSite.DBPool = a.Pool
		a.adminSite.Defaults()
		a.adminSite.Init()
		a.adminSite.Register(
			&auth.User{},
			&auth.Group{},
			&auth.Permission{},
		)
	}

	return a
}

// setupSessionManager sets up the session manager.
func (a *Application) setupSessionManager() {
	// Set up the session manager.
	var sessionManager = scs.New()
	sessionManager.Lifetime = 24 * time.Hour * 7
	sessionManager.IdleTimeout = 24 * time.Hour * 7
	sessionManager.Cookie.Name = "session_id"
	sessionManager.Cookie.Domain = ""
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = false
	var store scs.Store
	var err error
	if store, err = gormstore.New(a.config.DBConfig.DB()); err != nil {
		store = memstore.New()
	}
	sessionManager.Store = store
	a.sessionManager = sessionManager
}

// Setup the router.
func (a *Application) setupRouter() {
	// Provide the router with some of the app's settings.
	a.Router = router.NewRouter(true)
	a.Router.NotFoundHandler = a.config.Server.NotFoundHandler

	a.Router.Use(
		middleware.XFrameOptions(middleware.XFrameDeny),
		csrf.Middleware,
		middleware.AllowedHosts(a.config.AllowedHosts...),
		scsmiddleware.SessionMiddleware(a.sessionManager),
		auth.AddUserMiddleware(),
		LoggerMiddleware(a.Logger),
	)

	if a.config.RateLimitOptions != nil {
		a.Router.Use(
			middleware.RateLimiterMiddleware(a.config.RateLimitOptions),
		)
	}

	// Add the default middlewares.
	a.Router.Use(a.config.Middlewares...)

	if a.config.File != nil {
		// Get the registrars for the static/media files.
		var staticHandler, mediaHandler = a.config.File.Registrars()
		if staticHandler != nil {
			a.Router.AddGroup(staticHandler)
		}
		if mediaHandler != nil {
			a.Router.AddGroup(mediaHandler)
		}
	}

}

// Run the application.
//
// This will start the server and listen for requests.
//
// If the server is running in SSL mode, this will also start a redirect server.
func (a *Application) Run() error {
	if !a.initted {
		panic("You must call Init() before calling Run()")
	}

	if !a.flags.Ran() {
		a.flags.Run()
	}

	tracer.STACKLOGGER_UNSAFE = a.DEBUG

	if a.DEBUG {
		tracer.DisallowPackage("router/.*/middleware")
		var databases = make([]debug.DatabaseSetting, 0)
		for _, db := range a.Pool.Databases() {
			var settingDB = db.DB()
			var setting = debug.DatabaseSetting{
				ENGINE: settingDB.Config.Dialector.Name(),
				NAME:   settingDB.Name(),
				KEY:    string(db.Key()),
			}
			databases = append(databases, setting)
		}
		var settings = debug.AppSettings{
			DEBUG:     a.DEBUG,
			HOST:      a.config.Server.Host,
			PORT:      a.config.Server.Port,
			ROUTES:    a.Router.String(),
			DATABASES: databases,
		}
		a.Router.Use(debug.StacktraceMiddleware(&settings))
	} else {
		if a.config.ErrorHandler != nil {
			a.Router.Use(middleware.Recoverer(a.config.ErrorHandler))
		} else {
			a.Router.Use(middleware.Recoverer(func(err error, r *request.Request) {
				a.Logger.Error(err)
				r.Response.WriteHeader(http.StatusInternalServerError)
			}))
		}
	}

	if a.config.Templates != nil {
		var funcMap = make(template.FuncMap)
		if a.config.File != nil {
			funcMap["static"] = a.config.File.AsStaticURL
			funcMap["media"] = a.config.File.AsMediaURL
		}
		funcMap["url"] = a.Router.URLFormat
		for k, v := range a.config.DefaultTemplateFuncs {
			funcMap[k] = v
		}
		a.config.Templates.DEFAULT_FUNCS = funcMap
	}

	a.Router.AddGroup(a.adminSite.URLS())

	var server = a.server()
	if a.config.Server.CertFile != "" && a.config.Server.KeyFile != "" {
		// SSL is true, so we will listen on the TLS port.
		// This will also automatically redirect all HTTP requests to HTTPS.
		go a.Redirect()
		a.Logger.Infof("Listening on https://%s\n", server.Addr)

		return server.ListenAndServeTLS(a.config.Server.CertFile, a.config.Server.KeyFile)
	}
	a.Logger.Infof("Listening on http://%s\n", server.Addr)
	return server.ListenAndServe()
}

func (a *Application) server() *http.Server {
	var httpServer *http.Server = &http.Server{}
	if a.config.Server != nil && a.config.Server.Server != nil {
		httpServer = a.config.Server.Server
	}
	// Set the host and port.
	if a.config.Server.Host != "" && a.config.Server.Port != 0 {
		httpServer.Addr = fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
	}
	// Set the handler to the router.
	httpServer.Handler = a.Router
	return httpServer
}

// Set up a redirect server to redirect all HTTP requests to HTTPS.
//
// This is only used if the SSL config is set to true.
func (a *Application) Redirect() error {
	// SSL is true, so we will listen on the TLS port.
	//
	// This will also automatically redirect all HTTP requests to HTTPS.
	if a.config.RedirectServer == nil {
		var errMsg = "You must specify a redirect server configuration."
		a.Logger.Error(errMsg)
		//lint:ignore ST1005 This is an error message.
		return errors.New(errMsg)
	}
	// If a redirect server is specified, we will run that in a goroutine.
	//
	// This will redirect all HTTP requests to the HTTPS server.
	var s = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.RedirectServer.Host, a.config.RedirectServer.Port),
		Handler: http.RedirectHandler(a.config.RedirectServer.URL, a.config.RedirectServer.StatusCode),
	}
	return s.ListenAndServe()
}

// Main method to register models with the application.
//
// If an admin site is registered, the models will be registered with the admin site when specified.
//
// If the key is an empty string, the model will be registered with the default database.
func (a *Application) Register(toAdmin bool, key db.DATABASE_KEY, models ...any) {
	if a.Pool == nil {
		panic("You must call Init() before calling Register()")
	}
	if key == "" {
		key = db.DEFAULT_DATABASE_KEY
	}
	a.Pool.Register(key, models...)
	if toAdmin && a.adminSite != nil {
		a.adminSite.Register(models...)
	} else if toAdmin && a.adminSite == nil {
		panic("The admin site is nil!")
	}
}
