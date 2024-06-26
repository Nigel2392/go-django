package pages

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/contrib/pages/models"
	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	dj_models "github.com/Nigel2392/django/models"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

type PageAppConfig struct {
	*apps.DBRequiredAppConfig
	backend     dj_models.Backend[models.Querier]
	routePrefix string
}

func SetPrefix(prefix string) {
	if pageApp == nil {
		panic("app is nil")
	}

	pageApp.routePrefix = prefix
}

func (p *PageAppConfig) QuerySet() models.DBQuerier {
	if p.DB == nil {
		panic("db is nil")
	}

	var (
		querySet    models.DBQuerier
		driver      = p.DB.Driver()
		backend, ok = models.GetBackend(driver)
	)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	var qs, err = backend.NewQuerySet(p.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize queryset for backend %T", backend))
	}

	querySet = &Querier{
		Querier: qs,
		Db:      p.DB,
	}

	return querySet
}

var (
	pageApp *PageAppConfig
)

var (
	//go:embed assets
	assetsFS embed.FS
)

func App() *PageAppConfig {
	if pageApp == nil {
		panic("app is nil")
	}

	return pageApp
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
			DisplayLabel: fields.T("Edit Live Page"),
			ActionURL: fmt.Sprintf("%s?%s=%s",
				django.Reverse("admin:pages:edit", id),
				"next",
				r.URL.Path,
			),
		},
	}
}

func NewAppConfig() *PageAppConfig {
	var assetFileSys, err = fs.Sub(assetsFS, "assets/static")
	if err != nil {
		panic(err)
	}

	templateFileSys, err := fs.Sub(assetsFS, "assets/templates")
	if err != nil {
		panic(err)
	}

	staticfiles.AddFS(
		assetFileSys, tpl.MatchAnd(
			tpl.MatchPrefix("pages/"),
			tpl.MatchOr(
				tpl.MatchSuffix(".css"),
				tpl.MatchSuffix(".js"),
			),
		),
	)

	tpl.Add(tpl.Config{
		AppName: "pages",
		FS:      templateFileSys,
		Matches: tpl.MatchAnd(
			tpl.MatchPrefix("pages/"),
			tpl.MatchOr(
				tpl.MatchSuffix(".tmpl"),
			),
		),
	})

	if pageApp != nil {
		return pageApp
	}

	var initPageApp = func(settings django.Settings, db *sql.DB) error {

		if err := CreateTable(db); err != nil {
			return err
		}

		var driver = db.Driver()
		var backend, ok = models.GetBackend(driver)
		if !ok {
			return fmt.Errorf("no backend configured for %T", driver)
		}

		pageApp.backend = backend

		var hookFn = func(site *admin.AdminApplication, items components.Items[menu.MenuItem]) {
			items.Append(&PagesMenuItem{
				BaseItem: menu.BaseItem{
					Label:    fields.S("Pages"),
					ItemName: "pages",
					Ordering: -1,
				},
			})
		}

		goldcrest.Register(admin.RegisterMenuItemHook, 0, hookFn)

		admin.RegisterApp(
			AdminPagesAppName,
			pageAdminAppOptions,
			pageAdminModelOptions,
		)

		auditlogs.RegisterDefinition("pages:edit", newPageLogDefinition())

		// contenttypes.Register(&contenttypes.ContentTypeDefinition{
		// ContentObject:  &models.PageNode{},
		// GetLabel:       fields.S("Page"),
		// GetDescription: fields.S("A page in a hierarchical page tree- structure."),
		// GetObject:      func() any { return &models.PageNode{} },
		// })

		Register(&PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				ContentObject:  &models.PageNode{},
				GetLabel:       fields.S("Page"),
				GetDescription: fields.S("A page in a hierarchical page tree- structure."),
				GetObject:      func() any { return &models.PageNode{} },
			},
			GetForID: func(ctx context.Context, ref models.PageNode, id int64) (Page, error) {
				return &ref, nil
			},
		})

		return nil
	}

	pageApp = &PageAppConfig{
		DBRequiredAppConfig: &apps.DBRequiredAppConfig{
			AppConfig: apps.NewAppConfig("pages"),
			Init:      initPageApp,
		},
	}
	pageApp.AppConfig.Ready = func() error {
		var pagesRoute = admin.AdminSite.Route.Get(
			"/pages", nil, "pages",
		)

		// List all pages
		// Delibirately after the add page route
		pagesRoute.Get(
			"/<<page_id>>", pageHandler(listPageHandler), "list",
		)

		// Choose page type
		pagesRoute.Get(
			"/<<page_id>>/type", pageHandler(choosePageTypeHandler), "type",
		)

		// Add new page type to a parent page
		pagesRoute.Any(
			"/<<page_id>>/<<app_label>>/<<model_name>>/add", pageHandler(addPageHandler), "add",
		)

		// Edit page
		pagesRoute.Any(
			"/<<page_id>>/edit", pageHandler(editPageHandler), "edit",
		)

		// Unpublish page
		pagesRoute.Get(
			"/<<page_id>>/unpublish", pageHandler(unpublishPageHandler), "unpublish",
		)
		pagesRoute.Post(
			"/<<page_id>>/unpublish", pageHandler(unpublishPageHandler), "unpublish",
		)

		//// deleteURL for the pages admin site.
		//pagesRoute.Get(
		//	"<<page_id>>/delete", pageHandler(deletePageHandler), "delete",
		//)

		var pagesAPI = pagesRoute.Get(
			"/api", nil, "api",
		)

		pagesAPI.Get(
			"/menu", mux.NewHandler(pageMenuHandler), "menu",
		)

		return nil
	}
	QuerySet = pageApp.QuerySet

	return pageApp
}
