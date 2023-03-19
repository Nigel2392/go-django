package main

var defaultBaseHTMLTemplate = `{{define "base"}}
<!doctype html>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <title>{{block "title" .}}{{end}}</title>
        {{if .Messages}}{{template "messages.tmpl" .Messages}}{{end}}
        {{block "css" .}}{{end}}
    </head>
    <body>
        {{block "content" .}}{{end}}
        {{block "js" .}}{{end}}
    </body>
</html>
{{end}}`

var defaultMessagesTemplate = `<section class="messages text-center">
{{range .}}
	{{ if eq .Type "success" }}
		<div class="message bg-success removeself">
			<div class="font-l">{{.Text}}</div>
		</div>
	{{else if eq .Type "error" }}
		<div class="message bg-danger removeself">
			<div class="font-l">{{.Text}}</div>
		</div>
	{{else if eq .Type "warning" }}
		<div class="message bg-warning removeself">
			<div class="font-l">{{.Text}}</div>
		</div>
	{{else if eq .Type "info" }}
		<div class="message bg-info removeself">
			<div class="font-l">{{.Text}}</div>
		</div>
	{{end}}
{{end}}
</section>`

var defaultHTMLTemplate = `{{ template "base" . }}

{{ define "title" }} {{index .Data "title"}} {{end}}

{{ define "css" }}{{end}}

{{ define "content" }}
<h1>Index!</h1>
{{end}}

{{ define "js" }}{{end}}
{{template "base.tmpl"}}`

var mainTemplate = `package main

import (
	"github.com/Nigel2392/go-django/admin"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

var App = app.New(appConfig)

func main() {
	auth.USER_MODEL_LOGIN_FIELD = "Email"

	admin.Initialize("Admin", "/admin", App.Pool)
	admin.Register(
		&auth.User{},
		&auth.Group{},
		&auth.Permission{},
	)

	var urls = admin.URLS()
	App.Router.AddGroup(urls)

	App.Router.Get("/", index, "index")

	var err = App.Run()
	if err != nil {
		panic(err)
	}
}

func index(req *request.Request) {
	var err = response.Render(req, "app/index.tmpl")
	req.Data.Set("title", "Home")
	if err != nil {
		req.Error(500, err.Error())
		return
	}
}`

var appConfigTemplate = `package main

import (
	"os"
	"time"

	"github.com/Nigel2392/dotenv"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/middleware/csrf"
	"github.com/Nigel2392/router/v3/templates"
	"gorm.io/gorm"
)

var env = dotenv.NewEnv(".env")

var appConfig = app.Config{
	SecretKey:    env.Get("SECRET_KEY", time.Now().Format(time.RFC3339Nano)),
	AllowedHosts: env.GetAll("ALLOWED_HOSTS", []string{"*"}...),
	Server: &app.Server{
		// Server options.
		Host: env.Get("HOST", "127.0.0.1"),
		Port: env.GetInt("PORT", 8000),
		// SSL Options
		CertFile: env.Get("SSL_CERT_FILE", ""),
		KeyFile:  env.Get("SSL_KEY_FILE", ""),
	},
	Middlewares: []router.Middleware{
		csrf.Middleware,
		middleware.AddLogger,
		middleware.Printer,
	},
	RateLimitOptions: &middleware.RateLimitOptions{
		RequestsPerSecond: env.GetInt("REQUESTS_PER_SECOND", 10),
		BurstMultiplier:   env.GetInt("REQUEST_BURST_MULTIPLIER", 3),
		CleanExpiry:       5 * time.Minute,
		CleanInt:          1 * time.Minute,
	},
	Templates: &templates.Manager{
		TEMPLATEFS:             os.DirFS(env.Get("TEMPLATE_DIR", "assets/templates/")),
		BASE_TEMPLATE_SUFFIXES: env.GetAll("TEMPLATE_BASE_SUFFIXES", []string{".tmpl", ".html"}...),
		BASE_TEMPLATE_DIRS:     env.GetAll("TEMPLATE_BASE_DIRS", []string{"base"}...),
		TEMPLATE_DIRS:          env.GetAll("TEMPLATE_DIRS", []string{"templates"}...),
		USE_TEMPLATE_CACHE:     env.GetBool("TEMPLATE_CACHE", true),
	},
	File: &fs.Manager{
		FS_STATIC_ROOT:     env.Get("STATIC_DIR", "assets/static/"),
		FS_MEDIA_ROOT:      env.Get("MEDIA_DIR", "assets/media/"),
		FS_STATIC_URL:      env.Get("STATIC_URL", "/static/"),
		FS_MEDIA_URL:       env.Get("MEDIA_URL", "/media/"),
		FS_FILE_QUEUE_SIZE: env.GetInt("FS_FILE_QUEUE_SIZE", 100),
	},
	DBConfig: &db.DatabasePoolItem{
		DEFAULT_DATABASE: env.Get("DEFAULT_DATABASE", "sqlite3"),
		DB_NAME:          env.Get("DB_NAME", "db.sqlite3"),
		DB_USER:          env.Get("DB_USER", ""),
		DB_PASS:          env.Get("DB_PASS", ""),
		DB_HOST:          env.Get("DB_HOST", ""),
		DB_PORT:          env.GetInt("DB_PORT", 0),
		DB_SSLMODE:       env.Get("DB_SSLMODE", ""),
		Config:           &gorm.Config{},
	},
	Mail: &email.Manager{
		EMAIL_HOST:     env.Get("EMAIL_HOST", ""),
		EMAIL_PORT:     env.GetInt("EMAIL_PORT", 25),
		EMAIL_USERNAME: env.Get("EMAIL_USERNAME", ""),
		EMAIL_PASSWORD: env.Get("EMAIL_PASSWORD", ""),
		EMAIL_USE_TLS:  env.GetBool("EMAIL_USE_TLS", false),
		EMAIL_USE_SSL:  env.GetBool("EMAIL_USE_SSL", false),
		EMAIL_FROM:     env.Get("EMAIL_FROM", ""),
	},
}`

