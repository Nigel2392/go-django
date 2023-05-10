package app

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/app/tool"
	"github.com/Nigel2392/go-django/core/cache"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/netcache/src/client"
	logger "github.com/Nigel2392/request-logger"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/middleware/csrf"
	"github.com/Nigel2392/router/v3/middleware/sessions/scsmiddleware"
	"github.com/Nigel2392/router/v3/middleware/tracer"
	"github.com/Nigel2392/router/v3/middleware/tracer/debug"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/router/v3/templates"
	"github.com/Nigel2392/secret"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/go-sql-driver/mysql"
)

var __app *Application

// Returns the application object after it has been initialized.
func App() *Application {
	if __app == nil {
		panic("You must initialize the app before calling app.App()")
	}
	return __app
}

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
	// Loglevel to use.
	// If not specified, the default loglevel will be used.
	LogLevel logger.Loglevel

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

	// The application's cache.
	Cache client.Cache

	// Force the application to use a cache.
	//
	// If an error occurs connecting to the cache, do not proceed.
	ForceCache bool

	// Email settings:
	// Timeout to wait for a response from the server
	// Errors if the timeout is reached.
	// TLS Config to use when USE_SSL is true.
	// Default smpt authentication method
	// Hook for when an email will be sent
	Mail *email.Manager

	// Size of the file queue
	// Hook for when a file is read, or written in the media directory.
	File fs.Filer

	// The session store to use.
	SessionStore scs.Store

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

	// The database to use.
	Database *sql.DB

	// Use authentication?
	DisableAuthentication bool

	Middlewares          []router.Middleware
	DefaultTemplateFuncs template.FuncMap

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

	// The session store to use.
	sessionStore scs.Store

	// The cache to use.
	cache client.Cache

	Router *router.Router
	Logger request.Logger

	initted         bool
	useAuth         bool
	Auth            *auth.AuthApp
	config          *Config
	defaultDatabase *sql.DB
	flags           *flag.Flags
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

	// Initialize the cache.
	if config.Cache == nil && config.ForceCache {
		config.Cache = cache.NewInMemoryCache(cache.MedExpiration / 2)
	}

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
	var ll = config.LogLevel
	if ll == 0 {
		ll = logger.INFO
	}
	var lg = logger.NewBatchLogger(ll, 25, 5*time.Second, os.Stdout, "Go-Django ")
	lg.Colorize = true

	lg.Now(logger.DEBUG, "Initializing application...")

	// Initialize the application object.
	var a = &Application{
		SecretKey:       key,
		config:          config,
		Logger:          lg,
		flags:           flag.NewFlags("Go-Django", flag.ExitOnError),
		sessionStore:    config.SessionStore,
		cache:           config.Cache,
		useAuth:         !config.DisableAuthentication,
		defaultDatabase: config.Database,
	}
	a.flags.Info = `Go-Django is a web framework written in Go.
It is inspired by the Django web framework for Python.
This is Go-Django's default command line interface.`

	for _, f := range config.DefaultFlags {
		a.flags.RegisterCommand(f)
	}

	// Initialize default flags
	lg.Now(logger.DEBUG, "Initializing default flags...")
	a.flags.Register("startapp", "", "Initialize a new application with the given name.", tool.StartApp)
	a.flags.Register("newcompose", false, "Initialize a new composer with the given name.", tool.NewDockerCompose)
	a.flags.RegisterPtr(&a.DEBUG, false, "debug", "Run the application in debug mode.", nil)

	__app = a
	a.initted = true
	if a.useAuth && a.defaultDatabase != nil {
		lg.Now(logger.DEBUG, "Initializing auth...")
		a.Auth = auth.Initialize(a.defaultDatabase)
		a.flags.RegisterCommand(auth.CreateSuperUserCommand)

		admin.Register(admin.AdminOptions[*auth.User]{
			ListFields: []string{"Username", "Email", "FirstName", "LastName", "IsAdministrator", "IsActive"},
			FormFields: []string{"ID", "UploadAnImage", "Username", "Email", "FirstName", "LastName", "Password", "GroupSelect", "IsAdministrator", "IsActive"},
			Model:      &auth.User{},
		})

		admin.Register(admin.AdminOptions[*auth.Group]{
			ListFields: []string{"Name", "Description"},
			FormFields: []string{"ID", "Name", "PermissionSelect", "Description"},
			Model:      &auth.Group{},
		})

		admin.Register(admin.AdminOptions[*auth.Permission]{
			ListFields: []string{"Name", "Description"},
			FormFields: []string{"ID", "Name", "Description"},
			Model:      &auth.Permission{},
		})
	} else if a.useAuth && a.defaultDatabase == nil {
		panic("cannot use authentication without a database connection")
	}
	lg.Now(logger.DEBUG, "Initializing media manager...")
	if config.File != nil {
		a.config.File.Initialize()
	}

	lg.Now(logger.DEBUG, "Initializing email manager...")
	if config.Mail != nil {
		a.config.Mail.Init()
	}

	lg.Now(logger.DEBUG, "Initializing template manager...")
	a.config.Templates.Init()

	lg.Now(logger.DEBUG, "Initializing session manager...")
	a.setupSessionManager()

	lg.Now(logger.DEBUG, "Initializing router...")
	a.setupRouter()

	response.TEMPLATE_MANAGER = a.config.Templates

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
	var queryCreateSessionTable string
	var createSessionTableIndex string
	if a.defaultDatabase != nil {
		switch a.defaultDatabase.Driver().(type) {
		case *mysql.MySQLDriver:
			queryCreateSessionTable = `CREATE TABLE IF NOT EXISTS sessions (
				` + "`" + `token` + "`" + ` CHAR(43) PRIMARY KEY,
				` + "`" + `data` + "`" + ` BLOB NOT NULL,
				` + "`" + `expiry` + "`" + ` TIMESTAMP(6) NOT NULL
			)`
			createSessionTableIndex = `CREATE INDEX sessions_expiry_idx ON sessions (expiry)`
			store = mysqlstore.New(a.defaultDatabase)
		default:
			panic("Unsupported database driver, please use mysql")
		}
	}

	var err error
	if a.sessionStore != nil {
		store = a.sessionStore
	} else {
		if a.defaultDatabase == nil {
			store = memstore.New()
		} else {
			_, err = a.defaultDatabase.Exec(queryCreateSessionTable)
			if err != nil {
				panic(err)
			}
			_, err = a.defaultDatabase.Exec(createSessionTableIndex)
			if err != nil {
				if err, ok := err.(*mysql.MySQLError); ok && err.Number == 1061 {
					// Ignore error if index already exists.
				} else {
					panic(err)
				}
			}
		}
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
}

