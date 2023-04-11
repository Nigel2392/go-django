package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/orderedmap"
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
func (as *AdminSite) Init() {
	var asDB = as.DB()
	asDB.Register(
		&Log{},
		&LoggableUser{},
	)
	var err = asDB.AutoMigrate()
	if err != nil {
		panic(err)
	}

	as.templateManager().TEMPLATEFS = templateFS

	// Register the default admin site permissions.
	PermissionViewAdminSite.Save(asDB.DB())
	PermissionViewAdminInternal.Save(asDB.DB())
	PermissionViewAdminExtensions.Save(asDB.DB())

	// Register internal apps.
	// (Logs, etc...)
	var internalApp = as.internal_menu_items()

	// Register default URLs.
	// (Login, logout, etc...)
	as.registrar = router.Group(as.URL, "admin")
	as.registrar.Get("/unauthorized", as.wrapRoute(unauthorizedView), "unauthorized").Use(defaultDataMiddleware(as))
	as.registrar.Get("/login", as.wrapRoute(loginView), "login")
	as.registrar.Get("/logout", as.wrapRoute(logoutView), "logout")

	// Register the default admin site view.
	var rt = as.registrar.(*router.Route)
	rt.Method = router.GET
	rt.HandlerFunc = func(r *request.Request) {
		if !hasAdminPerms(as, r) {
			Unauthorized(as, r, "You do not have permission to access this page.")
			return
		}
		defaultDataFunc(as, r)
		indexView(as, r)
	}

	// Register the default admin site static file handler.
	var staticFileSysHTTP = http.FS(staticFileSystem)
	as.registrar.Get("/static/<<any>>",
		router.FromHTTPHandler(
			http.StripPrefix(fmt.Sprintf("%s/static/", as.URL),
				http.FileServer(staticFileSysHTTP))).ServeHTTP,
		"static")

	// Initialize/register the internal app route.
	as.internalRegistrar = as.registrar.Group(
		fmt.Sprintf("/%s", as.InternalAppName),
		"internal",
		adminRequiredMiddleware(as),
		defaultDataMiddleware(as),
		hasPerms(as, PermissionViewAdminInternal),
	)

	// Register the internal app's models.
	for _, m := range as.internal_models {
		if m != nil && m.model != nil {
			var dbItem db.PoolItem[*gorm.DB]
			var err error
			if dbItem, err = as.DBPool.Get(auth.DB_KEY); err != nil {
				dbItem, err = as.DBPool.Get(db.DEFAULT_DATABASE_KEY)
				if err != nil {
					panic(fmt.Sprintf("admin: could not get default database for %T: %s", m.model, err))
				}
			}

			dbItem.DB().AutoMigrate(m.model)
		}
		// Register the internal views/models.
		as.register_internal_model(m)
	}

	var internapAppChildren = internalApp.Children()

	// Compare length after registering internal models.
	if len(as.internal_models) != len(internapAppChildren) {
		panic("admin: internal app models length mismatch")
	}

	// Register the internal app's models.
	for i, menuItem := range internapAppChildren {
		var rt = as.internalRegistrar.Get(
			menuItem.URL.URLPart,
			menuItem.Data.(*internalModel).viewFunc,
			strings.ToLower(menuItem.Name),
		)

		if as.internal_models[i].registrar != nil {
			rt.AddGroup(as.internal_models[i].registrar)
		}
	}

}

// Wrap a route handler function.
func (as *AdminSite) wrapRoute(f func(as *AdminSite, r *request.Request)) func(r *request.Request) {
	return func(r *request.Request) {
		f(as, r)
	}
}

// Register a model to the admin site.
//
// These models will then be available in the admin site.
func (as *AdminSite) Register(m ...any) {
	for _, m := range m {
		// Create the model.
		var db, err = as.DBPool.ByModel(m)
		if err != nil {
			db = as.DB()
		}
		model, err := models.NewModel(as.URL, m, db.DB())
		if err != nil {
			as.Logger.Error(err)
			continue
		}
		// Add the model to the list of models.
		as.models = append(as.models, model)
	}
}

// Register an extension to the admin site.
//
// # Extensions are separate templates that can be used to add extra functionality
//
// These templates are embedded into the admin site's base template.
func (as *AdminSite) RegisterExtension(ext ...extensions.Extension) {
	var exts = make([]extensions.Extension, 0)
	for _, e := range ext {
		var ok bool = true
		for _, aE := range as.extensions {
			ok = ok && !(aE.Name() == e.Name())
		}
		if ok {
			exts = append(exts, e)
			continue
		}
		as.Logger.Warningf("admin: extension %s already registered\n", e.Name())
	}
	for _, e := range exts {
		as.register_extension(e)
	}
	as.extensions = append(as.extensions, exts...)
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
func (as *AdminSite) URLS() router.Registrar {

	var packages = orderedmap.New[string, router.Registrar]()
	for _, model := range as.models {
		// Get the package path of the model.
		var pkg router.Registrar
		var ok bool
		// Create the package if it doesn't exist.
		if pkg, ok = packages.GetOK(model.AppName()); !ok {
			pkg = router.Group(fmt.Sprintf("/%s", model.URLS.AppName), strings.Trim(model.URLS.AppName, "/"))
		}

		// Create the model group.
		var mdlRoute = pkg.Get(model.URLS.GroupPart, adminHandler(as, model, listView), model.Name)

		mdlRoute.Get(model.URLS.Detail, adminHandler(as, model, detailView), "detail")
		mdlRoute.Post(model.URLS.Detail, adminHandler(as, model, detailView), "detail")

		mdlRoute.Get(model.URLS.Create, adminHandler(as, model, createView), "create")
		mdlRoute.Post(model.URLS.Create, adminHandler(as, model, createView), "create")

		mdlRoute.Get(model.URLS.Delete, adminHandler(as, model, deleteView), "delete")
		mdlRoute.Post(model.URLS.Delete, adminHandler(as, model, deleteView), "delete")

		packages.Set(model.AppName(), pkg)
	}

	for _, mdl := range as.models {
		var pkg router.Registrar
		var ok bool
		pkg, ok = packages.GetOK(mdl.AppName())
		if !ok {
			continue
		}
		if !ok {
			panic("package not found")
		}
		as.registrar.AddGroup(pkg)
		packages.Delete(mdl.AppName())
	}

	var extensionManager = as.ExtensionsManager
	if extensionManager == nil {
		extensionManager = response.TEMPLATE_MANAGER
	}

	var extensionViewOptions = &extensions.Options{
		BaseManager:      as.templateMgr,
		ExtensionManager: extensionManager,
		TemplateName:     "base",
		BlockName:        "content",
	}

	var adminSite_ExtensionRegistrar = as.registrar.Group(
		as.ExtensionURL,
		"extensions",
		adminRequiredMiddleware(as), defaultDataMiddleware(as),
	)

	adminSite_ExtensionRegistrar.Use(
		hasPerms(as, PermissionViewAdminExtensions),
	)

	for _, ext := range as.extensions {
		adminSite_ExtensionRegistrar.Get(
			fmt.Sprintf("/%s", httputils.SimpleSlugify(ext.Name())),
			extensions.View(extensionViewOptions, ext),
			ext.Name(),
		)
	}

	return as.registrar
}
