package main

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
	"github.com/Nigel2392/router/v3/templates"
	"gorm.io/gorm"
)

var env = dotenv.NewEnv(".env")

var appConfig = app.Config{
	// Secret key for the server.
	SecretKey: env.Get("SECRET_KEY", time.Now().Format(time.RFC3339Nano)),
	// Allowed hosts for the server.
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
	File: &fs.Manager{
		FS_STATIC_ROOT:     env.Get("STATIC_DIR", "assets/static/"),
		FS_MEDIA_ROOT:      env.Get("MEDIA_DIR", "assets/media/"),
		FS_STATIC_URL:      env.Get("STATIC_URL", "/static/"),
		FS_MEDIA_URL:       env.Get("MEDIA_URL", "/media/"),
		FS_FILE_QUEUE_SIZE: env.GetInt("FS_FILE_QUEUE_SIZE", 100),
	},
	// Default database configuration.
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
	Admin: &app.AdminSite{
		Name: "Admin",
		URL:  "/admin",
	},
}