var env_template string = `# SECURITY WARNING: keep the secret key used in production secret!
SECRET_KEY = "SECRET-KEY-%v"

# Allowed hosts.
ALLOWED_HOSTS = "127.0.0.1"

# Address to host the server on.
HOST = "127.0.0.1"
PORT = 8000

# SSL Certificate and Key file.
SSL_CERT_FILE = None
SSL_KEY_FILE = None

# Requests per second to allow.
REQUESTS_PER_SECOND = 10
REQUEST_BURST_MULTIPLIER = 3

# Default database engine.
# Options are: 
# - (mysql, mariadb)
# - (sqlite, sqlite3)
# (NYI) - (postgres, postgresql)
# (NYI) - (mssql, sqlserver)
DEFAULT_DATABASE = "sqlite3"

# Database settings.
DB_NAME = "db.sqlite3"
DB_USER = None
DB_PASS = None
DB_HOST = None
DB_PORT = None
DB_SSLMODE = None

# Template settings.
TEMPLATE_DIR = "assets/templates"
TEMPLATE_BASE_SUFFIXES = ".html", ".htm", ".xml", "tmpl", "tpl"
TEMPLATE_BASE_DIRS = "base"
TEMPLATE_DIRS = "templates"
TEMPLATE_CACHE = False

# Email settings.
# EMAIL_HOST = "smtp.gmail.com"
# EMAIL_PORT = 465
# EMAIL_USERNAME = "test@gmail.com"
# EMAIL_PASSWORD = "password"
# EMAIL_USE_TLS = False
# EMAIL_USE_SSL = True
# EMAIL_FROM = $EMAIL_USERNAME

# Staticfiles/mediafiles options
STATIC_DIR = "assets/static"
MEDIA_DIR = "assets/media"
STATIC_URL = "/static/"
MEDIA_URL = "/media/"
FS_FILE_QUEUE_SIZE = 100

# Lifetime of the session cookie.
# Format: 1w3d12h (1 week, 3 days, 12 hours)
# 3d12h (3 days, 12 hours)
# 12h (12 hours)
# 1w (1 week)
SESSION_LIFETIME = "1w3d12h"
# Name to set for the session cookie. If not set, the default name is used.
SESSION_COOKIE_NAME = "sessionid"

# Idle timeout for the cookie
SESSION_IDLE_TIMEOUT = None

# Domain to set for the session cookie. If not set, the default domain is used.
SESSION_COOKIE_DOMAIN = None

# Path to set for the session cookie. If not set, the default path is used.
SESSION_COOKIE_PATH = "/"

# SESSION_COOKIE_SAMESITE options are:
# 1. SameSiteDefaultMode
# 2. SameSiteLaxMode
# 3. SameSiteStrictMode
# 4. SameSiteNoneMode
SESSION_COOKIE_SAME_SITE = 2

SESSION_COOKIE_SECURE = False
SESSION_COOKIE_HTTP_ONLY = True
SESSION_COOKIE_PERSIST = True`
