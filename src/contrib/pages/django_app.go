package pages

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
)

type PageAppConfig struct {
	*apps.DBRequiredAppConfig
	routePrefix        string
	useRedirectHandler bool
}

// SetRoutePrefix sets the route prefix for the pages app.
//
// Pages app will be served at the route prefix.
func SetRoutePrefix(prefix string) {
	if pageApp == nil {
		panic("app is nil")
	}

	pageApp.routePrefix = prefix
}

// SetUseRedirectHandler sets whether to use redirect handler for pages app.
//
// If set to true, a redirect handler will be registered at /__pages__/redirect/<page_id>
//
// This is useful when you want to redirect to a page when you only have the page id,
// using this handler will skip the need for a database query to get the page url.
func SetUseRedirectHandler(use bool) {
	if pageApp == nil {
		panic("app is nil")
	}

	pageApp.useRedirectHandler = use
}

// Returns the live URL path for the given page.
//
// This is the URL path that the page will be served at.
func URLPath(page Page) string {
	var ref *PageNode
	switch v := page.(type) {
	case *PageNode:
		ref = v
	default:
		ref = page.Reference()
	}
	return path.Join(pageApp.routePrefix, ref.UrlPath)
}

const (
	// The variable name generally used to refer to the page id in the URL.
	//
	// This is used in the URL patterns for the pages app, as well as javascript code and queries in URLs.
	PageIDVariableName = "page_id"
)

var (
	pageApp *PageAppConfig = &PageAppConfig{
		DBRequiredAppConfig: &apps.DBRequiredAppConfig{
			AppConfig: apps.NewAppConfig("pages"),
		},
	}
)

var (
	//go:embed assets
	assetsFS embed.FS

	//go:embed migrations/*
	migrationFS embed.FS
)

// Returns the pages app object itself
func App() *PageAppConfig {
	if pageApp == nil {
		panic("app is nil")
	}

	return pageApp
}

