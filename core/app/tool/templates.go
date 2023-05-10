package tool

var defaultBaseHTMLTemplate = `{{ define "base" }}
<!doctype html>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <title>{{ block "title" . }}{{ end }}</title>
        {{ if .Messages }}{{ template "messages.tmpl" .Messages }}{{ end }}
        {{ block "css" . }}{{ end }}
    </head>
    <body>
        {{ block "content" .}}{{ end }}
        {{ block "js" . }}{{ end }}
    </body>
</html>
{{ end }}`

var defaultMessagesTemplate = `<section class="messages text-center">
{{ range . }}
	{{ if eq .Type "success" }}
		<div class="message bg-success removeself">
			<div class="font-l">{{ .Text }}</div>
		</div>
	{{ else if eq .Type "error" }}
		<div class="message bg-danger removeself">
			<div class="font-l">{{ .Text }}</div>
		</div>
	{{ else if eq .Type "warning" }}
		<div class="message bg-warning removeself">
			<div class="font-l">{{ .Text }}</div>
		</div>
	{{ else if eq .Type "info" }}
		<div class="message bg-info removeself">
			<div class="font-l">{{ .Text }}</div>
		</div>
	{{ end }}
{{ end }}
</section>`

var defaultHTMLTemplate = `{{ template "base" . }}

{{ define "title" }} {{ index .Data "title" }} {{ end }}

{{ define "css" }}{{ end }}

{{ define "content" }}
<h1>Index!</h1>
{{ end }}

{{ define "js" }}{{ end }}
{{ template "base.tmpl" }}`

var mainTemplate = `package main

import (
	"fmt"

	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/app"
	logger "github.com/Nigel2392/request-logger"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

var App = app.New(appConfig)

func main() {
	// There are some default commandline flags registered.
	// You can add your own flags by using go-django's flag package.
	// See the help menu for more information. (go run ./src -h)

	auth.USER_MODEL_LOGIN_FIELD = "Email"
	App.Router.Get("/", router.HandleFunc(index), "index")

	var _, err = auth.CreateAdminUser(
		"developer@local.local", // Email
		"Developer",             // Username
		"root",                  // First name
		"toor",                  // Last name
		"Developer123!",         // Password
	)
	if err != nil {
		fmt.Println(logger.Colorize(fmt.Sprintf("Error creating superuser: %s", err.Error()), logger.Red))
	}

	err = App.Run()
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

var appConfigTemplateRegular = `//go:build !docker
// +build !docker

package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Nigel2392/dotenv"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/templates"

	_ "github.com/go-sql-driver/mysql"
)

var env = dotenv.NewEnv("environment/normal.env")

var db = func() *sql.DB {
	var DB_HOST = env.Get("DB_HOST", "127.0.0.1")
	var DB_PORT = env.GetInt("DB_PORT", 3306)
	var DB_NAME = env.Get("DB_NAME", "web")
	var DB_USER = env.Get("DB_USER", "root")
	var DB_PASSWORD = env.Get("DB_PASSWORD", "mypassword")
	var DSN_PARAMS = env.Get("DB_DSN_PARAMS", "charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")

	var dsn string
	if DB_PASSWORD == "" {
		dsn = "%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_HOST, DB_PORT, DB_NAME)
	} else {
		dsn = "%s:%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME)
	}

	if DSN_PARAMS != "" {
		dsn = dsn + "?" + DSN_PARAMS
	} else {
		dsn = dsn + "?charset=utf8mb4&parseTime=True&loc=Local"
	}

	var db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return db
}()

var appConfig = app.Config{
	// Secret key for the server.
	SecretKey: env.Get("SECRET_KEY", time.Now().Format(time.RFC3339Nano)),
	// Allowed hosts for the server.
	AllowedHosts: env.GetAll("ALLOWED_HOSTS", []string{"*"}...),
	Server: &app.Server{
		// Server options.
		Host: env.Get("HOST", "127.0.0.1"),
		Port: env.GetInt("PORT", 8080),
		// SSL Options
		CertFile: env.Get("SSL_CERT_FILE", ""),
		KeyFile:  env.Get("SSL_KEY_FILE", ""),
	},
	Middlewares: []router.Middleware{
		// Default router middleware to use
	},
	// Rate limit middlewares.
	RateLimitOptions: &middleware.RateLimitOptions{
		RequestsPerSecond: env.GetInt("REQUESTS_PER_SECOND", 10),
		BurstMultiplier:   env.GetInt("REQUEST_BURST_MULTIPLIER", 3),
		CleanExpiry:       5 * time.Minute,
		CleanInt:          1 * time.Minute,
	},
	// Template manager
	Templates: &templates.Manager{
		TEMPLATEFS:             os.DirFS(env.Get("TEMPLATE_DIR", "assets/templates/")),
		BASE_TEMPLATE_SUFFIXES: env.GetAll("TEMPLATE_BASE_SUFFIXES", []string{".tmpl", ".html"}...),
		BASE_TEMPLATE_DIRS:     env.GetAll("TEMPLATE_BASE_DIRS", []string{"base"}...),
		TEMPLATE_DIRS:          env.GetAll("TEMPLATE_DIRS", []string{"templates"}...),
		USE_TEMPLATE_CACHE:     env.GetBool("TEMPLATE_CACHE", true),
	},
	// File system manager (Static/media files)
	File: fs.NewFiler(env.Get("STATIC_DIR", "assets/static")),
	// Email settings.
	Mail: &email.Manager{
		EMAIL_HOST:     env.Get("EMAIL_HOST", ""),
		EMAIL_PORT:     env.GetInt("EMAIL_PORT", 25),
		EMAIL_USERNAME: env.Get("EMAIL_USERNAME", ""),
		EMAIL_PASSWORD: env.Get("EMAIL_PASSWORD", ""),
		EMAIL_USE_TLS:  env.GetBool("EMAIL_USE_TLS", false),
		EMAIL_USE_SSL:  env.GetBool("EMAIL_USE_SSL", false),
		EMAIL_FROM:     env.Get("EMAIL_FROM", ""),
	},
	// Database settings.
	Database: db,
}`

var appConfigTemplateDocker = `//go:build docker
// +build docker