// Instead of running the application, retrieve the handler/serve mux.
func (a *Application) Serve() (http.Handler, error) {
	if !a.initted {
		panic("You must call Init() before calling Run()")
	}

	if !a.flags.Ran() {
		a.flags.Run()
	}

	// Connect the cache.
	if a.cache != nil {
		var err = a.cache.Connect()
		if err != nil && a.config.ForceCache {
			return nil, err
		}
	}

	tracer.STACKLOGGER_UNSAFE = a.DEBUG

	if a.DEBUG {
		tracer.DisallowPackage("router/.*/middleware")
		var databases = make([]debug.DatabaseSetting, 0)
		var setting = debug.DatabaseSetting{
			ENGINE: reflect.TypeOf(a.defaultDatabase.Driver()).Name(),
		}
		databases = append(databases, setting)
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
		/*

			//if a.config.File != nil {
			//	funcMap["static"] = a.config.File.AsStaticURL
			//	funcMap["media"] = a.config.File.AsMediaURL
			//}

				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO
				TODO

			//if a.config.File != nil {
			//	funcMap["static"] = a.config.File.AsStaticURL
			//	funcMap["media"] = a.config.File.AsMediaURL
			//}

		*/
		funcMap["url"] = a.Router.URLFormat
		for k, v := range a.config.DefaultTemplateFuncs {
			funcMap[k] = v
		}
		a.config.Templates.DEFAULT_FUNCS = funcMap
	}

	if a.Auth != nil {
		a.Router.AddGroup(admin.Route())
	}

	return a.Router, nil
}

// Run the application.
//
// This will start the server and listen for requests.
//
// If the server is running in SSL mode, this will also start a redirect server.
func (a *Application) Run() error {
	var handler, err = a.Serve()
	if err != nil {
		return err
	}
	var server = a.server(handler)
	if a.config.Server.CertFile != "" && a.config.Server.KeyFile != "" {
		// SSL is true, so we will listen on the TLS port.
		// This will also automatically redirect all HTTP requests to HTTPS.
		go a.Redirect()
		a.Logger.Infof("Listening on https://%s", server.Addr)

		return server.ListenAndServeTLS(a.config.Server.CertFile, a.config.Server.KeyFile)
	}
	a.Logger.Infof("Listening on http://%s", server.Addr)
	return server.ListenAndServe()
}

func (a *Application) server(h http.Handler) *http.Server {
	var httpServer *http.Server = &http.Server{}
	if a.config.Server != nil && a.config.Server.Server != nil {
		httpServer = a.config.Server.Server
	}
	// Set the host and port.
	if a.config.Server.Host != "" && a.config.Server.Port != 0 {
		httpServer.Addr = fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
	}
	// Set the handler to the router.
	httpServer.Handler = h
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
