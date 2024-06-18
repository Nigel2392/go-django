package pages

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/mux"
)

type PageAppConfig struct {
	*apps.DBRequiredAppConfig
	backend models.Backend
}

func (p *PageAppConfig) QuerySet() models.DBQuerier {
	if p.DB == nil {
		panic("db is nil")
	}

	var querySet models.DBQuerier
	var driver = p.DB.Driver()
	var backend, ok = models.GetBackend(driver)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	var qs, err = backend.NewQuerySet(p.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize queryset for backend %T", backend))
	}

	if querySet == nil {
		querySet = &Querier{
			Querier: qs,
			Db:      p.DB,
		}
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

func NewAppConfig() *PageAppConfig {
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

		var fileSys, err = fs.Sub(assetsFS, "assets/static")
		if err != nil {
			return err
		}

		staticfiles.AddFS(
			fileSys, tpl.MatchAnd(
				tpl.MatchPrefix("pages/"),
				tpl.MatchOr(
					tpl.MatchSuffix(".css"),
					tpl.MatchSuffix(".js"),
				),
			),
		)

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