package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/templates"

	_ "github.com/go-sql-driver/mysql"
)

var _ = os.Chdir(filepath.Dir(os.Args[0]))

var db = func() *sql.DB {
	var DB_HOST = getSafeEnvVar("DB_HOST", "")
	var DB_PORT = getSafeEnvVarInt("DB_PORT", 3306)
	var DB_NAME = getSafeEnvVar("DB_NAME", "")
	var DB_USER = getSafeEnvVar("DB_USER", "")
	var DB_PASSWORD = getSafeEnvVar("DB_PASSWORD", "")
	var DSN_PARAMS = getSafeEnvVar("DB_DSN_PARAMS", "")

	var start = time.Now()
	var addr = fmt.Sprintf("%s:%d", DB_HOST, DB_PORT)
	for {
		var c, err = net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			fmt.Println("Database initialized!")
			break
		}
		if time.Since(start) > 30*time.Second {
			panic(err)
		}
		fmt.Printf("Waiting for database on %s to start...\n", addr)
		time.Sleep(500 * time.Millisecond)
	}

	var dsn string
	if DB_PASSWORD == "" {
		dsn = "%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_HOST, DB_PORT, DB_NAME)
	} else {
		dsn = "%s:%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME)
	}

	if DSN_PARAMS != "" {
		dsn = dsn + "?" + DSN_PARAMS
	} else {
		dsn = dsn + "?charset=utf8mb4&parseTime=True&loc=Local"
	}

	var db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return db
}()

var appConfig = app.Config{
	// Secret key for the server.
	SecretKey: getSafeEnvVar("SECRET_KEY", time.Now().Format(time.RFC3339Nano)),
	// Allowed hosts for the server.
	AllowedHosts: splitApart(getSafeEnvVar("ALLOWED_HOSTS", "127.0.0.1")),
	Server: &app.Server{
		// Server options.
		Host: getSafeEnvVar("HOST", "127.0.0.1"),
		Port: getSafeEnvVarInt("PORT", 8080),
		// SSL Options
		CertFile: getSafeEnvVar("SSL_CERT_FILE", ""),
		KeyFile:  getSafeEnvVar("SSL_KEY_FILE", ""),
	},
	Middlewares: []router.Middleware{
		// Default router middleware to use
	},
	// Rate limit middlewares.
	RateLimitOptions: &middleware.RateLimitOptions{
		RequestsPerSecond: getSafeEnvVarInt("REQUESTS_PER_SECOND", 10),
		BurstMultiplier:   getSafeEnvVarInt("REQUEST_BURST_MULTIPLIER", 3),
		CleanExpiry:       5 * time.Minute,
		CleanInt:          1 * time.Minute,
	},
	// Template manager
	Templates: &templates.Manager{
		TEMPLATEFS:             os.DirFS(getSafeEnvVar("TEMPLATE_DIR", "./assets/templates/")),
		BASE_TEMPLATE_SUFFIXES: splitApart(getSafeEnvVar("TEMPLATE_BASE_SUFFIXES", ".tmpl .html")),
		BASE_TEMPLATE_DIRS:     splitApart(getSafeEnvVar("TEMPLATE_BASE_DIRS", "base")),
		TEMPLATE_DIRS:          splitApart(getSafeEnvVar("TEMPLATE_DIRS", "templates")),
		USE_TEMPLATE_CACHE:     getSafeEnvVarBool("TEMPLATE_CACHE", true),
	},
	// File system manager (Static/media files)
	File: fs.NewFiler(getSafeEnvVar("STATIC_DIR", "./assets/static")),
	// Email settings.
	Mail: &email.Manager{
		EMAIL_HOST:     getSafeEnvVar("EMAIL_HOST", ""),
		EMAIL_PORT:     getSafeEnvVarInt("EMAIL_PORT", 25),
		EMAIL_USERNAME: getSafeEnvVar("EMAIL_USERNAME", ""),
		EMAIL_PASSWORD: getSafeEnvVar("EMAIL_PASSWORD", ""),
		EMAIL_USE_TLS:  getSafeEnvVarBool("EMAIL_USE_TLS", false),
		EMAIL_USE_SSL:  getSafeEnvVarBool("EMAIL_USE_SSL", false),
		EMAIL_FROM:     getSafeEnvVar("EMAIL_FROM", ""),
	},
	// Database settings.
	Database: db,
}

