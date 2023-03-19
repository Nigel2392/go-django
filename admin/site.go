package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/httputils/orderedmap"
	"github.com/Nigel2392/go-django/logger"
	"gorm.io/gorm"

	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/router/v3/templates/extensions"
)

// Create a new admin site.
//
// This function has to be called before any other admin functions.
//
// This will set up the adminsite for use.
func Initialize(name string, url string, p db.Pool[*gorm.DB], l ...request.Logger) {
	// Set up the adminsite
	AdminSite_Name = name
	AdminSite_URL = url
	AdminSite_DB_POOL = p
	if len(l) > 0 {
		AdminSite_Logger = l[0]
	} else {
		AdminSite_Logger = logger.NewLogger(logger.DEBUG, name+" ")
	}

	var admin_db_item = db.GetDefaultDatabase(auth.DB_KEY, p)

	admin_db = admin_db_item.DB()
	admin_db_item.Register(
		&Log{},
		&LoggableUser{},
	)
	var err = admin_db_item.AutoMigrate()
	if err != nil {
		panic(err)
	}

	// Register the default admin site permissions.
	PermissionViewAdminSite.Save(admin_db)
	PermissionViewAdminInternal.Save(admin_db)
	PermissionViewAdminExtensions.Save(admin_db)

	// Register internal apps.
	// (Logs, etc...)
	var internalApp = internal_menu_items()

	// Register default URLs.
	adminSite = router.Group(url, "admin")
	adminSite.Get("/unauthorized", unauthorizedView, "unauthorized").Use(defaultDataMiddleware)
	adminSite.Get("/login", loginView, "login")
	adminSite.Get("/logout", logoutView, "logout")

	// Register the default admin site view.
	var rt = adminSite.(*router.Route)
	rt.Method = router.GET
	rt.HandlerFunc = func(r *request.Request) {
		if !hasAdminPerms(r) {
			Unauthorized(r, "You do not have permission to access this page.")
			return
		}
		defaultDataFunc(r, AdminSite_Name)
		indexView(r)
	}

	// Register the default admin site static file handler.
	var staticFileSysHTTP = http.FS(staticFileSystem)
	adminSite.Get("/static/<<any>>",
		router.FromHTTPHandler(
			http.StripPrefix(fmt.Sprintf("%s/static/", AdminSite_URL),
				http.FileServer(staticFileSysHTTP))).ServeHTTP,
		"static")

	// Initialize/register the internal app route.
	internalRoutes = adminSite.Group(
		fmt.Sprintf("/%s", InternalAppName),
		"internal",
		adminRequiredMiddleware,
		defaultDataMiddleware,
		hasPerms(PermissionViewAdminInternal),
	)

	// Register the internal app's models.
	for _, m := range adminSite_Internal_Models {
		if m != nil && m.model != nil {
			var dbItem db.PoolItem[*gorm.DB]
			var err error
			if dbItem, err = AdminSite_DB_POOL.Get(auth.DB_KEY); err != nil {
				dbItem, err = AdminSite_DB_POOL.Get(db.DEFAULT_DATABASE_KEY)
				if err != nil {
					panic(fmt.Sprintf("admin: could not get default database for %T: %s", m.model, err))
				}
			}

			dbItem.DB().AutoMigrate(m.model)
		}
		// Register the internal views/models.
		register_internal_model(m)
	}

	var internapAppChildren = internalApp.Children()

	// Compare length after registering internal models.
	if len(adminSite_Internal_Models) != len(internapAppChildren) {
		panic("admin: internal app models length mismatch")
	}

	// Register the internal app's models.
	for i, menuItem := range internapAppChildren {
		var rt = internalRoutes.Get(
			menuItem.URL.URLPart,
			menuItem.Data.(*internalModel).viewFunc,
			strings.ToLower(menuItem.Name),
		)

		if adminSite_Internal_Models[i].registrar != nil {
			rt.AddGroup(adminSite_Internal_Models[i].registrar)
		}
	}

}

