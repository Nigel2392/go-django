package admin

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"

	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates"
	"github.com/Nigel2392/router/v3/templates/extensions"
	"gorm.io/gorm"
)

// The name of the admin site.
var AdminSite_Name string = "Admin"

// The URL for the admin site.
//
// This is the prefix for all admin site URLs.
var AdminSite_URL string = "/admin"

// The order of applications in the admin site.
//
// This is used to sort the applications in the admin site.
var ADMIN_APP_ORDER = []string{
	"Authentication",
	InternalAppName,
	ExtensionsAppName,
}

// The list of registered models.
//
// This is used to generate the admin site.
//
// There are certain requirements for a model to be registered:
//
// - The model must have an ID field.
//
// - The ID field can only be of types:
//   - int
//   - int8
//   - int16
//   - int32
//   - int64
//   - uint
//   - uint8
//   - uint16
//   - uint32
//   - uint64
//   - string
var adminSite_models []*models.Model

// Database connection pool.
//
// This is used to store the logs, and to fetch the models.
var AdminSite_DB_POOL db.Pool[*gorm.DB]

// Admin db connection.
//
// This is used to store the logs.
// This should be the default auth database!
var admin_db *gorm.DB

// The list of groups that are allowed to access the admin site.
//
// Allows bundles of groups.
//
// Example:
//
//	AdminSite_AllowedGroups = [][]string{
//		{"admin", "superuser"}, // Will check for both admin and superuser
//		{"admin"}, // Will check for admin
//	}
var AdminSite_AllowedGroups [][]string

// Logger is the logger that is used to log errors.
//
// If none is specified on admin.Initialize(),
//
// a new logger is automatically assigned with loglevel DEBUG.
var AdminSite_Logger request.Logger

// Admin site route.
//
// This is the route that is used to register the admin site.
//
// This is only used internally.
var adminSite router.Registrar

// Internal admin site routes.
//
// This is used internally to easily register internal routes.
var internalRoutes router.Registrar

// Internal app name
//
// Set this before Initialize()ing the admin site.
var InternalAppName = "internal"

// Internal models
//
// This is used internally to easily register internal models.
var adminSite_Internal_Models = []*internalModel{
	{&Log{}, logView, nil, logGroup()},
}

// Default adminsite extensions.
//
// The slice is only to be used internally.
//
// To add extensions, use the RegisterExtension() function.
//
// The extension interface is defined in the router/templates/extensions package.
var adminSite_Extensions []extensions.Extension = make([]extensions.Extension, 0)

// Per default, the admin site uses the request.TEMPLATE_MANAGER to fetch extensions.
//
// This can be overridden by setting this variable.
var AdminSite_ExtensionsManager *templates.Manager

// Extension URL prefix.
//
// This is the prefix for all extension URLs.
var EXTENSION_URL = "/ext"

// Application name for extensions.
//
// This is the name which will be displayed in the admin site.
var ExtensionsAppName = "extensions"

// The template file system for the admin site.
//
// This is where the admin site templates are stored.
//
//go:embed assets/templates/*
var templateFileSystem embed.FS

//go:embed assets/static/*
var sfs embed.FS

var staticFileSystem, _ = fs.Sub(sfs, "assets/static")

// The template manager for rendering templates.
//
// This can be used to arbitrarily render templates on the request,
//
// without touching the regular template FS.
//
// By default, this is the global request.TEMPLATE_MANAGER.
var adminSiteManager = &templates.Manager{
	USE_TEMPLATE_CACHE:     false,
	BASE_TEMPLATE_SUFFIXES: []string{".html", ".tmpl"},
	BASE_TEMPLATE_DIRS:     []string{"base"},
	DEFAULT_FUNCS: template.FuncMap{
		"title": func(s any) string {
			if s == nil {
				return AdminSite_Name
			}
			switch s := s.(type) {
			case string:
				return httputils.TitleCaser.String(s)
			}
			return formatFunc(s)
		},
		"lower":  strings.ToLower,
		"upper":  strings.ToUpper,
		"div":    divideFunc,
		"max":    maxStrLenFunc,
		"format": formatFunc,
		"join":   joinFunc,
	},
}

func init() {
	var TemplateFS, err = fs.Sub(templateFileSystem, "assets/templates")
	if err != nil {
		panic(err)
	}
	// var TemplateFS = os.DirFS("go-django/admin/assets/templates")
	adminSiteManager.TEMPLATEFS = TemplateFS
	// staticFileSystem = os.DirFS("go-django/admin/assets/static")
}
