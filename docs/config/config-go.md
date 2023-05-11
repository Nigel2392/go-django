# The default configuration explained

There is a pretty massive configuration for the Go-Django framework. 

We recommend you to experiment with some of the options.

```go
//go:build !docker
// +build !docker

package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Nigel2392/dotenv"
	"github.com/Nigel2392/go-django"
	"github.com/Nigel2392/go-django/core/app"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/templates"

	_ "github.com/go-sql-driver/mysql"
)

// Load in the ./environment/normal.env file.
var env = dotenv.NewEnv("environment/normal.env")

// Initialize the database.
var db = func() *sql.DB {
	// Fetch the environment variables.
	var DB_HOST = env.Get("DB_HOST", "127.0.0.1")
	var DB_PORT = env.GetInt("DB_PORT", 3306)
	var DB_NAME = env.Get("DB_NAME", "web")
	var DB_USER = env.Get("DB_USER", "root")
	var DB_PASSWORD = env.Get("DB_PASSWORD", "mypassword")
	var DSN_PARAMS = env.Get("DB_DSN_PARAMS", "charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true")

	var dsn string
	// Setup the DSN
	if DB_PASSWORD == "" {
		dsn = "%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_HOST, DB_PORT, DB_NAME)
	} else {
		dsn = "%s:%s@tcp(%s:%d)/%s"
		dsn = fmt.Sprintf(dsn, DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME)
	}
	// Setup DSN query parameters.
	if DSN_PARAMS != "" {
		dsn = dsn + "?" + DSN_PARAMS
	} else {
		dsn = dsn + "?charset=utf8mb4&parseTime=True&loc=Local"
	}
	// Open the connection to the database.
	var db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return db
}()

// Set up the application config.
var appConfig = django.AppConfig{
	// Secret key for the server.
	// 
	// We recommend you to set this to a static value.
	SecretKey: env.Get("SECRET_KEY", time.Now().Format(time.RFC3339Nano)),
	// Allowed hosts for the server.
	// If "*" is specified, any host is allowed.
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
	// Rate limit middleware options.
	RateLimitOptions: &middleware.RateLimitOptions{
		RequestsPerSecond: env.GetInt("REQUESTS_PER_SECOND", 10),
		BurstMultiplier:   env.GetInt("REQUEST_BURST_MULTIPLIER", 3),
		CleanExpiry:       5 * time.Minute,
		CleanInt:          1 * time.Minute,
	},
	// Template manager to load in templates from.
	Templates: &templates.Manager{
		// The file system in which the templates reside
		TEMPLATEFS:             os.DirFS(env.Get("TEMPLATE_DIR", "assets/templates/")),
		// The suffixes of templates allowed to be loaded.
		BASE_TEMPLATE_SUFFIXES: env.GetAll("TEMPLATE_BASE_SUFFIXES", []string{".tmpl", ".html"}...),
		// The directory of the template bases.
		BASE_TEMPLATE_DIRS:     env.GetAll("TEMPLATE_BASE_DIRS", []string{"base"}...),
		// The directory of the templates in the TEMPLATEFS
		TEMPLATE_DIRS:          env.GetAll("TEMPLATE_DIRS", []string{"templates"}...),
		// Load in the templates from a cache after loading them once
		USE_TEMPLATE_CACHE:     env.GetBool("TEMPLATE_CACHE", true),
	},
	// File system manager (Static/media files)
	// This is to easily create, read and delete files which can only reside in a specific specified directory.
	File: fs.NewFiler(env.Get("STATIC_DIR", "assets/static")),
	// Email settings.
	// This is used to send mail with the application.
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
	// This is used to enable authentication, and the adminsite within the application.
	Database: db,
	// When nil, the following parameter must be uncommented to avoid a panic.
	DisableAuthentication: true,
}
```