func splitApart(s string) []string {
	var parts = make([]string, 0)
	for _, part := range strings.Split(s, " ") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func getSafeEnvVar(key string, dflt ...string) string {
	var file = os.Getenv(key)
	if file == "" && len(dflt) == 0 {
		fmt.Printf("%s environment variable not set or is empty.\n", key)
		os.Exit(1)
	} else if file == "" && len(dflt) > 0 {
		return dflt[0]
	}
	return file
}

func getSafeEnvVarInt(key string, dflt int) int {
	var file = os.Getenv(key)
	if file == "" {
		return dflt
	}
	var i, err = strconv.Atoi(file)
	if err != nil {
		fmt.Printf("%s environment variable is not an integer: %s.\n", key, file)
		os.Exit(1)
	}
	return i
}

func getSafeEnvVarBool(key string, dflt bool) bool {
	var file = os.Getenv(key)
	if file == "" {
		return dflt
	}
	var b, err = strconv.ParseBool(file)
	if err != nil {
		fmt.Printf("%s environment variable is not a boolean.\n", key)
		os.Exit(1)
	}
	return b
}`

var Env_template_regular string = `# SECURITY WARNING: keep the secret key used in production secret!
SECRET_KEY = "SECRET-KEY-%v"

# Allowed hosts.
ALLOWED_HOSTS = "127.0.0.1"

# Address to host the server on.
HOST = "127.0.0.1"
PORT = 8080

# SSL Certificate and Key file.
SSL_CERT_FILE = None
SSL_KEY_FILE = None

# Requests per second to allow.
REQUESTS_PER_SECOND = 10
REQUEST_BURST_MULTIPLIER = 3

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

var Env_template_docker string = `# SECURITY WARNING: keep the secret key used in production secret!
SECRET_KEY = "%v"

# Allowed hosts.
ALLOWED_HOSTS = "127.0.0.1"

# Address to host the server on.
HOST = "0.0.0.0"
PORT = 8080

# Requests per second to allow.
REQUESTS_PER_SECOND = 10
REQUEST_BURST_MULTIPLIER = 3

# Template settings.
TEMPLATE_DIR = "assets/templates"
TEMPLATE_BASE_SUFFIXES = ".html .htm .xml tmpl tpl"
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
# EMAIL_FROM = test@gmail.com

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

var DockerComposeTemplate = `version: "3.9"
services:
  webapp:
    container_name: webapp
    command: /app/application
    build: 
      context: .
      dockerfile: Dockerfile
    env_file:
      - ./environment/docker.env
    environment:
      - PORT=8080
      - DB_HOST=database
      - DB_PORT=3306
      - DB_NAME=web
      - DB_USER=root
      - DB_PASSWORD=mypassword
      # - DB_DSN_PARAMS=""
      # - SSL_CERT_FILE=""
      # - SSL_KEY_FILE=""
    ports:
      - 8080:8080
    depends_on:
      - database
    networks:
      webnet:
        ipv4_address: 10.0.0.3
  database:
    image: mysql:latest
    restart: no
    stdin_open: true # docker run -i
    tty: true        # docker run -t
    environment:
      MYSQL_ROOT_PASSWORD: mypassword
      MYSQL_DATABASE: web
      # allow root login from any IP address (this is limited to the docker network.)
      MYSQL_ROOT_HOST: "%"
    command: '--default-authentication-plugin=mysql_native_password --bind-address=10.0.0.2' # --init-file /docker-entrypoint-initdb.d/init.sql 
    ports:
      - 3306:3306
    volumes:
      - db_data:/var/lib/mysql
      # - ./src/settings/sqlc/schema.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      webnet:
        ipv4_address: 10.0.0.2
volumes:
  db_data:

networks:
  webnet:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: "10.0.0.0/8"`

var DockerFileTemplate = `FROM golang:1.20.4-alpine3.17 as builder

COPY ./src /app/src
COPY ./assets /app/assets
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
WORKDIR /app

RUN go mod download
RUN go build -tags docker -o /app/application ./src

FROM scratch

COPY --from=builder /app/assets /app/assets
COPY --from=builder /app/application /app/application

EXPOSE 8080

ENTRYPOINT ["/app/application"]`