// Register a model to the admin site.
//
// These models will then be available in the admin site.
func Register(m ...any) {
	for _, m := range m {
		// Create the model.
		var db, err = AdminSite_DB_POOL.ByModel(m)
		if err != nil {
			db = AdminSite_DB_POOL.GetDefaultDB()
		}
		model, err := models.NewModel(AdminSite_URL, m, db.DB())
		if err != nil {
			AdminSite_Logger.Error(err)
			continue
		}
		// Add the model to the list of models.
		adminSite_models = append(adminSite_models, model)
	}
}

// Register an extension to the admin site.
//
// # Extensions are separate templates that can be used to add extra functionality
//
// These templates are embedded into the admin site's base template.
func RegisterExtension(ext ...extensions.Extension) {
	var exts = make([]extensions.Extension, 0)
	for _, e := range ext {
		var ok bool = true
		for _, aE := range adminSite_Extensions {
			ok = ok && !(aE.Name() == e.Name())
		}
		if ok {
			exts = append(exts, e)
			continue
		}
		AdminSite_Logger.Warningf("admin: extension %s already registered\n", e.Name())
	}
	for _, e := range exts {
		register_extension(e)
	}
	adminSite_Extensions = append(adminSite_Extensions, exts...)
}

// Generate a list of URL patterns for the admin site.
//
// This function has te be called after all models have been registered.
//
// This function returns a router.Registrar which can be used to add the
// admin site to a router.
//
// In practise, you could add this to the router, and any registrar.
//
// Adding it to a registrar will however break the admin site's URLs.
func URLS() router.Registrar {

	var packages = orderedmap.New[string, router.Registrar]()
	for _, model := range adminSite_models {
		// Get the package path of the model.
		var pkg router.Registrar
		var ok bool
		// Create the package if it doesn't exist.
		if pkg, ok = packages.GetOK(model.AppName()); !ok {
			pkg = router.Group(fmt.Sprintf("/%s", model.URLS.AppName), strings.Trim(model.URLS.AppName, "/"))
		}

		// Create the model group.
		var mdlRoute = pkg.Get(model.URLS.GroupPart, adminHandler(model, listView), model.Name)

		mdlRoute.Get(model.URLS.Detail, adminHandler(model, detailView), "detail")
		mdlRoute.Post(model.URLS.Detail, adminHandler(model, detailView), "detail")

		mdlRoute.Get(model.URLS.Create, adminHandler(model, createView), "create")
		mdlRoute.Post(model.URLS.Create, adminHandler(model, createView), "create")

		mdlRoute.Get(model.URLS.Delete, adminHandler(model, deleteView), "delete")
		mdlRoute.Post(model.URLS.Delete, adminHandler(model, deleteView), "delete")

		packages.Set(model.AppName(), pkg)

	}

	for _, mdl := range adminSite_models {
		var pkg router.Registrar
		var ok bool
		pkg, ok = packages.GetOK(mdl.AppName())
		if !ok {
			continue
		}
		if !ok {
			panic("package not found")
		}
		adminSite.AddGroup(pkg)
		packages.Delete(mdl.AppName())
	}

	var extensionManager = AdminSite_ExtensionsManager
	if extensionManager == nil {
		extensionManager = response.TEMPLATE_MANAGER
	}

	var extensionViewOptions = &extensions.Options{
		BaseManager:      adminSiteManager,
		ExtensionManager: extensionManager,
		TemplateName:     "base",
		BlockName:        "content",
	}

	var adminSite_ExtensionRegistrar = adminSite.Group(
		EXTENSION_URL,
		"extensions",
		adminRequiredMiddleware, defaultDataMiddleware,
	)

	adminSite_ExtensionRegistrar.Use(
		hasPerms(PermissionViewAdminExtensions),
	)

	for _, ext := range adminSite_Extensions {
		adminSite_ExtensionRegistrar.Get(
			fmt.Sprintf("/%s", httputils.SimpleSlugify(ext.Name())),
			extensions.View(extensionViewOptions, ext),
			ext.Name(),
		)
	}

	return adminSite
}