// NewAppConfig returns a new pages app config.
//
// This is used to initialize the pages app, set up routes, and register the admin application.
func NewAppConfig() django.AppConfig {

	attrs.RegisterModel(&PageNode{})

	var routePrefixSet = false

	pageApp.ModelObjects = []attrs.Definer{
		&PageNode{},
	}

	pageApp.Init = func(settings django.Settings, db drivers.Database) error {

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&PageNode{})
			if err := schemaEditor.CreateTable(table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		// contenttypes.Register(&contenttypes.ContentTypeDefinition{
		// ContentObject:  &PageNode{},
		// GetLabel:       trans.S("Page"),
		// GetDescription: trans.S("A page in a hierarchical page tree- structure."),
		// GetObject:      func() any { return &PageNode{} },
		// })

		if pageApp.routePrefix == "" {
			pageApp.routePrefix = "/"
		} else {
			routePrefixSet = true
		}

		if !strings.HasPrefix(pageApp.routePrefix, "/") {
			pageApp.routePrefix = "/" + pageApp.routePrefix
		}

		if strings.HasSuffix(pageApp.routePrefix, "/") && len(pageApp.routePrefix) > 1 {
			pageApp.routePrefix = pageApp.routePrefix[:len(pageApp.routePrefix)-1]
		}

		var handler = http.StripPrefix(pageApp.routePrefix, Serve(
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		))
		django.Global.Mux.Any(
			fmt.Sprintf("%s/", pageApp.routePrefix),
			handler, "pages_home",
		)
		django.Global.Mux.Any(
			fmt.Sprintf("%s/*", pageApp.routePrefix),
			handler, "pages",
		)

		Register(&PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				ContentObject:  &PageNode{},
				GetLabel:       trans.S("Page"),
				GetDescription: trans.S("A page in a hierarchical page tree- structure."),
				GetObject:      func() any { return &PageNode{} },
				GetInstance: func(identifier any) (interface{}, error) {
					var id int64
					switch v := identifier.(type) {
					case int:
						id = int64(v)
					case int64:
						id = v
					case string:
						var err error
						id, err = strconv.ParseInt(v, 10, 64)
						if err != nil {
							return nil, err
						}
					default:
						return nil, errs.ErrInvalidType
					}
					var node, err = NewPageQuerySet().GetNodeByID(id)
					if err != nil {
						return nil, err
					}
					return &node, nil
				},
				GetInstances: func(amount, offset uint) ([]interface{}, error) {
					var nodes, err = NewPageQuerySet().Offset(int(offset)).Limit(int(amount)).AllNodes()
					var items = make([]interface{}, 0)
					for _, n := range nodes {
						n := n
						items = append(items, &n)
					}
					return items, err
				},
			},
			GetForID: func(ctx context.Context, ref *PageNode, id int64) (Page, error) {
				return ref, nil
			},
		})

		// Return if the admin app is not installed
		// This is used to prevent the pages app's admin setup from running
		if !django.AppInstalled("admin") {
			return nil
		}

		admin.RegisterGlobalMenuItem(admin.RegisterMenuItemHookFunc(func(r *http.Request, site *admin.AdminApplication, items components.Items[menu.MenuItem]) {
			items.Append(&PagesMenuItem{
				BaseItem: menu.BaseItem{
					Label:    trans.S("Pages"),
					ItemName: "pages",
					Ordering: -1000,
					Hidden:   !permissions.HasPermission(r, "pages:list"),
				},
			})
		}))

		admin.RegisterHomePageComponent(admin.RegisterHomePageComponentHookFunc(func(r *http.Request, a *admin.AdminApplication) admin.AdminPageComponent {
			return &PagesAdminHomeComponent{
				AdminApplication: a,
				Request:          r,
			}
		}))

		admin.RegisterApp(
			AdminPagesAppName,
			pageAdminAppOptions,
			pageAdminModelOptions,
		)

		auditlogs.RegisterDefinition("pages:add", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:add_child", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:edit", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:publish", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:unpublish", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:delete", auditlogs.SimpleDefinition())
		return nil
	}

	// Only register staticfiles & templates if the admin app is installed
	if django.AppInstalled("admin") {
		var assetFileSys, err = fs.Sub(assetsFS, "assets/static")
		if err != nil {
			panic(err)
		}

		templateFileSys, err := fs.Sub(assetsFS, "assets/templates")
		if err != nil {
			panic(err)
		}

		staticfiles.AddFS(
			assetFileSys, filesystem.MatchAnd(
				filesystem.MatchPrefix("pages/"),
				filesystem.MatchOr(
					filesystem.MatchSuffix(".css"),
					filesystem.MatchSuffix(".js"),
				),
			),
		)

		pageApp.TemplateConfig = &tpl.Config{
			AppName: "pages",
			FS: filesystem.NewMultiFS(
				filesystem.NewMatchFS(
					templateFileSys,
					filesystem.MatchOr(
						filesystem.MatchAnd(
							filesystem.MatchPrefix("pages/"),
							filesystem.MatchOr(
								filesystem.MatchSuffix(".tmpl"),
							),
						),
					),
				),
				filesystem.NewMatchFS(
					admin.AdminSite.TemplateConfig.FS,
					admin.AdminSite.TemplateConfig.Matches,
				),
			),
			Bases: admin.AdminSite.TemplateConfig.Bases,
			Funcs: admin.AdminSite.TemplateConfig.Funcs,
		}
	}

	//	pageApp.Deps = []string{
	//		"revisions",
	//	}

	pageApp.AppConfig.Ready = func() error {

		// If the admin app is not installed we don't need to register the pages app's
		// routes and handlers
		if !django.AppInstalled("admin") {
			return nil
		}

		if !routePrefixSet {
			django.Global.Log.Fatal(1, "Route prefix was not set before calling django.App.Initialize().")
		}

		var pagesRoute = admin.AdminSite.Route.Get(
			"/pages", pageAdminAppHandler(listRootPageHandler), "pages",
		)

		// Choose new root page type
		pagesRoute.Any(
			"/type", pageAdminAppHandler(chooseRootPageTypeHandler), "root_type",
		)

		// Add new root page
		pagesRoute.Any(
			"/<<app_label>>/<<model_name>>/add", pageAdminAppHandler(addRootPageHandler), "root_add",
		)

		// List all pages
		// Delibirately after the add page route
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>", PageIDVariableName),
			pageHandler(listPageHandler), "list",
		)

		// Choose page type
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/type", PageIDVariableName),
			pageHandler(choosePageTypeHandler), "type",
		)

		// Delete page
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/delete", PageIDVariableName),
			pageHandler(deletePageHandler), "delete",
		)
		pagesRoute.Post(
			fmt.Sprintf("/<<%s>>/delete", PageIDVariableName),
			pageHandler(deletePageHandler), "delete",
		)

		// Add new page type to a parent page
		pagesRoute.Any(
			fmt.Sprintf("/<<%s>>/<<app_label>>/<<model_name>>/add", PageIDVariableName),
			pageHandler(addPageHandler), "add",
		)

		// Edit page
		pagesRoute.Any(
			fmt.Sprintf("/<<%s>>/edit", PageIDVariableName),
			pageHandler(editPageHandler), "edit",
		)

		// Unpublish page
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/unpublish", PageIDVariableName),
			pageHandler(unpublishPageHandler), "unpublish",
		)
		pagesRoute.Post(
			fmt.Sprintf("/<<%s>>/unpublish", PageIDVariableName),
			pageHandler(unpublishPageHandler), "unpublish",
		)

		var pagesAPI = pagesRoute.Get(
			"/api", nil, "api",
		)

		pagesAPI.Get(
			"/menu", mux.NewHandler(pageMenuHandler), "menu",
		)

		if pageApp.useRedirectHandler {
			var djangoMux = django.Global.Mux
			djangoMux.Get(
				fmt.Sprintf("/__pages__/redirect/<<%s>>", PageIDVariableName),
				mux.NewHandler(redirectHandler),
				"pages_redirect",
			)
		}

		return nil
	}

	return &migrator.MigratorAppConfig{
		AppConfig: pageApp,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/pages",
		),
	}
}

type pageLogDefinition struct {
	auditlogs.Definition
}

func newPageLogDefinition() *pageLogDefinition {
	return &pageLogDefinition{
		Definition: auditlogs.SimpleDefinition(),
	}
}

func (p *pageLogDefinition) GetActions(r *http.Request, l auditlogs.LogEntry) []auditlogs.LogEntryAction {
	var id = l.ObjectID()
	if id == nil {
		return nil
	}
	return []auditlogs.LogEntryAction{
		&auditlogs.BaseAction{
			DisplayLabel: trans.T("Edit Live Page"),
			ActionURL: fmt.Sprintf("%s?%s=%s",
				django.Reverse("admin:pages:edit", id),
				"next",
				r.URL.Path,
			),
		},
	}
}
