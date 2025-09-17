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
	"strconv"
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
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

type PageAppConfig struct {
	*apps.DBRequiredAppConfig
	routePrefix string
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

	CHOOSER_ROOT_PAGES_KEY = "pages.nodes.root"

	CHOOSER_PAGE_REVISIONS_KEY = "pages.page.revisions"
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

type PageChooserList struct {
	*chooser.WrappedModel[*PageNode]
	Models []*chooser.WrappedModel[*PageNode]
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

		Register(&PageDefinition{
			DisallowCreate: true,
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
					Ordering: 5,
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
		auditlogs.RegisterDefinition("pages:move", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:publish", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:unpublish", newPageLogDefinition())
		auditlogs.RegisterDefinition("pages:delete", auditlogs.SimpleDefinition())

		var chooserDefinitionAllNodes = chooser.ChooserDefinition[*PageNode]{
			Title: trans.S("Page Chooser"),
			Model: &PageNode{},
			MediaFn: func() media.Media {
				var m = media.NewMedia()
				m.AddCSS(media.CSS(django.Static("pages/admin/css/chooser.css")))
				return m
			},
			PreviewString: func(ctx context.Context, instance *PageNode) string {
				if !instance.IsPublished() {
					return html.EscapeString(instance.Title)
				}

				var _, site, err = SiteForRequest(ctx)
				if err != nil {
					return html.EscapeString(instance.Title)
				}

				if site.Root == nil || !strings.HasPrefix(instance.Path, site.Root.Path) {
					return html.EscapeString(instance.Title)
				}

				return fmt.Sprintf(
					"<a href=\"%s\" target=\"_blank\">%s</a>",
					URLPath(instance),
					html.EscapeString(instance.Title),
				)
			},
			ListPage: &chooser.ChooserListPage[*PageNode]{
				PerPage:  20,
				Template: "pages/chooser/list.tmpl",
				Fields: []string{
					"Title",
					"Slug",
					"ContentType",
					"CreatedAt",
					"UpdatedAt",
					// "ChooserChildren",
				},
				QuerySet: func(r *http.Request, model *PageNode) (*queries.QuerySet[*PageNode], error) {
					var parent = r.URL.Query().Get("parent")
					var depth int64 = 0
					var exprs = make([]expr.Expression, 0, 2)
					var parentObj *PageNode
					if parent != "" {
						var n, err = NewPageQuerySet().
							WithContext(r.Context()).
							Filter("PK", parent).
							Get()
						if err != nil {
							return nil, err
						}

						if r.URL.Query().Get("back") == "1" {
							if !n.Object.IsRoot() {
								n.Object, err = n.Object.Parent(r.Context())
								if err != nil {
									return nil, err
								}
							} else {
								parentObj = nil
								goto addDepthExpr
							}
						}

						exprs = append(
							exprs,
							expr.Q("PK", n.Object.PK),
						)

						depth = n.Object.Depth + 1
						parentObj = n.Object
					}

				addDepthExpr:
					var search = r.URL.Query().Get("search")
					if parentObj != nil {
						if search == "" {
							exprs = append(
								exprs,
								expr.And(
									expr.Q("Path__startswith", parentObj.Path),
									expr.Q("Depth", parentObj.Depth+1),
								),
							)
						} else {
							exprs = append(
								exprs,
								expr.And(
									expr.Q("Path__startswith", parentObj.Path),
									expr.Q("Depth__gte", parentObj.Depth),
								),
							)
						}
					} else {
						if search == "" {
							exprs = append(
								exprs,
								expr.Q("Depth", depth),
							)
						} else {
							exprs = append(
								exprs,
								expr.Q("Depth__gte", search),
							)
						}
					}

					var e expr.Expression
					if len(exprs) > 1 {
						e = expr.Or(exprs...)
					} else {
						e = exprs[0]
					}

					var qs = NewPageQuerySet().
						WithContext(r.Context()).
						Select("*").
						Filter(e).
						OrderBy("Depth").
						Annotate("IsParent", expr.Q("Depth", depth).Not(true)).
						Base()
					return qs, nil
				},
				NewList: func(req *http.Request, results []*PageNode, def *chooser.ChooserDefinition[*PageNode]) any {
					var parentNode *PageNode
					if len(results) > 0 && !attrs.IsZero(results[0].Annotations["IsParent"]) {
						parentNode = results[0]
						results = results[1:]
					}
					return &PageChooserList{
						WrappedModel: chooser.WrapModel(req.Context(), def, parentNode),
						Models:       chooser.WrapModels(req.Context(), def, results),
					}
				},
				SearchFields: []admin.SearchField{
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
		var chooserDefinitionRootNodesListPage = *chooserDefinitionAllNodes.ListPage
		chooserDefinitionRootNodes.ListPage = &chooserDefinitionRootNodesListPage
		chooserDefinitionRootNodes.ListPage.Template = ""
		chooserDefinitionRootNodes.ListPage.NewList = nil
		chooserDefinitionRootNodes.ListPage.QuerySet = func(r *http.Request, model *PageNode) (*queries.QuerySet[*PageNode], error) {
			return NewPageQuerySet().RootPages().Base(), nil
		}
		chooserDefinitionRootNodes.ListPage.Fields = []string{
			"Title",
			"Slug",
			"ContentType",
			"CreatedAt",
			"UpdatedAt",
		}
		chooser.Register(&chooserDefinitionAllNodes)
		chooser.Register(&chooserDefinitionRootNodes, CHOOSER_ROOT_PAGES_KEY)

		chooser.Register(
			&chooser.ChooserDefinition[*revisions.Revision]{
				Model: &revisions.Revision{},
				Title: trans.S("Page Revisions"),
				ListPage: &chooser.ChooserListPage[*revisions.Revision]{
					PerPage: 20,
					Fields:  []string{"ID", "Model", "CreatedAt"},
					QuerySet: func(r *http.Request, model *revisions.Revision) (*queries.QuerySet[*revisions.Revision], error) {
						var pks = r.URL.Query().Get("page-id")
						if pks == "" {
							return nil, fmt.Errorf("missing page-id parameter")
						}

						var pk, err = strconv.ParseInt(pks, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("invalid page-id parameter: %w", err)
						}

						p, err := NewPageQuerySet().
							WithContext(r.Context()).
							Filter("PK", pk).
							Get()
						if err != nil {
							return nil, fmt.Errorf("failed to get page for revision chooser queryset: %w", err)
						}

						return revisions.NewRevisionQuerySet().
							WithContext(r.Context()).
							ForObjects(p.Object).
							Base(), nil

					},
					Columns: map[string]list.ListColumn[*revisions.Revision]{
						"Model": list.FuncColumn(
							trans.S("Model"),
							func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) interface{} {
								var obj, err = row.AsObject(r.Context())
								if err != nil {
									logger.Errorf("failed to get revision object: %v", err)
									return ""
								}
								return attrs.ToString(obj)
							},
						),
					},
				},
			},
			CHOOSER_PAGE_REVISIONS_KEY,
		)

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

		goldcrest.Register(admin.RegisterHomePageDisplayPanelHook, 1, admin.RegisterHomePageDisplayPanelHookFunc(func(*http.Request, *admin.AdminApplication) []admin.DisplayPanel {
			return []admin.DisplayPanel{{
				IconName: "icon-file-earmark",
				Title: func(ctx context.Context, count int64) string {
					return trans.P(ctx, "Page", "Pages", count)
				},
				QuerySet: func(r *http.Request) *queries.QuerySet[attrs.Definer] {
					return queries.GetQuerySet[attrs.Definer](&PageNode{})
				},
				URL: func(r *http.Request) string {
					return django.Reverse("admin:pages")
				},
			}}
		}))

		tpl.RequestFuncs(func(r *http.Request) template.FuncMap {
			return template.FuncMap{
				"site_for_request": func() (*Site, error) {
					var _, site, err = SiteForRequest(r.Context())
					return site, err
				},
				"page_url": URLPath,
			}
		})

		pageApp.TemplateConfig = tpl.MergeConfig(
			&tpl.Config{
				AppName: "pages",
				FS:      templateFileSys,
				Matches: filesystem.MatchOr(
					filesystem.MatchAnd(
						filesystem.MatchPrefix("pages/"),
						filesystem.MatchOr(
							filesystem.MatchSuffix(".tmpl"),
						),
					),
				),
			},
			admin.AdminSite.TemplateConfig,
		)
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

		// List page revisions
		var revisionsRoute = pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/revisions", PageIDVariableName),
			pageHandler(listRevisionHandler), "revisions",
		)

		// Revision detail
		revisionsRoute.Any(
			"/<<revision_id>>",
			pageHandler(revisionDetailHandler), "detail",
		)

		// Compare revisions to current page state
		revisionsRoute.Get(
			"/<<revision_id>>/compare",
			pageHandler(revisionCompareHandler), "compare",
		)

		// Compare two revisions
		revisionsRoute.Get(
			"/<<revision_id>>/compare/<<other_revision_id>>",
			pageHandler(revisionCompareHandler), "compare_to",
		)

		// Choose page type
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/type", PageIDVariableName),
			pageHandler(choosePageTypeHandler), "type",
		)

		//	// Preview page
		//	pagesRoute.Get(
		//		fmt.Sprintf("/<<%s>>/preview", PageIDVariableName),
		//		pageHandler(previewPageHandler), "preview",
		//	)

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

		// Publish page
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/publish", PageIDVariableName),
			pageHandler(publishPageHandler), "publish",
		)
		pagesRoute.Post(
			fmt.Sprintf("/<<%s>>/publish", PageIDVariableName),
			pageHandler(publishPageHandler), "publish",
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

		// Move page
		pagesRoute.Get(
			fmt.Sprintf("/<<%s>>/move", PageIDVariableName),
			pageHandler(movePageHandler), "move",
		)
		pagesRoute.Post(
			fmt.Sprintf("/<<%s>>/move", PageIDVariableName),
			pageHandler(movePageHandler), "move",
		)

		var pagesAPI = pagesRoute.Get(
			"/api", nil, "api",
		)

		pagesAPI.Get(
			"/menu", mux.NewHandler(pageMenuHandler), "menu",
		)

		var djangoMux = django.Global.Mux
		djangoMux.Get(
			fmt.Sprintf("/__pages__/redirect/<<%s>>", PageIDVariableName),
			mux.NewHandler(redirectHandler),
			"pages_redirect",
		)

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

func (p *pageLogDefinition) GetLabel(r *http.Request, logEntry auditlogs.LogEntry) string {
	var cType *PageDefinition
	var unpublishChildren, _ = logEntry.Data()["unpublish_children"].(bool)
	var _, hasRevision = logEntry.Data()["revision_id"]
	var cTypeName, ok = logEntry.Data()["cType"].(string)
	if ok {
		cType = DefinitionForType(cTypeName)
	} else {
		cTypeName = fmt.Sprintf(
			"%s.%s",
			AdminPagesAppName,
			AdminPagesModelPath,
		)
		cType = DefinitionForType(cTypeName)
	}

	if cType != nil {
		cTypeName = cType.Label(r.Context())
	}

	switch logEntry.Type() {
	case "pages:add":
		return trans.T(r.Context(), "Page type %q added",
			cTypeName,
		)
	case "pages:add_child":
		return trans.T(r.Context(), "Child page type %q added",
			cTypeName,
		)
	case "pages:edit":
		if hasRevision {
			return trans.T(r.Context(), "Page type %q edited (new revision created)", cTypeName)
		}
		return trans.T(r.Context(), "Page type %q edited",
			cTypeName,
		)
	case "pages:publish":
		if hasRevision {
			return trans.T(r.Context(), "Page type %q published (new revision created)", cTypeName)
		}
		return trans.T(r.Context(), "Page type %q published",
			cTypeName,
		)
	case "pages:unpublish":
		if unpublishChildren {
			return trans.T(r.Context(), "Page type %q and it's children were unpublished",
				cTypeName,
			)
		}

		if hasRevision {
			return trans.T(r.Context(), "Page type %q unpublished (new revision created)", cTypeName)
		}

		return trans.T(r.Context(), "Page type %q unpublished",
			cTypeName,
		)
	}
	return trans.T(r.Context(), "Unknown page log entry type")
}

func (p *pageLogDefinition) GetActions(r *http.Request, l auditlogs.LogEntry) []auditlogs.LogEntryAction {
	var id = l.ObjectID()
	if id == nil {
		return nil
	}

	var actions = make([]auditlogs.LogEntryAction, 0)
	if permissions.HasPermission(r, "pages:edit") {
		actions = append(actions, &auditlogs.BaseAction{
			DisplayLabel: trans.T(r.Context(), "Edit Live Page"),
			ActionURL: fmt.Sprintf("%s?%s=%s",
				django.Reverse("admin:pages:edit", id),
				"next",
				r.URL.Path,
			),
		})

		var nextURL = r.URL.String()
		revId, ok := l.Data()["revision_id"]
		if ok {
			actions = append(actions, &auditlogs.BaseAction{
				DisplayLabel: trans.T(r.Context(), "View Revision"),
				ActionURL: addNextUrl(
					django.Reverse("admin:pages:revisions:detail", id, revId),
					nextURL,
				),
			})
		}
	}

	return actions
}

func (p *pageLogDefinition) FormatMessage(r *http.Request, logEntry auditlogs.LogEntry) any {
	var cTypeName, ok = logEntry.Data()["cType"].(string)
	var label, _ = logEntry.Data()["label"].(string)
	var cType *contenttypes.ContentTypeDefinition
	if ok {
		cType = contenttypes.DefinitionForType(cTypeName)
	} else {
		cTypeName = fmt.Sprintf(
			"%s.%s",
			AdminPagesAppName,
			AdminPagesModelPath,
		)
		cType = contenttypes.DefinitionForType(cTypeName)
	}

	if cType != nil {
		cTypeName = cType.Label(r.Context())
	}

	switch logEntry.Type() {
	case "pages:add":
		return trans.T(r.Context(), "Page %q with id %v and content type %q was added",
			label, logEntry.ObjectID(), cTypeName,
		)
	case "pages:add_child":
		return trans.T(r.Context(), "Child page %q with id %v and content type %q was added under parent with id %v",
			label, logEntry.Data()["page_id"], cTypeName, logEntry.ObjectID(),
		)
	case "pages:edit":
		return trans.T(r.Context(), "Page %q with id %v and content type %q was edited",
			label, logEntry.ObjectID(), cTypeName,
		)
	case "pages:publish":
		return trans.T(r.Context(), "Page %q with id %v and content type %q was published",
			label, logEntry.ObjectID(), cTypeName,
		)
	case "pages:unpublish":
		return trans.T(r.Context(), "Page %q with id %v and content type %q was unpublished",
			label, logEntry.ObjectID(), cTypeName,
		)
	}
	return trans.T(r.Context(), "Unknown page log entry type")
}
