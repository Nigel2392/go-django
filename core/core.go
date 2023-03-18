package core

import (
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/email"
	"github.com/Nigel2392/go-django/core/fs"

	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates"
	"gorm.io/gorm"
)

// Email settings:
// Timeout to wait for a response from the server
// Errors if the timeout is reached.
// TLS Config to use when USE_SSL is true.
// Default smpth authentication method
// Hook for when an email will be sent
//
//	EMAIL_HOST     string
//	EMAIL_PORT     int
//	EMAIL_USERNAME string
//	EMAIL_PASSWORD string
//	EMAIL_USE_TLS  bool
//	EMAIL_USE_SSL  bool
//	EMAIL_FROM     string
//
//	TIMEOUT      time.Duration
//	TLS_Config   *tls.Config
//	DEFAULT_AUTH smtp.Auth
//	OnSend func(e *email.Email)
var Mail = &email.Manager{}

// Database settings:
// GORM Config
// Database.DB will be initialized with Database.Init()
//
//	DEFAULT_DATABASE string // Default database to use (mysql, postgres, sqlite)
//	DB_NAME          string
//	DB_USER          string
//	DB_PASS          string
//	DB_HOST          string
//	DB_PORT          int
//	DB_SSLMODE       string
//
//	Config *gorm.Config
var Database = &db.Manager{Config: &gorm.Config{}}

// Size of the file queue
// Hook for when a file is read, or written in the media directory.
//
//	FS_STATIC_ROOT     string
//	FS_MEDIA_ROOT      string
//	FS_STATIC_URL      string
//	FS_MEDIA_URL       string
//	FS_FILE_QUEUE_SIZE int
//	OnReadFromMedia func(path string, buf *bytes.Buffer)
//	OnWriteToMedia  func(path string, b []byte)
var File = &fs.Manager{}

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
var Templates = &templates.Manager{}

func Init(pool db.Pool[*gorm.DB]) {
	// Initialize the database
	Database.Init()

	// Add the default database to the pool.
	pool.Add(db.DEFAULT_DATABASE_KEY, Database.DB)

	// Set up authentication manager.
	auth.Init(pool)

	// Set up email manager.
	Mail.Init()

	// Set up file manager and request template manager.
	File.Init()
	request.TEMPLATE_MANAGER = Templates

}
