package pages

import (
	"context"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"path"
	"reflect"
	"strings"
	"text/template"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
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

	var routePrefixSet = false

	pageApp.Deps = []string{
		"admin",
		"settings",
	}

	pageApp.ModelObjects = []attrs.Definer{
		&Site{},
		&PageNode{},
	}

	pageApp.Cmd = []command.Command{
		commandFixTree,
	}

	pageApp.Init = func(settings django.Settings, db drivers.Database) error {

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&PageNode{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
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

		pageApp.routePrefix = strings.TrimSuffix(pageApp.routePrefix, "/")
		var handler = http.StripPrefix(pageApp.routePrefix, Serve(
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		))
		django.Global.Mux.Any(
			fmt.Sprintf("%s/", pageApp.routePrefix),
			handler, "page",
		)
		django.Global.Mux.Any(
			fmt.Sprintf("%s/*", pageApp.routePrefix),
			&pageRouteResolver{handler}, "pages",
		)

		Register(&PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				ContentObject:  &PageNode{},
				GetLabel:       trans.S("Page"),
				GetDescription: trans.S("A page in a hierarchical page tree- structure."),
				GetInstanceLabel: func(a any) string {
					var page, ok = a.(*PageNode)
					if !ok {
						assert.Fail("object %T is not a PageNode", a)
					}
					return fmt.Sprintf(
						"%s (%d)",
						page.Title, page.ID(),
					)
				},
				GetObject: func() any { return &PageNode{} },
				GetInstance: func(ctx context.Context, identifier any) (interface{}, error) {
					var node, err = NewPageQuerySet().WithContext(ctx).Filter("PK", identifier).Get()
					if err != nil {
						return nil, err
					}
					return node.Object, nil
				},
				GetInstances: func(ctx context.Context, amount, offset uint) ([]interface{}, error) {
					var nodes, err = NewPageQuerySet().WithContext(ctx).Offset(int(offset)).Limit(int(amount)).AllNodes()
					return attrutils.InterfaceList(nodes), err
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

		reports.RegisterMenuItem(func(r *http.Request) []menu.MenuItem {
			return []menu.MenuItem{
				&menu.Item{
					BaseItem: menu.BaseItem{
						Ordering: 1000,
						Label:    trans.T(r.Context(), "Outdated Pages"),
					},
					Link: func() string {
						return django.Reverse("admin:pages:outdated")
					},
				},
			}
		})

		admin.RegisterGlobalMenuItem(admin.RegisterMenuItemHookFunc(func(r *http.Request, site *admin.AdminApplication, items components.Items[menu.MenuItem]) {
			items.Append(&PagesMenuItem{
				BaseItem: menu.BaseItem{
					Label:    trans.T(r.Context(), "Pages"),
					ItemName: "pages",
					Ordering: -1000,
					Hidden:   !permissions.HasPermission(r, "pages:list"),
				},
			})
		}))

		admin.RegisterHomePageComponent(admin.RegisterHomePageComponentHookFunc(func(r *http.Request, a *admin.AdminApplication) admin.AdminPageComponent {
			return &PagesAdminHomeComponent{
				ListFields:       []string{"Title", "ContentType", "Live", "UrlPath", "UpdatedAt", "Children"},
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

		var chooserDefinitionAllNodes = chooser.ChooserDefinition[*PageNode]{
			Title: trans.S("User Chooser"),
			Model: &PageNode{},
			PreviewString: func(ctx context.Context, instance *PageNode) string {
				if !instance.IsPublished() {
					return html.EscapeString(instance.Title)
				}

				return fmt.Sprintf(
					"<a href=\"%s\" target=\"_blank\">%s</a>",
					URLPath(instance),
					html.EscapeString(instance.Title),
				)
			},
			ListPage: &chooser.ChooserListPage[*PageNode]{
				PerPage: 20,
				SearchFields: []chooser.SearchField[*PageNode]{
					{
						Name:   "Title",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "Slug",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "UrlPath",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "ContentType",
						Lookup: expr.LOOKUP_ICONTANS,
					},
				},
			},
		}
		var chooserDefinitionRootNodes = chooserDefinitionAllNodes
		chooserDefinitionRootNodes.ListPage.Fields = []string{
			"Title",
			"Slug",
			"ContentType",
			"CreatedAt",
			"UpdatedAt",
		}
		chooserDefinitionRootNodes.ListPage.QuerySet = func(r *http.Request, model *PageNode) *queries.QuerySet[*PageNode] {
			return NewPageQuerySet().RootPages().Base()
		}

		chooser.Register(&chooserDefinitionAllNodes)
		chooser.Register(&chooserDefinitionRootNodes, "pages.nodes.root")

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

		tpl.Funcs(template.FuncMap{
			"PageURL": URLPath,
		})

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

		// Add outdated pages handler
		pagesRoute.Get(
			"/outdated", pageAdminAppHandler(outdatedPagesHandler), "outdated",
		)

		// Choose new root page type
		pagesRoute.Any(
			"/type", pageAdminAppHandler(chooseRootPageTypeHandler), "root_type",
		)

		// Add new root page
		pagesRoute.Any(
			"/<<app_name>>/<<model_name>>/add", pageAdminAppHandler(addRootPageHandler), "root_add",
		)

		// List all pages
		// Deliberately after the add page route
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
			fmt.Sprintf("/<<%s>>/<<app_name>>/<<model_name>>/add", PageIDVariableName),
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

var direct_page_methods = [...]string{
	"ID",
	"Reference",
}

func (p *PageAppConfig) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = p.AppConfig.Check(ctx, settings)
	for _, def := range pageRegistryObject.ListDefinitions() {
		var rTyp = reflect.TypeOf(def.ContentObject)
		for _, methodName := range direct_page_methods {
			if isPromoted(rTyp, methodName) {
				messages = append(messages, checks.Critical(
					"pages.method_promoted",
					fmt.Sprintf("promoted method %q is not allowed", methodName),
					def.ContentObject,
					"Please directly define the method on your page type.",
				))
			}
		}
	}
	return messages
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
			DisplayLabel: trans.T(r.Context(), "Edit Live Page"),
			ActionURL: fmt.Sprintf("%s?%s=%s",
				django.Reverse("admin:pages:edit", id),
				"next",
				r.URL.Path,
			),
		},
	}
}
