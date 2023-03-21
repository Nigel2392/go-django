package admin

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"

	"github.com/Nigel2392/go-django/admin/internal/menu"
	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/logger"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/templates"
	"github.com/Nigel2392/router/v3/templates/extensions"
	"gorm.io/gorm"
)

type AdminSite struct {
	// The name of the admin site.
	Name string

	// The URL for the admin site.
	//
	// This is the prefix for all admin site URLs.
	URL string

	// The order of applications in the admin site.
	//
	// This is used to sort the applications in the admin site.
	AppOrder []string

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
	AllowedGroups [][]string

	// Logger is the logger that is used to log errors.
	//
	// If none is specified on admin.Initialize(),
	//
	// a new logger is automatically assigned with loglevel DEBUG.
	Logger request.Logger

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
	models []*models.Model

	// Database connection pool.
	//
	// This is used to store the logs, and to fetch the models.
	DBPool db.Pool[*gorm.DB]

	// Admin site route.
	//
	// This is the route that is used to register the admin site.
	//
	// This is only used internally.
	registrar router.Registrar

	// Internal admin site routes.
	//
	// This is used internally to easily register internal routes.
	internalRegistrar router.Registrar

	// Internal models
	//
	// This is used internally to easily register internal models.
	internal_models []*internalModel

	// Internal app name
	//
	// Set this before Initialize()ing the admin site.
	InternalAppName string

	// Adminsite extensions.
	//
	// The slice is only to be used internally.
	//
	// To add extensions, use the RegisterExtension() function.
	//
	// The extension interface is defined in the router/templates/extensions package.
	extensions []extensions.Extension

	// Per default, the admin site uses the request.TEMPLATE_MANAGER to fetch extensions.
	//
	// This can be overridden by setting this variable.
	ExtensionsManager *templates.Manager

	// Extension URL prefix.
	//
	// This is the prefix for all extension URLs.
	ExtensionURL string

	// Application name for extensions.
	//
	// This is the name which will be displayed in the admin site.
	ExtensionsAppName string

	// Template manager for the admin site.
	templateMgr *templates.Manager

	// The internal application menu item.
	//
	// This will only be loaded once.
	//
	// This is where models such as the admin logs will be registered.
	internalApp *menu.Item

	// The extensions application menu item.
	//
	// This will only be loaded once.
	//
	// This is where all the extensions will be registered.
	extensionsApp *menu.Item
}

func NewAdminSite(name, url string, p db.Pool[*gorm.DB], l ...request.Logger) *AdminSite {
	var as = &AdminSite{
		Name:       name,
		URL:        url,
		DBPool:     p,
		models:     make([]*models.Model, 0),
		extensions: make([]extensions.Extension, 0),
		AppOrder:   make([]string, 0),
	}

	if len(l) > 0 {
		as.Logger = l[0]
	} else {
		as.Logger = logger.NewLogger(logger.DEBUG, name+" ")
	}
	return as
}

func (as *AdminSite) Defaults() {
	var internalAppName = "internal"
	var extensionsAppName = "extensions"

	if as.InternalAppName == "" {
		as.InternalAppName = internalAppName
		as.ExtensionsAppName = extensionsAppName
	}

	if as.Logger == nil {
		as.Logger = logger.NewLogger(logger.INFO, as.Name+" ")
	}

	if as.ExtensionURL == "" {
		var extensionURL = "/ext"
		as.ExtensionURL = extensionURL
	}
	if len(as.AppOrder) < 1 {
		as.AppOrder = []string{auth.AUTH_APP_NAME, internalAppName, extensionsAppName}
	}
	as.internal_models = []*internalModel{
		{&Log{}, logView(as), nil, logGroup(as)},
	}

}

func DefaultAdminSite(name, url string, p db.Pool[*gorm.DB], l ...request.Logger) *AdminSite {
	var as = NewAdminSite(name, url, p, l...)
	as.Defaults()
	return as
}

func (a *AdminSite) DB() db.PoolItem[*gorm.DB] {
	return db.GetDefaultDatabase(auth.DB_KEY, a.DBPool)
}

var templateFS fs.FS

func (a *AdminSite) templateManager() *templates.Manager {
	if a.templateMgr != nil {
		return a.templateMgr
	}
	var mgr = &templates.Manager{
		TEMPLATEFS:             templateFS,
		USE_TEMPLATE_CACHE:     true,
		BASE_TEMPLATE_SUFFIXES: []string{".html", ".tmpl"},
		BASE_TEMPLATE_DIRS:     []string{"base"},
		TEMPLATE_DIRS:          []string{"admin"},
		DEFAULT_FUNCS: template.FuncMap{
			"title": func(s any) string {
				if s == nil {
					return a.Name
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
	a.templateMgr = mgr
	return mgr
}

// The template file system for the admin site.
//
// This is where the admin site templates are stored.
//
//go:embed assets/templates/*
var templateFileSystem embed.FS

//go:embed assets/static/*
var sfs embed.FS

var staticFileSystem, _ = fs.Sub(sfs, "assets/static")

func init() {
	var TemplateFS, err = fs.Sub(templateFileSystem, "assets/templates")
	if err != nil {
		panic(err)
	}
	// var TemplateFS = os.DirFS("go-django/admin/assets/templates")
	templateFS = TemplateFS
	// staticFileSystem = os.DirFS("go-django/admin/assets/static")
}
